## 1. SQLite マイグレーション

- [x] 1.1 `internal/storage/sqlite/migrations/00005_rename_slot_to_meal.sql` を新設し、
      `ALTER TABLE meal_plans RENAME COLUMN slot TO meal;` を書く。goose の `-- +goose Up` /
      `-- +goose StatementBegin/StatementEnd` ヘッダは既存 `00004_meal_plans_batch.sql` に揃える。
      完了判定: ファイルが存在し、`grep '-- +goose Up' internal/storage/sqlite/migrations/00005_rename_slot_to_meal.sql`
      でヘッダが読める
- [x] 1.2 マイグレーションが動くことを検証する。**当初は `task dev` で確認としていたが、`go test
      ./internal/storage/sqlite/` が `sqlite.Open` 経由でテスト用一時 DB に全マイグレーションを
      流すため、実装差分レビュー時点では go test の全 pass が実質的な検証になる**。任意で
      `task dev` を叩けば `~/.local/share/meal-cycle/meal-cycle.db` にも 00005 が適用される
      (goose の自動 up)。完了判定: go test での migration up 実行がエラー無しで完了 (実測済み)

## 2. Go ドメイン層 (`internal/domain/mealplan/`) の型リネーム

- [x] 2.1 `internal/domain/mealplan/entry.go` を更新し、`Slot` 型を `Meal` に、定数
      `SlotBreakfast` / `SlotLunch` / `SlotDinner` を `MealBreakfast` / `MealLunch` / `MealDinner` に、
      `Entry.Slot` / `EntryInput.Slot` フィールドを `Meal` にリネームする。JSON タグの `"slot"` も
      `"meal"` に変更 (Wails bindings が JSON 経由なので、フィールド名変更のみでは足りない)。
      完了判定: `grep -n 'Slot\|slot' internal/domain/mealplan/entry.go` の残りが 0 件
- [x] 2.2 `internal/domain/mealplan/validate.go` を更新し、`ErrInvalidSlot` → `ErrInvalidMeal` に、
      `ValidateSlot` や引数名を meal 系に置換。エラーコード文字列 (`err.Code() == "mealplan: invalid slot"`
      など、frontend/error-messages.ts と揃える) も `invalid meal` に変更する。
      完了判定: `grep -n 'Slot\|slot' internal/domain/mealplan/validate.go` の残りが 0 件
- [x] 2.3 `internal/domain/mealplan/validate_test.go` を更新し、テスト名・アサーションを meal に追随
- [x] 2.4 `internal/domain/mealplan/copy.go` / `copy_test.go` を更新し、引数名・変数名・コメント中の
      `slot` / `Slot` を meal / Meal に置換
- [x] 2.5 `internal/domain/mealplan/batch.go` / `batch_test.go` を更新し、`slot` / `Slot` を meal /
      Meal に置換 (元々 slot 参照なし、no-op で完了)
- [x] 2.6 `internal/domain/mealplan/week.go` / `week_test.go` を更新し、`slot` / `Slot` を置換
      (元々 slot 参照なし、no-op で完了)
- [x] 2.7 `internal/domain/mealplan/` パッケージ全体で `grep -rn 'Slot\|slot' internal/domain/mealplan/`
      が 0 件になることを確認する。完了判定: exit code 0 (残 0 件で正常)

## 3. リポジトリ層 (`internal/storage/sqlite/`) の追随

- [x] 3.1 `internal/storage/sqlite/mealplan_repo.go` を更新し、SQL の `slot` カラム参照を `meal` に、
      Go 変数名・引数名の `slot` を meal に置換。バインド変数、`SELECT ... FROM meal_plans WHERE date = ?
      AND slot = ?` などの WHERE 節も対象。完了判定: `grep -n 'slot\|Slot' internal/storage/sqlite/mealplan_repo.go`
      が 0 件
- [x] 3.2 `internal/storage/sqlite/mealplan_repo_test.go` を更新し、SQL 直書きテストの `slot` 参照と
      Go テストコードの `Slot` 型使用を meal 系に置換

## 4. サービス層 (`internal/service/`) と Wails bindings

- [x] 4.1 `internal/service/mealplan.go` を更新し、公開メソッド (`AssignRecipe` / `ReplaceEntry` /
      `RemoveEntry` / `ClearSlot` (→ `ClearMeal`) / `ReorderSlot` (→ `ReorderMeal`) / `SplitBatch` /
      `MergeBatch` / `CopyLeftover` / `ListSlotBatches` (→ `ListMealBatches`)) の `slot string` 引数を
      `meal string` に、内部変数・エラー変換を meal 系に置換。**メソッド名の `Slot` を含むもの
      (ClearSlot / ReorderSlot / ListSlotBatches) は名前自体をリネームする**。
      完了判定: `grep -n 'slot\|Slot' internal/service/mealplan.go` が 0 件
- [x] 4.2 `internal/service/mealplan_test.go` の t.Run 名・アサーション・変数を meal に追随
- [x] 4.2b `internal/config/defaults.go` / `defaults_test.go` の `DefaultVisibleSlots` →
      `DefaultVisibleMeals`、`ItemsPerSlotHint` → `ItemsPerMealHint` を rename (proposal Impact 明記外
      だったが、フロントに bindings 経由で公開される値のため必須。task 8.5 の grep 0 件ゲート要件から
      追加)
- [x] 4.2c `internal/storage/sqlite/sqlite_test.go` の goose expected version を 4 → 5 に更新
      (00005 migration 追加に伴う調整)

## 5. フロントエンド TS の追随

- [x] 5.1 `frontend/src/components/mealplan/slot-visibility-checkboxes.tsx` を
      `meal-visibility-checkboxes.tsx` にリネームし、コンポーネント名 `SlotVisibilityCheckboxes` →
      `MealVisibilityCheckboxes`、内部変数 `slot` → `meal` に置換。参照する i18n key も 6 章の rename に
      合わせる。完了判定: 新ファイルが存在し、旧ファイルは無く、`grep -n 'Slot\|slot' <新ファイル>` が 0 件
- [x] 5.2 `frontend/src/components/mealplan/slot-menu.tsx` を `meal-menu.tsx` にリネームし、コンポーネント名
      `SlotMenu` → `MealMenu`、内部変数を meal に置換
- [x] 5.3 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、内部変数 (`slots`, `slot`,
      `slotLabelKey` 等)、mutation の引数キー、i18n key 参照、コメント中の `slot` を meal に置換。
      import 元を 5.1 / 5.2 の新ファイル名に更新
- [x] 5.4 `frontend/src/components/mealplan/meal-plan-cell.tsx` の `slot` 参照 (props / 変数 / コメント)
      を meal に置換
- [x] 5.5 `frontend/src/components/mealplan/mealplan-config.ts` の `SLOT_RENDER_ORDER` (→
      `MEAL_RENDER_ORDER`)、コメントを meal に置換
- [x] 5.6 `frontend/src/components/mealplan/copy-planner.ts` の `slot` 引数を meal に置換
- [x] 5.7 `frontend/src/components/mealplan/error-messages.ts` の `"mealplan: invalid slot"` →
      `"mealplan: invalid meal"`、i18n key 参照 `pages.meal_plan.errors.invalid_slot` →
      `pages.meal_plan.errors.invalid_meal`、コメントを更新
- [x] 5.8 `frontend/src/components/mealplan/batch-merge-dialog.tsx` のコメント (「同じ slot の他バッチ」)
      を meal に置換
- [x] 5.9 `frontend/src/routes/meal-plan.tsx` の import 元 (`SlotVisibilityCheckboxes`) を新名に更新
- [x] 5.10 `frontend/e2e/bindings-shim/mealplanservice.ts` を更新し、関数シグネチャ (`slot: string` →
      `meal: string`)、内部の HTTP call 引数を meal に置換。メソッド名の `ClearSlot` /
      `ReorderSlot` / `ListSlotBatches` も 4.1 の Go 側リネームに合わせる
- [x] 5.11 `frontend/e2e/mealplan-assign.spec.ts` の残 slot 参照が無いか確認する
      (Step 1 で見出しは書き換え済み)。完了判定: `grep -n 'slot\|Slot' frontend/e2e/mealplan-assign.spec.ts`
      が 0 件
- [x] 5.12 `frontend/e2e/mealplan-batch.spec.ts` のコメントにあった `ListSlotBatches` を
      `ListMealBatches` に追随 (task 一覧に無かったが grep 追随の一環で反映)

## 6. i18n key rename と en.json value 更新

- [x] 6.1 `frontend/src/locales/en.json` の key を更新: `pages.meal_plan.slot` → `pages.meal_plan.meal`、
      `slot_visibility` → `meal_visibility`、`slot_menu` (親 key) → `meal_menu`、
      `cell_actions.slot_menu` → `cell_actions.meal_menu`、`errors.invalid_slot` → `errors.invalid_meal`。
      同時に value 中の "Slot" を "Meal" に (`"Slot menu"` → `"Meal menu"`, `"Clear this slot"` →
      `"Clear this meal"`, `"Slot visibility"` → `"Meal visibility"`, `"Invalid slot"` → `"Invalid meal"`,
      `"Target slots already have dishes"` → `"Target meals already have dishes"`, `"{{count}} slot(s)
      already have dishes."` → `"{{count}} meal(s) already have dishes."`,
      `"No other batches in this slot to merge into"` → `"No other batches in this meal to merge into"`,
      `"Reorder request does not match current slot"` →
      `"Reorder request does not match current meal"`, `"Copy this slot for {{days}} days"` →
      `"Copy this meal for {{days}} days"`)。完了判定: `grep -n 'slot\|Slot' frontend/src/locales/en.json`
      が 0 件
- [x] 6.2 `frontend/src/locales/ja.json` の key を 6.1 と同じルールでリネーム。value は「時間帯」の
      まま維持 (Step 1 で確定)。完了判定: `grep -n 'slot' frontend/src/locales/ja.json` が 0 件
      (key も value も含めて完全 0)
- [x] 6.3 `frontend/src/` 配下の全 TS/TSX で `t("pages.meal_plan.slot*")` のような古い key 参照を
      検索し、新 key に置換する。完了判定: `grep -rn 't("pages.meal_plan.slot' frontend/src/` が 0 件
- [x] 6.4 `frontend/e2e/` 配下でも 6.3 と同じ検査 (テストコードが `ja.pages.meal_plan.slot*` を
      import している場合の追随)

## 7. spec / documentation の追随

- [x] 7.1 `openspec/specs/mealplan/spec.md` の残 "slot" を編集する (delta で MODIFIED していない
      11 Requirement 分)。散文の日本語「slot」は「時間帯」に、バッククォート囲みや `Err*` などの
      コード引用は `meal` / `ErrInvalidMeal` 等に置換。完了判定: `grep -n 'slot\|Slot'
      openspec/specs/mealplan/spec.md` が 0 件
- [x] 7.2 `docs/product.md` の残 "slot" (バッククォート囲みの `(date, slot)` 含む) を meal に置換。
      完了判定: `grep -n 'slot' docs/product.md` が 0 件
- [x] 7.3 `docs/design.md` §4 の MealPlan / meal_plans テーブル記述、および他章の "slot" 参照を meal に
      置換。完了判定: `grep -n 'slot\|Slot' docs/design.md` が 0 件
- [x] 7.4 `docs/scenarios/mealplan.feature` の残 "slot" (Step 1 で散文は「時間帯」に置換済み、残りは
      バッククォート囲みのコード引用 or 技術文脈) を meal に置換。完了判定:
      `grep -n 'slot' docs/scenarios/mealplan.feature` が 0 件 (Step 1 で既に 0 件、この change で
      再確認のみ)
- [x] 7.5 `openspec/specs/mealplan/spec.md` の Requirement 「入力バリデーション」内の
      `ErrInvalidSlot` を `ErrInvalidMeal` に (delta の対象外だが 7.1 の一部として実施)

## 8. lint / テスト / 最終ゲート

- [x] 8.1 `task lint` を実行し、Go / Frontend 両方でエラー 0 を確認する。完了判定: exit code 0
- [x] 8.2 `go test ./internal/... ./` を実行し、既存テストが全 pass することを確認する。完了判定:
      exit code 0
- [x] 8.3 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev` を実行し、
      TypeScript 型チェックと dev build が通ることを確認する。完了判定: exit code 0
- [x] 8.4 `task test:e2e` を実行し、E2E テストが引き続き pass することを確認する。完了判定: exit code 0
- [x] 8.5 リポジトリ全体で `grep -rin slot .` を実行し、以下**以外**の残り件数が 0 件になっていることを
      確認する:
      - `openspec/changes/archive/**/*` (archive 済み change の履歴)
      - `openspec/changes/rename-slot-to-meal/**/*` (この change 自身の planning artifacts —
        『slot を meal に rename した』という事実を文中で言及するため slot が残る)
      - `docs/wireframes/*.drawio.svg` (画像再生成コストのため触らない、Step 1 で決定)
      - `frontend/bindings/` (`wails3 generate bindings` 生成物、4.3 で再生成済みなら 0 のはず)
      - `frontend/node_modules/` / `frontend/package-lock.json` (依存パッケージ `@radix-ui/react-slot`
        は meal-plan の slot と無関係、shadcn/ui プリミティブ由来)
      - `build/android/` (Wails Android scaffolding、`AlertDialog slot` 等の Java コメント。MVP 対象外)
      - `internal/storage/sqlite/migrations/00003_meal_plans.sql` /
        `00004_meal_plans_batch.sql` / `00005_rename_slot_to_meal.sql` (歴史的マイグレーションの
        当時のスキーマ / rename 対象を指す文字列。手直しすると再現性が壊れる)
      - `.git/` (履歴)
      完了判定: 上記除外パスを差し引いた残り件数が 0 件
