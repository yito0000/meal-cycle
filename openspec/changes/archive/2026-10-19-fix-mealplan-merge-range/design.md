## Context

動機は proposal.md の Why を参照。

S09 の値は、各サービスがコンストラクタで受け取る `configSource` の `current()` から毎回読む (add-settings-screen design.md D3)。
`MealPlanService` は `LeftoverCopyPresets` をこの経路で読み、`CopyLeftover` の日数と `SpreadEntry` の対象日の上限に使っている。
一方、`MealPlanRepo` は `NewMealPlanRepo` で `config.Defaults()` を抱え、`mergeDayRange` でその最大値を読む。
`ListMergeCandidates` と `MergeBatch` の ±N 日はこの値で決まるので、S09 の値が届かない。
サービス層のテストのフェイクのリポジトリも `mergeDayRange` を 4 で決め打ちしており、この食い違いを拾えなかった。

`ListMergeCandidates` は、`WHERE recipe_id = ? AND ABS(julianday(date) - julianday(?)) <= ?` で絞ってから
`COUNT(*) OVER (PARTITION BY batch_id)` と `ROW_NUMBER() OVER (...)` を掛けている。
ウィンドウ関数は絞った後の行にしか掛からないので、範囲の外の行は件数からも代表行からも落ちる。

## Goals / Non-Goals

**Goals:**

- まとめる操作の探索範囲を、`MealPlanService` が `configSource` から読んだ値で決める
- 候補の代表行と件数を、範囲で絞る前の作り置きの全体から決める

**Non-Goals:**

- `MergeBatch` の可否の条件 (同じレシピの行が範囲内に 1 行でもあること) は変えない
- フロントは変えない。候補の値が正しくなれば、ダイアログの表示はそのまま直る

## Decisions

### D1. 探索範囲の日数をリポジトリへ渡す方法

**選択肢:**

- (a) サービスが日数を求め、`ListMergeCandidates` と `MergeBatch` の引数で渡す ← 採用
  — Pros: リポジトリが設定を知らずに済む。`maxSpreadOffsetDays` と同じ場所で同じ値から計算できる。
  フェイクのリポジトリも引数を使うだけになり、決め打ちが消える / Cons: interface の 2 メソッドの引数が 1 つ増える
- (b) `NewMealPlanRepo` に `configSource` 相当を渡し、リポジトリが毎回読む
  — Pros: 呼び出し側の引数が変わらない / Cons: リポジトリ層が設定の存在を知る。
  `configSource` は `package service` の非公開 interface なので、`sqlite` 側に同じ interface を切り直す必要がある
- (c) `NewMealPlanRepo(db, cfg config.Config)` で起動時の値を渡す
  — Pros: 変更が小さい / Cons: 起動時の値で固まり、S09 で保存しても再起動まで効かない。spec の「再起動せずに反映」に反する

**採用: (a)**。理由: 設定を読むのはサービスの責務というレイヤの分け方 (CLAUDE.md「可変値は 1 箇所に集約」) に合う。
`MealPlanService` に `mergeDayRange()` (候補の最大値) を足し、`maxSpreadOffsetDays()` は `mergeDayRange() - 1` で求める。
同じ最大値を 2 回ループで求めている今の重複も、これで 1 か所になる。

### D2. 候補の代表行と件数の求め方

**選択肢:**

- (a) 範囲内に同じレシピの行がある作り置きを先に選び、代表行と件数は、選んだ作り置きの同じレシピの全行から求める ← 採用
  — Pros: spec の「その作り置きの中でいちばん早い行」をそのまま満たす。範囲は判定だけに使うので、
  `MergeBatch` の可否の条件と同じ規則になる / Cons: クエリが 2 段になる
- (b) 範囲の条件を外して全期間の作り置きを数え、最後に「代表行が範囲内か」で絞る
  — Pros: クエリが 1 段で済む / Cons: 代表行が範囲の外にあり、残りの行だけが範囲内の作り置きが候補から落ちる。
  `MergeBatch` は受け付けるのに候補に出ない、という食い違いが新しく生まれる

**採用: (a)**。クエリは次の形にする。

```sql
WITH eligible AS (
  SELECT DISTINCT batch_id FROM meal_plans
  WHERE recipe_id = ? AND batch_id <> ?
    AND ABS(julianday(date) - julianday(?)) <= ?
)
SELECT ... FROM (
  SELECT m.batch_id, m.date, m.meal, m.recipe_id, m.position,
         COUNT(*) OVER (PARTITION BY m.batch_id) AS cnt,
         ROW_NUMBER() OVER (PARTITION BY m.batch_id ORDER BY m.date, <朝昼夕の順>, m.position) AS rn
  FROM meal_plans m JOIN eligible e ON e.batch_id = m.batch_id
  WHERE m.recipe_id = ?
) g ... WHERE g.rn = 1
```

### D3. 件数として数える行

**選択肢:**

- (a) その作り置きに属する同じレシピの行の数 ← 採用 — Pros: 今の実装と同じ対象を数え、範囲の制限だけを外す。
  候補の行はレシピ名と一緒に出るので、「肉じゃが (4 品)」が肉じゃがの行数を指すと読める / Cons: 時間帯全体のコピーで
  複数のレシピが 1 つの作り置きを共有しているとき、作り置きの全行数とは一致しない
- (b) その作り置きに属する全レシピの行の数 — Pros: 「作り置きをやめる」の確認が示す行数と同じ数え方になる /
  Cons: 候補の行に出ているレシピ以外の品まで数え、レシピ名と件数の組が読みにくくなる

**採用: (a)**。spec に「同じレシピの行の数」と明記し、あとから数え方がぶれないようにする。

### D4. テストの置き場所

探索範囲の日数がサービスから渡ることは、サービス層のテストで、`staticConfig` の `LeftoverCopyPresets` を変えて確かめる。
代表行と件数の集計は SQL にあるので、リポジトリ層のテストで確かめる。
settings の Scenario はサービス層で確かめ、mealplan の Scenario は E2E でも確かめる (tasks.md の 4 章)。

## Risks / Trade-offs

- [Risk] interface の引数が増え、フェイクのリポジトリと実装の引数の意味がずれる
  → 両方とも `dayRange` という同じ名前で受け、サービス層のテストに「候補の最大値を変えると範囲が変わる」ケースを足して押さえる
- [Risk] 候補の最大値を小さくすると、すでにある作り置きが候補に出にくくなる
  → add-settings-screen design.md の Risks に記載済みで、spec も「保存の前に作った献立は書き換えない」と定めている。今回は挙動を変えない
- [Risk] `eligible` を JOIN する形で、行の多い DB では集計が重くなる
  → 対象は同じレシピの行だけで、1 台 1 ユーザーの献立の規模では数十行に収まる

## Migration Plan

スキーマとデータは変わらない。既存の DB はそのまま使える。
