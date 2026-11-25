# Oxford Core Vocabulary — Personal Learning Application 仕様書 v1.0

- **Version**: **v1.0**（GPT Pro実装引継ぎ用・実装基準版）
- **更新日**: 2026-11-04
- **位置付け**: v0.91の既存仕様を全文継承し、採用済みのPC・Mobile各10画面A案、朱色・ゴシック体、問題作成画面のA〜Zを追加確定する。Go・SQLite・単一HTMLの実装へ進むための基準資料。**本書自体は実装済みアプリではない**。
- **目的**: 固定Corpusで任意の学習・検索・復習を実行し、Attempt・Actionの履歴をDataで振り返れる個人用Webアプリを実働させる。仮組みを優先し、運用しながら修正する。
- **読み方**: **§19〜§23をv1.0のUI採用・実装引継ぎとして優先**。§0〜§18はv0.91の機能仕様・決定経緯を継承。古い「UI未決定」「次はモック作成」という記述は当時の履歴であり、**v1.0では採用デザインを確定済み**として読み替える。既存の【確定】／【暫定採用】／【実装裁量】／【保留】の境界は維持する。

## v1.0の追加確定事項【v0.91からの差分】

1. **20件のUI選定を確定**：10 Viewとも`Oxford_Core_UI_Selections.json`のとおり**PC=A、Mobile=A**。B案は採用しない。採用参考HTMLは`oxford_ui_40_options/screens/{view}_pc_A.html`／`{view}_mobile_A.html`。
2. **色**：採用済み更新モックに合わせ、メインアクセントを**朱色**（参考CSS `--primary: #c64b2c`）とする。以前の緑／青アクセントを基準にしない。
3. **フォント**：明朝／Serifではなく、英文・和文とも**ゴシック／Sans-serif**。OS標準のSans-serifフォールバックを使用可能。外部Webフォントは必須としない。
4. **A〜ZはDictionaryとSubmode Selectionの両方に必要**：Quiz NEWとReviewで、問題抽出前の検索条件として`All`または`A..Z`を指定する。選択は共通検索へ反映され、Session作成時は既存`filter_spec`タグ7で保存する。モックの表示だけでは完成扱いにしない。
5. **UIは本番で単一の`index.html`へ統合**：20の参考HTMLはPC/Mobile各ViewのUI設計資料であって、本番配布用の別ページ20枚を要求するものではない。
6. **Go／SQLite／Action Log**：§1〜§18の契約と9カラムAction Log暫定案を実装基準として使用。DDL・HTTP endpoint・Package構造・CSSやJSの細部はGPT Proが技術的に決めてよい。独立した設計協議を無限に延ばさない。
7. **初期完了の判定**：Corpus 6,100行の検索、3種類のQuizとNEW/Review/Resume、Collection、Action/Attemptの保存、Dataによる後日の履歴閲覧、PC/Mobileの操作が実際に動くこと。Sentence GenerationはComing soon表示のまま。

---


## 0. v0.9 → v0.91 の変更点【Action Logの暫定採用とUI状態遷移の確定】

1. **v0.9の検索・データ・Quiz仕様を維持**：CSV 6,100行、固定Corpus ID、7テーブル、8桁ゼロ埋めID、`question_specs.parameters`、`filter_spec`のタグ付き整数列、Review／Resume／Collection、Oxford Reading HTMLに準じた検索規則を変更しない。【継承】
2. **Action Logは直近の9カラム案を暫定採用**：`action_id,occurred_at,activity_id,screen,event,target,value,session_id,attempt_id`。物理入力の毎フレームを記録するのではなく、意味のあるアプリ内操作と発生時刻を記録し、Dataで振り返る。【暫定採用】
3. **SPAの操作とSQLiteへのWriteを分離**：Frontendで操作を即時反映・Actionを蓄積し、画面遷移・回答確定・一定間隔・ページ非表示などを契機にGoへまとめて送る。送信はUIの応答をブロックしない。【暫定採用／実装裁量】
4. **回答確定とActionを対応付ける**：`attempt_logs.attempt_id`と、回答確定Actionの`action_logs.attempt_id`を対応させ、日時・時間分析に利用する。確定回答の内容はAttempt側が正本であり、Actionへ重複保存しない。【暫定採用】
5. **画面を10個のSPA View状態に整理**：Hero、View Selection、Quiz Selection、Submode Selection、Dictionary、Data、Word Recall、Cloze Test、Reading Review、Sentence Generation（Coming soon）。Quizの方向・variant・検索／件数・Resume／Collectionは原則既存View内の状態・パネルとして扱い、専用Viewを増やさない。【確定】
6. **Heroの自由入力からDictionaryへ直接遷移**：入力した検索語を引き継ぐ。通常の入口はHero → View／Quizで、Quizは形式選択 → NEW／Review → 設定 → 演習となる。【確定】
7. **PCとMobileの双方にハンバーガーメニュー**を設け、そこから既存View／機能への入口へ遷移できる。ハンバーガーは追加の独立Viewではなく、共通ナビゲーションパネル。【確定】
8. **Sentence Generationは画面入口のみComing soon**：問題生成・採点・Session作成へ進めない。新しい機能は要求しない。【確定／実装保留】
9. **UIの見た目は未確定**：v0.91は画面名・遷移・必要な状態を決めたもので、完成した`index.html`やデザインモックではない。次工程は最小HTMLモックの具体化。【確定】

### 0.1 本版で確定／暫定採用／保留とする範囲

| 区分 | 内容 |
|---|---|
| **確定** | v0.9のCorpus・Quiz・Session・検索・`filter_spec`・Collection、10 Viewの役割と主要遷移、PC／Mobile共通ハンバーガーメニュー |
| **暫定採用** | Action Logの9カラム、意味的操作の記録、Frontend側バッファ・バッチWrite、Attempt ID連携、簡易利用時間集計 |
| **実装裁量** | 具体的なイベントコードの網羅範囲、バッチ送信の閾値／間隔、SQL DDL、API Payload、画面の細かな配置・フォーカス・遷移アニメーション |
| **初期実装保留** | Sentence Generation本体・GPT採点、Score評価、忘却曲線、完全オフライン同期、日時カレンダーの完成UI（必要なら後付け） |

**設計の大きな未決定事項としてAction Logを残していた状態は終了**。ただし「暫定採用」を厳密な不可変更仕様と混同しない。Dataでの振り返りが実用上の基準である。

## 1. アーキテクチャ・責務境界【確定】

| 要素 | 採用方針 | 責務 |
|---|---|---|
| **Go Backend** | 複数ソース・Packageで開発し**単一バイナリ**にビルド | HTTP、Corpus Reader、共通検索、問題生成／再生成／採点、Session、Collection、統計、DB入出力 |
| **Frontend** | 最終的な実HTMLファイルは原則**`index.html`のみ**。CSS／JS内包 | PC／Mobileの表示、画面遷移、入力装置アダプター、一時的な未確定入力 |
| **Corpus** | `all_with_ID.csv`（固定6,100行・7列、ID付与済み） | 英単語・品詞・LEVEL・SECTION・日本語の語義ラベル・英語例文の正本 |
| **SQLite** | 外部に置く永続DB | Question Spec、Session、Action／Attemptログ、CollectionとSession抽出条件。**Corpus本文は格納しない** |
| **Go Meta／Import層** | 外部評価データの型・項目変換 | GPT評価等の所定数値を検証し、DBへ書き込める値にする。**初期実装は保留** |
| **PWA／通信** | Tailscale ServeのTailnet内HTTPS | 個人専用アクセス。Manifest／Service WorkerはGoからURL配信可能 |

- Goが提供するCorpus Readerの返却型は共通化する。**現時点はCSV Readerのみ**。将来SQLite由来の教材へ差し替え／追加できる余地は保持するが、外部CSV一括追加・複数Corpus管理は今は設計しない。【確定／保留】
- 外部CSVの本文、ビュー表示情報、ログ、推定された習熟度を混同しない。SQLiteに保存するのは**問題定義・確定した回答・操作履歴・進行状態・抽出条件・意図的に保存した集合**。【確定】
- `index.html`という物理ファイル数と、概念上の**10 View状態**の数は別。SPAとして状態で画面を切り替え、**PC／Mobile双方のハンバーガーメニューから遷移できる**。細かなUI操作のたびにAction Logを即時送信する必要はない。【確定】
- Service WorkerはHTML内の通常のScriptだけでは登録できないため独立URLを設けるが、物理`sw.js`ファイルはなくてもGo側からコードを配信できる。【実装案】
- PWAとしてインストールできることと完全オフライン動作は区別する。初期版はTailscale接続中に使えればよい。完全オフライン同期は保留。【確定】

---

## 2. Corpus・数値ID【ID付き実ファイルを正本に更新】

### 2.1 `all_with_ID.csv`の実ファイル検証【検証済み】

| 検査項目 | 実際の状態 |
|---|---|
| ヘッダー | `ID,WORD,POS,LEVEL,SECTION,JP,EXAMPLE` |
| データ行数 | **6,100行**（ヘッダーを除く） |
| 異なるWORD数 | **5,112** |
| `WORD + POS`の一意数 | **6,100組**（重複0件） |
| ID | **1〜6100、連続、欠番0、重複0** |
| LEVEL | `3000` = 3,811件、`5000` = 2,138件、空欄 = 151件 |
| SECTION | 空欄 = 4,591件。非空欄は複数のコードを` | `で連結する行を含む |
| JP・EXAMPLE | 空欄なし |
| `JP`の意味 | WORDの短い日本語語義ラベル。**EXAMPLE全体の日本語訳ではない** |
| SHA-256 | `9c871e809b082836339c665aa611476602301f3e6425eca0ded34185d8093f72` |

これは提出された**現物CSVに対する検証結果**であり、v0.7の「これからIDを付与する」という予定ではない。旧Reading HTMLに6,101件が埋め込まれていたというv0.7記録は履歴上の参考で、**現在のCorpus正本は6,100件**とする。

### 2.2 Corpus ID【確定】

- ヘッダーはデータIDに含めず、最初の行が`ID=1`、最後の行が`ID=6100`。**既にCSVへ付与済み**。
- Corpus IDは**CSVの1行（WORD・POS・JP・EXAMPLEを含む）に対応する固定ID**。学習記録が参照するので再採番しない。
- 表示順の並べ替えは自由だが、表示位置・行インデックスとCorpus IDを取り違えない。
- `WORD + POS`の一意性は整合性検査・意味的照合に残すが、Question SpecとAttemptは原則数値IDで参照する。
- 既存CSV内のIDの保存値は`1`〜`6100`のまま。**生成するDB識別子の8桁ゼロ埋め規則をCSVのデータ値へ遡及適用しない**。

#### 既存行の例

| Corpus ID | WORD | POS | 補足 |
|---:|---|---|---|
| 1 | `a, an` | indefinite article | WORD中にカンマを含む |
| 2 | `abandon` | verb | Clozeの例 |
| 3 | `ability` | noun | Reading Reviewの例 |
| 82 | `actually` | adverb | 四択の例 |
| 139 | `afterwards` | adverb | 四択の例 |
| 302 | `approximately` | adverb | 四択の例 |
| 1032 | `completely` | adverb | 四択の例 |

**元CSVは適切なCSVパーサーで読む。** `WORD="a, an"`のようにカンマを含む値を単純分割しない。DBの数値列`parameters`や`filter_spec`は、元CSVのCSV区切りとは別の表現である。

### 2.3 DBで生成するID系【確定・内部型は実装裁量】

- Session、Question Spec、Collection、Attempt、Action等の**独立識別子は8桁ゼロ埋め数字を使用**。例：`00000001`、`00000012`。
- IDの性質は**一意識別**であり、ID自体を日時と見なさない。日時の根拠はAction Logのイベント時刻へ分離する。
- DBでINTEGERとして保持しGoが8桁表示するか、TEXTとしてそのまま保持するかは実装判断。桁あふれ等の細部も仮組みに必要な範囲で扱う。
- `session_questions.position`やClozeの文字位置は、8桁識別子とは異なる**位置・数量**であり、無理に8桁化しない。

### 2.4 SQLiteの外にあるCorpusの参照【確定】

- SQLiteにはCorpusレコード本文を置かないため、CSV IDの存在はSQLiteの通常の外部キーだけでは検査できない。Goがロード済みCorpusから解決・検査する。
- `question_specs.parameters`とVocabulary CollectionはCorpus IDを使う。AttemptはSession内の位置を経由してSpec・Corpusへ接続する。
- 現在のID付与対象はこのCSVのみ。複数CorpusのID衝突解決は将来の別設計とする。

---

## 3. Frontendのモード・画面遷移【v0.91：10 Viewの最小構成】

### 3.0 View状態の一覧【確定】

| # | View状態 | 主な用途 | 補足 |
|---:|---|---|---|
| 1 | **Hero** | 起動・共通入口・自由入力検索 | 入力確定で検索文字列を保持したままDictionaryへ |
| 2 | **View Selection** | Dictionary／Dataの選択 | 「View」入口 |
| 3 | **Quiz Selection** | Cloze Test／Word Recall／Reading Review／Sentence Generationの形式選択 | 「Quiz」入口 |
| 4 | **Submode Selection** | 選んだQuiz形式についてNEW／Reviewを選択、必要な検索条件・件数を指定 | 方向・提示形式もこの画面内の設定。Review側にResume入口 |
| 5 | **Dictionary** | Corpusの検索、WORD・EXAMPLE・JPの閲覧、Collection操作 | Reading HTMLを参照 |
| 6 | **Data** | 学習履歴・Action履歴・日時・成績・概算利用時間を振り返る | 過去会話の「Date」表記も本アプリでは**Data**を指す |
| 7 | **Cloze Test** | Clozeの出題・回答・解説・問題移動 | 既存Sessionを使用 |
| 8 | **Word Recall** | EN→JP／JP→ENの四択出題・回答・解説 | 方向は設定値 |
| 9 | **Reading Review** | 英語例文を使ったカード型の自己評価 | Mobileは左右スワイプ、PCはボタン等でも同じ操作 |
| 10 | **Sentence Generation** | **Coming soon...**のみ表示 | 初期版で問題・Session・採点を実行しない |

**10個とは独立した画面ファイル数ではなく、同一`index.html`の画面状態の数**。検索パネル、Confidence、回答確定後、Collectionの管理、Resume一覧、ハンバーガーメニューは原則これらのView内のUI状態であり、View数に加えない。

### 3.0.1 遷移図【確定】

```text
Hero
├── 検索語を入力して確定 ───────────────────────────→ Dictionary（検索語を引き継ぐ）
├── View ──→ View Selection
│              ├── Dictionary ──────────────────→ Dictionary
│              └── Data ───────────────────────→ Data
└── Quiz ──→ Quiz Selection
               ├── Cloze Test ───────┐
               ├── Word Recall ──────┤
               ├── Reading Review ───┼────────→ Submode Selection
               │                      │           ├── NEW → 条件・件数設定 → Session生成 → 各Quiz View
               │                      │           └── Review → 履歴条件・件数設定 → 新Session → 各Quiz View
               │                      │                         └── Resume → 既存Sessionへ戻る
               └── Sentence Generation ────────→ Coming soon...（同一View内で終了）

すべての主要Viewに共通：
［☰ Hamburger Menu］→ Hero／View／Quiz／Dictionary／Data等の入口へ移動
                          ※Quizへの直接導線は選択・設定を経由。
                          ※進行中のSessionは復元可能な状態で保持。
```

- `Home`の従来表記を、今回の起点UIでは**Hero**と呼ぶ。**Heroの入力でDictionaryに直接遷移**する。【確定】
- 「Quizで形式選択」→「SubmodeでNEW／Review選択」は別のView。Word Recallの方向、Reading Reviewの二形式は独立の追加Viewにしない。【確定】
- Submode Selection内で、NEWはCorpus条件、ReviewはCorpus条件＋履歴条件を表示。Quiz形式別の細かな出題設定は同画面内に配置する。【確定】
- ResumeはReview経由で既存Sessionへ戻る。メニューから既存Sessionを選ぶ導線を設ける場合も、新Sessionを勝手に生成しない。【確定】
- Hamburger Menuは**どの端末でも同じ行き先を提供する**。主要View・モードの入口へショートカット移動できるが、Quiz画面へ直接進むときに必要な形式・Session・設定を飛ばさない。【確定・実装裁量】
- 画面を移動しても未完了Sessionを「回答済み」としない。`unknown`以外の単なる移動ではAttemptは生成しない。【確定】
- Collectionの作成／追加／削除や、DataのFilterは既存View内で扱う。**Collection専用の11番目のViewを必須としない**。【確定】
- Dictionary・Data・NEW・Reviewの共通検索は§3.2のとおり。検索の意味を統一しながら、画面ごとの配置は変えられる。【継承】

### 3.1 PC／Mobileと共通ナビゲーション【確定】

| 端末・向き | UI方針 | 主な入力 |
|---|---|---|
| PC | 横幅に応じたPC用レイアウト。Dictionary・Dataは一覧とフィルターを見やすく配置 | マウス・キーボード・ゲームパッド |
| スマートフォン | 縦向き・タッチを中心とするレイアウト。Quizは原則1画面1問 | タッチ・スワイプ・長押し |
| タブレット横 | PC UIの縮小 | 主にタッチ |
| タブレット縦 | Mobile UIの拡大 | 主にタッチ |

- **PCにもスマートフォンにもハンバーガー（☰）を設置**し、そこからHero、View、Quiz、Dictionary、Data等の既存行き先へ遷移できる。メニューは共通ナビゲーションの**開閉状態**であり、新しいページ／Viewではない。【確定】
- メニューの基本項目は**Hero／View／Dictionary／Data／Quiz**。Quiz側には**Cloze Test／Word Recall／Reading Review／Sentence Generation**への入口を表示できる。形式を選んだ後、必要なNEW／ReviewとSession設定をSubmode Selectionで行う。進行中のSessionがあれば、そのQuiz Viewへ戻ることもできる。【確定した遷移の具体化／実装裁量】
- メニューの中身と行き先の意味はPC／Mobileで共通。幅・表示方法（オーバーレイ／ドロワー等）はCSSで切り替えてよい。【確定／実装裁量】
- `index.html`のHTML／CSS／JavaScriptを1ファイルに収め、Responsive LayoutとSPA状態を使う。端末の回転やサイズ変更で、未確定の入力・進行中Sessionを破棄しない。【確定】
- ゲームパッドはPC UIのみを初期対象とし、既存の入力アダプター規則は§12に従う。【継承】
- Reading Reviewの左右スワイプは、PCでのボタン・ゲームパッド等と同一の`recalled`操作へ変換する。スワイプ中はカードを追従させ、取り消したジェスチャーはAttemptを作らない。スワイプ操作の具体的な閾値・アニメーションはUI試作で決めてよい。【確定／実装裁量】
- メニューの開閉、検索、Dictionaryの意味表示などはSPA内で直ちに応答し、Action LogのDB書き込み待ちで画面を止めない。【暫定採用】
- 本仕様は**状態遷移と必要機能の整理**であり、HTMLの装飾・画面寸法・完成モックを確定したものではない。次工程で一画面ずつ試作・調整する。【確定】

### 3.2 共通の検索・絞り込み【HTMLの実コード／画像／会話で確定】

| 検索条件 | Dictionary | Data | Quiz NEW | Quiz Review |
|---|:---:|:---:|:---:|:---:|
| WORD | ○ | ○ | ○ | ○ |
| POS | ○ | ○ | ○ | ○ |
| LEVEL | ○ | ○ | ○ | ○ |
| SECTION | ○ | ○ | ○ | ○ |
| Collection | ○ | ○ | ○ | ○ |
| 正誤 | — | ○ | — | ○ |
| Unknown | — | ○ | — | ○ |
| Confidence | — | ○ | — | ○ |
| Score | — | ○※ | — | ○※ |
| 回答日時 | — | ○ | — | ○ |

※Scoreは将来の数値評価導入後に有効。回答日時は日付範囲を指定できる構想だが、**初期版でカレンダーUIまで実装することは任意**。

**添付HTMLの実際の判定式（語義的に同等な擬似式）：**

```text
対象行が一致する条件 =
    (頭文字が未指定 OR 行の頭文字が指定値に一致)
AND (POSが未指定 OR 行のPOSが指定値に一致)
AND (LEVELが未指定 OR 行のLEVELが指定値に一致)
AND (自由入力を空白で区切った全ての語が検索対象に部分一致)
```

- **未指定の条件は評価を制限しない**。未指定属性をOR条件として全行へ加算するわけではない。**専用のAND／OR選択UIは作らない**。【HTML検証済み】
- 添付HTMLの自由入力は、`WORD,POS,LEVEL,SECTION,JP,EXAMPLE`を連結してNFKC正規化・小文字化した文字列に対し、空白区切りの各検索語を**すべて**含むかを確認する（`every`＋`includes`）。各検索語が一致するフィールドは異なってもよい。**部分一致、大小文字を区別しない**。【HTML検証済み】
- HTMLの`#pos`と`#level`は従来**単一選択**だが、新アプリでは**チェックリスト形式の複数選択**に変更する。`POS={noun,verb}`なら`noun OR verb`、`LEVEL={3000,5000}`なら`3000 OR 5000`。**POS条件とLEVEL条件を併用した場合はAND**。何もチェックしていなければ「すべて」と同じ。【今回確定】
- LEVELは`3000`・`5000`・**空欄**の3種で扱う。**3000／5000はCEFRではなく原CSVの所属ラベル**。【確定】
- SECTIONはCSVの単一欄に`S4 | W5`等の複数コードを含む。SECTION条件を選ぶ場合、**指定コードのいずれかを含む行に一致**させる。自由入力でもSECTIONの文字列を検索可能。【既存データ＋実装定義】
- 頭文字A〜Z／「すべて」はWORDの先頭文字で判定する。旧HTMLは埋め込みの`letter`属性を用いるが、新アプリではCSVのWORDから導出する。頭文字と他の有効条件はAND。【HTML検証済み／実装調整】
- Collection条件は保存項目をGoがCorpus行またはQuestion Specへ解決する。**Collection内の明示重複登録は保持**。一般的な候補閲覧と、Collection由来の繰返し出題は役割ごとに扱う。【継承】
- 正誤・Unknown・Confidence・Score・回答日時は**履歴条件**。Data／ReviewのUIが入力条件を決め、Goが対象Attemptを検索して既存Question Specへ結び付ける。過去に一度でも条件へ一致したAttemptがあれば対象となる（日時を指定すればその期間内）。Reading自己評価と客観正誤は分離する。【継承】
- 日本語一括表示／一時表示・ページ切替・絞り込み解除は**画面表示の操作**であり、自由入力やPOS等の検索条件と混同しない。旧HTMLの検索結果は75件単位のページ表示だが、新アプリの件数指定はUIで調整でき、75件の固定は必須としない。【HTML照合／実装裁量】
- 検索結果はDictionary／Data／NEW／Reviewで共有可能なGoの検索処理により得る。ただし、**一つの共通検索画面や一つのSQLに統一する義務はない**。各画面は同じ条件の意味を使い、結果をそれぞれの用途へ利用する。【確定】
- Session開始時に使用した検索条件は`filter_spec`へ記録する。**数値列の正式な文法・コード表は9.6を参照**。【v0.9確定】

**元HTMLの扱い**：`Oxford_Core_Vocabulary_Reading.html`は**検索挙動とUI参照の資料**であり、埋め込みデータ件数が旧版では6,101件と表示されていても、Corpusの正本は**`all_with_ID.csv`の6,100件**。旧HTMLの`VOC-A-0001`等のIDではなく、CSVの数値IDを使う。

---

## 4. Quizの種類と実装範囲【今回再検証】

| Quiz | 提示 | 回答 | 客観的正誤 | 初期実装 |
|---|---|---|---|---|
| **Word Recall** | `EN→JP`または`JP→EN`の四択 | 選択肢1つ（Corpus ID）／Unknown | TRUE／FALSE | **採用** |
| **Cloze Test** | `EXAMPLE`の一部分を空欄にする | 空欄に入る英語文字列／Unknown | TRUE／FALSE | **採用** |
| **Reading Review** | `WORD + EXAMPLE`または`JP + EXAMPLE` | できた／できなかった＋確信度 | **非適用：NULL** | **採用** |
| **Sentence Generation** | 日本語対訳から英文生成を想定 | 自由英文 | 主にGPT数値評価 | **保留** |

### 4.1 Word Recall（EN→JP／JP→EN）【確定】

- EN→JPは英語のWORDを提示し、JP意味ラベル4択。JP→ENはJP意味ラベルを提示し、英語WORD4択。
- **問題内の正答は、指定したCorpus行1件**。一般的な同義語判定や自由翻訳採点は行わない。表示が同一になる候補（同一JPラベル等）の**明らかな多重正答を生む衝突**だけは問題生成時に最低限回避する。
- 回答が正解・不正解・Unknownのいずれでも、**4候補すべてについてWORD／POS／JPの対応を解答後に表示**し、必要なら例文を見られるようにする。
- **誤答候補3件はQuestion Specに固定**しておく。**ABXYへの表示順は保存せず**、出題時・Review時にGoが正答を含む4候補をシャッフルする。
- 回答履歴には**押したABXYの位置ではなく、実際に選んだCorpus ID**を記録する。同じSpecを復習して配置が変わっても同じ誤答候補かどうか解析できる。
- `WORD + POS`はGoのCorpus Indexから参照し、問題定義やAttemptに文字列として何度も複製しない。

### 4.2 Cloze Test【確定】

- `all_with_ID.csv`の`EXAMPLE`を参照し、**指定した開始位置から指定文字数分を空欄化**する。
- 正答は、**まさにその空欄にされた文字列**。CSVの`WORD`の原形と一致しない場合でも、例文内の活用形を正答とする。専用`expected_answer`は不要。
- 例：ID=2（`abandon / verb`）、例文`We had to abandon the walk when the river covered the path.`。0基準の文字位置なら`abandon`の開始は10、長さは7。
- `WORD`が例文中にそのまま出現しない場合（活用形、辞書の語義注記、上付き番号など）があるので、**空欄の特定は単純なWORDの全文一致置換を前提にしない**。Go側で有効な位置と長さを確定・検証する。安全に特定できないレコードへの対応方法はGo側の実装に委ね、誤った位置を黙って出題しない。
- 文字位置は**GoのUnicodeコードポイント（rune）基準の0開始・長さ**を推奨。ブラウザ側にバイト／UTF-16コードユニットの位置計算を任せず、Goから空欄化済みの表示文を渡す。【実装案】
- 英字大小の正規化、余計な空白の扱い等の**採点許容ルールはGoの責務**。英語のWORDの意味を一般化した類義語判定は現時点で行わない。
- 既存例文の反復・記憶そのものに学習価値はある。ただし既知例文での想起と未知文脈の運用を同じ統計として主張しない。

### 4.3 Reading Review【確定：原案を復元】

- **Tinder型のスワイプで、理解・復元ができた／できなかったを自己報告する**。客観的な読解テストや英日翻訳問題ではない。
- 提示形式は二種類のみ。**英語例文は常に同じ`EXAMPLE`**。日本語例文の生成や英語への逆翻訳を求めない。

| 提示形式 | カードに表示する情報 | 判断すること |
|---|---|---|
| `word_and_example` | **`WORD`（英語）＋`EXAMPLE`（英語）** | 英単語と文から意味を自分の中で復元できたか |
| `jp_and_example` | **`JP`（語の日本語意味ラベル）＋`EXAMPLE`（英語）** | 中心的な語義を手掛かりに英文を理解できたか |

- 評価入力：**できた／できなかった**（TRUE／FALSEの自己評価）＋**Confidence 0〜100**。`correct`は**NULL**。どの形式で判断したかはQuestion Specの`variant`で保持する。
- `JP`は一意の全文対訳ではないため、例文全体の翻訳の一致／不一致を判定しない。
- スマートフォンでは右／左スワイプで自己評価、必要ならボタンでも同じ操作ができる。左右の割当てはできた＝右／できなかった＝左を基本とする。
- 日本語ラベルの**押している間だけの一時表示**は、意図しないネタバレを避けるよう、Quizの種類と提示形式に応じて有効化する。
- `reading_review`はQuiz形式名であり、`source_mode=review`は問題の選び方。両者を混同しない。

### 4.4 Sentence Generation【初期実装保留】

- 現在のCorpusにある`JP`は短い語義ラベルのみ。**`EXAMPLE`全文の日本語訳がないため、構想している日本語→英語の英文生成問題は初期実装しない**。ただしv0.91の**Sentence Generation Viewには「Coming soon...」を表示**する。
- 将来、対訳等が準備されたら問題仕様・回答形式・GPT評価Rubricを決める。4種類というQuiz構想は残すが、現段階では**選択後Coming soon...表示で止める**。
- GPTへ送る採点データは、**Goのメタ／翻訳層が指定項目へ変換し、所定の数値（0〜100）として受け取る**ことを想定する。GPT生JSONや自然言語の全文をSQLiteへ埋め込む必然性はない。Import Schema詳細と評価項目は保留。

---

## 5. Question Spec：JSONをやめ、数値列で表す【今回確定】

### 5.1 独立テーブルの目的

- Question Specは**どのような問題が生成されたか**を保持する独立した`question_specs`テーブル。
- 独立させる理由は、**NEWで問題定義を1度保存し、ReviewやQuestion Collectionがその定義を再参照でき、Attemptに同じ問題情報を毎回複写しないため**。
- **初見性や内容の重複排除が目的ではない**。NEWで偶然同じ問題を再生成しても、別の`question_id`を持つ別Specでよい。内容Hashは不要。
- `question_id`は、Corpus行の数値ID（1〜6100）とは**別**。Corpus行が1件でも、その行を使った異なるクイズはいくつでも存在する。

### 5.2 各Quizの保存形式

**初期実装対象の3形式は、Question SpecでJSONを必要としない。**

| Quiz | `quiz_type` | `variant` | `parameters`（`TEXT`） | Go側の解釈 |
|---|---|---|---|---|
| Word Recall EN→JP | `word_recall` | `en_to_jp` | `302,1032,82,139` | **先頭**が正答のCorpus ID、残る3件が誤答候補ID。**表示順を意味しない** |
| Word Recall JP→EN | `word_recall` | `jp_to_en` | `3,6,23,54` | 同じ規則。正答・誤答計4つのCorpus ID（候補の意味ラベルは明白な重複を避ける） |
| Cloze Test | `cloze` | `NULL` | `2,10,7` | Corpus ID=2、0開始の空欄位置=10、空欄長=7 |
| Reading Review | `reading_review` | `word_and_example` | `3` | Corpus ID=3。英語WORDと英語EXAMPLEを表示 |
| Reading Review | `reading_review` | `jp_and_example` | `3` | Corpus ID=3。JPラベルと英語EXAMPLEを表示 |
| Sentence Generation | `sentence_generation` | 未定 | 未定 | **保留**。現段階で保存契約を強制しない |

【確定】`parameters`の文字列を**どの形式でどのように解釈するかはGo側で定義**する。SQLiteは`TEXT`として保存し、`json_valid()`やJSONフィールドの存在を前提にしない。

【実装案】`parameters`の文法：10進整数をASCIIカンマ`,`で連結した文字列とする。半角空白を含めない。符号付き・空要素・末尾カンマは禁止し、Goが件数・範囲・重複等を検証する。厳密な文法の適用場所はGoのQuestion Specパーサーである。

- **正答のIDは常にWord Recallの第1値**。この順序は**保存形式上の意味**であり、画面のABXYの並び順ではない。誤答候補3件の記載順は表示順として使用しない。
- **4つのIDは別行**、すなわち値としてはすべて別のCorpus IDとする。EN→JPで`JP`表示が重複するなど、明らかに曖昧な選択肢はGoが再選定する。
- Clozeの`start,length`は**IDの後ろで固定順**。期待正答はこの2整数とCorpusのEXAMPLEから導く。単語表層形がWORDと違ってもよい。
- Reading Reviewの`parameters`はIDの1数値のみ。提示形式は通常カラムの`variant`で表せるため追加JSON不要。
- `parameters`は**学習回答ではなく問題定義**。正誤、本人の確信度、選択した誤答IDは保存しない。
- 同じSpecをReviewしても`parameters`を変えない。四択のABXYだけをその都度ランダムに並べる。

### 5.3 Word Recallで順序を保存しないことによる振る舞い

```text
question_specs: question_id=Q1 / variant=en_to_jp
  parameters = "302,1032,82,139"

1回目の表示：ABXY = [139, 302, 82, 1032]
Reviewで表示：[82, 1032, 139, 302]

どちらも同じQuestion Spec Q1。
選択した答えはABXYの「B」等ではなく、Corpus IDとしてAttemptに記録。
```

- **「同じQuestion Specを復習する」ことは「選択肢を同じ位置に並べる」ことではない**。教材行・誤答候補は同じでも配置は変わる。
- Resume時に同じ問題が再表示されても四択の配置は再シャッフルしてよい。【確定】ただし確定済みAttemptは元の選択Corpus IDで保持する。
- 画面の「何番を押したか」という位置だけでは、シャッフル後に選択語を特定できない。Frontend／Goは**選択Corpus ID**を回答確定時に送る。

### 5.4 Clozeの数値指定と文字位置

```text
Corpus ID = 2  → EXAMPLE = "We had to abandon the walk when the river covered the path."
parameters = "2,10,7"
                      ↓
0基準の文字位置10から7文字 = "abandon"
                      ↓
表示：We had to _______ the walk when the river covered the path.
正答：EXAMPLEのその範囲を取り出した "abandon"
```

- **空欄の位置は文字列先頭からの数値指定でよい**。別の正答文字列をSpecに複製しない。
- Unicodeコードポイント（Goのrune）の数え方を例として用いるが、Goが一貫して扱うのであれば、単位を後で明示的に選択してもよい。**文字位置をバイト／UTF-16等と無自覚に混在させない**。
- 1問題1空欄を初期形とする。将来複数空欄にする場合は数値列の文法を拡張するか、別のQuiz形式として改めて設計する。【保留】
- 0や負数の長さ、範囲外の切り出し、空白しかない空欄等はGoが拒否する。【実装案】

### 5.5 JSONとSQLiteの責務

- `question_specs.spec_json`を必須とした旧版案は廃止。初期実装3形式の問題定義は、**`parameters`のカンマ区切り数値列＋`quiz_type`・`variant`**で表す。【確定】
- SQLiteは数値列の中身を解析しない。Goが形式ごとにパース・検証して問題を再構成する。【確定】
- **Sessionの`filter_spec`も保存する**が、Question Specの`parameters`とは別物。Go側が`version,block_count,(tag,value_count,values...)`の**整数列**として解釈・復号する。詳細は9.6。【v0.9確定】
- WORDの部分一致を含む自由入力はUnicodeコードポイント列、日時範囲はUnixミリ秒として可逆的に保存する。**どの属性の値かを`tag`で区別し、可変長値を`value_count`で区切る**ため、元の文字列や複合条件を復元できる。【v0.9確定】
- HTTP通信や一時的なFrontend状態までJSONを禁止したわけではない。Go内部実装・HTTPのデータ形式は必要な範囲で選んでよい。


---

## 6. Attempt Log：JSONなしの回答保存【直近の議論を反映】

**各回答形式の情報は通常カラムで一意に表せるため、`attempt_logs.answer_json`を使わない。**

| Quiz | 回答時に確定する値 | `correct`の意味 |
|---|---|---|
| Word Recall | `selected_corpus_id`（1〜6100） | 選択したID＝Specの正答IDならTRUE。UnknownならFALSE |
| Cloze Test | `answer_text`（入力した語句） | EXAMPLEから切り出した文字列に採点規則上で一致すればTRUE。UnknownならFALSE |
| Reading Review | `recalled` = `1/0`（できた／できなかった） | **NULL**。自己評価と客観的採点は別 |
| Sentence Generation | 自由英文・評価方式とも詳細保留 | 評価前NULL。初期実装なし |

共通のカラム案：`attempt_id`（8桁識別子）、`session_id`、`position`、`outcome=answered/unknown`、`selected_corpus_id`、`answer_text`、`recalled`、`correct`（1/0/NULL）、`confidence`（0〜100/NULL）、`response_time_ms`（非負/NULL）、将来の`score`・`score_scheme`（任意）。**回答発生時刻はAction Logの回答確定イベントから参照**し、確定したAttempt IDを回答確定Actionの`attempt_id`に記録して対応付ける。【暫定採用】

- **回答・Unknownの確定時にのみAttemptをINSERT**。選択肢変更・確信度調整・押下中だけの意味表示はAttemptに保存しない。
- **Word Recallは答えの位置ABXYを保存しない**。選択された実データのIDだけ保存すれば、どの誤答候補を選んだかを分析できる。
- **Clozeの空欄解答文字列は本人の入力そのものを保存**。正答はQuestion SpecとCSVから再計算できるため、Attemptに正答文字列を複製しない。
- `outcome='unknown'`は意識的な「分からない」確定。通常の誤答（`answered`, `correct=0`）や未回答の問題移動（Attemptなし）とは別。
- **Reading Reviewの「できなかった」は`recalled=0,correct=NULL`という自己評価**。一般のUnknownと無条件には混同しない。
- Confidenceは**正解表示より前**に入力・確定。`0`と未設定`NULL`を区別し、値を勝手に埋めない。
- 過去の回答は追記保存する。後勝ちでSessionの位置が変わってもAttemptを消さない。
- 初期版ではスコアのある学習モードは存在しないため、**GPT評価点の保存・取り込み実装は保留**。拡張余地の説明だけ残す。

---

## 7. SQLiteのテーブル定義（概念スキーマ）【7テーブル維持】

> 現時点の**DB構造と保存責務の正本**。最終DDL（INTEGER/TEXT、各制約のSQL表現、Index等）ではない。学習機能の意味を変えない実装上の判断はGo／SQLite側に委ねる。**`filter_spec`は9.6で確定、Action Logの9カラムと記録・送信方針は7.5・11で暫定採用**。7テーブルを維持する。

### 7.1 `question_specs`：生成済み問題

| カラム | 内容 | 状態 |
|---|---|---|
| `question_id` | 8桁ゼロ埋めの一意識別子。Corpus IDとは別 | 確定（内部型は実装裁量） |
| `quiz_type` | `word_recall`／`cloze`／`reading_review` | 確定 |
| `variant` | Recall方向、Reading表示形式。ClozeはNULL等 | 確定 |
| `parameters` | カンマ区切りの整数列。問題形式別にGoが解釈 | 確定 |

- NEWで新Specを作り、Review／Question Collectionで同じSpecを参照する。**内容Hashによる同一問題排除はしない**。【確定】
- Corpus IDは`parameters`の中に保持。WORD・POS・JP・EXAMPLEをDBへ複写しない。【確定】
- 作成日時を`created_at`で別保存することは必須にしない。日時分析に必要なイベントはAction Log側へ寄せる。【確定方針／Actionは暫定採用】

### 7.2 `sessions`：学習単位と進行

| カラム | 内容 | 状態 |
|---|---|---|
| `session_id` | 8桁識別子 | 確定 |
| `quiz_type` | SessionのQuiz形式 | 確定 |
| `variant` | 方向／提示形式 | 確定 |
| `source_mode` | `new`／`review` | 確定 |
| `filter_spec` | そのSessionの抽出に使用した条件。**保存する** | 確定 |
| `current_position` | Session内の現在位置 | 確定（更新方法はGo裁量） |
| `status` | `active`／`paused`／`completed` | 確定 |

- **全てのSession問題位置に確定Attemptが1件以上存在すれば`completed`**。`unknown`も回答と数える。単なる表示・移動・選択中は未回答。【確定】
- 問題順はSession開始時に固定し、Resumeで再利用。Repeat回答は新Attemptを追加する。【確定】
- `filter_spec`の保存は必要。Goは9.6の**数字`,`区切り＋固定タグ／変換表**により文字列・日時を含む抽出条件を復元する。【v0.9確定】
- `started_at`／`updated_at`等の独立Timestamp列を必須化せず、時間の把握はAction Logに寄せる。【確定方針】
- 進行状態の同時更新はLast Write Winsを原則に、Attempt原記録は削除しない。【v0.7継承】

### 7.3 `session_questions`：出題順

| カラム | 内容 | 状態 |
|---|---|---|
| `session_id` | Session識別子 | 確定 |
| `position` | Session内の問題位置 | 確定 |
| `question_id` | 参照するQuestion Spec | 確定 |

- `(session_id, position)`を用いて一つの出題位置を特定できる。独立の8桁IDは不要。【確定方針】
- 同じQuestion Specを同一／別Sessionの複数位置へ置いてよい。【v0.7継承】
- **ここに保存するのは問題の順番**であり、Word Recallの四択ABXY配置ではない。【確定】

### 7.4 `attempt_logs`：確定した回答

| カラム | 内容 | 状態 |
|---|---|---|
| `attempt_id` | 8桁の固有識別子 | 確定方針 |
| `session_id` | 所属Session | 確定 |
| `position` | Session内で回答した問題位置 | 確定 |
| `outcome` | `answered`／`unknown` | 確定 |
| `selected_corpus_id` | Word Recallで選んだCorpus ID、他はNULL | 確定 |
| `answer_text` | Clozeで入力した文字列、他はNULL | 確定 |
| `recalled` | Reading Reviewの自己評価1／0、他はNULL | 確定 |
| `correct` | 正誤1／0／NULL。Reading ReviewはNULL | 確定 |
| `confidence` | 0〜100整数／未設定NULL | 確定 |
| `response_time_ms` | 問題への回答所要時間／NULL | 任意の実装案 |
| `score`・`score_scheme` | 将来のGPT評価 | 初期実装は保留 |

- **回答確定／Unknown確定／Reading Reviewの自己評価確定でAttemptを追記**。同一位置への再回答も別Attemptになる。【確定】
- 回答日時はAction Logの回答確定時刻を参照。**回答確定Actionの`attempt_id`でAttemptを特定**する。`attempt_id`とは別にAction自体の`action_id`を持つ。日時をIDとして兼用しない。【暫定採用】
- 問題情報は`session_questions → question_specs → all_with_ID.csv`から復元する。【確定】

### 7.5 `action_logs`：操作履歴・時間の基礎【v0.91暫定採用】

**目的**：アプリ上で何を操作したかと、**その操作が発生した日時**を後でDataから振り返る。Session・Attemptの日時連携、Dictionary／Quiz／Dataの概算利用時間も扱う。ログ基盤の完全性や業務用監査精度を求めない。

| カラム | 内容 | 例・扱い | 状態 |
|---|---|---|---|
| `action_id` | Action行を一意に識別する8桁ID | `00000001` | 暫定採用 |
| `occurred_at` | 操作が**実際に発生した日時** | Unix milliseconds。まとめて送信しても各イベントの発生時刻を保持 | 暫定採用 |
| `activity_id` | 一連の利用区間を区別する8桁ID | 画面をまたいだ同一の利用を識別する。厳密なSessionと別 | 暫定採用 |
| `screen` | 操作発生時のView | `hero`,`dictionary`,`data`,`quiz_selection`,`word_recall`,`cloze_test`等。具体的な文字列は§3のViewと整合させる | 暫定採用 |
| `event` | 意味的な操作の種類 | `enter`,`leave`,`search`,`reveal`,`submit`等 | 暫定採用 |
| `target` | 操作対象 | `corpus:00000302`、`question:00000012`、`collection:00000003`、UI識別子など／NULL | 暫定採用 |
| `value` | 操作に伴う値 | 検索文字列、選択した値など／NULL。**TEXT** | 暫定採用 |
| `session_id` | 関連Quiz Session ID | Session外はNULL | 暫定採用 |
| `attempt_id` | 関連する確定Attempt ID | **回答確定イベントで対応付ける**。無関係な操作はNULL | 暫定採用 |

- **操作とは意味的なアプリAction**。物理的なタップの座標やゲームパッドの毎フレーム状態を保存しない。タッチ／マウス／キーボード／PCゲームパッドからの同じ`SUBMIT`は、同じ意味のイベントへ正規化する。【暫定採用】
- Dictionaryの検索確定、意味の表示／非表示、フィルター変更、画面ENTER／LEAVE、Quiz選択・Confidence確定・SUBMIT／UNKNOWN、Collection追加／削除等を記録対象の候補とする。**検索入力の1文字ごと、Confidenceの全中間値を必須としない**。具体的な名称はGo側で一貫させる。【暫定採用／実装裁量】
- `target`は、複数種類の対象を`種別:ID`等で識別するための**汎用TEXT**とする。`value`も汎用TEXT。不要ならNULL。**Question Specの`parameters`やSessionの`filter_spec`をActionへ重複複写する義務はない**。【暫定採用】
- `attempt_logs`が**確定した回答の正本**、`action_logs`が**操作と発生時刻の正本**。回答確定時にはGoがAttemptを保存し、その`attempt_id`を回答確定Actionへ関連付ける。同一リクエスト／トランザクションでまとめて保存してよい。【暫定採用／実装裁量】
- **時刻とIDを混同しない**。`action_id`は8桁の一意ID、`occurred_at`はUnixミリ秒。`activity_id`も8桁IDだが、Quiz用の`session_id`とは別の概念。【暫定採用】
- Dataで**いつ、どの画面で何を行ったか**、回答の時刻、Dictionary／Quizの利用時間を振り返る。時間は主に画面ENTER／LEAVEの差による概算とし、読書中の無操作を直ちにIdle扱いにしない。【暫定採用】
- 画面非表示・ブラウザ終了・通信断では終了イベントや未送信Actionが欠落することがある。初期の仮組みでは**厳密な復旧・二重送信完全排除を要求しない**。集計で欠損があれば概算・不明を区別し、架空の利用時間を確定値として作らない。【暫定採用】

### 7.6 `collections`：保存Collection

| カラム | 内容 | 状態 |
|---|---|---|
| `collection_id` | 8桁識別子 | 確定 |
| `name` | 名前 | 確定 |
| `kind` | `vocabulary`／`question` | 確定 |

- 作成日時／更新日時の独立カラムは初期必須としない。時間の分析はAction Log側に寄せる。【確定方針】

### 7.7 `collection_items`：登録項目

| カラム／概念 | 内容 | 状態 |
|---|---|---|
| `item_id`等 | **同一IDの複数登録行を識別できる仕組み** | 必要性確定、具体的な実装はGo／SQLite裁量 |
| `collection_id` | 所属Collection | 確定 |
| `corpus_id` | Vocabulary用。Question用ならNULL | 確定 |
| `question_id` | Question用。Vocabulary用ならNULL | 確定 |

- **同一Collection・同一`corpus_id`／`question_id`を登録しようとしたら警告**。ユーザーが明示的に登録続行を選択すれば**重複登録を許可**。【確定】
- 重複行は独立の登録として残す。同一IDの登録をSQLiteのUNIQUE制約で全面禁止しない。【確定】
- 内容の同じ**別Question Spec ID**は、そもそも別問題として取り扱う。【v0.7継承】
- 個々の登録行を識別できることが必要だが、独立`item_id`を置くか別の仕組みにするかは実装判断。作成日時はAction Logに任せられる。【実装裁量】

### 7.8 参照関係【確定部分】

```text
all_with_ID.csv（SQLite外、GoがID解決）
  ├─ question_specs.parameters 内の数値ID
  └─ collection_items.corpus_id

sessions
  ├─ session_questions (session_id, position)
  │     └─ question_specs (question_id)
  ├─ attempt_logs (session_id, position) → session_questions
  └─ action_logs.session_id → sessions、回答確定時はaction_logs.attempt_id → attempt_logs

collections
  └─ collection_items（重複登録行を許容）
       └─ question_specs（Question Collectionの場合）
```

- 内部テーブル間の参照整合はSQLiteで構成できる。Corpusの実在・`parameters`の意味・`filter_spec`の解釈はGoの責務。【確定】
- SQL型・Index・CHECK・FOREIGN KEYの具体DDLは、学習機能の意味を変えない範囲で実装時に決定してよい。過度な厳密化を要求しない。【実装裁量】

---

## 8. DBのRead／Writeタイミング【Action一括送信を暫定採用】

| 操作 | 主な永続化対象 | 実装方針 |
|---|---|---|
| Heroの検索→Dictionary／Dictionary検索・意味の開閉／Data操作 | `action_logs`（必要なイベント） | **Frontendは即時応答**。Actionを一時バッファに記録、後から一括送信 |
| Hamburger Menu開閉・画面移動 | View状態更新、必要に応じAction | **新しいSessionやAttemptを勝手に作らない**。画面移動時はバッファ送信候補 |
| Quiz NEW開始 | 新Question Spec・Session・問題順 | Goが必要なDB更新を実行。Actionにも開始操作を記録可 |
| Quiz Review開始 | 既存Question Spec参照、新Session・問題順 | 新Specは生成しない |
| Resume | 既存Session・Attemptの読込、必要な進行UPDATE | Sessionを複製しない |
| 問題表示・未確定の選択・Confidence調整 | Attemptは未作成、必要なActionのみ | 未確定操作を回答扱いにしない |
| 通常回答／Unknown／Reading自己評価の確定 | `attempt_logs` INSERT＋Session位置更新＋回答確定Action | **回答内容と操作発生時刻を`attempt_id`で対応**。両者を同時にGoへ送ってよい |
| Collection作成／追加／削除 | Collection系の更新＋必要なAction | 重複追加時は警告後の明示操作を尊重 |
| GPT評価Import | 将来 | 初期保留 |

**推奨するActionの送信契機**【暫定採用】：①大きなView遷移、②Quiz回答確定、③30〜60秒程度の一定間隔、④ブラウザーが非表示になるときの可能な範囲での送信。実際のバッチ件数・間隔・実装APIはGo／HTMLで調整してよい。**画面遷移時しか送らない、または操作ごとに必ず送るという強制ではない。**

- Actionは**発生した瞬間の時刻**とともにFrontendで保持し、後からGoへ複数件まとめて送信する。SQLiteへのWrite待ちで検索・意味表示・カード操作を止めない。【暫定採用】
- Quiz回答確定など**確実に永続化したい意味的操作**は、Attemptと対応Actionを同一Goリクエストへ含めるなどして、Go側で対応する`attempt_id`を確定できるようにする。【暫定採用／実装裁量】
- 定期送信は**Actionの保存**が目的であり、DictionaryやDataの検索要求まで指定間隔で遅らせるものではない。検索にGo通信が必要なときは通常の検索APIで即時処理する。【確定】
- 初期版では端末強制終了時の未送信分が失われる可能性を許容する。後からDataが欠損を観測できる設計を優先し、監査品質を保証するものではない。【暫定採用】
- 日時の照合と利用時間の概算は§11による。**Sessionの`completed`判定はAttemptの存在により行い、Actionの到着回数からは判定しない**。【確定】

## 9. NEW／Review／Resume・Collection・集計【確定】

### 9.1 NEW：任意条件から新しい問題を作る

- 完全ランダムは選択肢の一つ。通常は自分で`noun`、`adverb`、`LEVEL=3000`などを自由に指定できる。
- 件数N／全件、対象の出題形式・Direction／Presentationを選ぶ。
- Corpus Readerが対象IDを検索し、Goが問形式ごとの`parameters`を作り、**Question Specを独立登録**する。
- 新Sessionには問題一覧とランダム順を確定して登録する。**Session内の問題順は固定**しResume時に未回答問題が欠けないようにする。
- **NEWは初見の単語だけを選ぶという意味ではない**。以前と同内容の別Question Specも作ってよい。

### 9.2 Review：過去のAttemptから対象を選ぶ【履歴検索の意味を確定】

| 条件 | 対象 | 備考 |
|---|---|---|
| `Incorrect` | `outcome=answered`かつ`correct=0` | **過去に一度でも該当するSpecを候補にできる**。後から正解しても残る |
| `Unknown` | `outcome=unknown` | 明示的に「回答できない」と確定した問題 |
| `Low Confidence` | `confidence<=N`（NULLを除く） | 0〜100の閾値指定 |
| Reading Review「できなかった」 | `reading_review`で`recalled=0` | 客観Incorrectではない |
| GPT `Score` | 将来の評価数値 | 初期実装は保留 |

- **過去のAttemptをすべて保持**し、該当Attemptが一度でもあるQuestion Specを抽出できるようにする。【確定】
- **回答日時**と組み合わせれば「全期間に一度でも」「直近7日間に一度でも」「特定日時範囲で」等の検索が可能。期間外のAttemptに一致しても、期間内に一致する履歴が無ければその期間検索には該当しない。【確定：検索意図】
- 正誤・Unknown・Confidence・Score・日時のうち存在しない評価値を、誤答や0に自動変換しない。【確定】
- WORD／POS／LEVEL／SECTION／Collection条件を併用できる。結果は**Question Spec ID**を基本単位として既存Specを選び、新Sessionへ登録する。ReviewでSpecを新規生成しない。【確定】
- 最後のAttemptだけで検索対象を確定する方式は採らない。**直近の時期を見たければ日時条件を指定する**。【確定】
- 重複登録を許したCollectionでは、その登録数がCollection由来の再演習機会へ反映され得る。Reviewの履歴検索でSpecを何回表示するかとは別の条件。【確定部分】
- 強制的な復習時期・回数制限は置かない。【確定】

### 9.3 Resume：旧Sessionへ戻る【Session完了条件を確定】

- 各QuizのReview UIからResumeへ進むが、ResumeはReviewによる新Session生成ではない。【確定】
- 同じSession ID、Question Spec、問題順、確定済みAttemptを復元する。【確定】
- **全問題位置に1件以上の確定AttemptがあるとCompleted**。Unknown確定は正式な回答なので完了条件を満たす。位置移動だけ・未確定の選択だけでは完了扱いにしない。【確定】
- 完了Sessionを後から復習したい場合、Unknown等でReview検索し、既存Question Specを用いた新Sessionを作成できる。【確定】
- 同じ問題を再回答した履歴も残す。Sessionの現在位置更新は後勝ちを基本とする。【v0.7継承】
- 入力途中のドラフト保存は初期版必須としない。日時記録はAction Logの方針による。【実装裁量】

### 9.4 Collection：手動で保存する集合【警告後の重複を許可】

| Collection | 保存対象 | 再演習時 |
|---|---|---|
| Vocabulary | 数値Corpus ID | 任意のQuiz形式で新Specを生成できる |
| Question | 既存Question Spec ID | 同じSpecを新Sessionで出題（四択表示順は毎回ランダム） |

- **同一Collectionに同一IDを再追加しようとしたときは警告する**。それでもユーザーが登録を選べば同一IDを複数登録できる。【確定】
- 重複登録が2件あればCollection内に2件の項目が存在し、そこから練習すると**同じ対象の練習機会が増える**。アプリが自動で1件へ統合しない。【確定】
- 同じIDを**別Collection**へ追加することも可能。【v0.7継承】
- **同内容で別Question Spec ID**の場合は、重複IDの扱いとは異なり、元から別の問題定義として認める。【v0.7継承】
- 個別項目の削除ができるよう、登録行そのものを識別できるようにする。具体的なDBキーは実装裁量。

### 9.5 Data：履歴・成績・時間の分析

- WORD／POS／LEVEL／SECTION／Collection／Quiz形式／variant／正誤／Unknown／Confidence／Score（将来）／回答日時によって検索・集計する。【確定】
- Attemptから回答件数、客観正解率、通常誤答数、Unknown数、Reading Reviewの自己評価、Confidence分布等を算出する。**Reading Reviewの`recalled`を客観的な正答として数えない**。【確定】
- 回答日時はAction Logの回答確定イベントにより取得し、`attempt_id`でAttemptと対応付ける。Session開始・画面利用時間もAction Logから概算する。【暫定採用】
- 同じCorpus IDでもQuiz形式・方向・表示形式を区別する。Attempt確定件数が基本の処理件数で、単なる表示は回答数でない。【確定】
- 利用時間は概算。短期間の正解率を忘却率や長期記憶定着度と同一視しない。【確定】

### 9.6 `filter_spec`の正式保存形式【v0.9確定】

**目的**：SessionをNEW／Reviewとして作った際に、**どの検索条件と件数指定で対象を選んだかを後から復元・分析する**。出題済みの問題一覧は`session_questions`に別保存される。`filter_spec`は**Session単位の元の選択条件**であり、Question Specの`parameters`（出題内容）とは違う。

#### 9.6.1 保存文法：タグ・長さ付き整数列

```text
filter_spec = version,block_count,tag,value_count,value1,...,tag,value_count,value1,...
version     = 1
block_count = 後続する属性ブロックの数
タグごとの値の数はvalue_countで指定
数値は10進整数。区切りはASCIIカンマ「,」。空白なし。
```

- SQLiteには**`TEXT`でそのまま保存**。Goが読書き時に数値列を解釈する。**JSONではない**。
- **指定しなかった条件のブロックは保存しない**。未指定は検索を制限しない。各`tag`は一つのSpec内で原則1回。同一タグの複数候補は一つのブロックの値として表す。
- **各ブロック内に並ぶ複数候補はOR**、ブロック同士は**AND**。ただし自由入力（タグ1）は旧HTMLどおり、空白で区切った検索語が**全部**一致する。範囲（タグ9〜11）は上下限を表す。これをGUIの個別演算子指定として公開する必要はない。
- コード値の割当ては**このv1仕様に固定**する。将来変える場合は`version`に応じてGo側の変換を行う。初期DB Schema Migration機構を要求するものではない。

#### 9.6.2 タグ一覧

| tag | 属性 | 値の並び | 説明 |
|---:|---|---|---|
| 1 | `free_text` | Unicodeコードポイントを順に列挙 | 元の自由入力文字列。部分一致検索用。**元の大小文字・空白を保持** |
| 2 | `corpus_ids` | Corpus IDを1個以上 | 特定のWORD＋POS行を直接指定する場合。**CSVの固定ID** |
| 3 | `pos_codes` | POSコードを1個以上 | 選んだ品詞のどれかに一致 |
| 4 | `level_codes` | LEVELコードを1個以上 | 3000／5000／空欄のいずれか |
| 5 | `section_codes` | SECTIONコードを1個以上 | S1〜S6／W1〜W12／空欄のいずれか |
| 6 | `collection_ids` | Collection IDを1個以上 | 既存Collection。8桁表示だが数値として参照 |
| 7 | `initial_letter` | A〜Zに対応する数値1個 | 先頭文字選択。「すべて」の場合はタグを保存しない |
| 8 | `history_result` | 結果コードを1個以上 | Data／Reviewでの正誤・Unknown・Reading自己評価の選択 |
| 9 | `confidence_range` | 下限,上限 | 双方0〜100。設定値の範囲内のAttemptを対象、NULLは含めない |
| 10 | `score_range` | 下限,上限 | 双方0〜100。**初期版は未使用、将来用の予約タグ** |
| 11 | `attempt_time_range` | 開始Unixミリ秒,終了Unixミリ秒 | 指定区間に回答確定したAttempt。端点未指定は`-1`。日時UIの初期実装は任意 |
| 12 | `requested_count` | 正の件数N、または`-1` | 出題数指定。`-1`は全件。**このタグは全Sessionで保存** |

- WORDの通常の**自由入力**（`theo`、`bank`など）はタグ1を用いる。Corpus IDは「その行が特定されている」場合だけタグ2で保存する。**部分一致を固定Corpus IDへ勝手に置換しない**。
- タグ1の値列はGoの`[]rune`に相当するUnicodeコードポイント整数。例えば`cat`→`99,97,116`、`猫`→`29483`。**値数で元の文字列を復元**できる。検索時だけNFKC正規化・大小文字無視を適用する。
- タグ5のSECTIONは`S4 | W5`等を` | `で分け、**コード単位の包含**で評価する。空欄は固有コードとする。
- タグ8で結果を選ぶUIはData／Reviewに置き、Goは履歴検索として判定する。正誤などのUI表現と内部コードを対応づける。
- タグ11のUnixミリ秒はタイムゾーンに依存しない区間端点。日付カレンダーの1日指定なら、UIの選んだローカル日付の開始と翌日開始直前を正しく変換して記録できる。**日時検索の記録元はAction Logの回答確定Actionで、`attempt_id`によってAttemptへ結び付ける**。【暫定採用】
- `sessions`には`quiz_type`・`variant`・`source_mode`が別にあるため、この3値を`filter_spec`に重複格納しない。実際に作られた問題順は`session_questions`にある。

#### 9.6.3 固定変換表

**POS：添付`all_with_ID.csv`に存在する16種類**（この順番の番号を固定）

| code | POS | code | POS |
|---:|---|---:|---|
| 1 | adjective | 9 | infinitive marker |
| 2 | adverb | 10 | linking verb |
| 3 | auxiliary verb | 11 | modal verb |
| 4 | conjunction | 12 | noun |
| 5 | definite article | 13 | number |
| 6 | determiner | 14 | preposition |
| 7 | exclam | 15 | pronoun |
| 8 | indefinite article | 16 | verb |

**LEVEL**：`1 = 3000`、`2 = 5000`、`3 = LEVEL空欄`。

**SECTION**：`1..6 = S1..S6`、`7..18 = W1..W12`（`W1=7`、`W10=16`、`W12=18`）、`19 = SECTION空欄`。

**頭文字**：`A=1, B=2, ..., Z=26`。現在のCorpusにX始まりの語がない場合でも、Xを選べば結果0件となる。

**履歴結果**（タグ8）：`1 = Correct`（客観`correct=1`）、`2 = Incorrect`（`outcome=answered, correct=0`）、`3 = Unknown`（`outcome=unknown`）、`4 = Reading Reviewでできた`（`recalled=1`）、`5 = Reading Reviewでできなかった`（`recalled=0`）。同一属性内の選択値はOR。Readingの自己評価を客観正誤へ変換しない。

**ID**：タグ2・6の値は、それぞれ`all_with_ID.csv`のCorpus ID、`collections.collection_id`の数値値を使用。画面上は`00000012`等の8桁で表示してよいが、`filter_spec`の数値欄では`12`で足りる。

#### 9.6.4 数値列の例

```text
例A：POS={noun,verb}、LEVEL={3000,5000}、頭文字R、50件
1,4,3,2,12,16,4,2,1,2,7,1,18,12,1,50

  version=1 / block_count=4
  tag3(POS)      = [12,16]
  tag4(LEVEL)    = [1,2]
  tag7(頭文字)   = [18]
  tag12(件数)    = [50]

例B：自由入力「cat」、POS=noun、10件
1,3,1,3,99,97,116,3,1,12,12,1,10

  tag1(自由入力) = [99,97,116] → "cat"
  tag3(POS)      = [12] → noun
  tag12(件数)    = [10]

例C：過去にIncorrectまたはUnknown、Confidence 0〜30、全件
1,3,8,2,2,3,9,2,0,30,12,1,-1

  tag8(結果)     = [2,3]
  tag9(確信度)   = [0,30]
  tag12(件数)    = [-1]

例D：完全ランダム全件（条件指定なし）
1,1,12,1,-1
```

- **「ランダム」は抽出後の選択・並べ替え動作**であり、検索条件自体ではない。`requested_count=-1`は対象候補の全件を意味する。件数Nの場合のランダム化はQuiz NEW／Reviewの処理に従う。
- タグ8〜11を組み合わせるときは、UIが指定した結果・数値・日時条件を**同じAttemptに適用**して既存Specを抽出する。これにより別々の日の無関係なAttempt条件を混ぜて1件一致と誤判定しない。タグ8内の複数結果はOR、他の有効条件とはANDとする。
- Goは数値列のバージョン・タグ・値数・数値域を確認して解釈するだけでよい。SQLiteに変換表用テーブルを追加する必要はない。**7テーブル構成を維持**する。
- この形式はあくまで**仮組み用に簡潔で後からSQLで変換可能な固定表現**として採用する。Query Builder、複雑な文法解析、JSON Schema、追加のDSLは要求しない。

---

## 10. 評価の意味・入力の確定タイミング【確定】

| 評価項目 | 型・値域 | 意味 |
|---|---|---|
| `correct` | `1 / 0 / NULL` | 客観的に正誤がある問題の判定。Reading ReviewではNULL |
| `confidence` | `INTEGER 0..100 / NULL` | 解答・自己判断時の主観的確信度。0と未入力を区別 |
| `recalled` | `1 / 0 / NULL` | Reading Reviewの「できた／できなかった」。他形式ではNULL |
| `outcome` | `answered / unknown` | 通常の回答と、明示的な「分からない」確定を区別 |
| `score` | `0..100 / NULL` | 将来のGPT数値評価。初期版では未使用 |
| `response_time_ms` | 非負整数／NULL | 必要に応じて回答所要時間を記録 |

- **正誤と確信度は別々に保存**。0とNULL、客観正誤と自己評価、Unknownを混同せず、数値×Booleanの単純積で情報を消さない。
- **Confidenceは正解の公開前**に確定する。未操作のスライダーから黙って50等を補わない。
- **Word Recall：** `selected_corpus_id`をGoが正答Corpus IDと比較する。ABXYの位置は学習統計の中心にしない。
- **Cloze：** `answer_text`をCorpusの空欄文字列と比較する。正答は別フィールドへ保存不要。
- **Reading Review：** `recalled=1/0, correct=NULL`。できなかった場合でもそれは客観的な誤答ではない。
- **Unknown：** `outcome='unknown'`。客観的正誤形式なら`correct=0`、Confidenceが未入力ならNULL。単なるスキップはAttemptを作らない。
- GPT評価は将来、Goのメタ層が指定項目を数値へ変換し、Attemptの定義済みスカラーカラムへ反映する。**DBにGPTの生JSONを保存・解釈させる必要はない**。

---

## 11. Action LogとSPAの関係【v0.91：最小案を暫定採用】

### 11.1 記録対象と粒度

- ユーザーが操作した**意味的なイベント**を、§7.5の9カラムで記録する。【暫定採用】
- 画面移動（`enter`／`leave`）、検索確定（`search`）、Dictionaryの意味表示／非表示（`reveal`／`hide`）、検索条件確定（`filter`）、Quizでの選択（`choose`）、Confidence確定（`confidence`）、回答確定（`submit`／`unknown`）、問題移動（`next`／`previous`）、Collection追加／削除（`add`／`remove`）などを扱う。**イベント語彙の厳密な一覧は実装時に揃える**。【暫定採用／実装裁量】
- 物理タップ座標、キーDownの全履歴、ゲームパッドの毎フレーム、検索入力1文字ずつ、スライダーの全中間値は初期保存必須にしない。【暫定採用】
- Action LogのためにDictionary検索やReading Reviewスワイプの操作性を損なわない。UIは即時更新され、ログは別の流れで送る。【確定】

### 11.2 Frontend→Go→SQLite

```text
PC／Mobile：単一index.htmlのSPA
  ├─ Dictionary・Data検索／意味開閉／メニュー操作 → 直ちに画面へ反映
  ├─ Quizの選択・Confidence・スワイプ → 直ちに画面へ反映
  └─ Actionイベント（発生日時付き）をFront側に一時蓄積
          ↓
  View遷移／回答確定／定期送信／画面非表示を契機にバッチ送信
          ↓
  Go Backend：受信イベントを検査し、必要なAttemptと結び付ける
          ↓
  SQLite：action_logs（操作・時刻）とattempt_logs（確定回答）を保存
```

- 一括送信でも**発生日時はイベントごとに保持**し、送信日時で置き換えない。【暫定採用】
- 回答確定Actionは、Goが生成／保存したAttemptの`attempt_id`と接続する。**同一HTTPリクエスト・同一トランザクション**で保存する方法を推奨するが、仕様上の目的は対応が一意に分かることである。【暫定採用／実装裁量】
- 入力未確定のまま画面を移ったとき、Attemptを捏造しない。Sessionの再開は§9.3に従う。【確定】

### 11.3 Dataでの振り返り：初期完成条件

- **いつ**利用したか、**どの画面**を使ったか、**何を操作したか**をActionから表示可能にする。
- Session・Quiz形式・問題／Corpusと回答確定時刻を照合し、Attemptの正誤／Unknown／Reading自己評価／Confidenceを参照できる。
- 画面ENTER／LEAVEを使い、Dictionary・Data・Quizの**概算利用時間**を出せる。終了記録が欠けた区間は概算／不明として扱い、精度を保証しない。
- 一度に多くの集計を完成させる必要はない。**最低限、保存した履歴を後からDataで閲覧できる**状態を優先する。
- Idleの厳密判定は保留。長い例文を読んでいる無操作時間を即時に非学習扱いにしない。

### 11.4 暫定採用の境界

この章は、**今回の実装に進めるための初期方式**であり、将来の改善禁止ではない。送信間隔、イベント名、表示粒度や追加のEvent列は、使用して不足があれば見直せる。**監査用の完全なイベントソーシング、イベント再生による全DB再構成、完全オフラインQueueは要求しない**。

## 12. 入力体系【v0.5〜v0.6を継承】

UIはタッチ／キーボード／ゲームパッドを内部共通Action（`NEXT`, `PREVIOUS`, `CHOOSE`, `REVEAL_START`, `REVEAL_END`, `SET_CONFIDENCE`, `SUBMIT`, `UNKNOWN`等）へ変換する。Goのドメイン処理は入力装置を知らなくてよい。

### 12.1 PCゲームパッド【確定】

| 入力 | 機能 |
|---|---|
| LB | 前の問題／語彙 |
| RB | 次の問題／語彙 |
| ABXY | Word Recall四択。画面上の4候補の選択 |
| A押下中 | 意味表示が可能なReader等の画面でのみ一時表示。**四択中のAは選択用** |
| LT／RT | Confidence減少／増加 |
| スティック | フォーカス移動・確信度スライダー調整 |
| R3 | 回答確定 |

- ゲームパッドはPC UIのみ。押下状態を常にDBへ書くことはしない。
- Trigger感度・スティックのデッドゾーン、Unknownボタン割当てはUI試作で調整。【保留】

### 12.2 スマートフォン／タッチ【確定】

- 左右スワイプ：通常は前後移動、**Reading Review中は右＝できた／左＝できなかった**。同じ操作をPCボタン・ゲームパッドでも行える。文脈により切り替える。
- **押している間だけ**日本語ラベルを一時表示。指を離すと隠す（Pointer Eventsで押下解除・キャンセルを扱う）。
- 四択タップ、Confidence 0〜100スライダー、回答確定ボタン、Unknownボタンを設ける。
- 画面回転・レイアウト変更後も、確定前の選択・ConfidenceはFrontend状態として保持する。

### 12.3 PCキーボード・マウス【実装案】

- マウスで検索・選択、キーボードでも問題移動・選択・確定・意味の一時表示を行える。
- 具体的キー割当てはUI試作時に決める。入力欄に文字を打っている間はショートカットを誤発火させない。

---

## 13. GPT評価・Meta Layer【方針のみ確定／実装保留】

- Sentence Generationは当面保留だが、将来の**評価軸（Grammar／Meaning／Usage／Overallなど）のスキーム定義**はGo側のメタ／翻訳層に設けられる。
- GPTにはそのスキームに従い**数値を出力させればよい**。外部表現形式はCSV等でも足り、**JSONを必須としない**。
- Meta Layerは列の同定・型変換・0〜100の範囲検証・対応Attemptへの紐付けを担当し、SQLiteには**確定したスカラー値**を保存する。
- どの項目を何点で評価するか、輸出入方式、`score`をAttemptに置くか別の数値テーブルに置くか、Rubric版や再評価履歴を保存するかは、Sentence Generation実装時に設計する。【保留】
- **「スキームを持つ」ことと「DBにそのスキームのJSONを保存すること」は別**。GPT原文／生JSONをSQLiteへ押し込まない。

---

## 14. 保留：忘却曲線・その他の拡張

- **忘却曲線／FSRS／Scheduler／次回復習日時／定着度推定は初期実装に含めない**。後付けの分析・候補提示として導入する余地のみ保持する。
- NEW・Reviewで自分の学習対象・練習回数・時期を自由に指定できることが優先。短期間の連続正解を長期記憶の定着と混同しない。
- Sentence GenerationとGPT評価Import詳細、例文全体のJP対訳作成は保留。
- 汎用CSV一括Import、複数Corpus統合・外部DBへの教材移行方式は保留（Corpus Reader差替え余地だけ残す）。
- 完全オフライン学習・同期待ちQueue・厳密な複数端末競合解決は保留。**Sessionの進行状態の後勝ちは採用**。
- 性能上の追加Indexは実測で必要になってから。PRIMARY KEY／UNIQUE／FOREIGN KEYなどの必要な整合性制約は先に定義する。
- DB Schema Migration／`user_version`等のバージョン管理実装は保留。GitでDDLの履歴は管理できるが、Gitのみで運用DBの構造が自動的に変わるわけではない。
- 8桁識別子の内部型・採番・重複防止はGo／SQLiteの実装判断とする。Timestampを識別子兼日時として使う旧案は必須ではなく、日時はAction Logの時刻を根拠にする。Action Logの暫定案は§7.5・§11。

---

## 15. 代表的な学習・データフロー

### 15.1 NEW：NounのJP→EN Word Recallを50問

1. `Hero → Quiz Selection → Word Recall → Submode Selection（JP→EN、NEWを選択）`。
2. 検索条件として`POS=noun`・件数50を指定。正本から候補IDを選ぶ。
3. 各問題に**正答Corpus ID＋誤答候補Corpus ID×3**を割り当て、`question_specs.parameters`へCSVではなくDB内のカンマ区切り整数列として保存。
4. Question Spec IDをランダムな問題順で新Sessionに配置。これは`session_questions`へ保存する。
5. 表示のたびGoが四択配置をシャッフル。正解前にConfidenceを入力し、ABXY／タップ等で回答確定。
6. Attemptに選択した**Corpus ID**、正誤、Confidenceを保存。**回答日時は`attempt_id`で結び付いたAction Logの回答確定時刻**。結果画面で4候補すべてのWORD／JP対応を表示。

### 15.2 Review：間違えたClozeを再演習

1. `Quiz Selection → Cloze Test → Submode Selection（Review → Incorrect）`。必要なら`POS=verb`等で絞り込む。
2. SQLiteのAttemptから既存Question Spec IDを取得。**Specは再INSERTしない**。
3. `parameters="2,10,7"`なら、CSVのID=2のEXAMPLEを参照し、文字位置10から7文字を空欄化。期待解答は該当文字列そのもの。
4. 新Sessionの問題順をランダム化し、回答を新Attemptとして記録。前回の誤答履歴はそのまま残る。

### 15.3 Reading Review：語彙を大量に走査

1. `Quiz Selection → Reading Review → Submode Selection（NEW）`。例として`variant=word_and_example`を選ぶ。
2. `parameters="3"`から`ability / noun`と英文例文を読み、カードとして表示。
3. 意味を復元できたかを左右スワイプ／ボタンで自己評価し、Confidenceを確定。
4. Attemptに`recalled=1/0, correct=NULL`を保存。Reviewで「できなかった」を条件に再度同じSpecを選べる。
5. `jp_and_example`では`JP=能力`と**同じ英語EXAMPLE**を表示し、語義を手掛かりに英文を理解できたかを判断する。

### 15.4 Resume：PCからMobileへ

1. PCで新Session開始。`session_questions`へ問題順を確定保存。
2. 途中で中断。確定済みAttemptとSessionの現在位置を保持。
3. Mobileで`Quiz Selection → 対応形式 → Submode Selection（Review → Resume Previous Session）`を選び、既存Sessionを復元。新たにQuestion Specを生成しない。
4. **Word Recallの四択表示位置は再シャッフルしてもよい**が、対象IDと誤答候補IDは同じ。Attemptは選んだCorpus IDを保持しているため、意味の整合は失われない。

### 15.5 Data：自分の学習履歴を参照

- Corpus ID／POS／LEVEL／Quiz形式／variant／期間／Collectionを指定して、回答回数・正答率・自己評価・Confidence・Unknownを検索・集計。
- Action LogからDictionary／Data／Quizの利用時間を概算し、操作・回答確定イベントから日時を取得する。**9カラム構成と`attempt_id`対応を暫定採用**。
- **出題はされたが回答が確定していない**問題は、Attemptが存在しない。Unknownはユーザーが明示的に押したときだけ記録される。

---

## 16. v0.7〜v0.9・Reading HTML・Corpusの照合とv0.91変更管理【検証報告】

### 16.1 照合対象

1. `Oxford_Core_Vocabulary_App_Specification_v0.7.md`：初期のQuiz・DB・画面／入力構想。
2. `Oxford_Core_Vocabulary_App_Specification_v0.8.md`：7テーブル、Collectionの警告後重複追加、Session完了、検索対象を整理。
3. `Oxford_Core_Vocabulary_App_Specification_v0.9.md`：**直接の継承正本**。Corpus検証、検索式、Filter Specの文法、保留事項を維持。
4. `Oxford_Core_Vocabulary_Reading.html`：Dictionaryの画面・自由入力・品詞／LEVEL・A〜Z・意味表示の参照。v0.9で検証済みの`filter()`の意味は変更しない。
5. `all_with_ID.csv`：Corpus正本。6,100行、7列、ID 1〜6100。**旧HTMLの埋め込み6,101件とは別**。
6. v0.9以後の会話：Action Logの9カラム案・一括送信・Dataでの振り返り、10 Viewの列挙、PC／Mobile双方のハンバーガーメニューを採用する指示。

### 16.2 v0.9との実質的な変更点

| 対象 | v0.9 | v0.91 |
|---|---|---|
| 主なUI起点 | `Home` | **Hero**。入力確定でDictionaryへ検索語を渡す |
| View状態 | 画面の概念・入口を記述 | **Hero／View Selection／Quiz Selection／Submode Selection／Dictionary／Data／Cloze／Recall／Reading／Generationの10状態** |
| ハンバーガー | 未規定 | **PC／Mobile双方**に設置、共通ナビゲーションを開閉 |
| Sentence Generation | 未実装表示／非表示の選択余地 | **Coming soon...のViewを表示** |
| Action Logカラム | 粒度・Payloadが未定 | **9カラムを暫定採用**。§7.5 |
| Action Write | 大画面遷移等が候補、詳細未定 | **Frontendバッファ→バッチ送信を暫定採用**。§8・§11 |
| Attemptの日時対応 | 方法未定 | **回答確定Actionの`attempt_id`で対応**を暫定採用 |
| Data履歴 | 構想済み、Action連結は未定 | **少なくとも操作・回答・日時を後から参照できることを初期基準に** |
| Corpus／Quiz／7テーブル／Collection | 確定済み | **変更なし** |
| 検索／Filter Spec | v0.9で確定 | **変更なし**。§3.2・§9.6を維持 |
| Forgetting Curve・GPT採点 | 保留 | **保留のまま** |

### 16.3 元の検索ロジックとの整合

- 元HTMLはNFKC正規化・大文字小文字無視・部分一致・自由入力の語間AND・有効属性間AND・未指定属性の無条件通過を用いる。v0.91でも**この意味を変更しない**。
- POS／LEVELの複数選択チェックリストはv0.9で導入した拡張。**同一属性の選択値はOR、属性間はAND**。A〜Z、意味表示、SECTION、Collection、Data／Review履歴条件もv0.9の定義を引き継ぐ。
- `sessions.filter_spec`はv0.9の**version 1・タグ／値数付き10進整数列**をそのまま使用。新しいタグ・変換表・テーブルを追加しない。

### 16.4 Corpus照合（変更なし）

```text
File: all_with_ID.csv
Header: ID,WORD,POS,LEVEL,SECTION,JP,EXAMPLE
Records: 6100
ID: 1..6100 (no gap, no duplicate)
Unique WORD: 5112
Unique WORD + POS: 6100
LEVEL: 3000=3811, 5000=2138, blank=151
SECTION blank: 4591
JP blank: 0
EXAMPLE blank: 0
SHA256: 9c871e809b082836339c665aa611476602301f3e6425eca0ded34185d8093f72
```

- `WORD="a, an"`等があるため正しいCSV Parserを使う。`WORD`の原形がEXAMPLEに文字列一致しない行もあり、Clozeは位置の検査を行う。
- v0.91では**仕様書だけを作成**。新Goコード・HTMLの完成モック・SQLite DDLは未作成。旧仕様書・CSV・参照HTMLは変更しない。

### 16.5 暫定採用と実装裁量を混同しない

**Action Logの9カラム・一括送信・Attempt連携はユーザーが「一旦それでいい」と採用した暫定仕様**。精密なIdle検出、厳格なイベント命名、DB復旧機構は追加要求ではない。**10 Viewと共通ハンバーガーは明示的に採用された画面構造**。UIの最終見た目・CSS・アニメーション・API・DDLは実装／試作段階で決める。

---

## 17. 次工程：単一HTMLの最小モックとGo実装

1. **UIの最小HTMLモック**：Hero・View／Quiz／Submode Selection・Dictionary・Data・3つの実働Quiz・Sentence GenerationのComing soonを、同じ`index.html`の状態遷移として試す。
2. **PC／Mobile共通ハンバーガー**：どの主要Viewからも既存の入口・画面へ移動できるようにする。新しい遷移Viewやルーターライブラリーは必須でない。
3. **Reading Review**：Mobileの右＝できた、左＝できなかったのスワイプ、PC入力を同じ自己評価Actionへ接続する。
4. **Action Log**：本版§7.5・§8・§11の暫定方式で実装し、まずはDataで日時・操作・回答履歴を後から見られることを確認する。
5. **Go実装**：固定CSV Reader・検索・Filter Spec整数列・Quiz生成／採点・Session／Collection・ログ入出力を実装する。複雑な実装前提は加えない。
6. **保留**：Sentence Generation本体／GPT評価・忘却曲線・完全オフライン同期は初期版に含めない。

---

## 18. 設計上の不変条件【v0.91】

1. ユーザーが**学習形式・対象・時期・回数を自由に選ぶ**。忘却曲線に拘束されない。
2. 正本Corpusは**`all_with_ID.csv`**。6,100行・7列、IDは固定の`1..6100`で再採番しない。Corpus本文はSQLiteへ複製しない。
3. DBで生成する識別子は**8桁ゼロ埋め**で扱う。IDをTimestampの代用品としない。
4. Question Specは独立、NEWで生成・Review／Question Collectionで再利用。**同内容の別Specも許す**。
5. Word Recallの`parameters`は**正答Corpus ID＋誤答候補3 ID**。ABXY順は保存せず、Goが並べ替える。
6. Clozeの`parameters`は**Corpus ID＋空欄開始位置＋長さ**。期待正答はEXAMPLEから切り出す。
7. Reading Reviewは**WORD+英語EXAMPLE／JPラベル+英語EXAMPLE**の二形式。`recalled`とConfidenceを保存し、`correct=NULL`。
8. Question Specの整数列、Attemptの通常カラム方式を維持。JSONを必須化しない。
9. Session内の問題順は開始時に確定しResumeで保持。**全問題で回答確定したら完了**し、Unknownも回答に含む。
10. Attemptは確定時に追記し、再回答も履歴を残す。未確定操作をAttemptに保存しない。
11. Reviewで**過去に一度でも間違えた問題**を検索できる。日時範囲との組合せも可能。Unknown・Reading自己評価は客観誤答と区別。
12. CollectionはVocabulary（Corpus ID）とQuestion（Question Spec ID）。**同一ID登録に警告し、ユーザーが望めば重複を許す**。
13. **共通検索はReading HTMLの実コードの意味を踏襲**。有効条件間AND、未指定は無条件、自由入力の検索語間AND・部分一致・大文字小文字無視。POS・LEVELは複数選択の同一属性内OR。専用AND／OR切替UIは不要。
14. **`filter_spec`は§9.6の整数列文法で保存・復号する**。Unicode自由入力、各属性コード、ID、履歴条件、日時区間、件数指定を可逆的に記録。7テーブルを維持する。
15. Action Logは**操作内容・発生時刻・利用時間の基礎**。§7.5の**9カラムを暫定採用**し、SPAの操作とDBへの送信は分離。`attempt_id`で回答確定と関連付ける。
16. **Dataでの後日振り返り**を優先する。操作・回答・日時の参照と概算利用時間を提供し、厳密なIdle／監査精度は求めない。
17. **10 View状態**：Hero／View Selection／Quiz Selection／Submode Selection／Dictionary／Data／Cloze／Word Recall／Reading Review／Sentence Generation（Coming soon）。
18. **PCとMobileの双方にハンバーガーメニュー**を設置し、主要View・入口へ移動できる。メニュー自体は11番目のViewにしない。
19. **Heroの自由入力からDictionaryに直接遷移**し検索条件を引き継ぐ。Quizでは形式選択→NEW／Review→設定→演習。Resumeは既存Sessionに戻る。
20. 共通検索の適用範囲は§3.2の表を正本とする。Scoreは将来の数値評価まで未使用。日時カレンダーUIの初期実装は任意。
21. 7テーブル、Go単一バイナリ、Frontend単一HTML、Tailscale Serve、PC Gamepadの方針を維持。
22. Sentence Generationの**本体**・GPT採点詳細・忘却曲線・複数Corpus・完全オフライン同期は保留。Sentence Generationの**Coming soon表示**は対象。
23. **学習のための仮組み装置**であり、実際に動くことを第一にする。バグゼロ保証や過度なCoding規範を追加しない。
24. 旧版との記述が衝突する場合、**検索とFilter Specはv0.9の定義、Action Logと画面遷移・ハンバーガーはv0.91の定義を優先**する。**暫定採用を永久不変仕様とみなさない**。

---

---

## 19. v1.0で採用したUIデザイン【確定】

### 19.1 採用選定の正本

ユーザーが提出した`Oxford_Core_UI_Selections.json`は、**10 ViewのPC／Mobileそれぞれ全件A案**を選択している。採用候補から実装用に参照する20件は以下とする。スクリーンの見た目・配置の参考に使い、モック内の偽データや仮リンクは実際のGo APIに差し替える。

| View | PCで採用する参考HTML | Mobileで採用する参考HTML |
|---|---|---|
| Hero | `hero_pc_A.html` | `hero_mobile_A.html` |
| View Selection | `view_selection_pc_A.html` | `view_selection_mobile_A.html` |
| Quiz Selection | `quiz_selection_pc_A.html` | `quiz_selection_mobile_A.html` |
| Submode Selection | `submode_selection_pc_A.html` | `submode_selection_mobile_A.html` |
| Dictionary | `dictionary_pc_A.html` | `dictionary_mobile_A.html` |
| Data | `data_pc_A.html` | `data_mobile_A.html` |
| Cloze Test | `cloze_pc_A.html` | `cloze_mobile_A.html` |
| Word Recall | `word_recall_pc_A.html` | `word_recall_mobile_A.html` |
| Reading Review | `reading_review_pc_A.html` | `reading_review_mobile_A.html` |
| Sentence Generation | `sentence_generation_pc_A.html` | `sentence_generation_mobile_A.html` |

- 参照元の親フォルダは`oxford_ui_40_options/screens/`。採用の基準は**朱色・ゴシック体・SubmodeのA〜Z選択を反映した更新版**（`Oxford_Core_UI_40_Proposals_v0.91_vermilion_gothic.zip`）。旧版UI ZIPやB案を誤って採用しない。
- 参考HTMLを独立20ページとして運用しない。**最終アプリは1つの`index.html`**。Media Queries／レスポンシブ・CSSとSPA状態でPC/Mobile表示を切り替える。10 Viewは別々のファイルではない。
- 画面サイズ、列数、スクロール、モーダル／ドロワーの細部は採用見た目の再現性と実操作性を優先して実装裁量とする。PC／Mobileの見た目を完全に同じ配置にする必要はない。
- 既存の`Oxford_Core_Vocabulary_Reading.html`は**Dictionary検索動作の元参照**。6,101件の古い埋め込みデータおよび`VOC-A-...`形式のIDは正本ではなく、Corpusは`all_with_ID.csv`の6,100行・固定数値IDを採用する。

### 19.2 色・文字・共通コンポーネント

| 項目 | v1.0採用基準 |
|---|---|
| アクセント | **朱色**。更新版A案の`--primary: #c64b2c`を目安とする |
| 明るい背景 | 更新版A案の温かいオフホワイト／紙色ベース |
| 日本語・英語 | ゴシック／Sans-serif。明朝／Georgia／Times等のSerifを主書体として使わない |
| ナビゲーション | PC／Mobile共通のハンバーガーメニュー。Hero／View／Dictionary／Data／Quiz／各Quiz形式の導線 |
| UI状態 | 共通の検索条件・選択・Session状態をSPAで保持。端末回転／幅変化で破棄しない |
| スマホ | 1カラムを基本に、横スワイプとタップ／長押しを対応 |
| PC | 余白のあるA案を基準に、検索欄・Quiz・履歴の操作性を保つ |

朱色は**メインアクセント**であり、誤答・注意・正答などの状態を必ず同じ色に塗るという意味ではない。表示意味を失わない配色は実装側で調整できる。

### 19.3 モックから実働アプリへの変換ルール

- 参考HTMLの`demo`, `NOT CONNECTED`, `UI PROPOSAL`表示、ハードコードされた数件の例文、`confirm()`や固定回答、固定集計値、ページ間`href`は**試作用**。本番ではCSV/SQLiteの実データ、Go API、SPAルーティング／状態更新に置換する。
- 参考HTMLの`<select>`／checkbox／スライダーなどの実装で、v0.91の全条件や操作がまだ入っていなくても、**機能仕様§3〜§12を優先**する。例：POSはCSVにある16種類、LEVELは3000／5000／空欄、SECTIONは実Corpus、QuizのA〜Zは本当に抽出に適用する。
- Quiz選択から直接問題画面へ飛ばず、**Quiz Selection → Submode Selection → NEW／Reviewの条件・件数 → Session開始 → Quiz View**を守る。Resumeは新Sessionを生成しない。
- Dictionary以外のA〜Zは**Submode Selectionの共通検索に組み込む**。単に押下色だけが変わる「飾り」ではなく、抽出結果、Start前の対象件数表示（実装する場合）、`filter_spec`の保存値へ反映する。
- Sentence Generationは**Coming soon...**を表示し、空のSessionを作成しない。

---

## 20. 10 Viewの画面責務と最小遷移【確定＋実装裁量】

| View | 初期動作の核 | PC／Mobileで共通化する内部状態 |
|---|---|---|
| Hero | クイック検索EnterでDictionaryへ。View／Quiz入口、☰ | 入力文字列／遷移先 |
| View Selection | Dictionary／Dataを選ぶ | 選択した入口 |
| Quiz Selection | Word Recall／Cloze／Reading／Sentence Generationを選ぶ | Quiz形式 |
| Submode Selection | NEW／Review、形式variant、問題数、WORD・POS・LEVEL・SECTION・Collection・**A〜Z**、Review履歴条件、Resume | 検索条件、選択Quiz、variant、source_mode、件数、resume対象 |
| Dictionary | CSV全文検索・語彙／例文・JP表示／一時表示・Collection操作・**A〜Z** | 共通検索条件、結果、ページ・開閉状態 |
| Data | Attempt／Action・学習日時・概算時間の閲覧／検索 | 履歴条件、集計／一覧 |
| Cloze Test | CSV EXAMPLEの空欄、回答／Unknown／Confidence、解答表示・前後移動 | Session ID／position／未確定入力 |
| Word Recall | EN→JP／JP→EN、ABXY四択・Confidence・Unknown、解答後の4候補対応 | Session ID／position／選択Corpus ID／Confidence |
| Reading Review | WORD＋EXAMPLE／JP＋EXAMPLE、自己評価、Confidence、スマホ左右スワイプ・PCボタン | Session ID／position／recalled／Confidence |
| Sentence Generation | Coming soon... | Viewのみ。Sessionを生成しない |

- `Data`が正式表記。過去の会話の`Date`は同じ画面の言い間違いとして扱い、新しい11番目の画面を追加しない。
- レイアウトは採用A案。履歴フィルタ、Collection管理、Resume一覧、回答後表示、メニュー開閉は**10 View内のパネル／状態**であり、それぞれ別Viewを作る必要はない。
- タッチ・マウス・キーボード・ゲームパッドからの同じ意味的操作は、共通Actionとして扱う。Reading Reviewの右＝できた／左＝できなかった。PCでもボタンで同じ結果へ到達できる。
- **アプリの動線をすべてJavaScriptでSPA遷移**させる。履歴が必要な場合のURL表現や`history`の具体構成は実装裁量とする。

---

## 21. Go／SQLite／Action Log実装への引継ぎ契約【v1.0】

### 21.1 コードの構成と責務

- Goは**単一バイナリにビルド**。内部は複数`.go`ファイル／Packageに分けてよい。責務はHTTP、CSV Reader、Corpus Index／Search、Question Spec・Quiz生成／採点、Session・Resume、Collection、Action／Attempt Log、Data集計。
- 配布HTMLは原則**`index.html`1枚（CSS／JS内包）**。採用された20参考HTMLの見た目・操作を、一つのSPAに統合する。巨大なWeb Frameworkや複雑なビルドチェーンは要求しない。
- `all_with_ID.csv`とSQLite DBは外部ファイル。Goが明示的に読み込み、CSVの7列と`ID=1..6100`の対応を保つ。**Corpus本文をSQLiteへ移さない**。
- SQLiteは§7の**7テーブル**を基準とし、必要な制約・Indexを実装する。カラム型・PK表現はGo側で決定してよいが、8桁表示、重複Collectionの保存、正誤／Unknown／NULL、Question SpecとAttemptの区別、9カラムActionを壊さない。
- `filter_spec`は§9.6の**version=1、タグと値数付き10進整数列**を正本にし、GoでEncode／Decodeする。`parameters`も§5の形式。前処理・検索中に独自JSON表現を使うことは許すが、これらのDB欄を勝手にJSONへ変更しない。
- Dataの正答率は**客観問題のみ**。Reading Reviewの`correct=NULL`と`recalled`を混同しない。

### 21.2 Action Logの実装判断（暫定採用のまま実働化）

1. §7.5の9カラム`action_id,occurred_at,activity_id,screen,event,target,value,session_id,attempt_id`を採用。`action_id`と`activity_id`は8桁識別子、`occurred_at`は操作が発生したUnixミリ秒。
2. FrontendはSPA操作を即時反映し、イベントを**メモリへ一時蓄積**。画面遷移／回答確定／一定間隔／非表示化などでGoに**バッチ送信**。送信は表示操作をブロックしない。
3. 検索確定、意味表示、フィルタ変更、入退場、回答確定、Unknown、Collection操作等を**意味的イベント**として記録する。入力文字1文字ずつやスライダー全中間値は不要。
4. **Quizの回答確定時はAttempt保存が正本**。Goが`attempt_id`を確定し、対応する`submit`／`unknown`等のActionと結び付ける。可能なら同一DBトランザクションで保存し、後から回答日時・Session・問題と照合できるようにする。
5. 初期版のDataではAction・Attemptの**時刻つき一覧、学習内容、概算滞在時間**を確認できればよい。精密なIdle推定・完全なログ監査・完全オフライン同期は要求しない。
6. バッファしたイベントの送信漏れ・重複の完全防止は初期要件ではない。ただし、確定回答の保存成否はユーザーへ誤認させず、**Attemptが保存されなかったのに回答済みと確定表示しない**。ログ送信失敗でDictionary検索を停止させない。

### 21.3 HTTP APIとDDL：形式は自由、必要な振る舞いは固定

以下は**機能的なAPI一覧**であり、パス名やリクエストJSON／SQLの完全なDDLを新たな確定規則にするものではない。GPT Proは実装しやすいエンドポイントと型を設計してよい。

| 操作 | Go側が実現する機能 |
|---|---|
| Corpus検索／単一行取得 | WORD・POS・LEVEL・SECTION・Collection・頭文字などの共通条件でCSV IDを解決 |
| NEW Session作成 | 抽出条件の保存、Question Spec生成、Session／出題順保存 |
| Review Session作成 | Attempt履歴条件で既存Question Specを抽出し、新Session／出題順を作る |
| Resume Session取得 | 既存Session／順番／回答済み位置／進行状況を返す |
| Quiz問題取得／回答確定 | 各形式の問題再構成、採点・自己評価、Attempt追記、進行更新、回答Action連携 |
| Collection作成・取得・項目追加／削除 | Vocabulary／Questionの両方に対応。警告後の同一ID複数登録を保持 |
| Action一括受信 | 時刻つきの意味的操作を受信して保存 |
| Data検索・集計 | 回答履歴とAction履歴の表示、基本指標、概算利用時間 |
| SPAの配信 | 単一`index.html`、必要に応じPWA用URL配信 |

- 配布・設定は個人利用を想定し、ローカルまたはTailnet HTTPSで動作すればよい。Tailscale Serve自体は既存仕様に従う。認証基盤やマルチテナントを新たな要件に加えない。
- HTTP、SQLite Driver、Go Package、Index・Cacheの採用は**実装裁量**。安定した簡潔な実装を優先する。利用する依存関係と起動方法はREADMEで明記する。
- 不明瞭な実装細部は仕様の意味を守る最小解で決める。機能の追加（SRS等）や完成の先延ばしにつながる過剰設計は不要。

---

## 22. 完成条件と検証チェック【v1.0実装用】

以下は**設計確認ではなく、GPT Proが実装後に確認する最低限のシナリオ**。モック内の固定値や装飾だけで通過扱いにしない。

1. **起動**：Goコマンド一つでWebアプリとSQLiteを使え、`index.html`が配信される。初回起動時に7テーブルが利用できる。
2. **Corpus**：CSV 6,100行・7列を読み込み、固定IDで指定行へ到達。`WORD="a, an"`等、カンマを含むCSVセルを壊さない。
3. **Hero**：検索入力確定でDictionaryへSPA遷移し、元の入力を検索条件として適用する。
4. **UI**：PC／Mobileで10 View・ハンバーガーメニューから必要な入口へ到達。選択済みA案の朱色・Sans-serifを維持。
5. **検索**：添付Reading HTMLの挙動どおり、NFKC／大小無視・部分一致、自由入力空白区切りAND、有効属性間AND、各チェック項目内OR、未指定無条件を満たす。
6. **A〜Z**：Dictionaryおよび**Submode SelectionのNEW／Review双方**で`All`／A〜Zが働き、問題抽出と保存する`filter_spec`タグ7へ反映される。
7. **Filter Spec**：§9.6の数値列Encode→Decodeが往復一致し、検索語・POS／LEVEL・SECTION・Collection・履歴条件・件数を復元できる。
8. **Word Recall**：EN→JP／JP→ENの4択、正答ID＋誤答候補3件を保存。提示ABXYは変わっても正誤判定・保存した選択Corpus IDは変わらない。
9. **Cloze**：EXAMPLEの指定位置と長さを空欄化。正答文字列をEXAMPLEから復元し、入力／Unknownを判定する。
10. **Reading Review**：2形式、Mobile左右スワイプ／PCボタン、`recalled=1/0`・`correct=NULL`・Confidenceを保存。スワイプ中断はAttemptにならない。
11. **Confidence**：正解表示前の値のみ確定。未設定`NULL`と0を区別する。
12. **Session**：全位置でAttemptを確定したらCompleted。Unknownも回答として数え、単なる移動は数えない。
13. **Review**：一度でもIncorrect／Unknown等の履歴条件を満たしたSpecを再出題でき、元の問題定義・過去Attemptは消えない。
14. **Resume**：中断Sessionを同じSession ID・問題順で復元。再開だけで新規Question SpecやAttemptを作らない。
15. **Collection**：同一IDの追加時に警告し、続行選択なら重複を保存して個別削除できる。
16. **Action Log**：画面移動・検索／閲覧・回答確定などを発生日時つきで記録。回答ActionとAttemptを`attempt_id`で照合でき、Dataで表示できる。
17. **Data**：回答履歴・Unknown・正誤・Confidence・日時・操作履歴を閲覧でき、利用時間は概算／不明を区別する。
18. **保留**：Sentence GenerationはComing soon表示のみで、GPT評価・忘却曲線・完全オフラインは初期完成の妨げにしない。

---

## 23. GPT Proへの成果物要求と優先順位【v1.0実装引継ぎ】

**提出してほしいもの**：起動可能なGoコード一式、実働用`index.html`1枚、SQLite初期化／操作処理、簡潔なREADME（起動・CSV/DB置き場所・初期化・依存・検証方法）。必要な自動テストや最小確認スクリプトはGo側の判断で添付する。**この仕様書の説明だけを言い換えた文章やモックのみを成果物としない**。

**優先順位**：

1. v1.0の**本書§19〜§23**（採用UI・実装基準・検証条件）
2. v0.91から継承した**本書§1〜§18**（検索、データ、Quiz、Action Log、`filter_spec`等）
3. **選択JSONに指定された採用済みA案20 HTML**（レイアウト・見た目・UI操作の参考）
4. `Oxford_Core_Vocabulary_Reading.html`（特にDictionary検索の既存動作参考）
5. `all_with_ID.csv`（Corpus値はこのファイルが唯一の正本。辞書項目の内容に関してはUIモックより優先）

**衝突時の原則**：Corpusの語句はCSV、機能・データ意味は仕様書、採用済みA案のレイアウト・朱色・Sans-serifはUI参照をそれぞれ正本とする。参考UIに不足する機能は仕様書に従って実装し、独自の新機能を勝手に追加しない。技術上の細部は、**実装を完成させるためGPT Proが決めてよい**。

**このv1.0の性質**：UIの選定と機能要求の整理が完了した**実装引継ぎ基準**である。Goソース・SQLite DDL・完成した単一SPAは、GPT Proがこれから実装する対象。Action Logはv0.91以来の**暫定採用方式を初期実装する**のであって、永久不変の設計へ格上げしたものではない。

---

## 最終要約【v1.0】

`all_with_ID.csv`の**6,100行・7列**を固定CorpusとしてGoが参照し、SQLiteの**7テーブル**へQuestion Spec・Session・Attempt・Action・Collection・Filter Specを保存する。Word Recall（2方向）、Cloze Test、Reading Review（2提示形式）を実装し、Sentence GenerationはComing soon表示に留める。

検索はReading HTMLの既存挙動を踏襲：未指定条件は制限なし／異なる有効条件間AND／同じ属性のチェック値間OR／自由入力はNFKC・大小無視の部分一致、空白区切り語間AND。**DictionaryだけでなくQuizのSubmode SelectionにもA〜Zの頭文字フィルター**を実装する。`filter_spec`は§9.6の数値列方式で保存・復元する。

**UIは10 Viewを本番`index.html`一枚のSPAへ統合**。採用デザインは`Oxford_Core_UI_Selections.json`どおり**全画面PC=A／Mobile=A**で、**朱色（参考`#c64b2c`）・ゴシック／Sans-serif**。両端末にハンバーガーメニューを設け、Reading Reviewのスマホ左右スワイプとPC入力を共通操作に接続する。

**Action Logはv0.91の9カラム・Frontendでの一時蓄積／バッチ送信・Attempt ID連携を暫定採用**し、Dataで操作・回答・日時・概算利用時間を後から振り返れるようにする。精密なIdle、SRS、GPT採点、完全オフライン同期は要求しない。

**Go／SQLite／HTTPの内部設計は本書が定める意味と検証条件の範囲内でGPT Proが決める**。今回の成果物は実装基準文書と採用UI参照であり、次の作業は実働するGoアプリと単一HTMLの生成・試験・納品である。
