package main

import "database/sql"

// HealthService reports basic runtime health to the frontend.
// Intentionally minimal — real domain services (RecipeService, MealPlanService, ...)
// will grow alongside proposals; this exists to prove the Wails binding +
// database wiring end-to-end.
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
