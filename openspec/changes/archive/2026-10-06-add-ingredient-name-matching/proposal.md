## Why

S07 買い物リストは、レシピの材料と冷蔵庫の在庫を「食材名 + 単位」の完全一致で突き合わせている。
ところが食材名の候補補完は S06 (冷蔵庫の入力行) にしかなく、S04 の材料行はただのテキスト入力 3 つである。
レシピで「タマネギ」、在庫で「たまねぎ」と書くだけで、S07 は在庫があっても「在庫なし」と表示し、材料テーブルにも別々の行を並べる
(docs/reviews/2026-09-25-design-ux-review.md R3)。

食材マスタ (S08) を作れば揺れは根本から無くなるが、マスタの画面と名寄せの運用は重い。
その手前で、揺れを入力の時点で生まれにくくし、突き合わせの時点で機械的に吸収できる分だけ吸収すれば、S07 の精度はかなり上がる。

## What Changes

- S04 の材料名の欄に、S06 と同じ候補補完を付ける
  - 候補は、在庫の食材名とレシピの材料名を合わせたもの (S06 と同じ)
  - 候補を選んだとき単位欄が空なら、その食材名で直近に使われた単位を入れる (S06 と同じ)
  - 候補にない名前もそのまま入力できる
- S04 の単位の欄に、既知の単位を候補として出す (S06 の追加行と同じ)
- S07 で材料どうしと、材料と在庫を突き合わせるとき、食材名と単位の表記をそろえてから比べる
  - そろえるのは、前後の空白、全角と半角 (英数字、記号、カタカナ)、カタカナとひらがな、英字の大文字と小文字
  - 表記をそろえた結果が同じなら、同じ食材 (同じ単位) として合算し、在庫を数える
  - 合算した行の表示は、期間内で最初に出てくる材料の表記を使う
- 漢字とかなの違い (「玉ねぎ」と「たまねぎ」) と、単位の違い (「1 個」と「1 玉」) はそろえない。
  これらは S04 の候補補完で、既存の表記を選んでもらう形で揺れを減らす

### Non-goals

- **食材マスタと別名の辞書 (S08)**：R19 のとおり、この proposal で不満が残ったときに「別名の辞書」程度の軽いものから始める
- **単位の換算**：「200 g」と「1 パック」を同じ量として比べることはしない
- **既存データの書き換え**：レシピと在庫に保存済みの表記は変えない。そろえるのは S07 で比べるときだけ
- **S05 の検索と並び順**：S05 の食材名検索と食材名順の並びは、今までどおり入力された表記で扱う
- **候補から削除済みレシピの材料名を除くこと**：S06 と同じく、削除済みのレシピの材料名も候補に出る。今の S06 の挙動を変えない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：S04 の材料名と単位の入力補助の要件「材料名と単位の候補補完」を ADDED で足す
- `shopping`：
  - 「同一食材の合算」の比較規則を、完全一致から表記をそろえたうえでの一致に改める。表示に使う表記も定める
  - 「量を持たない材料の欄」の、食材名と量る単位の比較規則を同じ規則に改める

## Impact

### 新設ファイル

- `internal/domain/shopping/normalize.go`：食材名と単位の表記をそろえる関数
- `internal/domain/shopping/normalize_test.go`：表記をそろえる関数の table-driven test
- `frontend/src/components/ingredient/ingredient-name-input.tsx`：候補補完つきの食材名の入力欄 (`fridge/stock-name-input.tsx` から移す)
- `frontend/src/components/ingredient/use-ingredient-suggestions.ts`：食材名と単位の候補を取る query (`fridge/use-fridge.ts` の `useStockSuggestions` から移す)
- `frontend/src/lib/ime.ts`：変換中のキー入力の判定 (`fridge/stock-form.ts` の `isImeComposing` から移す)

### 更新ファイル

- `internal/domain/shopping/aggregate.go`：材料と在庫を突き合わせるキーを、表記をそろえた食材名と単位にする
- `internal/domain/shopping/aggregate_test.go`：表記の揺れを吸収するケースを足す
- `internal/service/shopping_test.go`：表記の揺れを吸収するシナリオを足す
- `go.mod` / `go.sum`：`golang.org/x/text` を indirect から直接の依存に上げる
- `frontend/src/components/recipe/recipe-ingredient-row.tsx`：材料名を `IngredientNameInput` に替え、単位に候補の `datalist` を付ける
- `frontend/src/components/recipe/recipe-edit-form.tsx`：候補を取得して各行に渡す
- `frontend/src/routes/recipes/new.tsx` / `frontend/src/routes/recipes/$recipeId.edit.tsx`：保存の成功時に候補の query も捨てる (新しい材料名を候補に反映する)
- `frontend/src/components/fridge/fridge-add-row.tsx` / `fridge-edit-row.tsx`：移した入力欄と判定関数を import し直す
- `frontend/src/components/fridge/stock-form.ts`：`isImeComposing` を削除する
- `frontend/src/components/fridge/use-fridge.ts`：`useStockSuggestions` を削除し、候補の query を捨てる処理を新しい query key に合わせる
- `frontend/src/routes/fridge.tsx`：候補の取得を `useIngredientSuggestions` に替える
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：候補一覧の aria-label を `pages.fridge.add.suggestions_label` から `ingredient.suggestions_label` に移す
- `frontend/e2e/recipe-edit.spec.ts`：describe「材料名と単位の候補補完」を追加する
- `frontend/e2e/shopping.spec.ts`：describe「同一食材の合算」に表記の揺れの Scenario を足す
- `docs/product.md`：§4 S04 の材料行の説明、§4 S07 の材料テーブルの合算の説明を更新する
- `openspec/specs/recipe/spec.md` / `openspec/specs/shopping/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 削除ファイル

- `frontend/src/components/fridge/stock-name-input.tsx`：`ingredient/ingredient-name-input.tsx` に移す

### 生成物 (再生成のみ)

- なし。サービスのメソッドとバインディングの型は変えない

### migration

- なし

### 前提と順序

- 進行中の `revise-recipe-detail-page` が `recipe-edit-form.tsx` にパンくずを足している。そちらを先に archive してから apply する
- `add-mealplan-cooked-mark` は `shopping_repo.go` を、この change は `aggregate.go` を触る。ファイルは重ならないので、どちらが先でもよい

### 後続 proposal への引き継ぎ

- R19：漢字とかなの違いや、「個」と「玉」の違いが S07 の精度を下げ続けるなら、別名の辞書 (S08 の軽量版) を検討する
