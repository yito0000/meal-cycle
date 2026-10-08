## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/mealplan/spec.md](specs/mealplan/spec.md) と
[specs/shopping/spec.md](specs/shopping/spec.md) を参照。

現状は次のとおりである。

- `meal_plans` は `(date, meal, position)` を主キーに、`recipe_id` と `batch_id` を持つ。作り置きのテーブルは無く、
  同じ `batch_id` を持つ行の集まりが 1 つの作り置きを表す
- 入れ替え (`Replace`) は `batch_id` を変えずに `recipe_id` だけを差し替える。
  そのため 1 つの `batch_id` に別のレシピが混ざることがあり、S07 は `(batch_id, recipe_id)` の組を 1 回分の調理として数えている
  (`shopping_repo.go` の `loadPlannedGroups`)
- 作り置きを編集する操作は、`batch_id` を次のように扱う
  - 作り置きを広げる / 連続 N 日コピー (`CopyLeftover`、`CopyToOtherMeal`) の作り置きモード：元と同じ `batch_id` の行を足す。別調理モードは行ごとに新しい `batch_id`
  - 1 行移動 (`MoveEntry`)：`batch_id` と `recipe_id` を保ったまま、行を消して足し直す
  - 作り置きから外す (`SplitBatch`)：起点の行だけを残して他の行を消し、残った行に新しい `batch_id` を振る
  - 他の作り置きにまとめる (`MergeBatch`)：行の `batch_id` をまとめ先のものに付け替える。同じレシピの作り置きにしかまとめられない
- `recipes` は論理削除なので、`meal_plans.recipe_id` の `ON DELETE CASCADE` は実際には働かない

## Goals / Non-Goals

**Goals:**

- 「作った」印を、S07 が 1 回分の調理として数える単位と同じ単位で持つ
- 作り置きを編集する既存の操作に入れる変更を、できるだけ少なくする
- 後続の `add-mealplan-cook-consume` (在庫を減らす) と `add-mealplan-suggest-by-recency` (最終使用日) が、この印をそのまま使えるようにする

**Non-Goals:**

- 印を付けた日時を画面に出すこと
- 作り置きをテーブルとして持つように、`meal_plans` の構造を作り直すこと

## Decisions

### D1. 印を付ける単位

**選択肢:**

- (a) 行 (1 日の 1 品) ごとに付ける
  — Pros: データの形が単純
  / Cons: 3 日分の作り置きに 3 回印を付けることになる。S07 の「1 回分の調理」とも単位が合わない
- (b) `batch_id` ごとに付ける
  — Pros: 作り置きの単位に合う
  / Cons: 入れ替えで同じ `batch_id` に別のレシピが混ざると、作っていないレシピにまで印が付く
- (c) `(batch_id, recipe_id)` の組ごとに付ける ← 採用
  — Pros: S07 が数える単位とそろう。入れ替えた品が別の組になるので、作っていないレシピに印が付かない
  / Cons: 印を引くときに 2 つの列で突き合わせる

**採用: (c)**。
後続の `add-mealplan-cook-consume` で「作った」に合わせて在庫を減らすときも、減らす材料は 1 回分の調理の単位で決まる。
その単位と印の単位をそろえておけば、印 1 つにつき材料 1 回分を減らす、と対応づけられる。

### D2. 印の保存先

**選択肢:**

- (a) `meal_plans` に `cooked_at` の列を足し、同じ組のすべての行に同じ値を書く
  — Pros: 新しいテーブルが要らない。行が消えれば印も消える
  / Cons: 同じ組の行の値がそろっていることを、行を足す操作 (広げる、連続 N 日コピー、移動、まとめる) のすべてで保つ必要がある
- (b) `meal_plan_cooked (batch_id, recipe_id, cooked_at)` のテーブルを作り、印の付いた組だけを 1 行で持つ ← 採用
  — Pros: 同じ組の行を足しても動かしても、組が同じなら印はそのまま効く。広げる、連続 N 日コピー、移動には手を入れなくてよい
  / Cons: 組の行がすべて消えても印の行は残るので、掃除が要る。`SplitBatch` は `batch_id` を振り直すので、印を移す処理が要る

**採用: (b)**。
(a) だと行を足す操作 5 つすべてに印を写す処理が入るが、(b) なら `SplitBatch` と `MergeBatch` の 2 つで済む。

テーブルは次のとおりにする (`00008_meal_plan_cooked.sql`)。

```sql
CREATE TABLE meal_plan_cooked (
    batch_id  TEXT NOT NULL CHECK (length(batch_id) > 0),
    recipe_id TEXT NOT NULL,
    cooked_at TEXT NOT NULL,
    PRIMARY KEY (batch_id, recipe_id)
);
```

`meal_plans` への外部キーは張らない。`meal_plans` の主キーは `(date, meal, position)` で、`(batch_id, recipe_id)` は一意でないからである。
`cooked_at` は RFC3339 の時刻で、画面には出さない。後続の proposal で「いつ作ったか」が要るときのために残す。

### D3. 印の付け外しの API

`MealPlanService.SetCooked(ctx, date, meal string, position int, cooked bool) error` を足す。

他の品の操作 (`ReplaceEntry`、`RemoveEntry`、`SplitBatch` など) と同じく、品を `(date, meal, position)` で指す。
フロントは品の `batchId` と `recipeId` を持っているので組で指すこともできるが、
その形だと存在しない組を指定されたときの検証が別の経路になる。品の住所で指せば、既存の `validateMealAddress` と `ErrEntryNotFound` をそのまま使える。

repo の `SetCooked` は 1 トランザクションで次を行う。

1. 指定の行の `batch_id` と `recipe_id` を引く。無ければ `ErrEntryNotFound`
2. `cooked` が true なら `INSERT ... ON CONFLICT DO NOTHING`、false なら `DELETE` する

付けてある印をもう一度付けても、外してある印をもう一度外しても、エラーにしない。
S01 のメニューは現在の状態を見て項目を出し分けるが、二重に押された場合にエラーを出す理由が無いためである。

`ListPeriod` は、各行に `EXISTS (SELECT 1 FROM meal_plan_cooked c WHERE c.batch_id = m.batch_id AND c.recipe_id = m.recipe_id)` を
`cooked` として足して返す。

### D4. 作り置きの編集での印の扱い

D2 のテーブルにより、組が変わらない操作 (広げる / 連続 N 日コピーの作り置きモード、移動) は何もしなくても印を引き継ぐ。
別調理モードは新しい `batch_id` を振るので、何もしなくても印は付かない。
手を入れるのは次の 3 つである。

- `SplitBatch`：残った行に新しい `batch_id` を振るとき、元の組に印があれば、新しい組にも印の行を足す
- `MergeBatch`：行を付け替える前の組に印があり、まとめ先の組に印が無ければ、まとめ先の組に印の行を足す
- 掃除：行を消したり組を変えたりする操作の最後に、`meal_plans` に対応する行の無くなった印を消す

掃除は `pruneCookedMarks(ctx, tx)` として 1 つの SQL にまとめる。

```sql
DELETE FROM meal_plan_cooked
WHERE NOT EXISTS (
    SELECT 1 FROM meal_plans m
    WHERE m.batch_id = meal_plan_cooked.batch_id AND m.recipe_id = meal_plan_cooked.recipe_id
)
```

これを `Replace`、`Remove`、`ClearMeal`、`CopyLeftover`、`CopyToOtherMeal`、`SplitBatch`、`MergeBatch` のトランザクションの最後で呼ぶ。
正しさのために掃除が要るのは `Replace` である。
印の付いた品を別のレシピに入れ替えてから元のレシピに戻すと、掃除が無ければ古い印の行が残っていて、印が復活してしまう (spec の Scenario「入れ替えた品は印の無い状態になる」)。
他の操作での掃除は、使われない印の行をためないためである。
掃除の SQL は印のテーブル全体を走査するが、印の行は作り置きの数 (週に数件) しか増えないので問題にならない。

### D5. S07 の計上

`shopping.PlannedBatch` と `shopping.Dish` に `Cooked bool` を足し、`loadPlannedGroups` で D3 と同じ `EXISTS` を読む。

**選択肢 (印の付いた組を外す場所):**

- (a) `loadPlannedGroups` の SQL で除く
  — Pros: 材料を読み込む量が減る
  / Cons: 「この期間の献立」欄からも消えてしまい、spec の「印を添えて一覧に残る」を満たせない
- (b) `shopping.Aggregate` の走査で `Cooked` の組を飛ばす ← 採用
  — Pros: 献立欄 (`Dishes`) には残したまま、材料だけを外せる。domain の table-driven test で固定できる
  / Cons: 使わない材料も読み込む (期間の献立は多くても数十件なので問題にならない)

**採用: (b)**。

期間の献立がすべて印付きのときのメッセージは、フロントで `data.dishes.every((d) => d.cooked)` を見て出す。
この条件は `data.items.length === 0` (材料が無い) より先に判定する。すべて印付きのときは材料が 0 件になるので、
順を逆にすると「献立のレシピに材料が登録されていない」と誤った理由を出してしまうためである。

### D6. S01 の表示とメニュー

品の `⋯` メニューの先頭に「作った」(印の付いた品では「作った印を外す」) を置き、区切り線で他の項目と分ける。
調理した日に S01 を開いて押す、いちばん頻度の高い操作になるからである。
`revise-mealplan-reorder-menu` と合わせたメニューの並びは次のとおりにする。

```
作った / 作った印を外す
────────
入れ替え / この品を削除 / この品のレシピ詳細を見る / 上へ移動 / 下へ移動
────────
作り置きを広げる / 作り置きから外す / 他の作り置きにまとめる
```

印の付いた品は、レシピ名の前に lucide の `Check` アイコンを出す。
アイコンには `aria-label` (`pages.meal_plan.cell.cooked`) を付け、品行に `data-cooked="true"` を付ける (E2E の判定に使う)。
作り置きの 🔗 と並ぶときは、🔗、チェックの順にする。

印を付け外しした直後は、他の操作と同じく `ListPeriod` の query を捨てて取り直す。
楽観的更新は入れない。印は同じ組の他の日の行にも広がるので、フロントで組の全行を探して書き換えるより、取り直すほうが間違いが無い。

## Risks / Trade-offs

- [Risk] 利用者が印を付け忘れると、S07 は今までどおり作り終えた作り置きの材料も計上する → 印が無いときの挙動は今と同じなので、悪化はしない。
  S07 の献立欄に「作った」の印が出るので、付け忘れに気づく手がかりにはなる
- [Risk] 「他 N 件」に畳まれた品には、メニューから印を付けられない → 並び替え (R5) と同じ制約として受け入れる。期間を 1 週に戻せば見える
- [Risk] 掃除の呼び忘れで、印が復活する経路が残る → `Replace` の復活は spec の Scenario と repo のテストで固定する。
  他の操作は、組が消えたあとに同じ組が現れる経路が無い (新しい `batch_id` は乱数で振る) ので、呼び忘れても挙動は変わらない
- [Risk] `batch_id` が同じで `recipe_id` だけが違う組が、入れ替えを経て生まれる → D1 のとおり組ごとに印を持つので、入れ替えた品には印が付かない

## Migration Plan

- `00008_meal_plan_cooked.sql` の up でテーブルを作る。既存の行は変えないので、すべての献立は印の無い状態から始まる
- down でテーブルを drop する。印は失われるが、`meal_plans` には影響しない
- Windows 側と WSL 側の DB は別なので、どちらも起動時に goose が 00008 を当てる
