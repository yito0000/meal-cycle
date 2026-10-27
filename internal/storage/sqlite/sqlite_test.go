package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM schema_meta`).Scan(&n); err != nil {
		t.Fatalf("query schema_meta: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row in schema_meta, got %d", n)
	}

	var v int
	if err := db.QueryRow(`SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1`).Scan(&v); err != nil {
		t.Fatalf("query goose_db_version: %v", err)
	}
	if v != 11 {
		t.Fatalf("expected goose version 11, got %d", v)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate should be a no-op: %v", err)
	}
}

// seedMealPlansV8 は 00008 までの表に、献立 3 行 (うち 2 行は同じ作り置き) と作った印 1 行を直接入れる。
func seedMealPlansV8(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`INSERT INTO recipes (id, name, created_at, updated_at) VALUES ('r1', 'カレー', '2026-08-20T00:00:00Z', '2026-08-20T00:00:00Z')`,
		`INSERT INTO recipes (id, name, created_at, updated_at) VALUES ('r2', 'みそ汁', '2026-08-20T00:00:00Z', '2026-08-20T00:00:00Z')`,
		`INSERT INTO meal_plans (date, meal, position, recipe_id, created_at, batch_id)
		 VALUES ('2026-08-24', 'dinner', 0, 'r1', '2026-08-20T01:00:00Z', 'b1')`,
		`INSERT INTO meal_plans (date, meal, position, recipe_id, created_at, batch_id)
		 VALUES ('2026-08-24', 'dinner', 1, 'r2', '2026-08-20T02:00:00Z', 'b2')`,
		`INSERT INTO meal_plans (date, meal, position, recipe_id, created_at, batch_id)
		 VALUES ('2026-08-25', 'dinner', 0, 'r1', '2026-08-20T03:00:00Z', 'b1')`,
		`INSERT INTO meal_plan_cooked (batch_id, recipe_id, cooked_at) VALUES ('b1', 'r1', '2026-08-24T19:00:00Z')`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
}

func TestMigrate_MealPlansID(t *testing.T) {
	setup := func(t *testing.T) *sql.DB {
		t.Helper()
		db := openMigratedTo(t, filepath.Join(t.TempDir(), "test.db"), 8)
		seedMealPlansV8(t, db)
		if err := goose.UpTo(db, "migrations", 9); err != nil {
			t.Fatalf("UpTo(9): %v", err)
		}
		return db
	}

	t.Run("should give every existing row a distinct non-empty id and keep the other columns", func(t *testing.T) {
		db := setup(t)
		rows, err := db.Query(`SELECT id, date, meal, position, recipe_id, created_at, batch_id FROM meal_plans ORDER BY date, position`)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()
		type row struct {
			id, date, meal         string
			pos                    int
			recipe, created, batch string
		}
		got := make([]row, 0)
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.date, &r.meal, &r.pos, &r.recipe, &r.created, &r.batch); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got = append(got, r)
		}
		want := []row{
			{"", "2026-08-24", "dinner", 0, "r1", "2026-08-20T01:00:00Z", "b1"},
			{"", "2026-08-24", "dinner", 1, "r2", "2026-08-20T02:00:00Z", "b2"},
			{"", "2026-08-25", "dinner", 0, "r1", "2026-08-20T03:00:00Z", "b1"},
		}
		if len(got) != len(want) {
			t.Fatalf("want %d rows, got %d: %+v", len(want), len(got), got)
		}
		ids := map[string]bool{}
		for i, g := range got {
			if g.id == "" {
				t.Errorf("row %d: id is empty", i)
			}
			ids[g.id] = true
			w := want[i]
			w.id = g.id
			if g != w {
				t.Errorf("row %d = %+v, want %+v", i, g, w)
			}
		}
		if len(ids) != 3 {
			t.Errorf("want 3 distinct ids, got %d", len(ids))
		}
	})

	t.Run("should still reject a row that duplicates (date, meal, position)", func(t *testing.T) {
		db := setup(t)
		_, err := db.Exec(`INSERT INTO meal_plans (id, date, meal, position, recipe_id, created_at, batch_id)
			VALUES ('x', '2026-08-24', 'dinner', 0, 'r2', '2026-08-21T00:00:00Z', 'b9')`)
		if err == nil {
			t.Fatal("want a unique constraint error, got nil")
		}
	})

	t.Run("should keep the cooked marks", func(t *testing.T) {
		db := setup(t)
		var batch, recipe, at string
		if err := db.QueryRow(`SELECT batch_id, recipe_id, cooked_at FROM meal_plan_cooked`).Scan(&batch, &recipe, &at); err != nil {
			t.Fatalf("query meal_plan_cooked: %v", err)
		}
		if batch != "b1" || recipe != "r1" || at != "2026-08-24T19:00:00Z" {
			t.Errorf("mark = (%s, %s, %s), want (b1, r1, 2026-08-24T19:00:00Z)", batch, recipe, at)
		}
	})

	t.Run("should drop the id and keep the rows when migrated down", func(t *testing.T) {
		db := setup(t)
		if err := goose.DownTo(db, "migrations", 8); err != nil {
			t.Fatalf("DownTo(8): %v", err)
		}
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM meal_plans`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		if n != 3 {
			t.Errorf("want 3 rows, got %d", n)
		}
		if _, err := db.Exec(`SELECT id FROM meal_plans`); err == nil {
			t.Error("want the id column to be gone")
		}
	})
}
