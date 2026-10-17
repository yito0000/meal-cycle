package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

// positionShiftOffset は position を書き換えるあいだ、(date, meal, position) の一意性に当たらないための退避幅。
// 1 セルの品数がこの値より少ないことを前提にする (fix-mealplan-renumber-after-reorder design.md D1)。
const positionShiftOffset = 1000

type MealPlanRepo struct {
	db  *sql.DB
	cfg config.Config
}

func NewMealPlanRepo(db *sql.DB) *MealPlanRepo {
	return &MealPlanRepo{db: db, cfg: config.Defaults()}
}

// mergeDayRange は「他の作り置きにまとめる」で許容する日付レンジ (±N 日) を返す。
// CopyLeftover が生成する連続日数の最大値と揃える (design.md D4b)。
func (r *MealPlanRepo) mergeDayRange() int {
	max := 0
	for _, d := range r.cfg.LeftoverCopyPresets {
		if d > max {
			max = d
		}
	}
	return max
}

// ListPeriod は periodStart (月曜 "YYYY-MM-DD") から periodDays 日ぶんの Entry を position 順で返す。
// batch_member_count は期間外の行も含めて数える。期間で絞った後に数えると、週をまたぐ作り置きの端の行が単独に見えるため。
// 数え方は S07 の loadPlannedGroups とそろえる (fix-mealplan-batch-boundaries D1)。
func (r *MealPlanRepo) ListPeriod(
	ctx context.Context, periodStart string, periodDays int,
) ([]mealplan.Entry, error) {
	end, err := addDays(periodStart, periodDays)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.date, m.meal, m.position, m.recipe_id, r.name, m.batch_id,
		       (SELECT COUNT(*) FROM meal_plans a WHERE a.batch_id = m.batch_id) AS batch_member_count,
		       EXISTS (
		           SELECT 1 FROM meal_plan_cooked c
		           WHERE c.batch_id = m.batch_id AND c.recipe_id = m.recipe_id
		       ) AS cooked
		FROM meal_plans m
		JOIN recipes r ON r.id = m.recipe_id
		WHERE m.date >= ? AND m.date < ?
		ORDER BY m.date, m.meal, m.position
	`, periodStart, end)
	if err != nil {
		return nil, fmt.Errorf("query meal_plans: %w", err)
	}
	defer rows.Close()

	out := make([]mealplan.Entry, 0)
	for rows.Next() {
		var e mealplan.Entry
		var meal string
		if err := rows.Scan(&e.ID, &e.Date, &meal, &e.Position, &e.RecipeID, &e.RecipeName,
			&e.BatchID, &e.BatchMemberCount, &e.Cooked); err != nil {
			return nil, fmt.Errorf("scan meal_plans row: %w", err)
		}
		e.Meal = mealplan.Meal(meal)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meal_plans: %w", err)
	}
	return out, nil
}

// Assign は対象 (date, meal) の末尾 (position = max+1) に recipe_id を追加する。
// 単独割当は「新しい 1 バッチ」の意味なので、常に新 batch_id を発行する。
func (r *MealPlanRepo) Assign(ctx context.Context, date, meal, recipeID string) (mealplan.Entry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mealplan.Entry{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := ensureActiveRecipe(ctx, tx, recipeID); err != nil {
		return mealplan.Entry{}, err
	}
	pos, err := nextPosition(ctx, tx, date, meal)
	if err != nil {
		return mealplan.Entry{}, err
	}
	batchID := mealplan.NewBatchID()
	entry, err := insertMealPlan(ctx, tx, mealplan.NewEntryID(), date, meal, pos, recipeID, batchID)
	if err != nil {
		return mealplan.Entry{}, err
	}
	if err := tx.Commit(); err != nil {
		return mealplan.Entry{}, fmt.Errorf("commit tx: %w", err)
	}
	return entry, nil
}

// Replace は指定行の recipe_id を差し替える。該当行が無ければ ErrEntryNotFound。
// batch_id は変更しない (「同じバッチのまま中身のレシピを差し替える」意味)。
func (r *MealPlanRepo) Replace(ctx context.Context, entryID, recipeID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := loadEntryByID(ctx, tx, entryID); err != nil {
		return err
	}
	if err := ensureActiveRecipe(ctx, tx, recipeID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET recipe_id = ? WHERE id = ?
	`, recipeID, entryID)
	if err != nil {
		if isForeignKeyErr(err) {
			return mealplan.ErrUnknownRecipe
		}
		return fmt.Errorf("update meal_plans: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return mealplan.ErrEntryNotFound
	}
	// 掃除しないと、入れ替えてから元のレシピに戻したときに古い印が復活する (design.md D4)。
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Remove は 1 行削除し、後続 position を 1 つずつ詰め直す。消した行の内容を「元に戻す」用に返す。
func (r *MealPlanRepo) Remove(ctx context.Context, entryID string) (mealplan.RemovedEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mealplan.RemovedEntry{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	removed, err := loadRemovedEntries(ctx, tx, `m.id = ?`, entryID)
	if err != nil {
		return mealplan.RemovedEntry{}, err
	}
	if len(removed) == 0 {
		return mealplan.RemovedEntry{}, mealplan.ErrEntryNotFound
	}
	if err := removeRowTx(ctx, tx, removed[0].ID, removed[0].Date, string(removed[0].Meal), removed[0].Position); err != nil {
		return mealplan.RemovedEntry{}, err
	}
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return mealplan.RemovedEntry{}, err
	}
	if err := tx.Commit(); err != nil {
		return mealplan.RemovedEntry{}, fmt.Errorf("commit tx: %w", err)
	}
	return removed[0], nil
}

// MoveEntry は 1 行を別セル (targetDate, targetMeal) の末尾 (position = max+1) に移す。
// id / batch_id / recipe_id は据え置き。作り置きに属する行を動かしても batch は解体されない
// (spec: 「1 行 移動」)。同一セル (source == target) は責務分離のため ErrInvalidPosition を返し、
// UI 側で既存 ReorderMeal に流させる (design.md D2)。
func (r *MealPlanRepo) MoveEntry(ctx context.Context, entryID, targetDate, targetMeal string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	src, err := loadEntryByID(ctx, tx, entryID)
	if err != nil {
		return err
	}
	if src.Date == targetDate && string(src.Meal) == targetMeal {
		return mealplan.ErrInvalidPosition
	}
	pos, err := nextPosition(ctx, tx, targetDate, targetMeal)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET date = ?, meal = ?, position = ? WHERE id = ?
	`, targetDate, targetMeal, pos, entryID); err != nil {
		return fmt.Errorf("move meal_plans row: %w", err)
	}
	if err := closePositionGap(ctx, tx, src.Date, string(src.Meal), src.Position); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// removeRowTx は tx 内で 1 行削除し、同じセルの後続 position を詰め直す (自身では commit しない)。
func removeRowTx(ctx context.Context, tx *sql.Tx, id, date, meal string, position int) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM meal_plans WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete meal_plans: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return mealplan.ErrEntryNotFound
	}
	return closePositionGap(ctx, tx, date, meal, position)
}

// closePositionGap は removed の行を消した直後のセルで、後ろの行を 1 つずつ前へ詰める。
// SQLite は行ごとに (date, meal, position) の一意性を確かめ、処理順は position 順とは限らないので、
// position - 1 の 1 文にせず退避を挟む (fix-mealplan-renumber-after-reorder design.md D1)。
func closePositionGap(ctx context.Context, q querier, date, meal string, removed int) error {
	if _, err := q.ExecContext(ctx, `
		UPDATE meal_plans SET position = position + ?
		WHERE date = ? AND meal = ? AND position > ?
	`, positionShiftOffset, date, meal, removed); err != nil {
		return fmt.Errorf("shift positions: %w", err)
	}
	if _, err := q.ExecContext(ctx, `
		UPDATE meal_plans SET position = position - ? - 1
		WHERE date = ? AND meal = ? AND position >= ?
	`, positionShiftOffset, date, meal, positionShiftOffset); err != nil {
		return fmt.Errorf("renumber positions: %w", err)
	}
	return nil
}

// renumberMeal はセルの行を、いまの順を保ったまま position 0 から隙間なく振り直す。
// 複数行を消したあと用 (fix-mealplan-renumber-after-reorder design.md D2)。
func renumberMeal(ctx context.Context, q querier, date, meal string) error {
	rows, err := q.QueryContext(ctx, `
		SELECT position FROM meal_plans WHERE date = ? AND meal = ? ORDER BY position
	`, date, meal)
	if err != nil {
		return fmt.Errorf("query positions: %w", err)
	}
	positions := make([]int, 0)
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate positions: %w", err)
	}

	if _, err := q.ExecContext(ctx, `
		UPDATE meal_plans SET position = position + ? WHERE date = ? AND meal = ?
	`, positionShiftOffset, date, meal); err != nil {
		return fmt.Errorf("shift positions: %w", err)
	}
	for newPos, oldPos := range positions {
		if _, err := q.ExecContext(ctx, `
			UPDATE meal_plans SET position = ?
			WHERE date = ? AND meal = ? AND position = ?
		`, newPos, date, meal, oldPos+positionShiftOffset); err != nil {
			return fmt.Errorf("renumber position %d: %w", newPos, err)
		}
	}
	return nil
}

// ClearMeal は (date, meal) の全 position を削除し、消した行を position 昇順で返す。0 件でもエラーにしない。
func (r *MealPlanRepo) ClearMeal(ctx context.Context, date, meal string) ([]mealplan.RemovedEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	removed, err := loadRemovedEntries(ctx, tx, `m.date = ? AND m.meal = ?`, date, meal)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM meal_plans WHERE date = ? AND meal = ?
	`, date, meal); err != nil {
		return nil, fmt.Errorf("clear meal: %w", err)
	}
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return removed, nil
}

// RestoreEntries は Remove / ClearMeal で消した行を、元の ID のまま元のセルの元の position へ 1 トランザクションで戻す。
// 削除のあとにセルの品数が減っていれば末尾に置き、増えていれば後ろの行をずらす
// (revise-mealplan-destructive-actions design.md D2)。同じ ID の行が残っていれば ErrInvalidRestore を返す
// (add-mealplan-entry-id design.md D5)。
func (r *MealPlanRepo) RestoreEntries(ctx context.Context, entries []mealplan.RemovedEntry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, e := range sortRemovedEntries(entries) {
		if err := restoreEntryTx(ctx, tx, e); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// sortRemovedEntries は戻す順に並べる。position の小さい順に戻さないと、先に戻した行が後の行の position を押し出してしまう。
func sortRemovedEntries(entries []mealplan.RemovedEntry) []mealplan.RemovedEntry {
	sorted := append([]mealplan.RemovedEntry(nil), entries...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if a.Meal != b.Meal {
			return mealplan.MealRank(a.Meal) < mealplan.MealRank(b.Meal)
		}
		return a.Position < b.Position
	})
	return sorted
}

func restoreEntryTx(ctx context.Context, tx *sql.Tx, e mealplan.RemovedEntry) error {
	meal := string(e.Meal)
	var exists int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM meal_plans WHERE id = ?
	`, e.ID).Scan(&exists); err != nil {
		return fmt.Errorf("check restored id: %w", err)
	}
	if exists > 0 {
		return mealplan.ErrInvalidRestore
	}
	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM meal_plans WHERE date = ? AND meal = ?
	`, e.Date, meal).Scan(&count); err != nil {
		return fmt.Errorf("count meal rows: %w", err)
	}
	pos := min(e.Position, count)
	if err := openPositionGap(ctx, tx, e.Date, meal, pos); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO meal_plans (id, date, meal, position, recipe_id, created_at, batch_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.Date, meal, pos, e.RecipeID, e.CreatedAt, e.BatchID); err != nil {
		if isForeignKeyErr(err) {
			return mealplan.ErrUnknownRecipe
		}
		return fmt.Errorf("insert meal_plans: %w", err)
	}
	if e.CookedAt == "" {
		return nil
	}
	// 同じ作り置きの別の行が印を保っていれば、その印 (と日時) を優先する。
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO meal_plan_cooked (batch_id, recipe_id, cooked_at) VALUES (?, ?, ?)
		ON CONFLICT (batch_id, recipe_id) DO NOTHING
	`, e.BatchID, e.RecipeID, e.CookedAt); err != nil {
		return fmt.Errorf("insert meal_plan_cooked: %w", err)
	}
	return nil
}

// openPositionGap は at 以降の行を 1 つずつ後ろへずらし、at を空ける。
// closePositionGap と同じ理由で、+1 の 1 文にせず退避を挟む (fix-mealplan-renumber-after-reorder design.md D1)。
func openPositionGap(ctx context.Context, q querier, date, meal string, at int) error {
	if _, err := q.ExecContext(ctx, `
		UPDATE meal_plans SET position = position + ?
		WHERE date = ? AND meal = ? AND position >= ?
	`, positionShiftOffset, date, meal, at); err != nil {
		return fmt.Errorf("shift positions: %w", err)
	}
	if _, err := q.ExecContext(ctx, `
		UPDATE meal_plans SET position = position - ? + 1
		WHERE date = ? AND meal = ? AND position >= ?
	`, positionShiftOffset, date, meal, positionShiftOffset); err != nil {
		return fmt.Errorf("renumber positions: %w", err)
	}
	return nil
}

// loadRemovedEntries は where に合う行を、作った印の日時と一緒に position 昇順で読む。
func loadRemovedEntries(ctx context.Context, q querier, where string, args ...any) ([]mealplan.RemovedEntry, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT m.id, m.date, m.meal, m.position, m.recipe_id, m.batch_id, m.created_at, COALESCE(c.cooked_at, '')
		FROM meal_plans m
		LEFT JOIN meal_plan_cooked c ON c.batch_id = m.batch_id AND c.recipe_id = m.recipe_id
		WHERE `+where+`
		ORDER BY m.position
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("query removed rows: %w", err)
	}
	defer rows.Close()
	out := make([]mealplan.RemovedEntry, 0)
	for rows.Next() {
		var e mealplan.RemovedEntry
		var mealStr string
		if err := rows.Scan(&e.ID, &e.Date, &mealStr, &e.Position, &e.RecipeID, &e.BatchID, &e.CreatedAt, &e.CookedAt); err != nil {
			return nil, fmt.Errorf("scan removed row: %w", err)
		}
		e.Meal = mealplan.Meal(mealStr)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Reorder は orderedEntryIDs の順で position を 0..N-1 に採番し直す。
// 渡された ID の集合がセルの行の ID の集合と一致しない (重複・不足・他セルの ID) 場合は ErrReorderMismatch。
// (date, meal, position) の一意性に当たらないよう、一旦 positionShiftOffset ぶん退避する。
func (r *MealPlanRepo) Reorder(ctx context.Context, date, meal string, orderedEntryIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM meal_plans WHERE date = ? AND meal = ?
	`, date, meal)
	if err != nil {
		return fmt.Errorf("query current meal: %w", err)
	}
	current := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan current meal: %w", err)
		}
		current[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate current meal: %w", err)
	}

	if len(orderedEntryIDs) != len(current) {
		return mealplan.ErrReorderMismatch
	}
	requested := map[string]bool{}
	for _, id := range orderedEntryIDs {
		if !current[id] || requested[id] {
			return mealplan.ErrReorderMismatch
		}
		requested[id] = true
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET position = position + ? WHERE date = ? AND meal = ?
	`, positionShiftOffset, date, meal); err != nil {
		return fmt.Errorf("shift positions: %w", err)
	}
	for newPos, id := range orderedEntryIDs {
		if _, err := tx.ExecContext(ctx, `
			UPDATE meal_plans SET position = ? WHERE id = ?
		`, newPos, id); err != nil {
			return fmt.Errorf("reorder position %d: %w", newPos, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// CopyLeftover は targets の各日付 meal に sourceEntries を反映する。mode=overwrite は既存を消してから、
// mode=append は既存の末尾に追加する (二重追加防止のため domain.PlanCopy を経由)。
//
// shareBatch=true (作り置き) は「同じレシピ ID を持つコピー先行は全 target で 1 batch_id」を割り振る。
// この関数の入口で source[i] の RecipeID → 共通 batch_id の map を作り、各 target の PlanCopy に
// BatchID を注入した source を渡す。shareBatch=false (別調理) は PlanCopy 内で 1 行ごとに新 UUID を
// 発行する。
func (r *MealPlanRepo) CopyLeftover(
	ctx context.Context,
	meal string,
	targets []string,
	sourceEntries []mealplan.Entry,
	mode mealplan.CopyMode,
	shareBatch mealplan.ShareBatch,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	recipeIDs := make([]string, 0, len(sourceEntries))
	for _, e := range sourceEntries {
		recipeIDs = append(recipeIDs, e.RecipeID)
	}
	if err := ensureNotArchived(ctx, tx, recipeIDs); err != nil {
		return err
	}

	source := prepareCopySource(sourceEntries, shareBatch)

	for _, target := range targets {
		if mode == mealplan.CopyModeOverwrite {
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM meal_plans WHERE date = ? AND meal = ?
			`, target, meal); err != nil {
				return fmt.Errorf("clear target %s: %w", target, err)
			}
		}
		existing, err := loadMealEntries(ctx, tx, target, meal)
		if err != nil {
			return err
		}
		toInsert, err := mealplan.PlanCopy(target, source, existing, mode, shareBatch)
		if err != nil {
			return err
		}
		for _, e := range toInsert {
			batchID := e.BatchID
			if batchID == "" {
				batchID = mealplan.NewBatchID()
			}
			if _, err := insertMealPlan(ctx, tx, e.ID, e.Date, string(e.Meal), e.Position, e.RecipeID, batchID); err != nil {
				return err
			}
		}
	}
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// SplitBatch は「作り置きをやめる」= 連続 N 日コピーの逆演算を実行する。
// 対象行が属する作り置き全体を対象に、起点行 (日付、朝昼夕の順、position の順で最も早い行) を残し、
// 他の同バッチ行を全削除。残った起点行に新 batch_id を発行して返す。
// バッチが元々 1 行しか含まない場合は no-op で現行 batch_id を返す (design.md D4a)。
func (r *MealPlanRepo) SplitBatch(ctx context.Context, entryID string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	entry, err := loadEntryByID(ctx, tx, entryID)
	if err != nil {
		return "", err
	}
	currentBatchID := entry.BatchID

	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM meal_plans WHERE batch_id = ?
	`, currentBatchID).Scan(&count); err != nil {
		return "", fmt.Errorf("count batch: %w", err)
	}
	if count == 1 {
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("commit tx: %w", err)
		}
		return currentBatchID, nil
	}

	var originID, originRecipeID string
	if err := tx.QueryRowContext(ctx, `
		SELECT id, recipe_id FROM meal_plans
		WHERE batch_id = ?
		ORDER BY date,
		         CASE meal WHEN 'breakfast' THEN 0 WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END,
		         position
		LIMIT 1
	`, currentBatchID).Scan(&originID, &originRecipeID); err != nil {
		return "", fmt.Errorf("locate origin: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, date, meal FROM meal_plans
		WHERE batch_id = ? AND id <> ?
		ORDER BY date, meal, position
	`, currentBatchID, originID)
	if err != nil {
		return "", fmt.Errorf("query siblings: %w", err)
	}
	type cellRow struct{ id, date, meal string }
	toDelete := make([]cellRow, 0)
	for rows.Next() {
		var cr cellRow
		if err := rows.Scan(&cr.id, &cr.date, &cr.meal); err != nil {
			rows.Close()
			return "", fmt.Errorf("scan siblings: %w", err)
		}
		toDelete = append(toDelete, cr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate siblings: %w", err)
	}

	// 同じセルに兄弟が 2 行ある場合に備え、全部消してからセルごとに 1 回だけ詰め直す。
	type cell struct{ date, meal string }
	affected := make([]cell, 0)
	seen := map[cell]bool{}
	for _, cr := range toDelete {
		if _, err := tx.ExecContext(ctx, `DELETE FROM meal_plans WHERE id = ?`, cr.id); err != nil {
			return "", fmt.Errorf("delete sibling row: %w", err)
		}
		c := cell{cr.date, cr.meal}
		if !seen[c] {
			seen[c] = true
			affected = append(affected, c)
		}
	}
	for _, c := range affected {
		if err := renumberMeal(ctx, tx, c.date, c.meal); err != nil {
			return "", err
		}
	}

	newID := mealplan.NewBatchID()
	if _, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET batch_id = ? WHERE id = ?
	`, newID, originID); err != nil {
		return "", fmt.Errorf("assign new batch_id: %w", err)
	}
	if err := copyCookedMark(ctx, tx, currentBatchID, newID, originRecipeID); err != nil {
		return "", err
	}
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return newID, nil
}

// MergeBatch は対象行の batch_id を targetBatchID に付け替える。
// 2 条件 (同一レシピ / 対象行の日付から ±mergeDayRange 日以内) を防御的にチェックし、
// 違反時は ErrBatchMergeDenied、そもそも targetBatchID の行が他に無ければ
// ErrBatchTargetNotFound を返す (design.md D4b)。時間帯 (meal) の一致は要件から外れており、
// 別時間帯の作り置きにも合流できる (proposal: add-mealplan-cross-meal-leftover)。
func (r *MealPlanRepo) MergeBatch(ctx context.Context, entryID, targetBatchID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	src, err := loadEntryByID(ctx, tx, entryID)
	if err != nil {
		return err
	}
	sourceRecipeID, sourceBatchID := src.RecipeID, src.BatchID

	var targetExists int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM meal_plans WHERE batch_id = ? AND id <> ?
	`, targetBatchID, entryID).Scan(&targetExists); err != nil {
		return fmt.Errorf("check target batch: %w", err)
	}
	if targetExists == 0 {
		return mealplan.ErrBatchTargetNotFound
	}

	dayRange := r.mergeDayRange()
	var eligible int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM meal_plans
		WHERE batch_id = ? AND recipe_id = ? AND id <> ?
		  AND ABS(julianday(date) - julianday(?)) <= ?
	`, targetBatchID, sourceRecipeID, entryID, src.Date, dayRange).Scan(&eligible); err != nil {
		return fmt.Errorf("check merge eligibility: %w", err)
	}
	if eligible == 0 {
		return mealplan.ErrBatchMergeDenied
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET batch_id = ? WHERE id = ?
	`, targetBatchID, entryID)
	if err != nil {
		return fmt.Errorf("merge batch: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return mealplan.ErrEntryNotFound
	}
	if err := copyCookedMark(ctx, tx, sourceBatchID, targetBatchID, sourceRecipeID); err != nil {
		return err
	}
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// ListMergeCandidates は指定行から見た合流可能な作り置きの候補を列挙する。レシピと日付は行そのものから読む。
// フィルタ条件: 同一 recipeID / 起点行 (date) から ±mergeDayRange 日以内 /
// 起点行自身が属するバッチは除外 (design.md D4b)。各バッチにつき代表行 1 つを返す。
// 代表行は (日付, 朝昼夕, position) で最も早い行。時間帯を順序に含めないと同じ日の昼と夕が両方当たる (revise-mealplan-spread-api D6)。
// 候補の時間帯は問わない (別時間帯の作り置きも
// 候補に含める — proposal: add-mealplan-cross-meal-leftover)。BatchSummary の Meal
// フィールドで候補側の時間帯を返し、UI 側で「月曜・昼食」のように表示分岐できる。
func (r *MealPlanRepo) ListMergeCandidates(ctx context.Context, entryID string) ([]mealplan.BatchSummary, error) {
	own, err := loadEntryByID(ctx, r.db, entryID)
	if err != nil {
		return nil, err
	}
	ownBatchID, recipeID, date := own.BatchID, own.RecipeID, own.Date

	dayRange := r.mergeDayRange()
	rows, err := r.db.QueryContext(ctx, `
		SELECT g.batch_id, g.date, g.meal, g.recipe_id, r.name, g.cnt
		FROM (
			SELECT batch_id, date, meal, recipe_id, position,
			       COUNT(*) OVER (PARTITION BY batch_id) AS cnt,
			       ROW_NUMBER() OVER (
			           PARTITION BY batch_id
			           ORDER BY date,
			                    CASE meal WHEN 'breakfast' THEN 0 WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END,
			                    position
			       ) AS rn
			FROM meal_plans
			WHERE recipe_id = ?
			  AND ABS(julianday(date) - julianday(?)) <= ?
		) g
		JOIN recipes r ON r.id = g.recipe_id
		WHERE g.rn = 1 AND g.batch_id <> ?
		ORDER BY g.date,
		         CASE g.meal WHEN 'breakfast' THEN 0 WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END,
		         g.position
	`, recipeID, date, dayRange, ownBatchID)
	if err != nil {
		return nil, fmt.Errorf("query merge candidates: %w", err)
	}
	defer rows.Close()

	out := make([]mealplan.BatchSummary, 0)
	for rows.Next() {
		var s mealplan.BatchSummary
		var mealStr string
		if err := rows.Scan(&s.BatchID, &s.Date, &mealStr, &s.RecipeID, &s.RecipeName, &s.MemberCount); err != nil {
			return nil, fmt.Errorf("scan merge candidate: %w", err)
		}
		s.Meal = mealplan.Meal(mealStr)
		out = append(out, s)
	}
	return out, rows.Err()
}

// MealEntries は特定 (date, meal) の Entry 列を返す。CopyLeftover の domain 呼び出し用ヘルパ。
func (r *MealPlanRepo) MealEntries(ctx context.Context, date, meal string) ([]mealplan.Entry, error) {
	return loadMealEntries(ctx, r.db, date, meal)
}

// Spread は起点行の品を、targets の各セルに 1 行ずつ入れる。全セルを 1 トランザクションで反映する
// (revise-mealplan-spread-api design.md D3)。取り消しに使うため、追加した行の ID と上書きで消した行を返す
// (revise-mealplan-spread-safety design.md D2)。
func (r *MealPlanRepo) Spread(
	ctx context.Context,
	entryID string,
	targets []mealplan.SpreadTarget,
	mode mealplan.CopyMode,
	shareBatch mealplan.ShareBatch,
) (mealplan.SpreadResult, error) {
	result := mealplan.SpreadResult{AddedEntryIDs: []string{}, Removed: []mealplan.RemovedEntry{}}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	src, err := loadEntryByID(ctx, tx, entryID)
	if err != nil {
		return result, err
	}
	recipeID, sourceBatchID := src.RecipeID, src.BatchID
	if err := ensureNotArchived(ctx, tx, []string{recipeID}); err != nil {
		return result, err
	}

	sorted := append([]mealplan.SpreadTarget(nil), targets...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Date != sorted[j].Date {
			return sorted[i].Date < sorted[j].Date
		}
		return mealplan.MealRank(sorted[i].Meal) < mealplan.MealRank(sorted[j].Meal)
	})

	for _, tg := range sorted {
		if mode == mealplan.CopyModeOverwrite {
			removed, err := loadRemovedEntries(ctx, tx, `m.date = ? AND m.meal = ?`, tg.Date, string(tg.Meal))
			if err != nil {
				return result, err
			}
			result.Removed = append(result.Removed, removed...)
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM meal_plans WHERE date = ? AND meal = ?
			`, tg.Date, string(tg.Meal)); err != nil {
				return result, fmt.Errorf("clear target %s/%s: %w", tg.Date, tg.Meal, err)
			}
		} else {
			var dup int
			if err := tx.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM meal_plans
				WHERE date = ? AND meal = ? AND recipe_id = ?
			`, tg.Date, string(tg.Meal), recipeID).Scan(&dup); err != nil {
				return result, fmt.Errorf("check target duplicate: %w", err)
			}
			if dup > 0 {
				continue
			}
		}
		batchID := sourceBatchID
		if !shareBatch.IsShared() {
			batchID = mealplan.NewBatchID()
		}
		pos, err := nextPosition(ctx, tx, tg.Date, string(tg.Meal))
		if err != nil {
			return result, err
		}
		id := mealplan.NewEntryID()
		if _, err := insertMealPlan(ctx, tx, id, tg.Date, string(tg.Meal), pos, recipeID, batchID); err != nil {
			return result, err
		}
		result.AddedEntryIDs = append(result.AddedEntryIDs, id)
	}
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit tx: %w", err)
	}
	return result, nil
}

// UndoSpread は Spread の結果を 1 トランザクションで戻す。追加した行のうち別の操作で消えたものは飛ばし、
// 上書きで消した行は RestoreEntries と同じ規則で元のセルの元の並び順に戻す (revise-mealplan-spread-safety design.md D2)。
func (r *MealPlanRepo) UndoSpread(ctx context.Context, result mealplan.SpreadResult) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	type cell struct{ date, meal string }
	affected := make([]cell, 0)
	seen := map[cell]bool{}
	for _, id := range result.AddedEntryIDs {
		var c cell
		err := tx.QueryRowContext(ctx, `SELECT date, meal FROM meal_plans WHERE id = ?`, id).Scan(&c.date, &c.meal)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("locate added row: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM meal_plans WHERE id = ?`, id); err != nil {
			return fmt.Errorf("delete added row: %w", err)
		}
		if !seen[c] {
			seen[c] = true
			affected = append(affected, c)
		}
	}
	for _, c := range affected {
		if err := renumberMeal(ctx, tx, c.date, c.meal); err != nil {
			return err
		}
	}
	for _, e := range sortRemovedEntries(result.Removed) {
		if err := restoreEntryTx(ctx, tx, e); err != nil {
			return err
		}
	}
	if err := pruneCookedMarks(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// SetCooked は指定行の (batch_id, recipe_id) の組に作った印を付け外しする。該当行が無ければ ErrEntryNotFound。
// 二重に付けても外してもエラーにしない (design.md D3)。
func (r *MealPlanRepo) SetCooked(ctx context.Context, entryID string, cooked bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	entry, err := loadEntryByID(ctx, tx, entryID)
	if err != nil {
		return err
	}
	recipeID, batchID := entry.RecipeID, entry.BatchID

	if cooked {
		cookedAt := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO meal_plan_cooked (batch_id, recipe_id, cooked_at) VALUES (?, ?, ?)
			ON CONFLICT (batch_id, recipe_id) DO NOTHING
		`, batchID, recipeID, cookedAt); err != nil {
			return fmt.Errorf("insert meal_plan_cooked: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM meal_plan_cooked WHERE batch_id = ? AND recipe_id = ?
		`, batchID, recipeID); err != nil {
			return fmt.Errorf("delete meal_plan_cooked: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// --- 内部ヘルパ ------------------------------------------------------------

type querier interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
}

// Entry は ID で 1 行を読む。行が無ければ ErrEntryNotFound。
func (r *MealPlanRepo) Entry(ctx context.Context, entryID string) (mealplan.Entry, error) {
	return loadEntryByID(ctx, r.db, entryID)
}

// loadEntryByID は ID から行の住所 (日付・時間帯・並び順) と中身を読む。行が無ければ ErrEntryNotFound。
func loadEntryByID(ctx context.Context, q querier, id string) (mealplan.Entry, error) {
	var e mealplan.Entry
	var mealStr string
	err := q.QueryRowContext(ctx, `
		SELECT id, date, meal, position, recipe_id, batch_id FROM meal_plans WHERE id = ?
	`, id).Scan(&e.ID, &e.Date, &mealStr, &e.Position, &e.RecipeID, &e.BatchID)
	if errors.Is(err, sql.ErrNoRows) {
		return mealplan.Entry{}, mealplan.ErrEntryNotFound
	}
	if err != nil {
		return mealplan.Entry{}, fmt.Errorf("lookup meal_plans row: %w", err)
	}
	e.Meal = mealplan.Meal(mealStr)
	return e, nil
}

func loadMealEntries(ctx context.Context, q querier, date, meal string) ([]mealplan.Entry, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT m.id, m.date, m.meal, m.position, m.recipe_id, COALESCE(r.name, ''), m.batch_id
		FROM meal_plans m
		LEFT JOIN recipes r ON r.id = m.recipe_id
		WHERE m.date = ? AND m.meal = ?
		ORDER BY m.position
	`, date, meal)
	if err != nil {
		return nil, fmt.Errorf("query meal: %w", err)
	}
	defer rows.Close()
	out := make([]mealplan.Entry, 0)
	for rows.Next() {
		var e mealplan.Entry
		var mealStr string
		if err := rows.Scan(&e.ID, &e.Date, &mealStr, &e.Position, &e.RecipeID, &e.RecipeName, &e.BatchID); err != nil {
			return nil, fmt.Errorf("scan meal row: %w", err)
		}
		e.Meal = mealplan.Meal(mealStr)
		out = append(out, e)
	}
	return out, rows.Err()
}

// copyCookedMark は (fromBatchID, recipeID) に印があれば (toBatchID, recipeID) にも印を足す。
// cooked_at は元の印の値を引き継ぐ。
func copyCookedMark(ctx context.Context, q querier, fromBatchID, toBatchID, recipeID string) error {
	if _, err := q.ExecContext(ctx, `
		INSERT INTO meal_plan_cooked (batch_id, recipe_id, cooked_at)
		SELECT ?, recipe_id, cooked_at FROM meal_plan_cooked
		WHERE batch_id = ? AND recipe_id = ?
		ON CONFLICT (batch_id, recipe_id) DO NOTHING
	`, toBatchID, fromBatchID, recipeID); err != nil {
		return fmt.Errorf("copy cooked mark: %w", err)
	}
	return nil
}

// pruneCookedMarks は meal_plans に対応する行の無くなった印を消す (design.md D4)。
func pruneCookedMarks(ctx context.Context, q querier) error {
	if _, err := q.ExecContext(ctx, `
		DELETE FROM meal_plan_cooked
		WHERE NOT EXISTS (
			SELECT 1 FROM meal_plans m
			WHERE m.batch_id = meal_plan_cooked.batch_id AND m.recipe_id = meal_plan_cooked.recipe_id
		)
	`); err != nil {
		return fmt.Errorf("prune cooked marks: %w", err)
	}
	return nil
}

func nextPosition(ctx context.Context, q querier, date, meal string) (int, error) {
	var maxPos sql.NullInt64
	if err := q.QueryRowContext(ctx, `
		SELECT MAX(position) FROM meal_plans WHERE date = ? AND meal = ?
	`, date, meal).Scan(&maxPos); err != nil {
		return 0, fmt.Errorf("query max position: %w", err)
	}
	if !maxPos.Valid {
		return 0, nil
	}
	return int(maxPos.Int64) + 1, nil
}

func insertMealPlan(
	ctx context.Context,
	q querier,
	id string,
	date, meal string,
	position int,
	recipeID string,
	batchID string,
) (mealplan.Entry, error) {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := q.ExecContext(ctx, `
		INSERT INTO meal_plans (id, date, meal, position, recipe_id, created_at, batch_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, date, meal, position, recipeID, createdAt, batchID); err != nil {
		if isForeignKeyErr(err) {
			return mealplan.Entry{}, mealplan.ErrUnknownRecipe
		}
		return mealplan.Entry{}, fmt.Errorf("insert meal_plans: %w", err)
	}
	var name string
	if err := q.QueryRowContext(ctx, `SELECT name FROM recipes WHERE id = ?`, recipeID).Scan(&name); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return mealplan.Entry{}, fmt.Errorf("load recipe name: %w", err)
		}
	}
	return mealplan.Entry{
		ID:         id,
		Date:       date,
		Meal:       mealplan.Meal(meal),
		Position:   position,
		RecipeID:   recipeID,
		RecipeName: name,
		BatchID:    batchID,
	}, nil
}

// prepareCopySource は shareBatch=true のときに source[i].RecipeID → 共通 BatchID の
// map を作って source を書き換える。同じレシピは全 target で 1 バッチにまとめ、
// 異なるレシピ (肉じゃが / みそ汁 など) はそれぞれ別のバッチに割り振る。
// 起点行の既存 BatchID があればそれをそのまま共有する (spec 「起点行のバッチ ID は
// コピー元と共有される」)。無ければ新 UUID を発行する。
// shareBatch=false のときは PlanCopy が 1 行ごとに新 UUID を発行するので、source を触らず返す。
func prepareCopySource(source []mealplan.Entry, shareBatch mealplan.ShareBatch) []mealplan.Entry {
	if !shareBatch.IsShared() {
		return source
	}
	byRecipe := map[string]string{}
	out := make([]mealplan.Entry, len(source))
	for i, e := range source {
		bid, ok := byRecipe[e.RecipeID]
		if !ok {
			if e.BatchID != "" {
				bid = e.BatchID
			} else {
				bid = mealplan.NewBatchID()
			}
			byRecipe[e.RecipeID] = bid
		}
		out[i] = e
		out[i].BatchID = bid
	}
	return out
}

// ensureActiveRecipe は新たに献立へ入れるレシピが未削除かを確かめる。
// 削除済みは FK を満たしてしまうので、存在しない場合と同じ ErrUnknownRecipe に寄せる (add-recipe-deletion D4)。
func ensureActiveRecipe(ctx context.Context, q querier, recipeID string) error {
	var one int
	err := q.QueryRowContext(ctx, `
		SELECT 1 FROM recipes WHERE id = ? AND archived_at IS NULL
	`, recipeID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return mealplan.ErrUnknownRecipe
	}
	if err != nil {
		return fmt.Errorf("check recipe active: %w", err)
	}
	return nil
}

// ensureNotArchived は献立にある品を別のセルへ増やす前に、レシピが削除済みでないかを確かめる。
// Assign と違い利用者の目の前にその品が見えているので、存在しない場合と区別して ErrArchivedRecipe を返す (revise-mealplan-spread-safety D5)。
func ensureNotArchived(ctx context.Context, q querier, recipeIDs []string) error {
	for _, id := range recipeIDs {
		var archived int
		if err := q.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM recipes WHERE id = ? AND archived_at IS NOT NULL
		`, id).Scan(&archived); err != nil {
			return fmt.Errorf("check recipe archived: %w", err)
		}
		if archived > 0 {
			return mealplan.ErrArchivedRecipe
		}
	}
	return nil
}

// isForeignKeyErr は FK 違反 (存在しない recipe_id) を判定する。
// modernc.org/sqlite は Error interface を露出しないので、メッセージで拾う。
func isForeignKeyErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

func addDays(date string, days int) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", mealplan.ErrInvalidDate
	}
	return t.AddDate(0, 0, days).Format("2006-01-02"), nil
}
