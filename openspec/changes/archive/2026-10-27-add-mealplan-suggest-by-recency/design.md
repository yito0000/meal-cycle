## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/suggestion/spec.md](specs/suggestion/spec.md) と [specs/settings/spec.md](specs/settings/spec.md) を参照。

- `add-mealplan-suggest-by-stock` で、`SuggestionService.List` と `suggestion.Result`、ダイアログの `SuggestionSection` ができている
- 作った印は `meal_plan_cooked (batch_id, recipe_id, cooked_at)` に持ち、`meal_plans` の同じ組の全行に効く
- `config.Config` には `RecencyThresholdDays` (既定 10) が既にあり、どこからも読まれていない
- S09 の値は `settings.Values` に足し、`internal/service/settings.go` の `settingKeys` に key を登録すると保存と読み込みの経路に乗る

## Goals / Non-Goals

**Goals:**

- 最終調理日を、作った印の意味 (調理した事実) にそろえて求める

**Non-Goals:**

- 「久しぶりに」と「使い切り」の両方に当たるレシピに、両方に当たることを示すバッジを付けること。両方の欄に出るだけにする

## Decisions

### D1. 最終調理日の定義

**選択肢:**

- (a) 作った印の付いた行の日付のうち最も新しい日 ← 採用
  — Pros: [product.md §5](../../../docs/product.md) の定義どおり。作り置きなら最後に食べた日になり、「最後に食卓に出た日」からの日数として読める
  / Cons: 作り置きの最後の日が未来なら、経過日数が負になる
- (b) `meal_plan_cooked.cooked_at` (印を付けた日時)
  — Pros: 調理した時刻に近い / Cons: 前日にまとめて印を付けた場合や、あとから印を付けた場合に実際の日とずれる
- (c) 印の有無にかかわらず、今日より前の行の最も新しい日
  — Cons: 予定どおりに作らなかった料理も作ったことになる

**採用: (a)**。経過日数が負になるレシピは閾値を満たさないので、候補に出ない。

### D2. 履歴を読むクエリ

`internal/storage/sqlite/suggestion_repo.go` に `SuggestionRepo.ListRecipeHistory(ctx, today string) ([]suggestion.RecipeHistory, error)` を置く。
`RecipeHistory` は `RecipeID`、`LastCookedDate` (空なら一度も作っていない)、`HasUpcomingPlan` を持つ。

```sql
SELECT r.id,
       (SELECT MAX(m.date) FROM meal_plans m
          JOIN meal_plan_cooked c ON c.batch_id = m.batch_id AND c.recipe_id = m.recipe_id
         WHERE m.recipe_id = r.id) AS last_cooked,
       EXISTS (SELECT 1 FROM meal_plans m
                WHERE m.recipe_id = r.id AND m.date >= ?
                  AND NOT EXISTS (SELECT 1 FROM meal_plan_cooked c
                                   WHERE c.batch_id = m.batch_id AND c.recipe_id = m.recipe_id)) AS upcoming
FROM recipes r
WHERE r.archived_at IS NULL
```

`today` はサービスが `s.now()` のローカル日付で渡す。在庫の期限の判定と同じ「今日」を使う。

### D3. 予定のあるレシピを除く理由

今日以降に予定のあるレシピを候補に出すと、すでに来週入れたレシピを今週にも入れる二重の割り当てを誘う。
予定の側が作った印付き (前日に作り置きを作った場合など) なら、それは最終調理日の側で数えられるので、除く条件は「印の無い予定」に限る。

### D4. 並び順と件数

経過日数の降順、同じならレシピ名の昇順とし、`config.SuggestionLimit` 件に切る。
根拠は「N 日ぶり」とし、週や月への言い換えはしない。経過日数の大小がそのまま並び順なので、同じ単位で比べられるほうが読みやすい。

### D5. S09 の欄とワイヤー

献立の節の「作り置きを広げる日数の候補」の下に、数値の入力欄を 1 つ足す。
既存の 🔴 / 🟡 の日数と同じ部品 (整数の入力欄と、欄の下のエラー) を使う。
検証は `settings.ValidateRecencyThreshold` (1〜60) を足し、`Values.Validate` の列に加える。
上限の 60 は、週単位の献立で 2 か月より前の履歴を「久しぶり」の基準に使う場面が見当たらないためで、実装の上限として `validate.go` に置く。

S09 のワイヤーは更新せず、献立の節に欄が 1 つ増えたことをこの節に残す。

## Risks / Trade-offs

- [Risk] 作った印を付けない運用の利用者には、候補が 1 件も出ない
  → 欄は「該当なし」を出して残す。印を付けると候補に出ることは product.md の S01 の説明に書く
- [Risk] `add-mealplan-suggest-week-preset` も settings の同じ要件を MODIFIED するので、先に archive したほうの内容を後から取り込む必要がある
  → week-preset の proposal は、この proposal の archive のあとに delta を取り直す前提で書く

## Migration Plan

- スキーマは変えない。保存済みの設定に新しい key が無い DB は、既定値の 10 日で動く
