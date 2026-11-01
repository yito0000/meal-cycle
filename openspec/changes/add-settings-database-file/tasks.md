## 1. リポジトリ層: 書き出しと控えの一覧

- [ ] 1.1 `internal/storage/sqlite/backup.go` を更新し、控えのファイル名から版と日時と復元前かを読む `parseBackupName` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 1.2 `internal/storage/sqlite/restore.go` を新設し、`ExportTo(db, dest)` を `VACUUM INTO` (`.partial` → 改名) で書く。完了判定: `go build ./internal/...` が通る
- [ ] 1.3 `internal/storage/sqlite/restore.go` に `ListBackups(dbPath)` を書き、新しい順に返す。完了判定: `go build ./internal/...` が通る
- [ ] 1.4 `internal/storage/sqlite/restore_test.go` を新設する。完了判定: `go test ./internal/storage/sqlite/ -run 'TestExportTo|TestListBackups'` が pass する
  - ExportTo は WAL にだけある変更も含めて書き出す
  - ExportTo は書き込めない場所では .partial を残さずエラーを返す
  - ListBackups は更新前の控えと復元前の控えを新しい順に返す
  - ListBackups は backups フォルダが無いと空を返す

## 2. リポジトリ層: 予約と起動時の適用

- [ ] 2.1 `internal/storage/sqlite/restore.go` に `ValidateRestoreSource(path)` を design.md D2 の手順で書き、`ErrRestoreNotDatabase` / `ErrRestoreTooNew` を定義する。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.2 `internal/storage/sqlite/restore.go` に `ReserveRestore(dbPath, src)` / `PendingRestore(dbPath)` / `CancelRestore(dbPath)` を書く (design.md D1)。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.3 `internal/storage/sqlite/restore.go` に `applyPendingRestore(dbPath, now)` を design.md D3 の手順で書く。完了判定: `go build ./internal/...` が通る
- [ ] 2.4 `internal/storage/sqlite/startup.go` に `StageRestore` を足し、`OpenAndMigrate` の最初で `applyPendingRestore` を呼ぶ。完了判定: `go build ./internal/...` が通る
- [ ] 2.5 `internal/storage/sqlite/restore_test.go` にケースを足す。完了判定: `go test ./internal/storage/sqlite/ -run 'TestValidateRestoreSource|TestReserveRestore'` が pass する
  - SQLite でないファイルは ErrRestoreNotDatabase
  - goose の版の表が無い SQLite は ErrRestoreNotDatabase
  - 埋め込んだ最大より新しい版は ErrRestoreTooNew
  - 古い版は予約できる
  - 予約は元のファイルを消しても残る
  - 2 回予約すると後の予約に置き換わる
  - CancelRestore で PendingRestore が空になる
- [ ] 2.6 `internal/storage/sqlite/startup_test.go` にケースを足す。完了判定: `go test ./internal/storage/sqlite/ -run TestOpenAndMigrate` が pass する
  - 予約があると今の DB を復元前の控えに書き出してから置き換え、予約を消す
  - 古い版で予約すると復元前の控えと更新前の控えの両方を作って最新にする
  - backups に書き込めないと StageRestore を返し、DB と予約をそのまま残す
  - 予約が無ければ従来どおり動く

## 3. サービス層

- [ ] 3.1 `internal/service/filepicker.go` を新設し、`FilePicker` interface と Wails 実装 `NewWailsFilePicker` を書く (design.md D5)。完了判定: `go build ./internal/...` が通る
- [ ] 3.2 `internal/service/database.go` を新設し、`DatabaseService` と `Export` / `ListBackups` / `ReserveFromBackup` / `ReserveFromFile` / `Pending` / `CancelPending` / `Quit` を書く (design.md D4)。
  完了判定: `go build ./internal/...` が通る
- [ ] 3.3 `internal/service/database_test.go` を新設し、BDD 風に書く。完了判定: `go test ./internal/service/ -run TestDatabaseService` が pass する
  - should write the database to the picked path when export is called
  - should do nothing when the save dialog is closed
  - should reject a backup name that is not in the backup list
  - should reserve the picked file when it is a valid database
  - should return the pending restore after reserving
- [ ] 3.4 `main.go` の `application.Options.Services` に `service.NewDatabaseService(db, dbPath, service.NewWailsFilePicker())` を足す。完了判定: `go build .` が通る
- [ ] 3.5 `cmd/e2e-server/main.go` に固定のパスを返す `fixedFilePicker` を足し、`DatabaseService` を登録する。完了判定: `go build ./cmd/...` が通る
- [ ] 3.6 `cmd/e2e-server/dispatch_test.go` の登録するサービスの数を 1 つ増やす。完了判定: `go test ./cmd/...` が pass する
- [ ] 3.7 `wails3 generate bindings -clean=true -ts -i` で bindings を作り直す。完了判定: `cd frontend && npm run type-check` が通る

## 4. 画面

- [ ] 4.1 `frontend/e2e/bindings-shim/databaseservice.ts` を新設し、`frontend/e2e/bindings-shim/index.ts` に足す。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.2 `frontend/src/components/settings/use-database.ts` を新設する。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.3 `frontend/src/components/settings/restore-dialog.tsx` を新設し、控えの一覧、「ファイルを選ぶ」、確認ダイアログを描く。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.4 `frontend/src/components/settings/database-section.tsx` を新設し、書き出し、復元、予約中の表示を描く (design.md D6)。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.5 `frontend/src/routes/settings.tsx` の最後にデータの節を足す。保存ボタンの対象には含めない。完了判定: `task dev` の S09 で目視
- [ ] 4.6 `frontend/src/components/layout/startup-error-screen.tsx` に `restore` の段階の見出しを足す。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.7 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に文言を足す。完了判定: `cd frontend && npm run type-check` が通る

## 5. E2E

- [ ] 5.1 `frontend/e2e/settings-database.spec.ts` を新設する。完了判定: `task test:e2e` で pass する
  - describe「データの書き出し」
    - データを 1 つのファイルに書き出す
  - describe「復元の予約」
    - 書き出したファイルを選んで復元を予約する
    - データのファイルでないものは予約できない
  - describe「復元の予約の取り消しと終了」
    - 予約を取り消す
    - 予約したあとに S09 を開き直しても予約中と表示される
- 起動時の適用 (shell の Scenario) は 2.6 の Go のテストで確かめ、E2E には含めない

## 6. ドキュメント

- [ ] 6.1 `docs/wireframes/S09_settings.drawio.svg` にデータの節を足す (可視 SVG と編集 XML の両方)。完了判定: 目視
- [ ] 6.2 `docs/wireframes/S09a_restore_dialog.drawio.svg` を新設する。完了判定: 目視
- [ ] 6.3 `docs/wireframes/README.md` に S09a を足す。完了判定: 目視
- [ ] 6.4 `docs/product.md` の §4 S09 にデータの節を足す。完了判定: 目視
- [ ] 6.5 `docs/design.md` の §5「更新前の控え」の復元の段落を画面からの復元に書き換え、§6 のディレクトリ構成に `database.go` と `filepicker.go` を足す。完了判定: 目視
- [ ] 6.6 `docs/design.md` の §9「設定 UI（S09）の残り」から、控えからの復元を実装済みに直す。完了判定: 目視

## 7. 確認

- [ ] 7.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [ ] 7.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [ ] 7.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [ ] 7.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [ ] 7.5 `task test:e2e` を実行する。完了判定: 全 pass
- [ ] 7.6 `task lint` を実行する。完了判定: エラー 0
- [ ] 7.7 Windows 実機で、書き出し → 献立を変える → 書き出したファイルで予約 → 今すぐ終了 → 起動し直す、の順で復元を確かめる。完了判定: 目視
