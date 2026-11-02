package settings

// BackupFile は backups フォルダの控え 1 つ。BeforeRestore は復元の予約を当てる前に作った控えか
// (偽ならスキーマ更新の前の控え)。CreatedAt は RFC3339 (add-settings-database-file design.md D3)。
type BackupFile struct {
	Name          string `json:"name"`
	CreatedAt     string `json:"createdAt"`
	Version       int64  `json:"version"`
	BeforeRestore bool   `json:"beforeRestore"`
}

// PendingRestore は次の起動で当てる復元の予約。Name は選んだ控えかファイルの名前で、表示にだけ使う。
type PendingRestore struct {
	Pending bool   `json:"pending"`
	Name    string `json:"name"`
}

// DatabaseExportResult は書き出しの結果。保存ダイアログを閉じたときは Cancelled だけが真になる。
type DatabaseExportResult struct {
	Cancelled bool   `json:"cancelled"`
	Path      string `json:"path"`
}

// ReserveResult はファイルを選んで予約した結果。ファイルの選択を閉じたときは Cancelled だけが真になる。
type ReserveResult struct {
	Cancelled bool           `json:"cancelled"`
	Pending   PendingRestore `json:"pending"`
}
