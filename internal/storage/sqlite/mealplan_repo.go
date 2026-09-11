package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

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

// ListWeek は weekStart (月曜 "YYYY-MM-DD") を含む 7 日分の Entry を position 順で返す。
// recipes と JOIN してレシピ名を解決し、batch_member_count はウィンドウ関数で「同じ batch_id
// を持つ全行の件数」を返す (design.md D6)。週外の行も PARTITION に加算されるのが仕様。
func (r *MealPlanRepo) ListWeek(ctx context.Context, weekStart string) ([]mealplan.Entry, error) {
	end, err := addDays(weekStart, 7)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.date, m.slot, m.position, m.recipe_id, r.name, m.batch_id,
		       COUNT(*) OVER (PARTITION BY m.batch_id) AS batch_member_count
		FROM meal_plans m
		JOIN recipes r ON r.id = m.recipe_id
		WHERE m.date >= ? AND m.date < ?
		ORDER BY m.date, m.slot, m.position
	`, weekStart, end)
	if err != nil {
		return nil, fmt.Errorf("query meal_plans: %w", err)
	}
	defer rows.Close()

	out := make([]mealplan.Entry, 0)
	for rows.Next() {
		var e mealplan.Entry
		var slot string
		if err := rows.Scan(&e.Date, &slot, &e.Position, &e.RecipeID, &e.RecipeName,
			&e.BatchID, &e.BatchMemberCount); err != nil {
			return nil, fmt.Errorf("scan meal_plans row: %w", err)
		}
		e.Slot = mealplan.Slot(slot)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meal_plans: %w", err)
	}
	return out, nil
}

// Assign は対象 (date, slot) の末尾 (position = max+1) に recipe_id を追加する。
// 単独割当は「新しい 1 バッチ」の意味なので、常に新 batch_id を発行する。
func (r *MealPlanRepo) Assign(ctx context.Context, date, slot, recipeID string) (mealplan.Entry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mealplan.Entry{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	pos, err := nextPosition(ctx, tx, date, slot)
	if err != nil {
		return mealplan.Entry{}, err
	}
	batchID := mealplan.NewBatchID()
	entry, err := insertMealPlan(ctx, tx, date, slot, pos, recipeID, batchID)
	if err != nil {
		return mealplan.Entry{}, err
	}
	if err := tx.Commit(); err != nil {
		return mealplan.Entry{}, fmt.Errorf("commit tx: %w", err)
	}
	return entry, nil
}

// Replace は指定 (date, slot, position) の recipe_id を差し替える。該当行が無ければ ErrEntryNotFound。
// batch_id は変更しない (「同じバッチのまま中身のレシピを差し替える」意味)。
func (r *MealPlanRepo) Replace(ctx context.Context, date, slot string, position int, recipeID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE meal_plans SET recipe_id = ? WHERE date = ? AND slot = ? AND position = ?
	`, recipeID, date, slot, position)
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
	return nil
}

// Remove は 1 行削除し、後続 position を 1 つずつ詰め直す。
// 詰め直しは (date, slot, position) が PRIMARY KEY のため、下から上に UPDATE する必要は無く、
// 小さい position 側から順に上書きしていけば衝突しない (position=3 → 2、position=4 → 3、...)。
func (r *MealPlanRepo) Remove(ctx context.Context, date, slot string, position int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		DELETE FROM meal_plans WHERE date = ? AND slot = ? AND position = ?
	`, date, slot, position)
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
	if _, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET position = position - 1
		WHERE date = ? AND slot = ? AND position > ?
	`, date, slot, position); err != nil {
		return fmt.Errorf("renumber positions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// ClearSlot は (date, slot) の全 position を削除する。0 件でもエラーにしない。
func (r *MealPlanRepo) ClearSlot(ctx context.Context, date, slot string) error {
	if _, err := r.db.ExecContext(ctx, `
		DELETE FROM meal_plans WHERE date = ? AND slot = ?
	`, date, slot); err != nil {
		return fmt.Errorf("clear slot: %w", err)
	}
	return nil
}

// Reorder は orderedRecipeIDs の順で position を 0..N-1 に採番し直す。
// 既存 slot の recipe_id 集合と一致しない場合は ErrReorderMismatch。
// PRIMARY KEY (date, slot, position) の衝突を避けるため、一旦 +1000 のオフセットに退避する。
func (r *MealPlanRepo) Reorder(ctx context.Context, date, slot string, orderedRecipeIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `
		SELECT recipe_id FROM meal_plans WHERE date = ? AND slot = ?
	`, date, slot)
	if err != nil {
		return fmt.Errorf("query current slot: %w", err)
	}
	current := map[string]int{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan current slot: %w", err)
		}
		current[id]++
	}
	rows.Close()

	requested := map[string]int{}
	for _, id := range orderedRecipeIDs {
		requested[id]++
	}
	if len(current) != len(requested) {
		return mealplan.ErrReorderMismatch
	}
	for id, n := range current {
		if requested[id] != n {
			return mealplan.ErrReorderMismatch
		}
	}

	// 一時オフセット (+1000) に退避して PRIMARY KEY 衝突を避ける。slot 内の品数は最大でも
	// 数個〜十数個の想定で、+1000 で衝突する状況は現実的に無い。
	if _, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET position = position + 1000 WHERE date = ? AND slot = ?
	`, date, slot); err != nil {
		return fmt.Errorf("shift positions: %w", err)
	}

	// modernc.org/sqlite は UPDATE ... LIMIT 未対応。SELECT で最小 position の 1 件を取り、
	// 完全一致で UPDATE する 2 段構えにする。同一 recipe_id の重複を許容するため。
	for newPos, id := range orderedRecipeIDs {
		var oldPos int
		err := tx.QueryRowContext(ctx, `
			SELECT position FROM meal_plans
			WHERE date = ? AND slot = ? AND recipe_id = ? AND position >= 1000
			ORDER BY position LIMIT 1
		`, date, slot, id).Scan(&oldPos)
		if errors.Is(err, sql.ErrNoRows) {
			return mealplan.ErrReorderMismatch
		}
		if err != nil {
			return fmt.Errorf("locate row for reorder: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE meal_plans SET position = ?
			WHERE date = ? AND slot = ? AND position = ?
		`, newPos, date, slot, oldPos); err != nil {
			return fmt.Errorf("reorder position %d: %w", newPos, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// CopyLeftover は targets の各日付 slot に sourceEntries を反映する。mode=overwrite は既存を消してから、
// mode=append は既存の末尾に追加する (二重追加防止のため domain.PlanCopy を経由)。
//
// shareBatch=true (作り置き) は「同じレシピ ID を持つコピー先行は全 target で 1 batch_id」を割り振る。
// この関数の入口で source[i] の RecipeID → 共通 batch_id の map を作り、各 target の PlanCopy に
// BatchID を注入した source を渡す。shareBatch=false (別調理) は PlanCopy 内で 1 行ごとに新 UUID を
// 発行する。
func (r *MealPlanRepo) CopyLeftover(
	ctx context.Context,
	slot string,
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

	source := prepareCopySource(sourceEntries, shareBatch)

	for _, target := range targets {
		if mode == mealplan.CopyModeOverwrite {
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM meal_plans WHERE date = ? AND slot = ?
			`, target, slot); err != nil {
				return fmt.Errorf("clear target %s: %w", target, err)
			}
		}
		existing, err := loadSlotEntries(ctx, tx, target, slot)
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
			if _, err := insertMealPlan(ctx, tx, e.Date, string(e.Slot), e.Position, e.RecipeID, batchID); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// SplitBatch は「作り置きから外す」= 連続 N 日コピーの逆演算を実行する。
// 対象行が属する作り置き全体を対象に、起点日 (バッチ内で最も古い日付 → MIN(position) を tiebreaker
// にした行) を残し、他の同バッチ行を全削除。残った起点行に新 batch_id を発行して返す。
// バッチが元々 1 行しか含まない場合は no-op で現行 batch_id を返す (design.md D4a)。
func (r *MealPlanRepo) SplitBatch(ctx context.Context, date, slot string, position int) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentBatchID string
	err = tx.QueryRowContext(ctx, `
		SELECT batch_id FROM meal_plans WHERE date = ? AND slot = ? AND position = ?
	`, date, slot, position).Scan(&currentBatchID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", mealplan.ErrEntryNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lookup batch_id: %w", err)
	}

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

	var originDate, originSlot string
	var originPosition int
	if err := tx.QueryRowContext(ctx, `
		SELECT date, slot, position FROM meal_plans
		WHERE batch_id = ?
		ORDER BY date, position
		LIMIT 1
	`, currentBatchID).Scan(&originDate, &originSlot, &originPosition); err != nil {
		return "", fmt.Errorf("locate origin: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT date, slot, position FROM meal_plans
		WHERE batch_id = ? AND NOT (date = ? AND slot = ? AND position = ?)
		ORDER BY date, slot, position
	`, currentBatchID, originDate, originSlot, originPosition)
	if err != nil {
		return "", fmt.Errorf("query siblings: %w", err)
	}
	type cellRow struct {
		date, slot string
		position   int
	}
	toDelete := make([]cellRow, 0)
	for rows.Next() {
		var cr cellRow
		if err := rows.Scan(&cr.date, &cr.slot, &cr.position); err != nil {
			rows.Close()
			return "", fmt.Errorf("scan siblings: %w", err)
		}
		toDelete = append(toDelete, cr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate siblings: %w", err)
	}

	for _, cr := range toDelete {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM meal_plans WHERE date = ? AND slot = ? AND position = ?
		`, cr.date, cr.slot, cr.position); err != nil {
			return "", fmt.Errorf("delete sibling row: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE meal_plans SET position = position - 1
			WHERE date = ? AND slot = ? AND position > ?
		`, cr.date, cr.slot, cr.position); err != nil {
			return "", fmt.Errorf("renumber positions in cell: %w", err)
		}
	}

	newID := mealplan.NewBatchID()
	if _, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET batch_id = ?
		WHERE date = ? AND slot = ? AND position = ?
	`, newID, originDate, originSlot, originPosition); err != nil {
		return "", fmt.Errorf("assign new batch_id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return newID, nil
}

// MergeBatch は対象行の batch_id を targetBatchID に付け替える。
// 3 条件 (同 slot / 同一レシピ / 対象行の日付から ±mergeDayRange 日以内) を防御的にチェックし、
// 違反時は ErrBatchMergeDenied、そもそも targetBatchID の行が同 slot に無ければ
// ErrBatchTargetNotFound を返す (design.md D4b)。
func (r *MealPlanRepo) MergeBatch(
	ctx context.Context, date, slot string, position int, targetBatchID string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sourceRecipeID string
	err = tx.QueryRowContext(ctx, `
		SELECT recipe_id FROM meal_plans WHERE date = ? AND slot = ? AND position = ?
	`, date, slot, position).Scan(&sourceRecipeID)
	if errors.Is(err, sql.ErrNoRows) {
		return mealplan.ErrEntryNotFound
	}
	if err != nil {
		return fmt.Errorf("lookup source row: %w", err)
	}

	var targetExists int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM meal_plans
		WHERE slot = ? AND batch_id = ? AND NOT (date = ? AND position = ?)
	`, slot, targetBatchID, date, position).Scan(&targetExists); err != nil {
		return fmt.Errorf("check target batch: %w", err)
	}
	if targetExists == 0 {
		return mealplan.ErrBatchTargetNotFound
	}

	dayRange := r.mergeDayRange()
	var eligible int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM meal_plans
		WHERE slot = ? AND batch_id = ? AND recipe_id = ?
		  AND NOT (date = ? AND position = ?)
		  AND ABS(julianday(date) - julianday(?)) <= ?
	`, slot, targetBatchID, sourceRecipeID, date, position, date, dayRange).Scan(&eligible); err != nil {
		return fmt.Errorf("check merge eligibility: %w", err)
	}
	if eligible == 0 {
		return mealplan.ErrBatchMergeDenied
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE meal_plans SET batch_id = ?
		WHERE date = ? AND slot = ? AND position = ?
	`, targetBatchID, date, slot, position)
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
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// ListSlotBatches は (date, slot, recipeID) から見た合流可能な作り置きの候補を列挙する。
// フィルタ条件: 同じ slot / 同一 recipeID / 起点行 (date) から ±mergeDayRange 日以内 /
// 起点行自身が属するバッチは除外 (design.md D4b)。各バッチにつき代表行 1 つを返す。
func (r *MealPlanRepo) ListSlotBatches(
	ctx context.Context, date, slot, recipeID string,
) ([]mealplan.BatchSummary, error) {
	var ownBatchID string
	err := r.db.QueryRowContext(ctx, `
		SELECT batch_id FROM meal_plans
		WHERE date = ? AND slot = ?
		LIMIT 1
	`, date, slot).Scan(&ownBatchID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("query own batch: %w", err)
	}

	dayRange := r.mergeDayRange()
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.batch_id, m.date, m.slot, m.recipe_id, r.name, cnt
		FROM (
			SELECT batch_id, MIN(date || printf('%04d', position)) AS pk_key,
			       COUNT(*) AS cnt
			FROM meal_plans
			WHERE slot = ? AND recipe_id = ?
			  AND ABS(julianday(date) - julianday(?)) <= ?
			GROUP BY batch_id
		) g
		JOIN meal_plans m
		  ON m.batch_id = g.batch_id
		 AND (m.date || printf('%04d', m.position)) = g.pk_key
		JOIN recipes r ON r.id = m.recipe_id
		WHERE m.batch_id <> ?
		ORDER BY m.date, m.position
	`, slot, recipeID, date, dayRange, ownBatchID)
	if err != nil {
		return nil, fmt.Errorf("query slot batches: %w", err)
	}
	defer rows.Close()

	out := make([]mealplan.BatchSummary, 0)
	for rows.Next() {
		var s mealplan.BatchSummary
		var slotStr string
		if err := rows.Scan(&s.BatchID, &s.Date, &slotStr, &s.RecipeID, &s.RecipeName, &s.MemberCount); err != nil {
			return nil, fmt.Errorf("scan slot batch: %w", err)
		}
		s.Slot = mealplan.Slot(slotStr)
		out = append(out, s)
	}
	return out, rows.Err()
}

// SlotEntries は特定 (date, slot) の Entry 列を返す。CopyLeftover の domain 呼び出し用ヘルパ。
func (r *MealPlanRepo) SlotEntries(ctx context.Context, date, slot string) ([]mealplan.Entry, error) {
	return loadSlotEntries(ctx, r.db, date, slot)
}

// --- 内部ヘルパ ------------------------------------------------------------

type querier interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
}

func loadSlotEntries(ctx context.Context, q querier, date, slot string) ([]mealplan.Entry, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT m.date, m.slot, m.position, m.recipe_id, COALESCE(r.name, ''), m.batch_id
		FROM meal_plans m
		LEFT JOIN recipes r ON r.id = m.recipe_id
		WHERE m.date = ? AND m.slot = ?
		ORDER BY m.position
	`, date, slot)
	if err != nil {
		return nil, fmt.Errorf("query slot: %w", err)
	}
	defer rows.Close()
	out := make([]mealplan.Entry, 0)
	for rows.Next() {
		var e mealplan.Entry
		var slotStr string
		if err := rows.Scan(&e.Date, &slotStr, &e.Position, &e.RecipeID, &e.RecipeName, &e.BatchID); err != nil {
			return nil, fmt.Errorf("scan slot row: %w", err)
		}
		e.Slot = mealplan.Slot(slotStr)
		out = append(out, e)
	}
	return out, rows.Err()
}

func nextPosition(ctx context.Context, q querier, date, slot string) (int, error) {
	var maxPos sql.NullInt64
	if err := q.QueryRowContext(ctx, `
		SELECT MAX(position) FROM meal_plans WHERE date = ? AND slot = ?
	`, date, slot).Scan(&maxPos); err != nil {
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
	date, slot string,
	position int,
	recipeID string,
	batchID string,
) (mealplan.Entry, error) {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := q.ExecContext(ctx, `
		INSERT INTO meal_plans (date, slot, position, recipe_id, created_at, batch_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, date, slot, position, recipeID, createdAt, batchID); err != nil {
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
		Date:       date,
		Slot:       mealplan.Slot(slot),
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

// isForeignKeyErr は FK 違反 (存在しない recipe_id) を判定する。
// modernc.org/sqlite は Error interface を露出しないので、メッセージで拾う。
func isForeignKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "FOREIGN KEY constraint failed") ||
		strings.Contains(msg, "constraint failed")
}

func addDays(date string, days int) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", mealplan.ErrInvalidDate
	}
	return t.AddDate(0, 0, days).Format("2006-01-02"), nil
}
