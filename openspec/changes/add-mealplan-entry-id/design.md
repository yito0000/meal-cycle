## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/mealplan/spec.md](specs/mealplan/spec.md) を参照。

この change は、先行する 5 つの change が入ったあとの状態を前提にする (proposal.md の「前提と順序」)。
その時点の状態は次のとおりである。

- `meal_plans` の主キーは `(date, meal, position)` で、列は `recipe_id`、`created_at`、`batch_id` を持つ
- `MealPlanService` の 1 品に対するメソッドは、`(date, meal string, position int)` で行を指す。
  `ReplaceEntry`、`RemoveEntry`、`SetCooked`、`SplitBatch`、`MergeBatch`、`ListMergeCandidates`、`MoveEntry`、`SpreadEntry` が該当する
- `ReorderMeal(date, meal, orderedRecipeIDs)` は、レシピ ID の並びを受け取る。
  同じレシピが 2 つあるときは「古い並び順のものから順に割り当てる」ので、同じレシピの 2 品の順は入れ替えられない
- `CopyLeftover(date, meal, days, position, mode, shareBatch)` の `position` は、負なら時間帯全体、0 以上ならその 1 品を表す。
  画面は負の値しか渡さない (1 品を広げる経路は `SpreadEntry` に移っている)
- 行を消したあとの詰め直しは、`closePositionGap` / `renumberMeal` が退避の方式で行う
- `RemovedEntry` は削除した行の内容を持ち、`RestoreEntries` が元の並び順へ戻す
- スキーマの更新がある起動では、更新の前に DB の控えが `backups/` に作られる
- フロントは mutation のあと `["mealplan"]` のクエリを捨てて取り直す。楽観的更新はしていない

## Goals / Non-Goals

**Goals:**

- 1 品に対する操作が、並び順の変化に関係なく、指定した品に当たる
- 行を指す方法を、Go 側もフロント側も ID の 1 通りにする

**Non-Goals:**

- 楽観的更新の導入
- `batch_id` の持ち方の見直し

## Decisions

### D1. ID の形式

**選択肢:**

- (a) UUID v7 の文字列 ← 採用
  — Pros: レシピ、在庫、作り置きの ID と同じ形式になる。Go 側で発行でき、挿入の前に ID が決まる
  / Cons: 36 文字で、整数より大きい
- (b) SQLite の rowid (`INTEGER PRIMARY KEY AUTOINCREMENT`)
  — Pros: 小さい。発行を DB に任せられる
  / Cons: ほかの表と形式がそろわない。`RestoreEntries` で同じ ID を持つ行を戻すとき、AUTOINCREMENT の採番と明示の値が混ざる

**採用: (a)**。
`mealplan.NewEntryID()` を `NewBatchID()` の隣に置く。

既存の行への ID は、migration の中で `lower(hex(randomblob(16)))` で振る。
SQL だけでは UUID v7 を作れないためで、`00004_meal_plans_batch.sql` が `batch_id` を振ったときと同じ方法である。
ID は中身を読まない不透明な文字列として扱うので、形式が混ざっても動作に影響しない。

### D2. (日付, 時間帯, 並び順) の一意性を残すか

**選択肢:**

- (a) 一意インデックスとして残す ← 採用
  — Pros: 「並び順は 0 からの連番で欠番なし」という spec の不変条件 (「割当データの永続化」) を、DB が引き続き守る。
  詰め直しの抜けがあれば、黙って並びが崩れるのではなく、エラーとして表に出る
  / Cons: 詰め直しに退避の方式が要るままになる
- (b) 一意性を外し、ただのインデックスにする
  — Pros: 詰め直しが `position = position - 1` の 1 文で済む
  / Cons: 並び順の重複や欠番が入っても気づけない。表示の順は ID を第 2 キーにして安定させる必要がある

**採用: (a)**。
退避の方式は `fix-mealplan-renumber-after-reorder` ですでに入っており、追加の費用は無い。
不変条件を DB に持たせたままにするほうが、今後の操作の追加に強い。

### D3. メソッドの引数

行を指す引数を、すべて `entryID string` にする。

| メソッド | 変更後の引数 |
| --- | --- |
| `ReplaceEntry` | `(entryID, recipeID)` |
| `RemoveEntry` | `(entryID)` |
| `SetCooked` | `(entryID, cooked)` |
| `SplitBatch` | `(entryID)` |
| `MergeBatch` | `(entryID, targetBatchID)` |
| `ListMergeCandidates` | `(entryID)` |
| `MoveEntry` | `(entryID, targetDate, targetMeal)` |
| `SpreadEntry` | `(entryID, targets, mode, shareBatch)` |
| `ReorderMeal` | `(date, meal, orderedEntryIDs)` |
| `CopyLeftover` | `(date, meal, days, mode, shareBatch)` |
| `AssignRecipe` / `ClearMeal` / `ListPeriod` / `RestoreEntries` | 変更なし (セルや期間を指す操作) |

`ListMergeCandidates` は、これまで受け取っていた `recipeID` を行から読む。
呼び出し側が行とレシピの組を取り違える余地が無くなる。

空の `entryID` と、存在しない `entryID` は、どちらも `ErrEntryNotFound` にする。
他案 (空の ID 用に `ErrInvalidEntryID` を足す) は、画面から見た結果 (品が見つからない) が同じなので見送る。

`ReorderMeal` の検証は、「渡された ID の集合が、そのセルの行の ID の集合と一致する」に変わる。
重複する ID、足りない ID、ほかのセルの ID は `ErrReorderMismatch` にする。

### D4. `CopyLeftover` の `position` をどうするか

**選択肢:**

- (a) `position` を `entryID` (空なら時間帯全体) に置き換える
  — Pros: 「1 品だけを連続 N 日コピーする」経路が残る
  / Cons: 画面から使われない経路を、引数の形を変えてまで保つことになる
- (b) `position` を外し、時間帯全体のコピー専用にする ← 採用
  — Pros: メソッドの役割が 1 つになる。1 品を広げる経路は `SpreadEntry` だけと言い切れる
  / Cons: E2E のヘルパー `copyLeftover` が 1 品目だけをコピーする前提で使われている箇所を直す必要がある

**採用: (b)**。
E2E のヘルパーは、1 品だけのセルに対して使われている。時間帯全体のコピーに変えても結果は同じである。
2 品以上あるセルで 1 品だけを広げたい箇所があれば、`SpreadEntry` を呼ぶヘルパーに置き換える。

### D5. 元に戻した行の ID

`RemovedEntry` に `ID` を足し、`RestoreEntries` は同じ ID で行を入れる。
削除の前に開いていたメニューや、取り直す前の画面が持っている ID が、元に戻したあとも同じ品を指すようにするためである。
同じ ID の行がすでにある場合 (「元に戻す」の二重押し) は、S05 の `Restore` と同じく拒否し、`ErrInvalidRestore` を返す。

### D6. migration の形

SQLite は主キーの変更を `ALTER TABLE` でできないので、`00004_meal_plans_batch.sql` と同じく表を作り直す。

1. `meal_plans_new` を、`id TEXT PRIMARY KEY`、既存の列、`UNIQUE (date, meal, position)` で作る
2. 既存の行を、`lower(hex(randomblob(16)))` の ID を付けてコピーする
3. `meal_plans` を消し、`meal_plans_new` を改名する
4. `idx_meal_plans_date` と `idx_meal_plans_batch_id` を張り直す

`meal_plan_cooked` は `meal_plans` へ外部キーを張っていないので、作り直しの影響を受けない。
`recipes` への外部キー (`ON DELETE CASCADE`) は、新しい表でも同じ定義にする。

Down は、`id` を落として `(date, meal, position)` を主キーに戻す。

### D7. フロントの持ち方

- `DragSource` を `{ id, date, meal, position }` にする。`id` で操作し、`date` / `meal` / `position` は同じセルかどうかの判定と、ドロップ位置の計算に使う
- 品行の `key` を `entry.id` にする。並び替えで行が入れ替わっても、メニューの開閉の state が別の品へ移らない
- E2E のヘルパーは、テストの読みやすさのために (日付, 時間帯, 並び順) を受け取る形を保ち、内部で `ListPeriod` から ID を引く

## Risks / Trade-offs

- [テーブルを作り直す migration が途中で失敗する] → goose は migration 1 本を 1 トランザクションで当てるので、失敗すれば巻き戻る。
  加えて `add-shell-startup-safety` の控えが更新の前に取られる。失敗した場合はエラー画面に控えの場所が出る
- [メソッドの引数の変更が、サービス層とリポジトリ層のテスト (合わせて 2,500 行) に広く及ぶ] → 挙動は変えず、行の指し方だけを変える。
  テストの前提づくりに `Assign` の戻り値の ID を使う小さなヘルパーを先に足し、機械的に置き換える
- [バインディングの生成形式 (`.ts` / `.js`) で引数の型の出方が違う] → shim の型を狭く書く既存の方針に従い、`npm run type-check` を両方の生成形式で確かめる必要は無い
  (引数は `string` と `boolean` と配列だけで、nullable の差が出ない)
- [WSL と Windows で DB が別なので、migration は Windows 側でも 1 度当たる] → 実機確認のタスクで、Windows 側の既存の DB に対して起動し、献立が残っていることを確かめる

## Migration Plan

1. この change を入れた版を起動すると、`OpenAndMigrate` が更新を検出し、`backups/meal-cycle-v8-<日時>.db` を作る
2. migration 00009 が `meal_plans` を作り直し、既存の行に ID を振る
3. 失敗した場合は起動時のエラー画面が出る。控えを `meal-cycle.db` に上書きコピーし、前の版の `.exe` で起動すれば元の状態に戻る

前の版の `.exe` は `id` 列を知らないので、更新後の DB をそのまま前の版で開くことはできない。戻すときは必ず控えを使う。
