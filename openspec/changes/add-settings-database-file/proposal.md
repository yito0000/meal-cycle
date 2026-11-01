## Why

meal-cycle のデータは PC の 1 つの DB ファイルにしか無い。
PC の買い替えや故障に備えて手元に写しを残す手段は、今はレシピの書き出し (add-recipe-export) だけで、献立、在庫、常備品、設定は残せない
(docs/reviews の R18「バックアップの手段がない」)。

スキーマ更新前の控え (`backups/`) は自動で作られるが、戻すにはアプリを終了し、エクスプローラーで `%APPDATA%` を開いて、
WAL のファイルを消してから控えをコピーする必要がある ([design.md §5](../../../docs/design.md)「更新前の控え」)。
利用者がこの手順を覚えている前提は置けない。

[design.md §5](../../../docs/design.md) と [§9](../../../docs/design.md) は「画面からの復元と任意のタイミングでの書き出しは、S09 に節を足す別の proposal で扱う」としている。
本 proposal がその proposal である。

## What Changes

- S09 の最後に「データ」の節を足す
- 「データを書き出す」：OS の保存ダイアログで場所を選び、DB 全体を 1 つの `.db` ファイルに書き出す
  - 既定のファイル名は `meal-cycle-<YYYYMMDD-HHMMSS>.db`
  - 書き出したら場所を通知で知らせる。ダイアログを閉じたときは何もしない
- 「復元する」：復元に使うファイルを選ぶダイアログを開く
  - `backups/` の控えを新しい順に一覧し (作成日時と、どの版の更新の前か)、1 つを選べる
  - 「ファイルを選ぶ」で、書き出したファイルなど任意の `.db` ファイルを選べる
  - 選ぶと確認ダイアログを出し、確認すると**次の起動で復元する予約**をする。今のデータはその場では変えない
- 予約すると、S09 に予約中である旨と「予約を取り消す」「今すぐ終了する」を出す
- 次の起動では、DB を開く前に予約を当てる
  - 今の DB を `backups/` に控えとして書き出してから、予約したファイルで置き換える
  - 置き換えたファイルのスキーマが古ければ、いつもの起動と同じく控えを作ってから更新する
  - 予約を当てられなかったときは、今の DB を変えずに起動時のエラー画面を出す
- 予約の時点で、選んだファイルが meal-cycle の DB として開けるか、このアプリより新しい版でないかを確かめ、だめなら予約しない

### Non-goals

- **起動中の DB の差し替え**：サービスが DB の接続を持ったまま動いているので、ファイルを差し替えると接続と中身が食い違う。再起動を挟む
- **アプリの自動の再起動**：Wails v3 にプロセスを起動し直す API は無く、自前で exe を起動し直すと Windows で二重起動の扱いが要る。終了だけを行い、起動は利用者に任せる
- **書き出しの自動化 (毎週など)**：定期的な実行はアプリを起動していない間に動けない。必要になったら起動時に書き出す形で別の proposal にする
- **一部のデータだけの書き出しと取り込み (献立だけ、在庫だけ)**：形式を決める必要がある。全体の書き出しで足りないと分かってから考える
- **控えの削除や名前の変更を画面から行うこと**

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `settings`：S09 にデータの節を足し、書き出しと復元の予約を行う
- `shell`：起動時に復元の予約を当てる。予約を当てられないときを起動時の失敗に加える

## Impact

**新設ファイル**

- `internal/storage/sqlite/restore.go`：`ExportTo` / `ListBackups` / `ValidateRestoreSource` / `ReserveRestore` / `CancelRestore` / `PendingRestore` / `applyPendingRestore`
- `internal/storage/sqlite/restore_test.go`
- `internal/service/database.go`：`DatabaseService` (書き出し、控えの一覧、復元の予約と取り消し、終了)
- `internal/service/database_test.go`
- `internal/service/filepicker.go`：保存先と開くファイルを選ぶ `FilePicker` interface と Wails 実装
- `frontend/src/components/settings/database-section.tsx`：S09 のデータの節
- `frontend/src/components/settings/restore-dialog.tsx`：控えの一覧、ファイルを選ぶ、確認
- `frontend/src/components/settings/use-database.ts`：TanStack Query の hook
- `frontend/e2e/bindings-shim/databaseservice.ts`
- `frontend/e2e/settings-database.spec.ts`

**更新ファイル**

- `internal/storage/sqlite/startup.go`：`OpenAndMigrate` の最初で `applyPendingRestore` を呼ぶ。`StageRestore` を足す
- `internal/storage/sqlite/startup_test.go`：予約を当てる起動のテストを足す
- `internal/storage/sqlite/backup.go`：控えのファイル名を読む関数を `ListBackups` と共有する。復元前の控えも `pruneBackups` の数に入れる
- `main.go`：`application.Options.Services` に `service.NewDatabaseService(db, dbPath, service.NewWailsFilePicker())` を足す
- `cmd/e2e-server/main.go`：`DatabaseService` を固定の保存先と開くファイルを返す picker で登録する
- `cmd/e2e-server/dispatch_test.go`：登録するサービスの数を 1 つ増やす
- `frontend/e2e/bindings-shim/index.ts`：`DatabaseService` を足す
- `frontend/src/routes/settings.tsx`：データの節を足す
- `frontend/src/components/layout/startup-error-screen.tsx`：`restore` の段階の見出しを引けるようにする
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`
- `docs/design.md`：§5「更新前の控え」の復元の段落を、画面からの復元に書き換える。§6 のディレクトリ構成に `database.go` と `filepicker.go` を足す
- `docs/product.md`：§4 S09 にデータの節を足す
- `docs/wireframes/S09_settings.drawio.svg`：データの節を足す
- `docs/wireframes/S09a_restore_dialog.drawio.svg`：新設。復元に使うファイルを選ぶダイアログ
- `docs/wireframes/README.md`

**生成物 (再生成のみ)**

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`DatabaseService` が増える

**migration**

- なし

**後続 proposal への引き継ぎ**

- `add-mcp-server` と両方が入ったら、予約中は MCP の書き込みのツールを止めるかを決める (予約中に MCP で書いた内容は次の起動で消える)
- 書き出しを定期的に行う声が出たら、起動時に N 日おきに書き出す設定を S09 に足す
- 起動時のエラー画面から「予約を取り消して起動する」操作を足すかは、予約の失敗が実際に起きてから決める
