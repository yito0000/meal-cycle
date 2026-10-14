## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/shell/spec.md](specs/shell/spec.md) を参照。

現状は次のとおりである。

- `frontend/src/index.css` は `@media (prefers-color-scheme: dark)` で `body` の `background-color` と `color` だけを変える。
  部品は `bg-white` / `bg-neutral-50` を固定しており、`text-*` を持たない要素は `body` の白い文字色を継承する
- `main.go` は `sqlite.DefaultPath` → `sqlite.Open` → `sqlite.Migrate` の順に呼び、どれかが失敗すると `log.Fatalf` で終了する
- 配布用ビルド (`build/windows/Taskfile.yml`) は `-H windowsgui` を付けるので、標準エラー出力は利用者に見えない
- `HealthService.Ping()` は DB への疎通だけを返し、サイドバーの下端に「DB OK」と出している
- DB は WAL モードで開いている。DB 本体のファイルだけをコピーしても、WAL に残った書き込みは含まれない

## Goals / Non-Goals

**Goals:**

- OS の配色設定に関係なく、全画面が読める
- 起動時の失敗を、利用者がウィンドウの中で読める
- スキーマの更新が壊れても、更新前のデータが 1 ファイルで残っている

**Non-Goals:**

- 復元や書き出しの UI (S09)
- 起動時以外のエラーの見せ方

## Decisions

### D1. 配色の固定のしかた

**選択肢:**

- (a) `dark` のブロックを消し、`:root { color-scheme: light; }` を置く ← 採用
  — Pros: 数行で済む。`color-scheme` で日付入力やスクロールバーも明るい側にそろう
  / Cons: OS がダークモードの利用者には、アプリだけが明るく見える
- (b) 全部品に `text-*` を明示する
  — Pros: 継承に頼らなくなる
  / Cons: 部品を足すたびに付け忘れる。`body` の背景は黒のまま残る
- (c) ダークテーマを作る
  — Pros: OS の設定に合う
  / Cons: 作る予定が無い (利用者の判断)

**採用: (a)**。

### D2. 起動時の失敗をどこに出すか

**選択肢:**

- (a) ネイティブのメッセージダイアログ (`app.Dialog.Error()`) を出して終了する
  — Pros: フロントに手を入れない
  / Cons: 文言を Go 側に持つことになり、「Go は日本語メッセージを持たない」方針 (docs/design.md §7) に反する。場所のパスをコピーしにくい
- (b) ウィンドウは通常どおり開き、フロントがエラー画面を出す ← 採用
  — Pros: 文言は i18n に載る。パスを選択してコピーできる。`HealthService` という既存の経路を使える
  / Cons: DB が無い状態でほかのサービスを登録できないので、登録の分岐が要る
- (c) ログファイルに書いて終了する
  — Pros: 実装が最小
  / Cons: 利用者はログファイルの存在を知らない

**採用: (b)**。
失敗したときは `HealthService` だけを登録する。
フロントは `RootLayout` で `Ping` の結果を見て、起動時のエラーがあればエラー画面だけを描く。
ルート配下のページは描かないので、登録していないサービスが呼ばれることはない。

`Health` には、段階を表すコード (`open` / `backup` / `migrate`) と、Go のエラー文字列 (詳細) を別々に載せる。
フロントはコードから見出しを引き、詳細はそのまま等幅で出す。

### D3. 控えの取り方

**選択肢:**

- (a) `VACUUM INTO '<path>'` で、整合の取れた 1 ファイルを書き出す ← 採用
  — Pros: WAL の中身を含んだ状態で 1 ファイルになる。SQL 1 文で済み、Cgo も追加の依存も要らない
  / Cons: DB 全体を読み直すので、ファイルコピーよりは遅い (数 MB の DB では体感できない)
- (b) `PRAGMA wal_checkpoint(TRUNCATE)` のあとで DB 本体のファイルをコピーする
  — Pros: 速い
  / Cons: チェックポイントとコピーのあいだに書き込みが入ると壊れる。起動時は書き込みが無いとはいえ、前提がコメント頼みになる
- (c) SQLite の Online Backup API を使う
  — Pros: 本来の手段
  / Cons: `database/sql` 越しでは呼べず、ドライバ固有の接続を取り出す必要がある

**採用: (a)**。
`VACUUM INTO` は書き出し先が既にあると失敗するので、ファイル名に秒までの日時を入れる
(`meal-cycle-v<版>-<YYYYMMDD-HHMMSS>.db`)。

### D4. 「更新がある」の判定

`goose.GetDBVersion` で現在の版を取り、`goose.CollectMigrations("migrations", current, goose.MaxVersion)` が 1 件以上返れば更新ありとする。
初回の起動かどうかは、`sqlite.Open` の前に DB ファイルの有無を `os.Stat` で見て決める。
他案 (現在の版が 0 なら初回とみなす) は、`goose_db_version` の表を作る副作用のあとでしか判定できず、順序が絡むので見送る。

### D5. 控えを何個残すか

5 個を `backup.go` の定数にする。
`internal/config/defaults.go` には置かない。
利用者が画面から変えたい値ではなく、ディスクを食い潰さないための実装上の歯止めだからである
(レシピのインポート件数の上限と同じ扱い)。

### D6. 処理の置き場所

`sqlite.OpenAndMigrate(path string) (*sql.DB, error)` に、開く、判定する、控えを取る、更新する、をまとめる。
失敗は `*StartupError{Stage, Err}` で返し、`main.go` は `errors.As` で段階を取り出して `HealthService` に渡す。
`cmd/e2e-server` は一時ディレクトリの新しい DB しか開かないので、既存の `Open` + `Migrate` のままにする。

## Risks / Trade-offs

- [エラー画面の配信に失敗する (フロントの資産が壊れている) と、やはり何も見えない] → ログへの出力は残す。資産の破損は DB の失敗とは別の事故として扱う
- [`VACUUM INTO` の途中で落ちると、不完全な控えが残る] → 一時的な名前 (`.partial`) で書き出し、成功してから改名する。起動時に `.partial` を見つけたら消す
- [migrate に失敗した DB は途中の版で止まる] → goose は migration 1 本を 1 トランザクションで当てるので、失敗した 1 本は巻き戻る。
  エラー画面に控えの場所を出し、復元はファイルのコピーで行う
- [起動時の失敗は E2E で再現しにくい] → リポジトリ層のテスト (壊れた migration、書き込めない控えフォルダ) とサービス層のテストで確かめ、
  画面は Windows 実機で 1 度だけ目視する (tasks.md)

## Migration Plan

スキーマの変更は無い。
この change を入れた版の最初の起動では更新が無いので、控えは作られない。
次にスキーマを変える change (`add-mealplan-entry-id`) の最初の起動で、初めて控えが作られる。
