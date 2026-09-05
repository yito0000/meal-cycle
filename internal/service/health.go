// Package service は Wails IPC でフロントエンドに公開する各種サービスを
// まとめる。1 capability = 1 ファイル (例: recipe.go / mealplan.go /
// health.go) で、いずれも `application.NewService(...)` に渡して `main.go`
// で登録する。
//
// サービスは domain / storage 層の上に薄く乗る境界層で、
// - フロント向けの型付き入出力 (Wails バインディングに出る shape) を確定させる
// - 入力バリデーションを施す
// - Repository / Domain を呼び出す
// の 3 つに責務を絞る。フレームワーク非依存にしたいドメインロジックは
// `internal/domain/<capability>/` に置き、DB アクセスは
// `internal/storage/sqlite/` に置く。
package service

import "database/sql"

// HealthService reports basic runtime health to the frontend.
// Intentionally minimal — real domain services (RecipeService,
// MealPlanService, ...) grow alongside proposals; this exists to prove
// the Wails binding + database wiring end-to-end.
type HealthService struct {
	db *sql.DB
}

// NewHealthService constructs a HealthService bound to db.
func NewHealthService(db *sql.DB) *HealthService {
	return &HealthService{db: db}
}

// Health describes the current runtime state visible to the UI.
type Health struct {
	DBConnected bool   `json:"dbConnected"`
	DBError     string `json:"dbError,omitempty"`
}

// Ping checks the database connection and returns the current Health.
func (h *HealthService) Ping() Health {
	if err := h.db.Ping(); err != nil {
		return Health{DBConnected: false, DBError: err.Error()}
	}
	return Health{DBConnected: true}
}
