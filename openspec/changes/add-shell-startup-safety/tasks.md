## 1. 配色の固定

- [x] 1.1 `frontend/src/index.css` (更新) から `@media (prefers-color-scheme: dark)` のブロックを消す。完了判定: 差分を目視で確認する
- [x] 1.2 `frontend/src/index.css` (更新) に `:root { color-scheme: light; }` を足す (design.md D1)。完了判定: `cd frontend && npm run build:dev` が通る
- [x] 1.3 `frontend/e2e/shell.spec.ts` (新設) に describe「配色の固定」を足す (`test.use({ colorScheme: "dark" })` で OS のダークモードを再現する)
    - OS がダークモードでも見出しは暗い文字で表示される
    - OS がダークモードでも入力欄の文字は暗い色で表示される
    - 完了判定: `task test:e2e` で上記が pass する

## 2. 控えと起動処理 (リポジトリ層)

- [x] 2.1 `internal/storage/sqlite/backup.go` (新設) に `pendingMigrations(db *sql.DB) (current int64, pending bool, err error)` を書く (design.md D4)。
  完了判定: `go build .` が通る
- [x] 2.2 `internal/storage/sqlite/backup.go` (更新) に `backupBeforeMigrate(db *sql.DB, dbPath string, version int64, now time.Time) (string, error)` を書く
    - 書き出し先は `<dbPath のディレクトリ>/backups/meal-cycle-v<version>-<YYYYMMDD-HHMMSS>.db`
    - `VACUUM INTO` で `.partial` 付きの名前に書き出し、成功してから改名する (design.md D3)
    - 完了判定: `go build .` が通る
- [x] 2.3 `internal/storage/sqlite/backup.go` (更新) に `pruneBackups(dir string, keep int) error` と定数 `backupKeep = 5` を書く。
  `.partial` の残りもここで消す。完了判定: `go build .` が通る
- [x] 2.4 `internal/storage/sqlite/backup_test.go` (新設) にテストを書く
    - 最新の版の DB では `pendingMigrations` が false を返す
    - migration を途中の版まで当てた DB では `pendingMigrations` が true を返す
    - `backupBeforeMigrate` が作ったファイルを開くと、更新前の行が読める
    - WAL に残った書き込みも控えに含まれる
    - 控えが 5 個あるとき、6 個目を作って `pruneBackups` を呼ぶと、いちばん古い 1 個が消える
    - `.partial` のファイルは `pruneBackups` で消える
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'Backup|Pending'` が pass する
- [x] 2.5 `internal/storage/sqlite/startup.go` (新設) に `StartupError{Stage StartupStage; Err error}` と段階の定数 (`open` / `backup` / `migrate`) を書く。
  完了判定: `go build .` が通る
- [x] 2.6 `internal/storage/sqlite/startup.go` (更新) に `OpenAndMigrate(path string) (*sql.DB, error)` を書く (design.md D6)
    - `os.Stat` で DB の有無を見てから `Open` する
    - DB が既にあり、更新があるときだけ `backupBeforeMigrate` と `pruneBackups` を呼ぶ
    - 控えに失敗したら `Migrate` を呼ばずに `StartupError{Stage: backup}` を返す
    - 完了判定: `go build .` が通る
- [x] 2.7 `internal/storage/sqlite/startup_test.go` (新設) にテストを書く
    - should create no backup when the database file does not exist yet
    - should create no backup when no migration is pending
    - should create one backup and migrate when migrations are pending
    - should return a backup-stage error and leave the schema version unchanged when the backups directory is not writable
    - should return an open-stage error when the path is a directory
    - 完了判定: `go test ./internal/storage/sqlite/ -run OpenAndMigrate` が pass する

## 3. サービス層と起動

- [x] 3.1 `internal/service/health.go` (更新) の `Health` に `StartupError` / `StartupDetail` / `DBPath` / `BackupDir` を足す。完了判定: `go build .` が通る
- [x] 3.2 `internal/service/health.go` (更新) の `NewHealthService` を `NewHealthService(db *sql.DB, startup StartupState)` に変え、
  `db` が nil でも `Ping` が落ちないようにする。完了判定: `go build .` が通る
- [x] 3.3 `internal/service/health_test.go` (新設) にテストを書く
    - should report dbConnected and no startupError when startup succeeded
    - should report the stage, detail, dbPath and backupDir when startup failed
    - should not panic when db is nil
    - 完了判定: `go test ./internal/service/ -run TestHealthService` が pass する
- [x] 3.4 `main.go` (更新) の `Open` と `Migrate` の呼び出しを `sqlite.OpenAndMigrate` に置き換え、`log.Fatalf` での終了をやめる。完了判定: `go build .` が通る
- [x] 3.5 `main.go` (更新) で、起動に失敗したときは `HealthService` だけを `application.Options.Services` に登録する (design.md D2)。完了判定: `go build .` が通る
- [x] 3.6 `main.go` (更新) の「CSS の font-size は意図的に触らず」のコメントを、`index.css` が `html` に 18px を指定している実態に合わせる。完了判定: 差分を目視で確認する
- [x] 3.7 `cmd/e2e-server/main.go` (更新) の `service.NewHealthService(db)` を新しい引数に合わせる。完了判定: `go test ./cmd/...` が pass する
- [x] 3.8 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: `frontend/bindings/` の `Health` に 4 項目が増えている

## 4. エラー画面

- [x] 4.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` (更新) に `startup_error.title`、`startup_error.stage.open` / `backup` / `migrate`、
  `startup_error.detail`、`startup_error.db_path`、`startup_error.backup_dir`、`startup_error.hint` を足す。完了判定: 両言語で同じキーがそろっている
- [x] 4.2 `frontend/src/components/layout/startup-error-screen.tsx` (新設) に、段階の見出し、詳細 (等幅)、DB の場所、控えの場所を出す画面を書く。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/components/layout/root-layout.tsx` (更新) で `HealthService.Ping()` を `useQuery({ queryKey: ["health"] })` で読み、
  `startupError` があれば `StartupErrorScreen` だけを描く。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.4 `frontend/src/components/layout/sidebar.tsx` (更新) の `useEffect` での `Ping` を、同じ `["health"]` のクエリに置き換える。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.5 `frontend/e2e/bindings-shim/healthservice.ts` (更新) を `Health` の項目の追加に合わせる。完了判定: `task test:e2e` の既存テストが pass する

## 5. ドキュメント

- [x] 5.1 `CLAUDE.md` (更新) の「Wails 固有の注意点」にある「CSS の `font-size` は意図的に触っていない」を、`html` に 18px を指定している実態に合わせる。完了判定: 差分を目視で確認する
- [x] 5.2 `docs/design.md` (更新) §5 に「更新前の控え」の節を足す (場所、ファイル名、残す個数、復元の手順)。完了判定: 差分を目視で確認する
- [x] 5.3 `docs/product.md` (更新) §4 共通シェルに、配色の固定と起動時のエラー画面を足す。完了判定: 差分を目視で確認する
- [x] 5.4 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R24 と R26 にチェックを入れ、`add-shell-startup-safety` で対応した旨を追記する。完了判定: 差分を目視で確認する

## 6. 検証

- [x] 6.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 6.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 6.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 6.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 6.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [ ] 6.6 `task build:win` で作った `.exe` を、`%APPDATA%\meal-cycle\meal-cycle.db` を同名のフォルダに置き換えた状態で起動する。
  完了判定: エラー画面に「データベースを開けませんでした」と DB の場所が出ることを目視で確認する
- [ ] 6.7 Windows の設定をダークモードにして `.exe` を起動する。完了判定: 4 画面の見出しと入力欄の文字が読めることを目視で確認する
- [x] 6.8 `openspec validate add-shell-startup-safety --strict` を実行する。完了判定: エラー 0 件
