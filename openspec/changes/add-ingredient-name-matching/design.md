## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/recipe/spec.md](specs/recipe/spec.md) と
[specs/shopping/spec.md](specs/shopping/spec.md) を参照。

現状は次のとおりである。

- S06 の食材名の候補補完は `StockNameInput` (`frontend/src/components/fridge/stock-name-input.tsx`) が持つ。
  WAI-ARIA の combobox で、変換中の Enter は `isImeComposing` (`fridge/stock-form.ts`) で無視する
- 候補は `FridgeService.ListSuggestions` が返す。在庫の食材名とレシピの材料名 (削除済みレシピを含む) を名前ごとにまとめ、直近の単位を添えている。
  単位の候補 (`units`) も同じ呼び出しで返る
- フロントの候補の query は `useStockSuggestions` (`fridge/use-fridge.ts`)、key は `["fridge", "suggestions"]`。在庫の mutation のたびに捨てている
- S04 の材料行 (`RecipeIngredientRow`) は、名前 / 量 / 単位の 3 つのテキスト入力で、候補を持たない。
  add-recipe-edit の design.md D9 は「食材マスタが無いと候補が空で意味が無い」として候補補完を見送ったが、
  今は在庫とレシピの材料名から候補を作れる
- S07 の突き合わせは `shopping.Aggregate` (`internal/domain/shopping/aggregate.go`) が行う。
  キーは `strings.TrimSpace` した食材名と単位の組で、在庫の合計 (`stockByKey`、`stockByName`) も同じキーで引く。
  量る単位の判定は `slices.Contains(measuringUnits, k.unit)` の完全一致である
- `golang.org/x/text` は go.mod に indirect として入っている

## Goals / Non-Goals

**Goals:**

- S04 と S06 で、同じ候補補完の部品と同じ候補を使う
- S07 の突き合わせの規則を `shopping` のドメイン層の 1 か所で決め、table-driven test で固定する
- サービスのメソッドとバインディングの型を変えない

**Non-Goals:**

- S05 の食材名検索と並び順に、表記をそろえる規則を持ち込むこと
- 候補の並び順や件数の上限 (8 件) を変えること

## Decisions

### D1. 表記の揺れを吸収する時点

**選択肢:**

- (a) 保存の時点で表記をそろえて保存する
  — Pros: S07 の突き合わせは今の完全一致のままでよい
  / Cons: 利用者が書いた「タマネギ」が「たまねぎ」に書き換わって表示される。既存のデータの移行も要る
- (b) S07 で比べる時点でだけ表記をそろえる ← 採用
  — Pros: 保存済みのデータと表示が変わらない。そろえる規則をあとで変えても、移行が要らない
  / Cons: 比べるたびに変換の計算が入る (材料と在庫は数十〜数百件で、問題にならない)
- (c) 表記をそろえたキーを別の列として保存する
  — Pros: SQL で突き合わせができる
  / Cons: レシピの材料と在庫の両方にマイグレーションが要る。規則を変えるたびに列を作り直す

**採用: (b)**。
S07 は Go のドメイン層でメモリ上に合算しているので、キーを作る関数を差し替えるだけで済む。

### D2. 表記のそろえ方

`internal/domain/shopping/normalize.go` に `normalizeKey(s string) string` を置き、次の順で変換する。

1. `strings.TrimSpace`
2. `norm.NFKC` (`golang.org/x/text/unicode/norm`)：全角英数字と記号と空白を半角に、半角カタカナを全角にする
3. カタカナ (U+30A1〜U+30F6) を 0x60 だけ引いてひらがなにする。長音符「ー」はそのまま残す
4. `strings.ToLower`

**選択肢 (2 の手段):**

- (a) `norm.NFKC` ← 採用
  — Pros: 全角と半角の両方向の変換が 1 回で済む。x/text はすでに依存に入っている
  / Cons: 「㎏」が「kg」に、「①」が「1」になるなど、全角と半角以外の変換も起きる
- (b) `golang.org/x/text/width` の `Fold`
  — Pros: 幅の変換だけに絞れる
  / Cons: 半角カタカナの濁点が分かれたまま残る (「ｶﾞ」が「ガ」にならない)

**採用: (a)**。
NFKC で起きる幅以外の変換 (「㎏」→「kg」) は、比べるときに同じものとして扱いたいものがほとんどである。

漢字とかなの違いは、辞書が無いとそろえられないので扱わない (R19 の別名の辞書に回す)。

### D3. キーと表示の表記を分ける

`Aggregate` の中では、合算と在庫の引き当てに `normalizeKey` を通したキーを使う。
行の表示用の `Item.Name` と `Item.Unit` には、そのキーで最初に行を作ったときの材料の表記 (`TrimSpace` したもの) を入れる。
材料は献立の順 (`sortedBatches`) に走査しているので、最初に行を作った材料が spec の「最初に出てくる材料」になる。

在庫の側 (`stockByKey`、`stockByName`) も同じ `normalizeKey` でキーを作る。
量る単位の判定も、`measuringUnits` の各要素と材料の単位の両方を `normalizeKey` に通してから比べる。

行を並べ替える最後の比較 (食材名、単位の昇順) は、表示の表記のままで比べる。
`Item` の JSON の形は変わらないので、フロントの `itemKey` (食材名 + 単位) も一意のままになる。

### D4. 候補補完の部品を共有する方法

**選択肢:**

- (a) S04 から `fridge/stock-name-input.tsx` をそのまま import する
  — Pros: ファイルの移動が無い
  / Cons: レシピの部品が冷蔵庫の部品に依存する。aria-label の文言キーも `pages.fridge.*` のまま使うことになる
- (b) `frontend/src/components/ingredient/` を作り、入力欄と候補の query をそこへ移して S04 と S06 の両方から使う ← 採用
  — Pros: 食材名という両方の画面にまたがる概念の置き場ができる。S08 の食材マスタを作るときの置き場にもなる
  / Cons: S06 の import を書き換える

**採用: (b)**。

- `ingredient/ingredient-name-input.tsx`：`StockNameInput` を `IngredientNameInput` と改名して移す。中身は変えない
- `ingredient/use-ingredient-suggestions.ts`：`useStockSuggestions` を `useIngredientSuggestions` と改名して移す。
  query key を `["ingredient", "suggestions"]` にし、`INGREDIENT_SUGGESTIONS_KEY` として export する
- `lib/ime.ts`：`isImeComposing` を移す。入力欄と S06 の行の両方が使う

`FridgeService.ListSuggestions` はそのまま使い、レシピ用のサービスメソッドは足さない。
在庫とレシピの材料名を合わせた候補を返すという中身は、S04 でもそのまま使えるからである。

### D5. 候補を捨てる時点

候補の query は、次の mutation の成功時に捨てる。

- 在庫の追加、編集、削除、元に戻す (今の `useInvalidateFridge`。key を差し替える)
- レシピの新規保存と編集の保存 (`routes/recipes/new.tsx` と `routes/recipes/$recipeId.edit.tsx` の成功時の処理に足す)

レシピの削除では捨てない。削除済みレシピの材料名も候補に残る (proposal の Non-goals) ので、捨てても結果が変わらないためである。

### D6. S04 の単位欄の候補

S06 の追加行と同じく、単位欄に `<datalist>` を付けて `units` を並べる。
材料行ごとに `useId` で datalist の id を作る。
combobox の部品にしなかったのは、単位は数が少なく、候補を選んだときに他の欄を埋める処理も無いためである。

## Risks / Trade-offs

- [Risk] NFKC とひらがな化で、利用者が別物のつもりで書いた食材が 1 行にまとまる (例：「ハム」と「はむ」) → 同じ読みの別の食材を、かなの種類だけで書き分けることはまず無い。まとまった行の用途の列でどのレシピの材料かは分かる
- [Risk] S04 の材料名で候補の一覧が開いているとき、Enter が保存の送信と競合する → `IngredientNameInput` は候補の確定時に `preventDefault` と `stopPropagation` を呼んでいる (S06 と同じ)。S04 のフォームは Enter で送信しないので、競合は起きない
- [Risk] 材料行が多いレシピで、行ごとに候補の絞り込みを計算する → 絞り込みは入力中の行だけで走り、候補は数百件程度なので問題にならない

## Migration Plan

- データの移行は無い。表記をそろえるのは S07 の計算の中だけで、戻すときはコミットを revert する
