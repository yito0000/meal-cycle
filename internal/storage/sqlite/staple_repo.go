package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
)

// StapleRepo は staples を扱う。名前の重複検査はサービス層で行う。
type StapleRepo struct {
	db *sql.DB
}

func NewStapleRepo(db *sql.DB) *StapleRepo {
	return &StapleRepo{db: db}
}

const stapleColumns = `id, name, state, added_at, updated_at`

func (r *StapleRepo) List(ctx context.Context) ([]fridge.Staple, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+stapleColumns+` FROM staples ORDER BY added_at, id`)
	if err != nil {
		return nil, fmt.Errorf("query staples: %w", err)
	}
	defer rows.Close()

	out := make([]fridge.Staple, 0)
	for rows.Next() {
		s, err := scanStaple(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staples: %w", err)
	}
	return out, nil
}

func (r *StapleRepo) GetByID(ctx context.Context, id string) (fridge.Staple, error) {
	s, err := scanStaple(r.db.QueryRowContext(ctx, `SELECT `+stapleColumns+` FROM staples WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return fridge.Staple{}, fridge.ErrStapleNotFound
	}
	return s, err
}

func (r *StapleRepo) Create(ctx context.Context, name string, state fridge.StapleState) (fridge.Staple, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return fridge.Staple{}, fmt.Errorf("generate staple id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	s := fridge.Staple{ID: id.String(), Name: name, State: state, AddedAt: now, UpdatedAt: now}
	if err := insertStaple(ctx, r.db, s); err != nil {
		return fridge.Staple{}, err
	}
	return s, nil
}

// UpdateState は updated_at を必ず進めるため、同一秒に落ちたときは既存値 +1s にする (RFC3339 秒精度)。
func (r *StapleRepo) UpdateState(ctx context.Context, id string, state fridge.StapleState) (fridge.Staple, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return fridge.Staple{}, err
	}
	now := time.Now().UTC()
	if prev, perr := time.Parse(time.RFC3339, existing.UpdatedAt); perr == nil && !now.Truncate(time.Second).After(prev) {
		now = prev.Add(time.Second)
	}
	updatedAt := now.Format(time.RFC3339)
	if _, err := r.db.ExecContext(ctx,
		`UPDATE staples SET state = ?, updated_at = ? WHERE id = ?`, string(state), updatedAt, id,
	); err != nil {
		return fridge.Staple{}, fmt.Errorf("update staple: %w", err)
	}
	existing.State = state
	existing.UpdatedAt = updatedAt
	return existing, nil
}

func (r *StapleRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM staples WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete staple: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete staple rows affected: %w", err)
	}
	if affected == 0 {
		return fridge.ErrStapleNotFound
	}
	return nil
}

// Restore は削除直後の「元に戻す」用に、渡された ID / AddedAt / State のまま作り直す。
func (r *StapleRepo) Restore(ctx context.Context, input fridge.Staple) (fridge.Staple, error) {
	s := input
	s.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := insertStaple(ctx, r.db, s); err != nil {
		return fridge.Staple{}, err
	}
	return s, nil
}

func scanStaple(s rowScanner) (fridge.Staple, error) {
	var out fridge.Staple
	if err := s.Scan(&out.ID, &out.Name, &out.State, &out.AddedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fridge.Staple{}, err
		}
		return fridge.Staple{}, fmt.Errorf("scan staple: %w", err)
	}
	return out, nil
}

func insertStaple(ctx context.Context, e execer, s fridge.Staple) error {
	if _, err := e.ExecContext(ctx, `INSERT INTO staples (`+stapleColumns+`) VALUES (?, ?, ?, ?, ?)`,
		s.ID, s.Name, string(s.State), s.AddedAt, s.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert staple: %w", err)
	}
	return nil
}
