# Oxford Core Vocabulary v1.0 — 実行検証結果

検証日：2026年11月4日。添付のv1.0仕様書、固定Corpus CSV、全10画面のPC A／Mobile Aの選定を基準に実装しました。

## 1. 結果と検証環境

**Linux上でGoアプリを起動し、実際のHTTP API、外部SQLite、Chromium上のSPAを操作して確認しました。** APIだけの成功を、画面や実機タッチの成功とは扱っていません。

| 検証層 | 実行結果 | 主な証跡 |
|---|---|---|
| Go単体・結合テスト | 8件成功、失敗0件 | `test-results/go-test-final.log` |
| 独立HTTP API・SQLite検証 | 15群成功、失敗0群 | `test-results/api-verification.json` |
| Frontend非同期制御の回帰検証 | 6群成功、失敗0群 | `test-results/frontend-control-flow-verification.json` |
| 実ブラウザー操作 | 下表と操作記録の範囲で確認 | `test-results/browser-ui-verification.json` |
| 狭いCSS画面幅での表示 | 10 Viewへ到達、横方向の文書はみ出しなし | `test-results/narrow-layout-checks.json` |

実行環境はLinux amd64、Go 1.24.9、Chromiumです。同梱バイナリは静的リンク済みです。Goソース・依存定義・DDLが同一であることを確認したうえで、最後のHTMLだけの修正後はGo8件を不要に再実行せず、最終バイナリのAPI15群とFrontend6群を再実行しました。その区別を`final-runtime-verification.json`に記録しています。

READMEの単一`go run`起動も実行しました。初回はコンパイル込み42.574秒で起動し、HTML・HTTP・新規DBを確認しました。試験ハーネスの最初の終了処理では`/proc`の子PID参照が利用できず、終了手順を再確認しました。再実行でSIGINT後の待受け停止を確認していますが、`go run`ラッパーの終了コードは1でした。**この起動検証だけは`PASS_WITH_NOTE`**として、その経緯を`go-run-smoke.json`へ残しています。終了コード0を確認したとは報告していません。

ブラウザーはPC幅約1,363 CSS pxと、ブラウザーズームによる**340 CSS px幅**で確認しました。狭幅検証時の高さは234 CSS pxです。これはレスポンシブ幅の検証であり、スマートフォン実機・端末回転・タッチエミュレーションの検証ではありません。この検証環境ではDevToolsによる端末エミュレーションが管理設定で禁止されていました。

## 2. 仕様書§22：18項目の実行結果

「確認」は記載した方法と範囲で成功したことを表します。**4と10には実機Mobileで未検証の部分が残るため、18項目すべてを無条件に合格とは報告しません。**

| # | 検証項目 | 結果 | 実行した確認と証跡 |
|---:|---|---|---|
| 1 | 起動 | 確認 | 新規の外部DBでサーバー起動。HTML配信、7テーブルの自動作成、9カラムAction Logを確認。READMEの`go run`経路は`go-run-smoke.json`、配布バイナリ経路はAPI検証と配布ZIPの起動確認に記録。 |
| 2 | Corpus | 確認 | CSV 6,100行・7列、固定ID、全件ページ取得、`a, an`のカンマ保持、正本SHA-256一致。本文をSQLiteへ複製するテーブルはない。 |
| 3 | Hero | 確認 | `ＢＡＮＫ`を入力してEnter。DictionaryへSPA遷移し、元の全角入力を保持して7件を表示。最終版ではページ再読込直後の入力でも確認。 |
| 4 | PC／Mobileの10 View・A案UI | **PC・狭幅確認／実機Mobile未検証** | 10 ViewをPC幅と340 CSS px幅で開き、ハンバーガーメニューで移動。朱色、Sans-serif、単一SPAを確認。実機の高さ・タッチ・Mobile Safariでの表示は未検証。 |
| 5 | 検索 | 確認 | 独立したPython検索計算との40条件比較。NFKC、大小無視、部分一致、語ごとのAND、属性間AND、属性内OR、未指定無制限、SECTION複数コードを確認。ブラウザーでも全角検索と結果を確認。 |
| 6 | A〜Z | 確認 | DictionaryのBでbankの2件、Xで0件。NEWはAの候補424件から出題。ReviewのUnknown＋Aは1件、Xは0件。APIと保存DBでタグ7を確認。Allではタグ7を省略。 |
| 7 | Filter Spec | 確認 | §9.6の例、原文のUnicode・空白、タグ1〜9・11・12を往復検証。タグ10は将来用。実施した条件と数値列はAPI検証JSONに保存。 |
| 8 | Word Recall | 確認 | EN→JP／JP→ENの四択を実操作。正答・誤答・Unknown、4候補の解説表示を確認。選択位置ではなくCorpus IDを保存。Reviewで候補表示順が変わっても既存の4候補IDと判定が保たれることをAPI・DBで確認。 |
| 9 | Cloze | 確認 | abandonの空欄へ`ABANDON`を入力し、正答判定と入力原文の保存を確認。Unknownも確定。6,100件すべてのrune位置・長さ・単語境界を検査し、有効6,100件／除外0件。 |
| 10 | Reading Review | **共通操作・保存確認／実機タッチ未検証** | WORD＋EXAMPLE、JP＋EXAMPLEの両形式を確認。PCキー入力と左右ポインタードラッグが同じ自己評価へ接続。短いドラッグは未回答、十分な左右ドラッグもSubmit前は未確定。保存後は`recalled=1/0`、`correct=NULL`。実機スマートフォンのタッチスワイプは未検証。 |
| 11 | Confidence | 確認 | 未操作NULL、明示0、100を実操作で保存。回答結果表示後にスライダーを無効化。APIでもNULLと0、範囲外の拒否を検証。幅を340からPC幅へ戻しても未確定の自己評価とConfidence 0を保持。 |
| 12 | Session | 確認 | 次へ／前へだけでは回答数0。Unknownを1回答と数え、全位置で確定後にCompleted。回答位置の順序を変えた場合と再回答追記もAPIで検証。 |
| 13 | Review | 確認 | 過去のUnknownからall rightを再出題。areaを誤答→正答した後でもIncorrect＋Aで対象1件。APIで新Question Specを生成しないこと、元のAttempt保持、結果・日時・Confidenceが同じAttemptへ適用されることを確認。 |
| 14 | Resume | 確認 | Session 00000003を未回答で離れ、ReviewのResumeから同じ問題へ復帰。APIでは同じSession ID・順序・現在位置、Spec／Attemptの増分0、実プロセス再起動後の永続性を確認。 |
| 15 | Collection | 確認 | Vocabularyへ同じIDを再追加すると警告。明示続行で2登録となり、片方だけ削除して1登録を保持。Question CollectionのartisticをReviewへ接続。APIで両種の重複出題回数、個別削除、既存Spec再利用を確認。 |
| 16 | Action Log | 確認 | 送信待ち表示→画面遷移等で送信→Data表示を実操作。回答Actionと11件の実ブラウザーAttemptを照合。APIでは5イベントのバッチ、発生・送信時刻の分離、再送重複防止を確認。DBは指定9カラム。 |
| 17 | Data | 確認 | 回答・操作履歴、日時、正誤、Unknown、自己評価、NULL／0／100 Confidenceを表示。Unknown条件で2件へ絞り込み。11回答中、客観問題8件・正答5件で62.5%、Reading3件を分母に含めない。概算時間と終了不明1区間を区別。 |
| 18 | 保留機能 | 確認 | Sentence GenerationはComing soon...のみ。APIでもこの形式のSession作成を拒否。SRS・忘却曲線・GPT採点・完全オフライン同期を完成条件へ混入させていない。 |

## 3. 実ブラウザーで確認した具体例

- **検索**：全角`ＢＡＮＫ`の元入力を保持し、7件を取得。B指定でbank (money)とbank (river)の2件、X指定で0件。
- **回答と移動**：2問Sessionで前後移動だけを行い、0 answeredを維持。all rightをUnknown・Confidence未設定、artisticを正答・Confidence 0として確定するとCompleted。
- **逆方向**：`心配している`からanxiousを選び、Confidence 100で正答。
- **復習と再開**：Unknown＋Aでall rightをReviewし、中断後にResume。既存の4候補を使用し、新しい位置順でも正答を判定。
- **Cloze**：`We had to ________ the walk when the river covered the path.`に`ABANDON`を入力し、正答を確認。別SessionではUnknownを確定。
- **Reading**：PCの左キーで「できなかった」を選択しEnterで確定。JP＋EXAMPLE形式では、25 CSS px程度の短いドラッグは評価を選択せず、約200 CSS pxの左右ドラッグで評価を選択。Submit後だけ回答数が増加。
- **誤答履歴の保持**：areaでstyleを選んだ誤答と、再回答の正答を両方保存。後から正解しても過去のIncorrectで検索可能。
- **Collection**：Vocabularyの同一IDを警告後に重複登録し、登録項目ID単位で削除。Question Collectionからartisticの既存問題へ接続。
- **Data**：回答履歴と操作履歴を切り替え、`session_start`で9件の開始Actionへ絞り込み。回答日時は送信時刻に置き換わらず、回答ActionをAttemptへ対応付けて表示。

ブラウザー試験用DBから、確定Attempt、対応Action、Session、Question Spec参照を読み取った記録が`test-results/browser-db-evidence.json`です。**試験用DBそのものは配布しません。** 配布後の学習履歴は初回起動で新規作成されます。

## 4. 検証中に修正し、再確認した箇所

- 問題移動の遅い応答が、移動先の別ViewをQuizに戻さないようにした。
- 遅い回答成功／失敗が、新しく表示した問題の未確定回答を上書きしないようにした。
- Collectionから練習するときと選択中Collectionを削除したときに、古い画面の条件が復活しないようにした。
- NEWで使わないReview専用の入力を、NEWの開始エラーにしないようにした。
- 問題数変更のblurから不要な候補再取得を除き、直後の最初のStartクリックを実ブラウザーで確認した。
- 起動完了時にHeroの入力DOMを作り直さず、再読込直後の入力・Enterが失われないようにした。
- Dataの操作フィルターのイベント名を、実際に保存される`session_start`等へ合わせた。

最初の4点を含む6群の非同期制御は、実際のFrontend関数と制御したPromise／DOM代替で再現しました。この結果を実ブラウザー試験と混同しないため、証跡を分けています。

## 5. 未実装・未検証の範囲

### 仕様どおり実装対象外

Sentence Generationの出題本体と採点、GPT採点、SRS、忘却曲線、完全オフライン回答・同期、複数Corpus、厳密な複数端末の同時編集競合解決。

### 実装はあるが、実機／配備環境で未検証

- iPhone／Androidの実機タッチ、Mobile Safari、端末回転、仮想キーボード表示時の使い勝手。
- 物理ゲームパッドの標準マッピング、長押し、スティック操作。
- macOS／Windowsでの実行。同梱の実行ファイルはLinux x86-64用で、macOSではソースからビルドする。
- 利用者のTailscale Tailnet上での接続と複数端末からの利用。
- 実機でのPWAインストール。manifest、アイコン、Service Workerの配信実装と、実機インストールの成否は別。

強制終了や端末停止時に未送信の操作ログが一切失われない保証はありません。完全オフライン同期は対象外です。回答保存に失敗した場合は成功として扱わず、保存結果を確認する導線を表示します。

## 6. 再実行

プロジェクトルートで以下を実行します。GoビルドにはGo 1.24以降とCコンパイラーが必要です。API検証はPython標準ライブラリーのみ、Frontend制御検証にはNode.jsを使います。

```sh
CGO_ENABLED=1 go test -buildvcs=false -mod=vendor -count=1 -v ./...
CGO_ENABLED=1 go build -buildvcs=false -mod=vendor -trimpath -tags sqlite_omit_load_extension -o bin/oxford-core .
python3 tests/verify_api.py --binary ./bin/oxford-core --corpus ./data/all_with_ID.csv --output ./test-results/api-verification.json
node tests/frontend_control_flow_regression.cjs
```

API検証は一時DBで実行し、通常の学習DBには書き込みません。実ブラウザー手順は上記の具体例を用いて再現できます。自動検証の成功を、未検証の実機環境の成功とは解釈しないでください。

## 7. 配布版の識別

| 対象 | SHA-256 |
|---|---|
| 固定Corpus CSV | `9c871e809b082836339c665aa611476602301f3e6425eca0ded34185d8093f72` |
| 本番HTML | `caaf7914bc2c8673d2979ed43c05cb7de1e5cad5cde96f1883b36208a63a3b95` |
| 同梱Linuxバイナリ | `59fd861d8abdd4614a29e9131e92f6035b64099bacad90037acb79423c59325b` |

現在の本番HTML 128,443 bytesが、同梱バイナリに完全一致で埋め込まれていることを確認済みです。その他のファイルは`SHA256SUMS.txt`で確認できます。
