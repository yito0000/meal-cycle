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

// HealthService はフロントに DB 疎通を返す最小サービス。Wails バインディングと
// DB 接続の end-to-end 疎通を確認するために置いてあり、ドメイン機能は持たない。
type HealthService struct {
	db *sql.DB
}

func NewHealthService(db *sql.DB) *HealthService {
	return &HealthService{db: db}
}

type Health struct {
	DBConnected bool   `json:"dbConnected"`
	DBError     string `json:"dbError,omitempty"`
}

func (h *HealthService) Ping() Health {
	if err := h.db.Ping(); err != nil {
		return Health{DBConnected: false, DBError: err.Error()}
	}
	return Health{DBConnected: true}
}
