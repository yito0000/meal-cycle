## Why

「使い切り」と「久しぶりに」の候補は、セルを 1 つずつ埋めるときの助けになる。
だがペルソナの「1 週間の献立を勝手に考える、または案を提示してほしい」([product.md §1](../../../docs/product.md)) に応えるには、セルごとにダイアログを開く手間がまだ残る。
作り置きで回すペルソナは、月〜金の 5 コマを 2〜3 品で埋めている。
この形に合わせて、週の空いたコマを少数のレシピの作り置きで埋めた案を出せば、利用者は案を選んで微調整するだけで済む。

[product.md §5](../../../docs/product.md) は、この「週プリセット」を 3 つめの提示先として置いている。
`add-mealplan-suggest-by-stock` と `add-mealplan-suggest-by-recency` の候補を材料に、週の案を 2〜3 個組み立てる。

## What Changes

- S01 の下部、「この期間の買い物リストを見る」の横に「✨ この週の候補を出す」を置く
  - 対象は S01 で表示中の期間の最初の 1 週間 (開始日から 7 日間) とする
  - 対象のコマは、S09 で決める曜日と時間帯 (既定は月〜金の夕食) のうち、品が 1 つも無いセルとする
- 押すと「週の候補」ダイアログを開き、対象のコマを埋める案を最大 3 個並べる
  - 案は「使い切り優先」「久しぶり優先」「バランス」の 3 種類で、同じ内容になる案は 1 つにまとめる
  - 各案は、対象のコマを日付の順に 1 品あたり S09 の上限 (既定 3 日) までの作り置きに分け、各作り置きに 1 つのレシピを当てる
  - 1 つの案の中で同じレシピは 2 回使わない
  - 各作り置きの行に、レシピ名、割り当てる日付、候補の根拠 (期限の近い在庫、何日ぶり) を出す
  - 対象のコマが無いときは、案の代わりにその旨を出す
- 案の「この案を適用」で、その案の作り置きを S01 にまとめて割り当てる
  - 1 つの作り置きは同じ作り置きの行として割り当て、S01 の枠色で 1 組に見える
  - 既存の品は消さない。適用までに品が入ったセルには、品を追加する
  - 全行を 1 回で反映し、1 行でも反映できなければどの行も割り当てない
  - 反映後は割り当てた行の数と「元に戻す」を通知し、「元に戻す」で割り当てた行を消す
- S09 の献立の節に、週の候補で埋める曜日と時間帯、1 品を続ける日数の上限 (1〜7 日、既定 3 日) を足す

### Non-goals

- **副菜の候補を添えること**：1 時間帯に複数品を入れる案は、品の役割 (主菜、副菜) をレシピが持たないので作れない。1 コマ 1 品だけを埋める
- **案の保存や、案を直してから適用すること**：適用したあとに S01 の既存の操作で直す
- **乱数で毎回違う案を出すこと**：同じ在庫と履歴なら同じ案を出す。押し直しても変わらない
- **4 週表示の 2 週目以降を埋めること**：対象は表示中の最初の 1 週間だけにする

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `suggestion`：週の候補の算出と、週の候補ダイアログの要件を追加する
- `mealplan`：S01 の「この週の候補を出す」と、案のまとめての割り当てと元に戻す要件を追加する
- `settings`：S09 の献立の節に、週の候補で埋める曜日と時間帯、1 品を続ける日数の上限を足す

## Impact

**前提**

- `add-mealplan-suggest-by-stock` と `add-mealplan-suggest-by-recency` を先に入れる。settings の delta は by-recency の delta を取り込んだ形で書いてある

**新設ファイル**

- `internal/domain/suggestion/week.go`：`WeekSlot` / `WeekPlan` / `WeekPlanItem` と、案を組み立てる `WeekPlans`
- `internal/domain/suggestion/week_test.go`
- `internal/domain/mealplan/apply.go`：`PlanItem` (日付、時間帯、レシピ、作り置きの組の番号) と検証
- `internal/domain/mealplan/apply_test.go`
- `frontend/src/components/mealplan/week-plan-dialog.tsx`：週の候補ダイアログ
- `frontend/e2e/suggestion-week.spec.ts`

**更新ファイル**

- `internal/config/defaults.go` と `defaults_test.go`：`WeeklyFillPattern` (文字列) を `WeeklyFillDays []int` と `WeeklyFillMeal string` に置き換え、
  `WeeklyPresetRecipeCount` を `WeeklyPresetMaxRunDays int` (既定 3) に置き換える
- `internal/domain/suggestion/suggestion.go`：`WeekPlans` の返り値の型
- `internal/service/suggestion.go` と `suggestion_test.go`：`WeekPlans(ctx, weekStart string)`
- `internal/storage/sqlite/mealplan_repo.go` と `mealplan_repo_test.go`：作り置きの組ごとに行を足す `ApplyPlan`
- `internal/service/mealplan.go` と `mealplan_test.go`：`ApplyPlan(ctx, items []mealplan.PlanItem) (mealplan.SpreadResult, error)`
- `internal/domain/settings/settings.go` / `validate.go` / `errors.go` / `settings_test.go`
- `internal/service/settings.go`：key `weeklyFill` と `weeklyPresetMaxRunDays`
- `frontend/src/routes/meal-plan.tsx`：下部の CTA
- `frontend/src/components/settings/settings-form.tsx` と `frontend/src/components/settings/error-messages.ts`
- `frontend/src/locales/ja.json` と `frontend/src/locales/en.json`
- `frontend/e2e/bindings-shim/suggestionservice.ts` と `mealplanservice.ts`
- `frontend/e2e/settings.spec.ts`
- `docs/product.md`：§1 の可変値の表 (曜日パターンと品数の行)、§4 S01 の CTA、§4 S09、§5 の週単位の提示先
- `docs/design.md`：§8 の `add-mealplan-suggest-week-preset` の行 (「5 コマ分を貪欲生成」を空きコマと日数の上限の形に改める)
- `docs/wireframes/S01b_week_plan_dialog.drawio.svg` を新設し、`docs/wireframes/README.md` に行を足す

**生成物 (再生成のみ)**

- `frontend/bindings/`：`wails3 generate bindings -clean=true -ts -i`

**migration**

- なし

**後続 proposal への引き継ぎ**

- `add-mealplan-templates` は、テンプレートの適用に `MealPlanService.ApplyPlan` と `UndoSpread` を使う。上書きの指定はテンプレートの proposal で足す
- 副菜の候補を添えるには、レシピに品の役割を持たせる proposal が要る
