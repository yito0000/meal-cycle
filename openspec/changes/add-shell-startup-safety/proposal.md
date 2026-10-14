## Why

アプリの土台に、利用者が自分では直せない穴が 2 つある (docs/reviews/2026-10-14-design-ux-review.md R24、R26)。

1 つめは配色である。
`index.css` は OS がダークモードのとき `body` の文字を白に変えるが、画面の部品はすべて明るい背景を固定している。
色を指定していない文字 (各画面の見出し、入力欄の文字、テーブルのセル) だけが白くなり、明るい背景の上で読めなくなる。
ダークテーマを作る予定は無い。

2 つめは起動時の失敗である。
データベースを開けない、またはスキーマの更新に失敗すると、アプリはログを出して終了する。
配布用ビルドにはコンソールが無いので、利用者から見ると「ダブルクリックしても何も起きない」になる。
しかもスキーマの更新は、起動のたびに本番のデータベースへ直接当たる。
更新が途中で壊れても、戻せる控えがどこにも無い (前回レビューの R18)。

`add-mealplan-entry-id` で献立のテーブルを作り直す前に、この 2 つを塞いでおく。

## What Changes

- 配色を明るいテーマに固定する。OS がダークモードでも、文字と背景は変わらない
- スキーマの更新がある起動では、更新の前にデータベースの控えを作る
  - 控えはデータベースと同じ場所の `backups/` に置き、新しいものから 5 個だけ残す
  - 初回起動と、更新の無い起動では作らない
  - 控えを作れなかったときは、更新を当てずに止める
- 起動時にデータベースを開けない、控えを作れない、更新に失敗した、のいずれかが起きたら、アプリのウィンドウにエラー画面を出す
  - 何が起きたか、データベースの場所、控えの場所を示す
  - 通常の画面 (サイドバーと各ページ) は出さない
- CLAUDE.md と `main.go` のコメントにある「CSS の font-size は触っていない」を実態に合わせる (R35 の一部)

### Non-goals

- **ダークテーマ**：作らない
- **控えからの復元の UI**：エラー画面は控えの場所を示すだけで、復元はファイルのコピーで行う。画面からの復元は S09 で扱う
- **任意のタイミングでの書き出し**：「バックアップを書き出す」ボタンは S09 (設定) で扱う
- **控えの個数の設定**：5 個は実装上の上限として固定する。S09 の可変値には入れない

## Capabilities

### New Capabilities

- `shell`：全画面に共通する土台 (配色、起動時の失敗の見せ方、データベース更新前の控え) を定める

### Modified Capabilities

(なし)

## Impact

### 新設ファイル

- `internal/storage/sqlite/startup.go`：`OpenAndMigrate` と `StartupError` (段階つきのエラー)
- `internal/storage/sqlite/backup.go`：`backupBeforeMigrate`、`pendingMigrations`、`pruneBackups`
- `internal/storage/sqlite/startup_test.go` / `internal/storage/sqlite/backup_test.go`
- `frontend/src/components/layout/startup-error-screen.tsx`：起動時のエラー画面
- `frontend/e2e/shell.spec.ts`：配色のシナリオ
- `openspec/specs/shell/spec.md`：archive 時に delta から作られる (手で直接は触らない)

### 更新ファイル

- `frontend/src/index.css`：`prefers-color-scheme: dark` のブロックを消し、`:root { color-scheme: light; }` を足す
- `main.go`：`log.Fatalf` での終了をやめ、`sqlite.OpenAndMigrate` の結果を `HealthService` に渡す。
  失敗したときは `HealthService` だけを登録する。font-size のコメントを直す
- `internal/service/health.go`：`Health` に `startupError` (段階のコード)、`startupDetail`、`dbPath`、`backupDir` を足す
- `internal/service/health_test.go` (新設)：起動時の状態が `Ping` にそのまま出ることを確かめる
- `cmd/e2e-server/main.go`：`service.NewHealthService` の引数の変更に合わせる
- `frontend/src/components/layout/root-layout.tsx`：`HealthService.Ping()` を TanStack Query で読み、起動時のエラーがあればエラー画面だけを出す
- `frontend/src/components/layout/sidebar.tsx`：自前の `useEffect` での `Ping` を、`root-layout.tsx` と同じクエリに置き換える
- `frontend/e2e/bindings-shim/healthservice.ts`：`Health` の項目の追加に合わせる
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`startup_error.*` を足す
- `CLAUDE.md`：「Wails 固有の注意点」の font-size の記述を直す
- `docs/design.md`：§5 に「更新前の控え」の節を足す
- `docs/product.md`：§4 共通シェルに、配色の固定と起動時のエラー画面を足す
- `docs/reviews/2026-10-14-design-ux-review.md`：R24、R26 にチェックを入れる

### 生成物 (再生成のみ)

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`Health` の項目が増える

### migration

- なし

### 前提と順序

- ほかの change に依存しない。`add-mealplan-entry-id` (テーブルを作り直す migration を含む) より先に apply する

### 後続 proposal への引き継ぎ

- S09 (設定)：控えの場所を開く、任意のタイミングで書き出す、控えから復元する、の 3 つをここで作った `backups/` の上に載せる
- `add-mealplan-entry-id`：migration 00009 を当てる起動で、この change の控えが働く
