// Package service は Wails IPC でフロントに公開する各種サービスをまとめる。
// 1 capability = 1 ファイル (recipe.go / mealplan.go / health.go) で、いずれも
// application.NewService(...) に渡して main.go で登録する。
//
// サービスは domain / storage 層の境界層。フロント向けの型付き入出力を確定させ、
// 入力バリデーションを施し、Repository / Domain を呼び出す 3 つに責務を絞る。
// ドメインロジックは internal/domain/<capability>/、DB アクセスは
// internal/storage/sqlite/ に置く。
package service

import "database/sql"

// HealthService は DB 疎通と起動時の失敗をフロントに返す。起動に失敗したときは
// これだけが登録されるので、db が nil でも動く必要がある (design.md D2)。
type HealthService struct {
	db      *sql.DB
	startup StartupState
}

// StartupState の Stage は sqlite.StartupStage のコード (open / backup / migrate)。空なら起動に成功している。
type StartupState struct {
	Stage     string
	Detail    string
	DBPath    string
	BackupDir string
}

func NewHealthService(db *sql.DB, startup StartupState) *HealthService {
	return &HealthService{db: db, startup: startup}
}

type Health struct {
	DBConnected   bool   `json:"dbConnected"`
	DBError       string `json:"dbError,omitempty"`
	StartupError  string `json:"startupError,omitempty"`
	StartupDetail string `json:"startupDetail,omitempty"`
	DBPath        string `json:"dbPath,omitempty"`
	BackupDir     string `json:"backupDir,omitempty"`
}

func (h *HealthService) Ping() Health {
	out := Health{
		StartupError:  h.startup.Stage,
		StartupDetail: h.startup.Detail,
		DBPath:        h.startup.DBPath,
		BackupDir:     h.startup.BackupDir,
	}
	if h.db == nil {
		return out
	}
	if err := h.db.Ping(); err != nil {
		out.DBError = err.Error()
		return out
	}
	out.DBConnected = true
	return out
}
