## 1. 既定値と設定

- [x] 1.1 `internal/config/defaults.go` を更新する (design.md D1 / D2)。完了判定: `go test ./internal/config/` が pass する
    - `WeeklyFillPattern string` を `WeeklyFillDays []int` (既定 `[1, 2, 3, 4, 5]`) と `WeeklyFillMeal string` (既定 `"dinner"`) に置き換える
    - `WeeklyPresetRecipeCount [2]int` を `WeeklyPresetMaxRunDays int` (既定 3) に置き換える
    - `defaults_test.go` の検査を新しい値に合わせる
- [x] 1.2 `internal/domain/settings/settings.go` の `Values` に `WeeklyFillDays` / `WeeklyFillMeal` / `WeeklyPresetMaxRunDays` を足し、`FromConfig` / `ApplyTo` / `Normalize` (曜日を昇順にそろえる) に通す。
  完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/domain/settings/validate.go` に `ValidateWeeklyFill(days []int, meal string)` (曜日は 1〜7 で 1 つ以上、重複なし、時間帯は breakfast / lunch / dinner) と
  `ValidateWeeklyPresetMaxRunDays(days int)` (1〜7) を足し、`errors.go` に対応するエラーを足す。完了判定: `go build ./internal/...` が通る
- [x] 1.4 `internal/domain/settings/settings_test.go` に table-driven のケースを追加する。完了判定: `go test ./internal/domain/settings/` が pass する
    - 曜日が空だとエラーになる
    - 曜日に 0 や 8 があるとエラーになる
    - 同じ曜日が 2 回あるとエラーになる
    - 時間帯が未知の値だとエラーになる
    - 続ける日数の上限が 0 と 8 だとエラーになり、1 と 7 は通る
- [x] 1.5 `internal/service/settings.go` の `settingKeys` に `weeklyFill` (曜日と時間帯をまとめた JSON) と `weeklyPresetMaxRunDays` を足す。
  完了判定: `go test ./internal/service/ -run TestSettingsService` が pass する

## 2. まとめて割り当てる操作

- [x] 2.1 `internal/domain/mealplan/apply.go` を新設し、`PlanItem` (`Date` / `Meal` / `RecipeID` / `Group`) と `ValidatePlanItems(items)` (空、不正な日付と時間帯、空のレシピ ID を拒否) を定義する (design.md D5)。
  完了判定: `go build ./internal/...` が通る
- [x] 2.2 `internal/domain/mealplan/apply_test.go` を新設し、`ValidatePlanItems` を table-driven でテストする。完了判定: `go test ./internal/domain/mealplan/` が pass する
    - 空の配列はエラーになる
    - 実在しない日付はエラーになる
    - 未知の時間帯はエラーになる
    - レシピ ID が空の行はエラーになる
- [x] 2.3 `internal/storage/sqlite/mealplan_repo.go` に `ApplyPlan(ctx, items []mealplan.PlanItem) (mealplan.SpreadResult, error)` を追加する。
  1 トランザクションで `Group` ごとに `batch_id` を発行し、各セルの末尾の position で行を足す。レシピが無ければ `ErrUnknownRecipe`、削除済みなら `ErrArchivedRecipe` を返してロールバックする。
  完了判定: `go build ./internal/...` が通る
- [x] 2.4 `internal/storage/sqlite/mealplan_repo_test.go` に `TestMealPlanRepo_ApplyPlan` を追加する。完了判定: `go test ./internal/storage/sqlite/ -run TestMealPlanRepo_ApplyPlan` が pass する
    - 同じ Group の行は同じ batch_id になり、別の Group は別の batch_id になる
    - 既存の品のあるセルでは末尾に足される
    - 削除済みのレシピを含むとどの行も足されない
    - 返り値の AddedEntryIDs を UndoSpread に渡すと足した行が消える
- [x] 2.5 `internal/service/mealplan.go` に `ApplyPlan` を追加し、`ValidatePlanItems` を通してから repo を呼ぶ。`internal/service/mealplan_test.go` に BDD 風のテストを足す。
  完了判定: `go test ./internal/service/ -run TestMealPlanService_ApplyPlan` が pass する
    - should add rows grouped as leftover batches when items are valid
    - should add nothing when any recipe is archived
    - should return error when items are empty

## 3. 週の候補の算出

- [x] 3.1 `internal/domain/suggestion/week.go` を新設し、`WeekSlot` / `WeekPlanItem` / `WeekPlan` / `WeekPlanResult` を定義する (design.md D4)。完了判定: `go build ./internal/...` が通る
- [x] 3.2 `internal/domain/suggestion/week.go` に `SplitSlots(slots []WeekSlot, maxRunDays int) [][]WeekSlot` を実装する (design.md D2)。完了判定: `go build ./internal/...` が通る
- [x] 3.3 `internal/domain/suggestion/week.go` に `WeekPlans(groups [][]WeekSlot, byStock, byRecency []Candidate, rest []recipe.Recipe) []WeekPlan` を実装する (design.md D3)。完了判定: `go build ./internal/...` が通る
- [x] 3.4 `internal/domain/suggestion/week_test.go` を新設し、table-driven でテストする。完了判定: `go test ./internal/domain/suggestion/` が pass する
    - `SplitSlots`：5 コマ・上限 3 は 3, 2 になる
    - `SplitSlots`：4 コマ・上限 3 は 2, 2 になる
    - `SplitSlots`：7 コマ・上限 3 は 3, 2, 2 になる
    - `SplitSlots`：上限 1 はコマの数だけの組になる
    - `SplitSlots`：0 コマは空になる
    - `WeekPlans`：使い切り優先は byStock の順に当てる
    - `WeekPlans`：久しぶり優先は byRecency の順に当てる
    - `WeekPlans`：バランスは byStock と byRecency を交互に当てる
    - `WeekPlans`：1 案の中で同じレシピを 2 回使わない
    - `WeekPlans`：レシピが足りない組は案に含めない
    - `WeekPlans`：前の案と同じ案は除かれる
- [x] 3.5 `internal/service/suggestion.go` に `WeekPlans(ctx, weekStart string)` を追加する。対象のコマは `MealPlanRepo` の期間の行と S09 の曜日と時間帯から求め、候補は `ByStock` と `ByRecency` を limit なしで呼ぶ。
  完了判定: `go build .` が通る
- [x] 3.6 `internal/service/suggestion_test.go` に BDD 風のテストを追加する。完了判定: `go test ./internal/service/ -run TestSuggestionService_WeekPlans` が pass する
    - should return empty slots when every target cell has entries
    - should skip cells that already have entries when building slots
    - should follow saved fill days and meal when settings change
    - should return ErrInvalidDate when weekStart is not a date
    - should return the same plans when called twice with unchanged data
- [x] 3.7 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/e2e/bindings-shim/suggestionservice.ts` に `WeekPlans`、`mealplanservice.ts` に `ApplyPlan` を足す。
  完了判定: `cd frontend && npm run type-check` が通る

## 4. S01 の画面と S09 の欄

- [x] 4.1 `frontend/src/components/mealplan/week-plan-dialog.tsx` を新設する (design.md D6)
    - 開いたときに `SuggestionService.WeekPlans(periodStart)` を取り、コマが無ければ `pages.meal_plan.week_plan.no_slots` を出す
    - 案ごとのカードに種類の名前、組の行 (レシピ名、`formatDishDates` で組んだ日付、根拠)、「この案を適用」を置く。カードに `data-testid="week-plan"` を付ける
    - 「この案を適用」で `ApplyPlan` を呼び、成功したら `onApplied(result)` を呼ぶ
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/src/routes/meal-plan.tsx` の下部の CTA の横に「この週の候補を出す」を置き、`WeekPlanDialog` を開く。
  適用後は献立を invalidate し、件数と「元に戻す」(`UndoSpread`) のトーストを出す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/components/settings/settings-form.tsx` の献立の節に、曜日の 7 つのチェックボックス、時間帯の選択、続ける日数の上限の整数入力欄を足す。
  `frontend/src/components/settings/error-messages.ts` に新しいエラーを足す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.4 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.meal_plan.week_plan.*` と S09 の新しい欄の文言を足す
    - `cta`：「✨ この週の候補を出す」、`title`：「この週の候補」
    - `kind.stock`：「使い切り優先」、`kind.recency`：「久しぶり優先」、`kind.balanced`：「バランス」
    - `apply`：「この案を適用」、`applied`：「{{count}} 行を割り当てました」、`no_slots`：「この週に埋める空きはありません」
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 5. E2E

- [x] 5.1 `frontend/e2e/suggestion-week.spec.ts` を新設する。describe 名は Requirement 名、test 名は Scenario 名と一字一句そろえる。完了判定: `task test:e2e` で以下が pass する
    - describe「週の候補の対象」
      - 空いた 5 コマは 3 日と 2 日の 2 組になる
      - 品のあるセルは対象にしない
      - 空いたコマが無いとその旨を示す
    - describe「週の候補の案」
      - 使い切り優先の案は期限の近い在庫を使うレシピから当てる
      - 同じ内容の案はまとめる
- [x] 5.3 `frontend/e2e/mealplan-week-plan.spec.ts` を新設する。シナリオ検査はファイル名の capability で照合するので、mealplan の Requirement はこちらに置く。
  完了判定: `task test:e2e` で以下が pass する
    - describe「週の候補の呼び出し」
      - 週の候補ダイアログを開く
    - describe「週の候補の適用」
      - 案を適用すると作り置きとして割り当たる
      - 適用を元に戻すと割り当てた行が消える
- [x] 5.2 `frontend/e2e/settings.spec.ts` にテストを足す。完了判定: `task test:e2e` で以下が pass する
    - describe「設定値の検証」：週の候補で埋める曜日を空にすると保存されない
    - describe「保存した値の各画面への反映」：1 品を続ける日数の上限を変えると週の候補の品数が変わる

サービス層とドメイン層で確かめるシナリオ (「1 つの案で同じレシピは使わない」「押し直しても同じ案が出る」「削除済みのレシピを含む適用は何も割り当てない」) は E2E に含めない。

## 6. ドキュメントと検証

- [x] 6.1 `docs/wireframes/S01b_week_plan_dialog.drawio.svg` を新設し、案のカード 3 枚 (種類、組の行、根拠、「この案を適用」) を描く。可視 SVG と `content=` 内の drawio XML の両方に入れる。
  `docs/wireframes/README.md` に行を足す。完了判定: ブラウザで SVG を開いて目視で確認する
- [x] 6.2 `docs/product.md` を更新する。完了判定: 差分を目視で確認する
    - §1 の可変値の表で、曜日パターンを S09 で変更可にし、「週プリセット案の 1 週間内品数」を「週の候補で 1 品を続ける日数の上限」に置き換える
    - §4 S01 の「MVP 後拡張」の CTA の文言と対象の週を改める
    - §4 S09 に献立の節の欄を足す
    - §5「提示先」の週単位の項を、空きコマ、組の分け方、3 種類の案、適用と元に戻すの形に改め、副菜を添える記述を Non-goal として外す
- [x] 6.3 `docs/design.md` §8 の `add-mealplan-suggest-week-preset` の行を改める。完了判定: 差分を目視で確認する
- [x] 6.4 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 6.5 `cd frontend && npm run test` と `npm run type-check` と `npm run build:dev` を実行する。完了判定: 3 つとも成功する
- [x] 6.6 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 6.7 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 6.8 `openspec validate add-mealplan-suggest-week-preset --strict` を実行する。完了判定: エラー 0 件
