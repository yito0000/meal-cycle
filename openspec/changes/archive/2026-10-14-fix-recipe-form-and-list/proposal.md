## Why

レシピの画面に、入力を失う穴と、細かな使いにくさが残っている (docs/reviews/2026-10-14-design-ux-review.md R28、R32、R33)。

- S04 の未保存の確認は、「キャンセル」とパンくずにしか掛かっていない。
  手順を書いている途中でサイドバーから献立を見に行くと、入力は確認なしで捨てられる
- S04 の人数欄は、空にすると即座に 1 へ戻る。Backspace で消して 4 と打つと 14 になる。材料の量も同じ作りで、空にすると 0 が入る
- S02 の更新日の列は、`2026-10-13T03:21:45Z` のような UTC の時刻の文字列をそのまま出している
- S02 の行はクリックでしか開けず、キーボードではフォーカスが当たらない

1 つめは、入力した文章を失うので影響が大きい。
残りは小さいが、どれも同じ 3 ファイル前後に収まるので、まとめて直す。

## What Changes

- S04 から別の画面へ移るすべての操作 (サイドバーを含む) で、未保存の変更があれば確認ダイアログを出す
  - 「破棄」を選ぶと、押した先の画面へ移る
  - 保存に成功したあとの移動では出さない
- S04 の人数欄と材料の量の欄で、入力途中の内容をそのまま保持する
  - 人数が空のまま保存しようとすると、人数のエラーにする
  - 量が空のまま保存すると、量 0 (量を持たない材料) として保存する
- S02 の更新日を、PC の時刻で見た年月日 (`yyyy/mm/dd`) で表示する
- S02 の各行のレシピ名を、Tab でフォーカスでき、Enter で S03 を開けるボタンにする
- product.md の S02 の列の記述 (「材料数」がある) を、spec と実装の 3 列に合わせる (R35 の一部)

### Non-goals

- **アプリを閉じるときの確認**：ウィンドウを閉じる操作での未保存の確認は扱わない
- **材料の量の負の値の検証**：入力欄の `min=0` に任せ、保存時の検証は足さない
- **材料行の並び替え**：扱わない
- **手順の持ち方の記述の修正** (R12)：product.md §4 S04 と design.md §4 の修正は、design.md の全体的な見直し (R20) と一緒に行う
- **S02 の検索対象の拡張** (R13)

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：
  - 「未保存変更のキャンセル確認」を、サイドバーを含むすべての画面の移動に広げる
  - 「レシピ一覧の初期表示」に、更新日の表示の形を足す
  - 「S02 の行のキーボード操作」と「数値欄の入力途中の保持」を足す

## Impact

### 新設ファイル

- なし

### 更新ファイル

- `frontend/src/components/recipe/recipe-edit-form.tsx`
  - `pendingLeave` をやめ、TanStack Router の `useBlocker` で画面の移動を止める
  - 人数と量を文字列で持つ形に合わせ、保存時に数値へ直す
- `frontend/src/components/recipe/use-recipe-form.ts`：`RecipeFormValues` の `servings` と、材料行の `amount` を文字列にする
- `frontend/src/components/recipe/recipe-ingredient-row.tsx`：量の欄を文字列で受け渡す
- `frontend/src/components/recipe/recipe-table.tsx`：更新日を整形し、レシピ名をボタンにする
- `frontend/src/components/recipe/recipe-detail.tsx`：更新日の整形を共通の関数に置き換える
- `frontend/src/lib/format-date.ts`：`formatTimestampDate(rfc3339)` を足す
- `frontend/e2e/recipe-edit.spec.ts`：describe「未保存変更のキャンセル確認」にサイドバーのシナリオを足し、describe「数値欄の入力途中の保持」を足す
- `frontend/e2e/recipe-listing.spec.ts`：describe「レシピ一覧の初期表示」に更新日のシナリオを足し、describe「S02 の行のキーボード操作」を足す
- `docs/product.md`：§4 S02 の列を「名前 / タグ / 更新日」に直し、§4 S04 の未保存の確認にサイドバーを足す
- `docs/reviews/2026-10-14-design-ux-review.md`：R28、R32、R33 にチェックを入れる
- `openspec/specs/recipe/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- なし。Go 側とバインディングは変えない

### migration

- なし

### 前提と順序

- ほかの change に依存しない。献立まわりの change とはファイルが重ならないので、どの順でも apply できる

### 後続 proposal への引き継ぎ

- S06 / S07a：在庫の入力欄はすでに文字列で保持している。フォームの持ち方がそろうので、数値欄の共通部品を作るならここを起点にできる
- R12 / R20：design.md §4 のデータモデルを実装に合わせる作業は、別の `docs:` コミットで行う
