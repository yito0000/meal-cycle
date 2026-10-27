## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/suggestion/spec.md](specs/suggestion/spec.md) を参照。

- S01 のレシピ選択ダイアログ (`recipe-picker-dialog.tsx`) は、`RecipeService.List` の結果を名前で絞り込んで並べるだけの部品で、
  空セルへの追加、品の追加、品の入れ替えの 3 つの経路から開く
- 在庫の期限の状態は `fridge.Evaluate` が今日と S09 の 🔴 / 🟡 の日数から決める
- 食材名の表記をそろえる関数は `add-fridge-staples` で `internal/domain/ingredient.NormalizeName` に移っている
- ワイヤー [S01a_recipe_picker_dialog.drawio.svg](../../../docs/wireframes/S01a_recipe_picker_dialog.drawio.svg) は、
  検索欄の下に軸ごとの欄 (見出し、件数、候補の行と「選択」) を縦に並べ、下に全レシピの一覧への導線を置いている

## Goals / Non-Goals

**Goals:**

- 3 つの軸を順に足せるよう、候補を返すサービスと画面の欄を軸に依らない形で作る
- 候補を出しても、ダイアログで名前を検索して選ぶ今の操作は変えない

**Non-Goals:**

- 候補の算出結果の保存。ダイアログを開くたびに算出する

## Decisions

### D1. capability とサービスの置き場所

**選択肢:**

- (a) `mealplan` capability の要件にし、`MealPlanService` にメソッドを足す
  — Pros: 画面は S01 のダイアログなので、利用者から見た置き場所と一致する
  / Cons: `MealPlanService` はリポジトリ 1 つを受け取る形で、在庫とレシピを読むにはコンストラクタを変えることになる
- (b) `suggestion` capability を新設し、`SuggestionService` を足す ← 採用
  — Pros: 3 つの軸と週プリセットの要件を 1 か所に積める。在庫、レシピ、献立の履歴を読む依存がこのサービスに閉じる
  / Cons: サービスの登録 (`main.go`、`cmd/e2e-server`、shim) が 1 つ増える

**採用: (b)**。[product.md §5](../../../docs/product.md) は献立候補を独立した章として扱っており、後続の 2 本もここに足す。

### D2. 返り値の形

`SuggestionService.List(ctx) (suggestion.Result, error)` とし、`Result` は軸ごとの候補の配列を持つ。この proposal では `ByStock []Candidate` だけを持つ。
`Candidate` は `RecipeID` / `RecipeName` と、軸ごとの根拠の値 (この軸では `Stocks []StockReason`) を持つ。
`StockReason` は在庫の `Name` と `DaysLeft` を持つ。

根拠の文言は Go で組み立てず、フロントが i18n で組む (design.md §7 の「Go は日本語を持たない」)。

**選択肢:**

- (a) 軸ごとにメソッドを分ける (`ListByStock`、`ListByRecency`)
  — Pros: 軸ごとに取得を分けられる / Cons: ダイアログを開くたびに呼び出しが軸の数だけ増える
- (b) 1 つの `List` で全軸を返す ← 採用
  — Pros: 在庫とレシピを 1 回読めば全軸を算出できる / Cons: 軸が増えるたびに `Result` の型が変わる

**採用: (b)**。候補はダイアログを開いたときに一度に見せるので、呼び出しを 1 回にそろえるほうが表示がずれない。

### D3. 「使い切り」の算出

`internal/domain/suggestion/by_stock.go` の `ByStock(recipes, stocks, today, red, yellow, limit)` が純関数で算出する。

1. 在庫のうち `fridge.Evaluate` が `urgent` か `soon` を返すものだけを残す (期限切れと期限なしは除く)
2. 同じ食材名 (表記をそろえる) の在庫が複数あれば、最も近い期限の 1 件にまとめる
3. 削除済みでないレシピごとに、材料の食材名と一致する在庫を集める。単位と量は見ない
4. 一致が 1 件以上のレシピを候補にし、一致した在庫の数の降順、最も近い期限の昇順、レシピ名の昇順に並べて `limit` 件に切る

単位を見ないのは、候補の目的が「この在庫を使う料理を思い出させる」ことにあり、使い切れるかの判定ではないためである。
S07 の在庫判定と違い、量る単位の材料 (醤油 大さじ 2) も、在庫の醤油が期限間近なら候補の根拠に数える。

`limit` は `config.SuggestionLimit` (既定 5) から読む。ワイヤーの欄の高さで 5 件までが一度に見える。

### D4. ダイアログの組み方

候補の欄は `SuggestionSection` として軸に依らない部品にし、見出し、件数、候補の行 (レシピ名と根拠の 1 行)、該当なしの表示を持つ。
候補の行を押すと `onSelect` を呼び、下の一覧の行を押したときと同じ経路でそのレシピを選ぶ。
検索欄に文字があるときは候補の欄を隠す。名前で探している間は候補が一覧を押し下げるだけになるためである。

ダイアログの幅は `max-w-md` から `max-w-lg` に広げる。根拠の 1 行がレシピ名の下に収まるようにする。
候補の取得は `useQuery(["suggestions"])` で、ダイアログが開いている間だけ有効にする。
在庫や献立の変更から invalidate はしない。ダイアログを開くたびに `staleTime: 0` で取り直す。

## Risks / Trade-offs

- [Risk] 材料に「玉ねぎ」、在庫に「たまねぎ」のように漢字とかなが違うと候補に出ない
  → S07 と同じ制約で、別名の辞書を入れるまでは変えない
- [Risk] レシピの数が多いと、ダイアログを開くたびの算出が重くなる
  → ペルソナのレパートリーは 20〜50 品で、材料と在庫の突き合わせはメモリ上の map で済む
- [Trade-off] 在庫の量を見ないので、玉ねぎ 1/4 個の残りでも、玉ねぎを 3 個使うレシピが候補に出る

## Migration Plan

- スキーマは変えない
