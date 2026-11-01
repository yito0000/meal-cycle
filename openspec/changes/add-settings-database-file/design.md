## Context

動機は proposal.md の Why を参照。

- DB は `sqlite.OpenAndMigrate(path)` が起動時に 1 回開き、`*sql.DB` を全サービスに渡している。WAL モードなので、本体のファイルの横に `-wal` / `-shm` がある
- 更新前の控えは `backupBeforeMigrate` が `VACUUM INTO` で `.partial` に書いてから改名し、`pruneBackups` が新しい 5 個を残す (`backups/meal-cycle-v<版>-<日時>.db`)
- 起動時の失敗は `*StartupError{Stage, Err}` で返し、`HealthService` を通してフロントの起動時のエラー画面に出す
- フォルダの選択は `FolderPicker` interface にして、E2E では固定のフォルダを返す実装を渡している (add-recipe-export)
- `SettingsService` は `NewSettingsService(db)` で作られ、`add-mcp-server` の MCP サーバーからも同じ形で呼ばれる予定

## Goals / Non-Goals

**Goals:**

- 献立、在庫、常備品、設定を含めた全データを、利用者が選んだ場所に 1 ファイルで残せるようにする
- 書き出したファイルと自動の控えを、ファイル操作を知らなくても画面から戻せるようにする
- 復元を失敗しても今のデータを失わないようにする

**Non-Goals:**

- 起動中の差し替えと自動の再起動 (proposal.md の Non-goals)

## Decisions

### D1. 復元は「次の起動で当てる予約」にする

**選択肢:**

- (a) 起動中に DB を閉じて差し替え、開き直す — Pros: 再起動が要らない /
  Cons: 全サービスが `*sql.DB` を持っており、差し替えにはサービスを作り直すか、接続を間接参照に変える改修が要る。
  処理中の呼び出しと競合したときの扱いも決める必要がある
- (b) 予約して次の起動で当てる ← 採用 — Pros: DB を誰も開いていない時点で差し替えられる。既存の起動の流れ (控え → 更新) をそのまま通せる /
  Cons: 利用者が終了と起動を行う必要がある
- (c) 終了の直前 (`OnShutdown`) に差し替える — Pros: 次の起動を待たずに済む /
  Cons: 終了の処理中に失敗すると、知らせる画面が無い

**採用: (b)**。理由: 失敗したときにエラー画面で知らせられ、今の DB を残したまま止まれる。終了の手間は「今すぐ終了する」ボタンで減らす。

予約は、選んだファイルを DB と同じフォルダの `restore-pending.db` に `.partial` 経由でコピーして表す。
選んだファイルの場所を覚えるだけにしないのは、起動までの間に元のファイルが動いたり消えたりしても予約が壊れないようにするためである。
`backups/` の控えも同じくコピーする (控えは `pruneBackups` で消されうる)。

### D2. 予約の前の確かめ

`ValidateRestoreSource(path)` は次を確かめ、どれかに当たれば予約しない。

1. 読み取り専用で開けて、`PRAGMA quick_check` が `ok` を返す
2. goose の版の表があり、版を読める (meal-cycle の DB である)
3. 版が、このアプリに埋め込んだ migration の最大の版以下である (新しい版の DB は、古いアプリでは読めない列を持ちうる)

版が古いファイルは予約できる。次の起動で、いつもの更新の流れが控えを作ってから最新にする。

### D3. 起動時に予約を当てる手順

`OpenAndMigrate` の最初、`os.Stat(path)` より前に `applyPendingRestore(path)` を呼ぶ。`restore-pending.db` が無ければ何もしない。

1. 今の DB があれば `Open` で開き、`VACUUM INTO` で `backups/meal-cycle-v<版>-<日時>-before-restore.db` に書き出して閉じる (WAL の中身も含めるため)
2. `-wal` / `-shm` を消す
3. `restore-pending.db` を DB の場所へ `os.Rename` で置き換える (同じフォルダなので改名で済み、途中で止まっても片方は必ず残る)
4. `pruneBackups` を呼ぶ

1〜3 のどこかで失敗したら `StageRestore` の `StartupError` を返す。予約のファイルは消さずに残し、次の起動でもう一度試す。
1 で失敗した場合は今の DB を変えていない。3 で失敗した場合も、1 の控えと今の DB の両方が残っている。

控えのファイル名に `-before-restore` を付けるのは、S09 の一覧で更新前の控えと見分けるためである。
`pruneBackups` は `meal-cycle-v` で始まる `.db` を数えるので、復元前の控えも 5 個の中に数えられる。

### D4. サービスの分け方: `DatabaseService` を新設する

**選択肢:**

- (a) `SettingsService` にメソッドを足す — Pros: capability と 1 対 1 のファイルのまま /
  Cons: コンストラクタに DB のパスとファイルの picker が要り、`main.go`、E2E サーバー、`add-mcp-server` の呼び出しがすべて変わる。
  設定の値の保存と、ファイルの操作という別の関心が 1 つの型に入る
- (b) `internal/service/database.go` に `DatabaseService` を新設する ← 採用 — Pros: `SettingsService` の形を変えない。
  MCP サーバーは `DatabaseService` を登録しないだけで書き出しと復元を公開しない形になる /
  Cons: settings capability のサービスが 2 つになる

**採用: (b)**。理由: 既存の呼び出し元を変えずに済み、MCP の Non-goals (DB の操作は公開しない) とも素直に合う。
CLAUDE.md の「1 capability = 1 ファイル」は、サービスの名前の衝突を避けるための規則で、1 capability に 2 ファイルを置くことは妨げない。

`DatabaseService` のメソッド:

- `Export(ctx, dialogTitle) (ExportResult, error)`：保存ダイアログを開き、選ばれた場所へ `VACUUM INTO` (`.partial` → 改名)。閉じられたら空の結果
- `ListBackups(ctx) ([]BackupFile, error)`：`backups/` の控えを新しい順に返す。`BackupFile{Name, CreatedAt, Version, BeforeRestore}`
- `ReserveFromBackup(ctx, name) (PendingRestore, error)`：`name` は `ListBackups` の結果の名前だけを受け付ける (パスの指定を許さない)
- `ReserveFromFile(ctx, dialogTitle) (PendingRestore, error)`：開くダイアログを開き、選ばれたファイルで予約する
- `Pending(ctx) (PendingRestore, error)` / `CancelPending(ctx) error`
- `Quit(ctx)`：`application.Get().Quit()` を呼ぶ

`DatabaseService` は `*sql.DB` を受け取り、書き出しはその接続で `VACUUM INTO` を実行する (起動中の WAL の中身も含めるため)。

### D5. ファイルの選択: `FilePicker` interface

`FolderPicker` と同じく interface にし、E2E では固定のパスを返す実装を渡す。

```go
type FilePicker interface {
    SaveFile(title, defaultName string) (string, error) // 閉じられたら ""
    OpenFile(title string) (string, error)              // 閉じられたら ""
}
```

開くダイアログは拡張子 `.db` で絞る。

### D6. 予約中の表示

S09 を開くたびに `Pending` を呼び、予約があれば節の先頭に「次の起動で <ファイル名> から復元します」と「予約を取り消す」「今すぐ終了する」を出す。
予約中も書き出しと別のファイルでの予約はできる (別のファイルで予約し直すと上書きする)。

## Risks / Trade-offs

- [Risk] 予約したまま気付かずに使い続け、次の起動で予約の後に入れた献立が消える
  → 予約の確認ダイアログと S09 の予約中の表示で、予約の後の変更は消える旨を出す。消える分も復元前の控え (D3) に残る
- [Risk] `add-mcp-server` が予約中に書き込む → proposal.md の後続への引き継ぎに残す。復元前の控えには残るので、取り返しはつく
- [Risk] 書き出し先がクラウド同期のフォルダで、書き出しの途中のファイルが同期される → `.partial` に書いてから改名する (既存の控えと同じ)
- [Risk] Windows で `os.Rename` の置き換えがウイルス対策ソフトのロックで失敗する → D3 のとおり予約を残して起動時のエラー画面を出し、次の起動で再試行する
