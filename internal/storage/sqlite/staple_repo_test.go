package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
)

func newStapleRepoForTest(t *testing.T) *StapleRepo {
	t.Helper()
	_, db := newStockRepoForTest(t)
	return NewStapleRepo(db)
}

func TestStapleRepo(t *testing.T) {
	ctx := context.Background()

	t.Run("created staple keeps state and added_at in List", func(t *testing.T) {
		repo := newStapleRepoForTest(t)
		created, err := repo.Create(ctx, "牛乳", fridge.StapleLow)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0] != created {
			t.Errorf("want [%+v], got %+v", created, got)
		}
		if created.State != fridge.StapleLow || created.AddedAt == "" {
			t.Errorf("unexpected created: %+v", created)
		}
	})

	t.Run("UpdateState changes state and advances updated_at keeping added_at", func(t *testing.T) {
		repo := newStapleRepoForTest(t)
		orig, err := repo.Restore(ctx, fridge.Staple{ID: "s1", Name: "牛乳", State: fridge.StapleEnough, AddedAt: "2026-01-01T00:00:00Z"})
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		got, err := repo.UpdateState(ctx, "s1", fridge.StapleOut)
		if err != nil {
			t.Fatalf("UpdateState: %v", err)
		}
		if got.State != fridge.StapleOut || got.AddedAt != orig.AddedAt || got.UpdatedAt <= orig.UpdatedAt {
			t.Errorf("unexpected updated: orig=%+v got=%+v", orig, got)
		}
		read, _ := repo.GetByID(ctx, "s1")
		if read != got {
			t.Errorf("read back mismatch: %+v vs %+v", read, got)
		}
	})

	t.Run("UpdateState and Delete return ErrStapleNotFound for missing id", func(t *testing.T) {
		repo := newStapleRepoForTest(t)
		if _, err := repo.UpdateState(ctx, "missing", fridge.StapleLow); !errors.Is(err, fridge.ErrStapleNotFound) {
			t.Errorf("UpdateState: want ErrStapleNotFound, got %v", err)
		}
		if err := repo.Delete(ctx, "missing"); !errors.Is(err, fridge.ErrStapleNotFound) {
			t.Errorf("Delete: want ErrStapleNotFound, got %v", err)
		}
	})

	t.Run("Restore recreates with given id and added_at", func(t *testing.T) {
		repo := newStapleRepoForTest(t)
		created, _ := repo.Create(ctx, "食パン", fridge.StapleLow)
		if err := repo.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		restored, err := repo.Restore(ctx, created)
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		got, _ := repo.GetByID(ctx, created.ID)
		if got.ID != created.ID || got.AddedAt != created.AddedAt || got.State != fridge.StapleLow || got != restored {
			t.Errorf("unexpected restored: %+v", got)
		}
	})

	t.Run("state outside enumeration fails CHECK constraint", func(t *testing.T) {
		repo := newStapleRepoForTest(t)
		_, err := repo.db.ExecContext(ctx, `INSERT INTO staples (id, name, state, added_at, updated_at)
			VALUES ('x', '牛乳', 'plenty', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
		if err == nil {
			t.Error("want CHECK constraint error, got nil")
		}
	})
}
