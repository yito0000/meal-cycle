## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/mealplan/spec.md](specs/mealplan/spec.md) を参照。

`meal_plans` の主キーは `(date, meal, position)` である。
行を消したあとの詰め直しは、次の 1 文で行っている (`removeRowTx` と `SplitBatch`)。

```sql
UPDATE meal_plans SET position = position - 1 WHERE date = ? AND meal = ? AND position > ?
```

SQLite は 1 行を書き換えるたびに主キーの重複を確かめる。
行を処理する順は position 順とは限らず、このクエリでは rowid 順 (挿入した順) になる。
`Reorder` は行を消さずに position だけを書き換えるので、並べ替えたセルでは rowid 順と position 順が食い違う。
A、B、C を B、C、A に並べ替えて B を消すと、rowid の小さい A (position 2) が先に 1 へ動き、まだ position 1 にいる C と重なる。

レビューでは、リポジトリ層の一時的なテストでこの失敗を再現した。

## Goals / Non-Goals

**Goals:**

- 詰め直しが、行を処理する順に左右されないようにする
- サービスのメソッド、引数、スキーマを変えない

**Non-Goals:**

- 並び順を主キーから外す (`add-mealplan-entry-id` で検討する)

## Decisions

### D1. 詰め直しの方式

**選択肢:**

- (a) 後ろの行をいったん大きな値へ退避し、そこから目的の値へ戻す 2 文にする ← 採用
  — Pros: どちらの文も、書き換え前の値の範囲と書き換え後の値の範囲が重ならないので、処理する順に関係なく重複しない。
  `Reorder` がすでに同じ退避 (+1000) を使っている
  / Cons: 1 セルに 1000 品以上あると退避先が重なる (現実には起きない)
- (b) Go 側で後ろの行を position 昇順に読み、1 行ずつ UPDATE する
  — Pros: 退避幅の仮定が要らない
  / Cons: 行数ぶんのクエリになる。コードが長くなる
- (c) `UPDATE … ORDER BY position` で処理順を指定する
  — Pros: 1 文で済む
  / Cons: `modernc.org/sqlite` は `UPDATE … ORDER BY / LIMIT` を有効にしていない (`Reorder` のコメントにある制約と同じ)
- (d) 行に ID を足し、並び順を主キーから外す
  — Pros: 重複の検査そのものが無くなる
  / Cons: スキーマと全操作の引数が変わる。止血としては大きすぎる

**採用: (a)**。
具体的には次の 2 文を `closePositionGap(ctx, q, date, meal, removed)` にまとめる。

```sql
UPDATE meal_plans SET position = position + :offset WHERE date = ? AND meal = ? AND position > :removed;
UPDATE meal_plans SET position = position - :offset - 1 WHERE date = ? AND meal = ? AND position >= :offset;
```

`:offset` は `Reorder` と共有する定数 `positionShiftOffset = 1000` にする。
`Reorder` は退避した行を自分で振り直すので、退避中の行が残った状態でこの関数が呼ばれることはない。

### D2. `SplitBatch` の詰め直し

`SplitBatch` は、起点以外の行を 1 行ずつ消し、そのつどセルを詰めている。
同じセルに同じ作り置きの行が 2 つ以上あると、1 つめを詰めた時点で 2 つめの position がずれ、覚えておいた position では消せなくなる。

**選択肢:**

- (a) 消す行を先にすべて消し、影響を受けたセルごとに 0 から振り直す ← 採用
  — Pros: 消す順と詰める順が絡まない
  / Cons: 振り直しの関数がもう 1 つ要る
- (b) 消す行を position の大きい順に処理する
  — Pros: 既存の流れを保てる
  / Cons: 「大きい順なら安全」という前提がコメント頼みになり、今回と同じ種類の見落としを招く

**採用: (a)**。
`renumberMeal(ctx, q, date, meal)` を足す。
セルの行を position 昇順に読み、全行を退避してから、読んだ順に 0 から振り直す。
`closePositionGap` は 1 行を消した直後専用、`renumberMeal` は複数行を消したあと用と使い分ける。

### D3. `isForeignKeyErr` の絞り込み

現在は `constraint failed` を含むエラーをすべて外部キー違反として扱い、`ErrUnknownRecipe` に読み替える。
今回の失敗 (主キーの重複) が `Assign` や `Replace` の経路で起きていたら、「存在しないレシピ」と表示されていた。
`FOREIGN KEY constraint failed` を含む場合だけに絞る。他案 (エラーコードの数値で判定する) は、ドライバの内部型に依存するので見送る。

## Risks / Trade-offs

- [1 セルに 1000 品以上あると退避先が重なる] → 表示上限が最大 4 品のセルで起きる状況ではない。`positionShiftOffset` のコメントに前提を書く
- [`isForeignKeyErr` を絞ると、外部キー違反以外の制約違反が汎用エラーになる] → 意図した変更である。原因の違うエラーを同じ文言で見せない
- [E2E は 1 シナリオだけで、移動と解体はリポジトリ層のテストで確かめる] → 失敗の原因は SQL にあり、画面の経路による差は無い
