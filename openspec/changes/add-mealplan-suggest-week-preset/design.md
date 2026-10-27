## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/suggestion/spec.md](specs/suggestion/spec.md)、
[specs/mealplan/spec.md](specs/mealplan/spec.md)、[specs/settings/spec.md](specs/settings/spec.md) を参照。

- `SuggestionService.List` が「使い切り」と「久しぶりに」の候補を返す (by-stock / by-recency)
- `config.Config` には `WeeklyFillPattern: "mon-fri-dinner"` (自由テキスト)、`WeeklyPresetRecipeCount: [2, 3]`、`ItemsPerMealHint: [1, 3]` があり、どれも読まれていない
- 献立を複数行まとめて足す操作は `SpreadEntry` (1 品を広げる) と `CopyLeftover` (時間帯をコピー) だけで、どちらも起点の行が要る。
  `SpreadEntry` の取り消しは、足した行の ID と上書きで消えた行を持つ `SpreadResult` を `UndoSpread` に渡して行う
- S01 の期間の開始日は常に月曜で、`＜` / `＞` は期間の長さ単位で動く

## Goals / Non-Goals

**Goals:**

- 同じ在庫と履歴なら同じ案を出す決定的な組み立てにし、テストで固定できるようにする
- 献立をまとめて割り当てる操作を、テンプレートの適用でも使える形で作る

**Non-Goals:**

- 案の良し悪しの評価 (栄養や味の偏り)。2 つの軸の候補の並びに従うだけにする

## Decisions

### D1. 対象のコマ

表示中の期間の開始日から 7 日間のうち、S09 の曜日 (`WeeklyFillDays`、月曜 = 1 〜 日曜 = 7) に当たる日の、S09 の時間帯 (`WeeklyFillMeal`) のセルを見る。
そのうち品が 1 つも無いセルを、日付の順に対象のコマとする。
品が 1 つでもあるセルは、利用者がすでに決めたものとして埋めない。

`WeeklyFillPattern` の自由テキストは、S09 で直せる形にするため、曜日の配列と時間帯の 2 つの値に置き換える。
時間帯を 1 つに限るのは、ペルソナが夕食だけを組むことと、3 食を埋める案は時間帯ごとに向くレシピが違い、候補の並びだけでは組めないためである。

### D2. コマを作り置きに分ける規則

対象のコマが n 個、1 品を続ける日数の上限が m のとき、作り置きの数を k = ⌈n / m⌉ とし、コマを日付の順に k 個の組に分ける。
組の大きさは前から順に ⌈n / k⌉ と ⌊n / k⌋ の 2 種類に均す (n = 5、m = 3 なら 3 と 2)。

**選択肢:**

- (a) 上限まで詰めて余りを最後に置く (5 → 3, 2 / 4 → 3, 1)
  — Cons: 4 コマで 1 日だけの組ができ、作り置きの形から外れる
- (b) 組の数を最小にしたうえで大きさを均す ← 採用
  — Pros: 4 → 2, 2、7 → 3, 2, 2 のように、どの組も上限以下で大きさがそろう
- (c) 品数の目標 (`WeeklyPresetRecipeCount` の 2〜3 品) から組の数を決める
  — Cons: 空きコマの数が週ごとに変わると、目標の品数と上限の日数を同時には満たせない

**採用: (b)**。品数の目標は上限の日数から決まるので、`WeeklyPresetRecipeCount` は `WeeklyPresetMaxRunDays` に置き換える。
上限を 1 にすれば毎日別のレシピの案になり、作り置きをしない利用者にも合う。

組の中のコマが連続した日でなくてもかまわない (水曜だけ既に埋まっていれば、月・火・木で 1 組になりうる)。
作り置きの行は日付が飛んでもよい ([mealplan spec](../../../openspec/specs/mealplan/spec.md) の作り置きの要件)。

### D3. 案の組み立て

3 種類の案は、組に当てるレシピの順位表だけが違い、組への当て方は同じとする。

- 使い切り優先：「使い切り」の候補の順、続けて「久しぶりに」の候補の順、続けて残りのレシピ
- 久しぶり優先：「久しぶりに」の候補の順、続けて「使い切り」の候補の順、続けて残りのレシピ
- バランス：「使い切り」と「久しぶりに」の候補を 1 件ずつ交互に、続けて残りのレシピ

候補の数は `SuggestionLimit` で切らず、条件を満たすレシピをすべて使う。
「残りのレシピ」は、削除済みと今日以降に印の無い予定のあるレシピを除き、最終調理日の古い順 (一度も作っていないレシピは最後)、レシピ名の昇順とする。
順位表の上から、まだ案で使っていないレシピを組の順に当てる。
レシピが組の数より少なければ、当てられなかった組は案に含めない。
3 つの案のうち、組とレシピの対応が前の案と同じものは除く。

乱数を使わないのは、押し直しても同じ案が出ることで、案の根拠 (順位表) が利用者から見て一貫するためである。

### D4. 返り値の形

`SuggestionService.WeekPlans(ctx, weekStart string) (suggestion.WeekPlanResult, error)` とする。
`WeekPlanResult` は `Slots []WeekSlot` (対象のコマ) と `Plans []WeekPlan` を持つ。
`WeekPlan` は `Kind` (`stock` / `recency` / `balanced`) と `Items []WeekPlanItem` を持ち、
`WeekPlanItem` は `RecipeID`、`RecipeName`、`Dates`、`Meal`、根拠 (`Stocks` と `DaysSince`) を持つ。
`weekStart` が実在する日付でなければ `mealplan.ErrInvalidDate` を返す。

### D5. まとめて割り当てる操作

**選択肢:**

- (a) `MealPlanService.ApplyPlan(ctx, items []mealplan.PlanItem) (mealplan.SpreadResult, error)` を足す ← 採用
  — Pros: 取り消しは既存の `UndoSpread` がそのまま使える。テンプレートの適用でも同じ操作を使える
  / Cons: 名前に Spread を含む型を別の用途に使う
- (b) フロントから組ごとに `AssignRecipe` と `SpreadEntry` を順に呼ぶ
  — Cons: 途中で失敗すると一部だけ割り当たる。取り消しも組の数だけ要る

**採用: (a)**。`PlanItem` は `Date`、`Meal`、`RecipeID`、`Group` を持ち、同じ `Group` の行は 1 つの `batch_id` を共有する。
リポジトリは 1 トランザクションで、組ごとに新しい `batch_id` を発行し、各セルの末尾に行を足す。
レシピが存在しない、削除済み、日付や時間帯が不正な行が 1 つでもあれば、何も足さない。
返り値の `SpreadResult` は `AddedEntryIDs` だけを埋め、`Removed` は空にする。上書きはテンプレートの proposal で足す。

### D6. ダイアログの組み方

S01 の下部に CTA を置き、押すと `WeekPlans` を呼んでダイアログを開く。
ダイアログは案ごとにカードを並べ、カードに案の種類、作り置きの行 (レシピ名、日付の範囲、根拠)、「この案を適用」を置く。
日付の表記は S07 の献立の欄と同じ (`9/24〜26`、`9/24, 9/26`) にする。
適用が成功したらダイアログを閉じ、S01 の献立を invalidate し、件数と「元に戻す」をトーストで出す。

ワイヤーは `docs/wireframes/S01b_week_plan_dialog.drawio.svg` として新しく作る (product.md §5 が「proposal 着手時に追加」としている)。

## Risks / Trade-offs

- [Risk] レシピが少ないと、どの案も同じになり 1 案しか出ない
  → 同じ内容の案はまとめると決めてあるので、1 案だけでも正しい表示になる
- [Risk] 案を出してから適用するまでに S01 の別の操作でセルが埋まると、そのセルに 2 品目が入る
  → ダイアログは S01 の上に開くモーダルなので、通常は別の操作を挟めない。挟んだ場合も品の追加になるだけで、既存の品は消えない
- [Trade-off] `WeeklyPresetRecipeCount` と `ItemsPerMealHint` の意味を product.md から外す。品数は日数の上限から決まり、副菜は Non-goal になる

## Migration Plan

- スキーマは変えない。`WeeklyFillPattern` と `WeeklyPresetRecipeCount` は S09 で保存したことが無い値なので、既定値の置き換えだけで済む
