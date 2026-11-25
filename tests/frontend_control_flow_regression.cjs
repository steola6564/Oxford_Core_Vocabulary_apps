// Regressions for concrete review findings, using actual functions with controlled
// DOM/network substitutes. This does not claim real browser or touch verification.
const fs = require('node:fs');
const vm = require('node:vm');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');
const pathTools = require('node:path');
const projectRoot = pathTools.resolve(__dirname, '..');
const path=pathTools.join(projectRoot, 'web', 'index.html');
const source=fs.readFileSync(path,'utf8'), lines=source.split('\n'), checks=[];
function fn(name){const line=lines.find(x=>x.startsWith('function '+name+'(')||x.startsWith('async function '+name+'('));assert.ok(line,name);return line;}
function ctx(extra={}){return vm.createContext({console,structuredClone,Promise,Number,String,Math,...extra});}
function fresh(){return {free_text:'',corpus_ids:[],pos_codes:[],level_codes:[],section_codes:[],collection_ids:[],initial_letter:0,history_result:[],confidence_range:null,attempt_time_range:null,requested_count:20};}
async function check(name,run){try{await run();checks.push({name,status:'passed'});}catch(error){checks.push({name,status:'failed',error:String(error),stack:error.stack});}}
(async()=>{
 await check('Late Next response does not render over a newer view',async()=>{
  let resolve;const response=new Promise(r=>resolve=r),renders=[];
  const S={view:'word_recall',session:{session_id:'00000001',total:3},question:{position:1},draft:{pending:false},quizLoading:false};
  const c=ctx({S,sessionOperation:0,routeVersion:0,api:()=>response,log:()=>{},targetQuestion:()=>null,renderQuiz:()=>renders.push(S.view),loadQuestion:async()=>{},isQuiz:()=>['word_recall','cloze_test','reading_review'].includes(S.view)});
  vm.runInContext(fn('moveQuestion'),c);const p=vm.runInContext('moveQuestion(2)',c);
  S.view='hero';c.routeVersion++;resolve({session_id:'00000001',total:3,current_position:2});await p;
  assert.deepEqual(renders,[]);assert.equal(S.view,'hero');
 });
 await check('Practise Collection preserves newly selected collection after dialog close',async()=>{
  const stale='00000001',desired='00000002',S={view:'submode_selection',filters:{setup:fresh()},collections:[{collection_id:stale},{collection_id:desired}],collectionPanel:{},setup:{}};
  const dialog={open:true,close(){this.open=false;}},panel={};
  const c=ctx({S,$:s=>s==='#collectionDialog'?dialog:s==='[data-filter-panel="setup"]'?panel:s==='[data-query="setup"]'?{value:'old query'}:null,$$:s=>s==='[data-filter="collection_ids"]'?[{value:stale,checked:true}]:[],log:()=>{},renderSetup:()=>{},loadCandidates:async()=>{},freshFilter:fresh,route:()=>{}});
  const branch=lines.find(x=>x.includes('if(button.dataset.practiseCollection)'));
  vm.runInContext(fn('readFilters')+'\n'+fn('closeCollections')+'\nfunction invoke(button){'+branch+'}',c);
  vm.runInContext("invoke({dataset:{practiseCollection:'00000002'}})",c);
  assert.deepEqual(Array.from(S.filters.setup.collection_ids),[desired]);assert.equal(S.filters.setup.free_text,'');
 });
 await check('Deleted selected Collection is not reintroduced by stale underlying checkboxes',async()=>{
  const stale='00000001',S={view:'submode_selection',filters:{setup:fresh()},collections:[{collection_id:stale}],collectionPanel:{deleteConfirm:stale,busy:false}};
  S.filters.setup.collection_ids=[stale];const dialog={open:true,close(){this.open=false;}},panel={};
  const c=ctx({S,$:s=>s==='#collectionDialog'?dialog:s==='[data-filter-panel="setup"]'?panel:s==='[data-query="setup"]'?{value:''}:null,$$:s=>s==='[data-filter="collection_ids"]'?[{value:stale,checked:true}]:[],log:()=>{},renderSetup:()=>{},loadCandidates:async()=>{},api:async()=>({deleted:true}),action:()=>({}),refreshCollections:async()=>{S.collections=[];},toast:()=>{},renderCollections:()=>{}});
  vm.runInContext(fn('readFilters')+'\n'+fn('closeCollections')+'\n'+fn('deleteCollection'),c);
  await vm.runInContext('deleteCollection()',c);vm.runInContext('closeCollections()',c);
  assert.deepEqual(S.filters.setup.collection_ids,[]);
 });
 await check('NEW validates its effective filter without hidden Review conditions',async()=>{
  let calls=0;const S={view:'submode_selection',setup:{source_mode:'new'},filters:{setup:{...fresh(),confidence_range:[80,20]}}};
  const c=ctx({S,candidateVersion:0,paintCandidates:()=>{},ensureReady:async()=>{},api:async()=>{calls++;return{total:6100};},$:()=>({innerHTML:''}),noticeError:s=>s});
  vm.runInContext(fn('setupFilter')+'\n'+fn('setupPayload')+'\n'+fn('validateFilter')+'\n'+fn('loadCandidates'),c);
  await vm.runInContext('loadCandidates()',c);assert.equal(calls,1);assert.equal(S.setup.error,'');assert.equal(S.setup.total,6100);
 });
 for(const fail of [false,true])await check(fail?'Late submit failure does not paint error on another quiz':'Late submit success does not replace a newer draft at the same Session position',async()=>{
  let resolve,reject;const response=new Promise((r,j)=>{resolve=r;reject=j;}),renders=[];
  const oldDraft={selected_corpus_id:2,answer_text:'',recalled:null,confidence:20,saved:null,pending:false,started:0},oldQuestion={position:1,quiz_type:'word_recall'};
  const S={view:'word_recall',session:{session_id:'00000001',quiz_type:'word_recall'},question:oldQuestion,draft:oldDraft,quizError:''};
  const c=ctx({S,performance:{now:()=>100},api:()=>response,action:()=>({}),targetQuestion:()=>null,renderQuiz:()=>renders.push(S.draft),flushActions:async()=>{},toast:()=>{},isQuiz:()=>true,$:()=>null});
  vm.runInContext(fn('submitAttempt'),c);const p=vm.runInContext('submitAttempt(false)',c);
  const newDraft={saved:{attempt_id:'00000002'},selected_corpus_id:3},newSession={session_id:fail?'00000002':'00000001',quiz_type:'word_recall'};
  S.draft=newDraft;S.session=newSession;S.question={position:1,quiz_type:'word_recall'};
  if(fail){const e=new Error('old server error');e.status=500;reject(e);}else resolve({session:{session_id:'00000001'},attempt:{attempt_id:'00000001',selected_corpus_id:2},feedback:{}});
  await p;assert.equal(S.draft,newDraft);assert.equal(S.session,newSession);assert.equal(S.quizError,'');assert.equal(renders.length,1);
 });
 const result={scope:'Extracted current JS functions, controlled promises and DOM substitutes; not browser or touch verification',source:'web/index.html',source_sha256:crypto.createHash('sha256').update(source).digest('hex'),checks,passed:checks.filter(x=>x.status==='passed').length,failed:checks.filter(x=>x.status==='failed').length};
 console.log(JSON.stringify(result,null,2));
 fs.mkdirSync(pathTools.join(projectRoot, 'test-results'), {recursive:true});
 fs.writeFileSync(pathTools.join(projectRoot, 'test-results', 'frontend-control-flow-verification.json'),JSON.stringify(result,null,2)+'\n');
 process.exitCode=result.failed?1:0;
})().catch(e=>{console.error(e);process.exitCode=1;});
