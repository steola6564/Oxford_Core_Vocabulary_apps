# API contract — Oxford Core Vocabulary v1.0

JSON uses snake_case. Generated DB IDs are 8-digit strings, corpus IDs are integers. Session positions are **1-based**. Errors use HTTP 400/404/409/500 and `{"error":"message","code":"optional_machine_code"}`. All successful list fields are arrays, not null.

## Corpus and filters
`GET /api/meta` → `{count:6100,sha256, pos:[{code,label}], levels:[{code,label}], sections:[{code,label}],cloze_eligible:integer,version:"1.0"}`.
Corpus records: `{id,word,pos,level,section,jp,example,letter}`.
`POST /api/search` body `{filter:Filter,offset:0,limit:50}` → `{items:[Record],total,offset,limit}`. `GET /api/corpus/{integer-id}` → Record.
Filter:
```json
{"free_text":"","corpus_ids":[],"pos_codes":[],"level_codes":[],"section_codes":[],"collection_ids":[],"initial_letter":0,"history_result":[],"confidence_range":null,"attempt_time_range":null,"requested_count":20}
```
collection_ids may be 8-digit strings (preferred) or numeric values; backend accepts both. initial_letter 0=All, A=1..Z=26. confidence_range `[min,max]`, time range `[from_unix_ms,to_unix_ms]`, unspecified endpoints -1. Codes per spec §9.6. POS 1 adjective,2 adverb,3 auxiliary verb,4 conjunction,5 definite article,6 determiner,7 exclam,8 indefinite article,9 infinitive marker,10 linking verb,11 modal verb,12 noun,13 number,14 preposition,15 pronoun,16 verb. LEVEL 1=3000,2=5000,3=blank. SECTION 1..6=S1..6,7..18=W1..12,19=blank. History 1=correct,2=answered incorrect,3=unknown,4=recalled yes,5=recalled no. requested_count -1=all or positive N. Unspecified values omitted/zero/empty. No score support in v1.

## Activities and Actions
`POST /api/activity` body `{}` → `{activity_id:"00000001"}`. Persist activity ID in memory for current page lifetime. Activity start is recorded by backend to reserve ID.
Action input `{occurred_at:Date.now(), activity_id, screen, event, target:null, value:null, session_id:null}`; optional `attempt_id` only for an existing attempt. No input physical coordinates. target/value strings or null. `POST /api/actions` body `{events:[Action], sent_at:Date.now()}` → `{saved:integer,received_at:unixms}`. Batch buffered events every ~30 seconds, navigation and visibilitychange. sent_at is envelope only; occurred_at is persisted and never overwritten. Basic exact-event deduplication handles retries. UI may retain failed batch in memory.

## Sessions / candidates
Quiz types `word_recall`,`cloze`,`reading_review`; variants `en_to_jp`,`jp_to_en`,`word_and_example`,`jp_and_example`; cloze variant null. source_mode `new` or `review`.
`POST /api/candidates` body `{quiz_type,variant,source_mode,filter}` → `{total,cloze_skipped}`. Same as actual creation; collections preserve repeated registrations for quiz candidates.
`POST /api/sessions` body `{quiz_type,variant,source_mode,filter,action:Action}` → Session object. New creates specs; Review reuses historical specs; a question collection reuses its existing specs even in NEW. No matches returns 400 code `no_candidates`.
`GET /api/sessions?quiz_type=word_recall` → `{items:[SessionSummary]}` including incomplete and completed, newest first. Summary `{session_id,quiz_type,variant,source_mode,status,current_position,total,answered,created_at,filter,filter_spec}`.
`GET /api/sessions/{id}` → Session object = Summary plus `questions:[{position,question_id,answered,attempt_count}]`.
`PATCH /api/sessions/{id}` body `{current_position:integer,status:"active"|"paused"}` → Session object. Completed remains completed if all positions answered. No Attempt is created.
`GET /api/sessions/{id}/questions/{position}` → Question:
```
{session_id,position,total,question_id,quiz_type,variant,
 corpus:{id,word,pos,level,section,jp,example,letter},
 prompt:string,cloze_text:string|null,
 options:[{id,label}],
 latest_attempt:Attempt|null,
 feedback:Feedback|null}
```
`corpus` is study material (client must render only relevant fields before answer). No expected answer/parameters/correct ID in question response. options are four shuffled corpus IDs and labels; stable within a session+position, may differ in Review. For cloze prompt=blanked EXAMPLE. For reading prompt=word or JP. feedback is returned only if this position was previously answered.
`POST /api/sessions/{id}/attempts`:
```
{position,outcome:"answered"|"unknown",selected_corpus_id:int|null,
 answer_text:string|null,recalled:0|1|null,confidence:0..100|null,
 response_time_ms:nonnegative|null,action:Action}
```
→ `{attempt:Attempt,feedback:Feedback,session:Session}`. Backend transaction commits Attempt plus answer Action (submit/unknown and attempt_id); client must wait for success before claiming saved. Supplied action's occurred_at/activity_id are preserved. Other buffered actions may be flushed separately. Repeat answering creates new Attempt. Optional `request_id` not used; do not auto-retry attempts blindly.
Attempt fields `{attempt_id,session_id,position,outcome,selected_corpus_id,answer_text,recalled,correct,confidence,response_time_ms,occurred_at}`. Nullable fields remain null. Unknown objective=correct 0; reading=correct null, recalled null. Confidence null is distinct from zero.
Feedback `{correct_corpus_id,expected_answer,corpus,options:[Record]}` (all four mappings always for recall). current_position is kept at answered position until explicit next PATCH; status completed derives from answered positions.

## Collections
`GET /api/collections` → `{items:[{collection_id,name,kind,count}]}`.
`POST /api/collections` body `{name,kind:"vocabulary"|"question",action?:Action}` → collection.
`GET /api/collections/{id}` → `{collection_id,name,kind,count,items:[{item_id,corpus_id,question_id,corpus,quiz_type,variant}]}`.
`POST /api/collections/{id}/items` body `{corpus_id:int}` or `{question_id:string}`, plus optional `allow_duplicate:true`, `action:Action` → item. Duplicate returns HTTP409 `{code:"duplicate",error:"..."}`; ask user then repeat with allow_duplicate true. Repeated rows are retained and each has item_id.
`DELETE /api/collections/{id}/items/{item_id}` → `{deleted:true}`. May use request body `{action:Action}` or separately enqueue remove Action.
`DELETE /api/collections/{id}` → `{deleted:true}` (UI explicit confirmation).

## Data
`POST /api/data` body `{filter:Filter,quiz_type:""|quiz_type,variant:""|variant,offset:0,limit:50,action_offset:0,action_limit:50,screen:"",event:""}` →
```
{attempts:[Attempt augmented {question_id,quiz_type,variant,corpus}],total,
 actions:[{action_id,occurred_at,activity_id,screen,event,target,value,session_id,attempt_id}],action_total,
 stats:{attempts,objective_count,correct,incorrect,unknown,reading_yes,reading_no,accuracy:null|0..100,confidence_average:null|number},
 usage:{total_ms,by_screen:[{screen,duration_ms,intervals}],unknown_intervals,approximate:true}}
```
Attempt history filters apply to SAME Attempt; historical attempts retained. Corpus/collection/quiz/variant filters apply to attempts. Action list filters by screen/event and time range; action overview includes navigation/search actions without corpus links. usage derives from paired enter/leave (or hidden/visible) events by activity; unclosed intervals unknown, no invented durations.

## Static
`GET /` embeds web/index.html. `/manifest.webmanifest`, `/sw.js`, `/icon.svg` are Go-generated URLs, not additional frontend files. Backend defaults `-addr 127.0.0.1:8080 -csv ./data/all_with_ID.csv -db ./data/learning.db`; flags `-check-corpus`, `-version` also supported.
