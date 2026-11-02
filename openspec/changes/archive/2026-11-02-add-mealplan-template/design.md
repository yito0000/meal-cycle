## Context

動機は proposal.md の Why を参照。

- 献立は `meal_plans` の 1 行 = 1 品で、`(date, meal)` 内の順を `position`、作り置きを `batch_id` で表す (design.md §4)
- 週の候補の適用は `MealPlanService.ApplyPlan(items []PlanItem)` で、`PlanItem{Date, Meal, RecipeID, Group}` の同じ `Group` を同じ作り置きにし、
  各セルの末尾に足す。取り消しは返り値の `SpreadResult` を `UndoSpread` に渡す
- `ApplyPlan` は削除済みのレシピを含むと全体を拒む (mealplan spec「週の候補の適用」)
- レシピの削除は論理削除で、`recipes` の行は残る
- 週の候補は「表示中の期間の最初の 1 週間」を対象にしている

## Goals / Non-Goals

**Goals:**

- 気に入った週の組み方を、作り置きのつながりごと別の週に再現できるようにする
- 適用は既存の `ApplyPlan` / `UndoSpread` の経路に載せ、元に戻すの振る舞いを週の候補とそろえる

**Non-Goals:**

- 名前の変更、中身の編集、ほかの長さ (proposal.md の Non-goals)

## Decisions

### D1. 保存の形: 曜日のずれと作り置きの組の番号

`meal_plan_template_items` に、週の最初の日からのずれ (`day_offset` 0〜6)、`meal`、`position`、`recipe_id`、`group_no` を持つ。

```sql
CREATE TABLE meal_plan_templates (
  id         TEXT PRIMARY KEY,   -- UUID v7
  name       TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE meal_plan_template_items (
  template_id TEXT    NOT NULL REFERENCES meal_plan_templates(id) ON DELETE CASCADE,
  day_offset  INTEGER NOT NULL CHECK (day_offset BETWEEN 0 AND 6),
  meal        TEXT    NOT NULL,
  position    INTEGER NOT NULL,
  recipe_id   TEXT    NOT NULL REFERENCES recipes(id),
  group_no    INTEGER NOT NULL,
  PRIMARY KEY (template_id, day_offset, meal, position)
);
```

**選択肢:**

- (a) 日付をそのまま保存し、適用時に週の差でずらす — Pros: 元の週が分かる / Cons: 「どの週から作ったか」は使わない情報で、ずらす計算が適用側に残る
- (b) 曜日のずれで保存する ← 採用 — Pros: 適用は「対象の週の最初の日 + ずれ」で済む / Cons: 元の週は残らない
- (c) JSON の 1 列に行を詰める — Pros: 表が 1 つ / Cons: `recipe_id` に外部キーを張れず、レシピの存在を SQL で確かめられない

**採用: (b)**。

`group_no` は、週の中の行の `batch_id` を、最初に出てくる順に 0, 1, 2… と振り直したもの。
週の外に続く作り置き (日曜から翌週の月曜まで) は、週の中の行だけが同じ組になる。
外部キーの `recipes(id)` は `ON DELETE` を付けない。レシピは論理削除なので行は消えない。

名前の重複は表記をそろえずに前後の空白を除いて比べ、サービスで拒む (常備品の名前と同じく UNIQUE は付けない。エラーの種類をサービスで選ぶため)。

### D2. 適用は `ApplyPlan` に変換して流す

`ApplyTemplate(ctx, templateID, weekStart)` は、テンプレートの行を `PlanItem{Date: weekStart + day_offset, Meal, RecipeID, Group: group_no}` に変換し、
削除済みのレシピの行を除いてから `repo.ApplyPlan` を呼ぶ。返り値は `TemplateApplyResult{Result SpreadResult, SkippedArchived int}`。

**選択肢:**

- (a) テンプレート専用の書き込みをリポジトリに足す — Pros: `position` まで再現できる / Cons: 末尾に足す、作り置きを作る、取り消しの記録、の 3 つを二重に持つ
- (b) `ApplyPlan` に流す ← 採用 — Pros: 末尾に足す規則と取り消しが週の候補と同じになる /
  Cons: `ApplyPlan` はセルの末尾に足すので、`position` は「同じセルに入れる順」としてだけ使う

**採用: (b)**。`PlanItem` を `(day_offset, meal, position)` の順に並べて渡せば、同じセル内の順はテンプレートのとおりになる。

削除済みのレシピを除くのは `ApplyTemplate` の中で行う。`ApplyPlan` の「削除済みを含めば全体を拒む」はそのまま残し、除いた後の行に念のため効かせる。
除いた結果が 0 行なら `ErrTemplateAllArchived` を返す。

### D3. 保存の対象と UI の置き場所

対象は週の候補と同じ「表示中の期間の最初の 1 週間」。2 週表示で後半の週を保存したいときは、前後ナビで動かしてから保存する。
時間帯表示 (夕食のみ / 3 食) にかかわらず、全時間帯の品を保存する (隠れた時間帯の品が黙って落ちないように)。

S01 の下部の CTA が「この期間の買い物リストを見る」「この週の候補を出す」に加えて増えるので、ボタンを並べずに「テンプレート」の 1 つのメニューにまとめる。

### D4. 作った印は保存しない

テンプレートは予定の組み方で、適用した行は作っていない予定として入る。

## Risks / Trade-offs

- [Risk] テンプレートを作ってから時間がたち、削除済みのレシピが増える → 一覧のダイアログで削除済みの品に印を付け、適用時に除いた数を通知する
- [Risk] 対象の週に既に品があると、セルに品が積み重なる → 週の候補の適用と同じ振る舞いで、「元に戻す」で戻せる。上書きは Non-goals
- [Risk] migration の番号が `add-mealplan-suggest-by-time` と衝突する → 実装した順に番号を振り、後の change の proposal.md と tasks.md を直す
