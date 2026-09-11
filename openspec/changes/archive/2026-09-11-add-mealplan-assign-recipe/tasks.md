## 1. ドメイン層 (internal/domain/mealplan)

- [x] 1.1 `internal/domain/mealplan/entry.go` を更新し、`Entry` struct に `RecipeID string` フィールドを追加する
      (`RecipeName` は表示用としてそのまま残す。完了判定: `go build ./internal/...` が通る)
- [x] 1.2a `internal/domain/mealplan/validate.go` を新設し、以下の型付きエラーを宣言する: `ErrInvalidDate` /
      `ErrInvalidSlot` / `ErrInvalidPosition` / `ErrEntryNotFound` / `ErrReorderMismatch` / `ErrInvalidCopyDays` /
      `ErrInvalidCopyMode` (完了判定: `go build ./internal/...` が通る)
- [x] 1.2b `internal/domain/mealplan/validate.go` に `ValidateDate(s string) error` (RFC3339 の date-only 形式のみ
      許可) を実装する (完了判定: 下記 1.2d のテストが pass する)
- [x] 1.2c `internal/domain/mealplan/validate.go` に `ValidateSlot(s string) error` (breakfast|lunch|dinner のみ
      許可) と `ValidatePosition(p int) error` (p >= 0 のみ許可) を実装する (完了判定: 下記 1.2d のテストが pass
      する)
- [x] 1.2d `internal/domain/mealplan/validate_test.go` を新設し、以下の table-driven テストを追加する
    - ValidateDate: "2026-08-24" は nil、"2026/08/24" は ErrInvalidDate、"20260824" は ErrInvalidDate、
      "2026-13-01" (存在しない月) は ErrInvalidDate、空文字は ErrInvalidDate
    - ValidateSlot: "breakfast"/"lunch"/"dinner" は nil、"snack" は ErrInvalidSlot、空文字は ErrInvalidSlot、
      大文字 "Dinner" は ErrInvalidSlot
    - ValidatePosition: 0 は nil、1 は nil、-1 は ErrInvalidPosition
    - (完了判定: `go test ./internal/domain/mealplan/...` が pass する)
- [x] 1.2 `internal/domain/mealplan/copy.go` を新設し、`CopyDates(startDate string, days int) ([]string, error)` を
      実装する (起点日を含む合計 days 日ぶんの "YYYY-MM-DD" 配列を返す。完了判定: 下記 1.3 のテストが pass する)
- [x] 1.3 `internal/domain/mealplan/copy_test.go` を新設し、`CopyDates` の table-driven テストを追加する
    - days=1 は起点日 1 個だけを返す
    - days=3 は起点日を含む 3 日連続の日付を返す
    - 月末をまたぐ (例: 2026-01-30 起点で 3 日) と正しく翌月に繰り越す
    - days=0 は空配列を返す
    - days<0 は `ErrInvalidCopyDays` エラーを返す
    - startDate が "YYYY-MM-DD" 形式でない場合は `ErrInvalidDate` エラーを返す
    - (完了判定: `go test ./internal/domain/mealplan/...` が pass する)
- [x] 1.4 `internal/domain/mealplan/copy.go` に `PlanCopy(target string, sourceEntries []Entry, existing []Entry, mode
      string) []Entry` を追加する (mode="append" なら existing に続けて position を採番、mode="overwrite" なら
      existing を無視して 0 から採番。完了判定: 下記 1.5 のテストが pass する)
- [x] 1.5 `internal/domain/mealplan/copy_test.go` に `PlanCopy` の table-driven テストを追加する
    - existing が空・mode=append なら sourceEntries が position=0.. の順で返る
    - existing が 1 品・mode=append なら sourceEntries は position=1.. から採番される
    - existing が 2 品・mode=overwrite なら sourceEntries が position=0.. で返り、既存 2 品は返り値に含まれない
    - 起点日と target が同一日の場合、既存品と重複する RecipeID の source は返り値に含まれない (二重追加防止)
    - mode が append/overwrite 以外は `ErrInvalidCopyMode` エラーを返す
    - (完了判定: `go test ./internal/domain/mealplan/...` が pass する)

## 2. SQLite migration + repository (internal/storage/sqlite)

- [x] 2.1 `internal/storage/sqlite/migrations/00003_meal_plans.sql` を新設し、`meal_plans` テーブルを作る
      (カラム: `date TEXT NOT NULL`, `slot TEXT NOT NULL`, `position INTEGER NOT NULL`, `recipe_id TEXT NOT NULL
      REFERENCES recipes(id) ON DELETE CASCADE`, `created_at TEXT NOT NULL`。PRIMARY KEY (date, slot, position)。
      INDEX idx_meal_plans_date ON meal_plans(date)。goose Up / Down を書く。完了判定: `task dev` 起動時に
      goose migrate が成功する)
- [x] 2.2 `internal/storage/sqlite/mealplan_repo.go` を新設し、`MealPlanRepo` struct と `NewMealPlanRepo(db *sql.DB)`
      を追加する (完了判定: `go build ./internal/...` が通る)
- [x] 2.3 `MealPlanRepo.ListWeek(ctx, weekStart string) ([]mealplan.Entry, error)` を実装する (`recipes` と JOIN して
      `RecipeName` を解決、7 日ぶんを 1 クエリで返す、ORDER BY date, slot, position。完了判定: 下記 2.10 のテストが
      pass する)
- [x] 2.4 `MealPlanRepo.Assign(ctx, date, slot, recipeID string) (mealplan.Entry, error)` を実装する (対象 slot の
      現在 position 最大値 +1 で行を追加。完了判定: 下記 2.10 のテストが pass する)
- [x] 2.5 `MealPlanRepo.Replace(ctx, date, slot string, position int, recipeID string) error` を実装する
      (完了判定: 下記 2.10 のテストが pass する)
- [x] 2.6 `MealPlanRepo.Remove(ctx, date, slot string, position int) error` を実装する (削除後、より大きい position
      の行を 1 つずつ詰め直す。1 トランザクション内で下から順に UPDATE することで PRIMARY KEY 衝突を避ける。
      完了判定: 下記 2.10 のテストが pass する)
- [x] 2.7 `MealPlanRepo.ClearSlot(ctx, date, slot string) error` を実装する (完了判定: 下記 2.10 のテストが pass する)
- [x] 2.8 `MealPlanRepo.Reorder(ctx, date, slot string, orderedRecipeIDs []string) error` を実装する
      (既存 slot と recipeID セットが一致することを確認した上で position を新順序で再採番する。1 トランザクション内で
      一旦大きなオフセット (例: +1000) に退避してから採番し直すことで PRIMARY KEY 衝突を避ける。完了判定: 下記 2.10
      のテストが pass する)
- [x] 2.9 `MealPlanRepo.CopyLeftover(ctx, sourceDate, slot string, targets []string, sourceEntries []mealplan.Entry,
      mode string) error` を実装する (targets の各日付 slot に対し、mode に応じて append / overwrite を 1
      トランザクションで反映。完了判定: 下記 2.10 のテストが pass する)
- [x] 2.10 `internal/storage/sqlite/mealplan_repo_test.go` を新設し、`t.TempDir()` で実 SQLite を作って BDD 風テストを
      書く
    - should return empty slice when no meal plans exist
    - should return entries JOINed with recipes.name for the requested week
    - should ignore entries outside the requested week
    - Assign should append with position = max+1 when the slot already has items
    - Assign should return error when recipeID does not reference an existing recipe
    - Replace should update only the target row and keep other positions untouched
    - Remove should renumber subsequent positions to be contiguous
    - ClearSlot should delete every row in the target (date, slot) and leave other slots untouched
    - Reorder should renumber positions according to orderedRecipeIDs
    - Reorder should return error when orderedRecipeIDs mismatches the current slot content
    - CopyLeftover with mode=append should append source entries to targets without removing existing rows
    - CopyLeftover with mode=overwrite should first delete existing target rows and then insert source entries
    - CopyLeftover should skip duplicate recipe when source and target are the same date
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)

## 3. Wails サービス (internal/service/mealplan.go)

- [x] 3.1 `internal/service/mealplan.go` を更新し、`MealPlanService` に `repo *sqlite.MealPlanRepo` フィールドと
      `recipeRepo *sqlite.RecipeRepo` を持たせて `NewMealPlanService(mp *sqlite.MealPlanRepo, r *sqlite.RecipeRepo)`
      に差し替える (完了判定: `go build ./` が通る)
- [x] 3.2 `MealPlanService.ListWeek(ctx, weekStart string) ([]mealplan.Entry, error)` を repo 呼び出しに差し替え、
      固定 dataset を返すコードを削除する (完了判定: 下記 3.9 のテストが pass する)
- [x] 3.3 `MealPlanService.AssignRecipe(ctx, date, slot, recipeID string) (mealplan.Entry, error)` を追加する
      (完了判定: 下記 3.9 のテストが pass する)
- [x] 3.4 `MealPlanService.ReplaceEntry(ctx, date, slot string, position int, recipeID string) error` を追加する
      (完了判定: 下記 3.9 のテストが pass する)
- [x] 3.5 `MealPlanService.RemoveEntry(ctx, date, slot string, position int) error` を追加する (完了判定: 下記 3.9
      のテストが pass する)
- [x] 3.6 `MealPlanService.ClearSlot(ctx, date, slot string) error` を追加する (完了判定: 下記 3.9 のテストが pass
      する)
- [x] 3.7 `MealPlanService.ReorderSlot(ctx, date, slot string, orderedRecipeIDs []string) error` を追加する
      (完了判定: 下記 3.9 のテストが pass する)
- [x] 3.8 `MealPlanService.CopyLeftover(ctx, date, slot string, days int, position int, mode string) error` を追加
      する (position < 0 なら slot 全体、position >= 0 ならその 1 品のみコピー元とする。内部で `CopyDates` /
      `PlanCopy` を呼び、repo.CopyLeftover にまとめて渡す。完了判定: 下記 3.9 のテストが pass する)
- [x] 3.8a 全割当系メソッド (Assign / Replace / Remove / ClearSlot / Reorder / CopyLeftover) の入口で
      `ValidateDate` / `ValidateSlot` / `ValidatePosition` を呼び、違反時は即座にエラーを返す配線を追加する
      (完了判定: 下記 3.9 のバリデーション系ケースが pass する)
- [x] 3.8b `MealPlanService.CopyLeftover` の入口で days を `config.LeftoverCopyPresets` と照合し、含まれなければ
      `ErrInvalidCopyDays` を返す (call site にリテラル閾値を書かない。完了判定: 下記 3.9 の
      「days=5 で ErrInvalidCopyDays」ケースが pass する)
- [x] 3.9 `internal/service/mealplan_test.go` を更新し、既存の固定値テストを削除して BDD 風テストを追加する
    - should return entries from the repository when the week is requested
    - AssignRecipe should return an entry with a fresh position when the slot already has items
    - ReplaceEntry should replace only the target position and keep others
    - RemoveEntry should renumber positions when a middle entry is removed
    - ClearSlot should delete every entry in the target (date, slot)
    - ReorderSlot should return error when orderedRecipeIDs contain unknown recipe
    - CopyLeftover with position=-1 and mode=append should append every source entry into each target slot
    - CopyLeftover with position=0 and mode=overwrite should replace target slot with the single source item
    - CopyLeftover with days=1 should be a no-op (target list contains only the source date itself)
    - AssignRecipe with malformed date should return ErrInvalidDate and leave meal_plans untouched
    - AssignRecipe with slot="snack" should return ErrInvalidSlot and leave meal_plans untouched
    - ReplaceEntry with position=-1 should return ErrInvalidPosition and leave meal_plans untouched
    - ReplaceEntry against a non-existent position should return ErrEntryNotFound
    - RemoveEntry against a non-existent position should return ErrEntryNotFound
    - ReorderSlot with recipeIDs missing or duplicating an existing entry should return ErrReorderMismatch
    - CopyLeftover with days=5 should return ErrInvalidCopyDays when preset is [2, 3, 4]
    - (完了判定: `go test ./internal/service/...` が pass する)
- [x] 3.10 `main.go` を更新し、`NewMealPlanService` の呼び出しに `sqlite.NewMealPlanRepo(db)` と既存 `recipeRepo` を
      渡す (完了判定: `go build ./` が通り、`task dev` で起動が成功する)

## 4. Wails バインディング再生成

- [x] 4.1 `wails3 generate bindings` を実行し、`frontend/bindings/github.com/yito0000/meal-cycle/internal/service/`
      の `MealPlanService` に `AssignRecipe` / `ReplaceEntry` / `RemoveEntry` / `ClearSlot` / `ReorderSlot` /
      `CopyLeftover` メソッドが生成されることを確認する (完了判定: `frontend/bindings/` に該当メソッドが存在する)

## 5. i18n (frontend/src/locales)

- [x] 5.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に以下のキー群を追加する
    - `mealplan.picker.title` / `mealplan.picker.search_placeholder` / `mealplan.picker.empty` /
      `mealplan.picker.cancel`
    - `mealplan.cell.actions.add_first` / `mealplan.cell.actions.add_more` / `mealplan.cell.actions.slot_menu` /
      `mealplan.cell.actions.item_menu`
    - `mealplan.cell.item_menu.replace` / `mealplan.cell.item_menu.remove` / `mealplan.cell.item_menu.view_detail` /
      `mealplan.cell.item_menu.copy_days`
    - `mealplan.cell.slot_menu.clear` / `mealplan.cell.slot_menu.copy_days` / `mealplan.cell.slot_menu.reorder`
    - `mealplan.reorder.move_up` / `mealplan.reorder.move_down` / `mealplan.reorder.done`
    - `mealplan.copy.title` / `mealplan.copy.days_option` (with `{{days}}` interpolation) / `mealplan.copy.confirm`
    - `mealplan.copy.conflict.title` / `mealplan.copy.conflict.body` / `mealplan.copy.conflict.append` /
      `mealplan.copy.conflict.overwrite` / `mealplan.copy.conflict.cancel`
    - `mealplan.errors.invalid_date` / `mealplan.errors.invalid_slot` / `mealplan.errors.invalid_position` /
      `mealplan.errors.entry_not_found` / `mealplan.errors.reorder_mismatch` / `mealplan.errors.invalid_copy_days` /
      `mealplan.errors.unknown_recipe` / `mealplan.errors.unknown` (最後は未マップ用フォールバック)
    - (完了判定: `cd frontend && npm run type-check` が通り、`cd frontend && npm run build:dev` が成功する)

## 6. フロントエンド: レシピ選択ダイアログ

- [x] 6.1 `frontend/src/components/mealplan/recipe-picker-dialog.tsx` を新設し、`open` / `onOpenChange` /
      `onSelect(recipeId, recipeName)` の props を持つコンポーネントを実装する (`RecipeService.List` を TanStack
      Query で取得、名前部分一致で絞り込み。完了判定: `cd frontend && npm run type-check` が通る)
- [x] 6.2 shadcn/ui の Dialog プリミティブが `frontend/src/components/ui/dialog.tsx` に無ければ手動でコピーして追加
      する (完了判定: `import { Dialog } from "../ui/dialog"` が解決する)

## 7. フロントエンド: セル操作 UI

- [x] 7.1 `frontend/src/components/mealplan/meal-plan-cell.tsx` を更新し、空セル中央に `＋` ボタンを描画する
      (クリックで `recipe-picker-dialog` を開く。完了判定: `task dev` の S01 で空セルに `＋` が表示される)
- [x] 7.2 `frontend/src/components/mealplan/meal-plan-cell.tsx` を更新し、割当済セル右上に品追加 `＋` ボタンを追加
      する (クリックで `recipe-picker-dialog` を開く。完了判定: 目視で 1 品ある slot に右上 `＋` が表示される)
- [x] 7.3 `frontend/src/components/mealplan/cell-item-menu.tsx` を新設し、1 品行の `⋯` メニューを実装する (項目:
      入れ替え / 削除 / 詳細 / 連続 N 日。完了判定: 目視でメニューが開き、各項目がクリック可能)
- [x] 7.4 `frontend/src/components/mealplan/slot-menu.tsx` を新設し、slot 右下の `⋯` メニューを実装する (項目:
      全クリア / 全体連続 N 日 / 並び替え。完了判定: 目視で slot 右下 `⋯` からメニューが開く)
- [x] 7.5 shadcn/ui の DropdownMenu プリミティブが `frontend/src/components/ui/dropdown-menu.tsx` に無ければ手動で
      コピーして追加する (完了判定: `import { DropdownMenu } from "../ui/dropdown-menu"` が解決する)

## 8. フロントエンド: 作り置きコピーの衝突ダイアログ

- [x] 8.1 `frontend/src/components/mealplan/copy-conflict-dialog.tsx` を新設し、コピー先の (日付, slot) に既存割当が
      ある場合に「追加」「上書き」「キャンセル」の 3 択を出す (props: `conflictTargets: string[]`, `onSelect(mode)`。
      完了判定: `cd frontend && npm run type-check` が通る)
- [x] 8.2 `frontend/src/components/mealplan/copy-planner.ts` を新設し、フロント側で衝突予測を行うヘルパ
      `previewCopy(sourceDate, slot, days, allEntries) : { targets: string[], conflicts: string[] }` を実装する
      (完了判定: 下記 8.3 のテストが必要になったら追加)
- [x] 8.3 (テストランナー未導入のため保留) `copy-planner.ts` の単体テストは Vitest 導入 proposal 後に追加する。本
      proposal では behavior を `task test:e2e` の E2E で担保する (完了判定: 判断が本文に明記されている)

## 9. フロントエンド: ページ配線 (routes/meal-plan.tsx)

- [x] 9.1 `frontend/src/routes/meal-plan.tsx` を更新し、`useMutation` で `AssignRecipe` / `ReplaceEntry` /
      `RemoveEntry` / `ClearSlot` / `ReorderSlot` / `CopyLeftover` をラップし、成功時に
      `queryClient.invalidateQueries({ queryKey: ["mealplan", "week", weekStartKey] })` を呼ぶ (完了判定:
      `cd frontend && npm run type-check` が通り、`cd frontend && npm run build:dev` が成功する)
- [x] 9.1a `frontend/src/components/mealplan/error-messages.ts` を新設し、Go 側の型付きエラーコード
      (`ErrInvalidDate` 等) を i18n key (`mealplan.errors.*`) にマップするヘルパを実装する (未マップのエラーは
      `mealplan.errors.unknown` を返す。完了判定: `cd frontend && npm run type-check` が通る)
- [x] 9.1b 各 `useMutation` の `onError` でこのヘルパを介してトースト / インライン表示に文字列を渡す (完了判定:
      目視で、不正日付を強引に叩き込んだ場合に日本語メッセージが表示される)
- [x] 9.2 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、セルに `recipe-picker-dialog` /
      `cell-item-menu` / `slot-menu` / `copy-conflict-dialog` を配線する (完了判定: 目視で空セル `＋` からレシピを
      選ぶと画面に反映される)

## 10. E2E シナリオ (frontend/e2e)

- [x] 10.1 `frontend/e2e/mealplan.spec.ts` (無ければ新設) に、`docs/scenarios/mealplan.feature` の「献立への
      レシピ割当」機能ブロックの各シナリオ名で `test(...)` を追加する
    - 空セルに 1 品目を割り当てる
    - 既に 1 品ある slot にもう 1 品追加する
    - 品数が 3 を超えたら「他 N 件」で省略される
    - 1 品だけ入れ替える
    - 1 品だけ削除する
    - slot 全体をクリアする
    - 割当済みの 1 品から S03 詳細を開き、S01 に戻る
    - (完了判定: `task test:e2e` が pass する。まだ実装が追いつかないシナリオは `test.skip` で名前だけ残す)
- [x] 10.2 `frontend/e2e/mealplan.spec.ts` に、「作り置きコピー」機能ブロックのシナリオ名で `test(...)` を追加する
    - 1 品を連続 3 日に作り置きコピー
    - slot 全体（複数品）を連続 N 日コピー
    - コピー先に既存の割当があると追加 or 上書きを確認する
    - (完了判定: `task test:e2e` が pass する)

## 11. lint / format / 最終テスト

- [x] 11.1 `task lint` を実行し、Go / Frontend の両方でエラーがないことを確認する (完了判定: exit code 0)
- [x] 11.2 `task format` を実行し、差分が無い / 整形結果が意図どおりであることを確認する (完了判定: `git diff` が
      整形差分のみで、意味のある変更を上書きしていない)
- [x] 11.3 `go test ./internal/... ./` を実行し、全パッケージの単体テストが pass することを確認する
      (完了判定: exit code 0)
- [x] 11.4 `cd frontend && npm run type-check && npm run build:dev` を実行し、フロントの型チェックとビルドが通る
      ことを確認する (完了判定: exit code 0)
- [x] 11.5 `task test:e2e` を実行し、E2E シナリオが pass することを確認する (完了判定: exit code 0)
- [ ] 11.6 `task dev` で起動し、S01 で空セル → 割当 → 入替 → 削除 → 作り置きコピー → 衝突ダイアログの
      一連の流れを目視で確認する (完了判定: 各操作が UI に反映され、再起動後も残る)
