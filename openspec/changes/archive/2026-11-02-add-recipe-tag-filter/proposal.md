## Why

レシピにはタグ (「和食」「簡単」など) を付けられるが、S02 で絞り込めるのはレシピ名の部分一致だけである ([product.md §4 S02](../../../docs/product.md))。
「今日は簡単なものにしたい」「中華の気分」というときに、名前を思い出せないレシピを探す手段が無い。
レパートリーが 20〜50 品 (product.md §1 ペルソナ) あると、一覧を目で追うのは手間になる。

[design.md §9](../../../docs/design.md) は MVP 後の候補に「レシピのタグフィルタ」を挙げており、
docs/reviews の R13 は「S10 (検索画面) は S02 の検索拡張で足りる」としている。本 proposal は S02 の上部にタグの絞り込みを足す。

## What Changes

- S02 の検索欄の下に、削除済みでないレシピに付いているタグをすべてチップで並べる (タグの名前の昇順)
- チップを押すと選択と解除が切り替わる。選んだタグを**すべて**持つレシピだけをテーブルに残す
- タグの絞り込みは、レシピ名の検索と並び替えと同時に効く
- 「タグの選択を解除」で選択をすべて外す。1 つも選んでいないときは表示しない
- S03 から S02 に戻ったときは、検索語と並び順と同じく、選んでいたタグを保つ。サイドバーから開き直したときは選択なしに戻る
- 選んだタグを持つレシピが無いときは、検索と同じ「一致するレシピがない」旨を出す
- タグが 1 つも無いときは、チップの並びを出さない

### Non-goals

- **材料名での絞り込み**：「玉ねぎを使うレシピ」は、在庫を起点にした「使い切り」の候補で一部を満たしている。必要になったら別の proposal にする
- **OR 条件での絞り込み**：絞り込みは選ぶほど狭まる AND にそろえる。OR の需要が出たら切替を足す
- **S01 のレシピ選択ダイアログでのタグの絞り込み**：ダイアログは候補の欄と名前の検索で選ぶ。必要になったら同じ部品を流用する
- **タグの名前の変更や統合、タグの管理画面**
- **テーブルのタグ列のタグを押して絞り込むこと**：行を押すと S03 へ移る操作と衝突する

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：S02 にタグでの絞り込みを足し、S03 から戻ったときに選んだタグを保つ

## Impact

**新設ファイル**

- `frontend/src/components/recipe/recipe-tag-filter.tsx`：タグのチップの並びと選択の解除
- `frontend/src/components/recipe/recipe-filter.ts`：レシピからタグの一覧を作る関数と、名前とタグで絞り込む関数
- `frontend/src/components/recipe/recipe-filter.test.ts`
- `frontend/e2e/recipe-tag-filter.spec.ts`

**更新ファイル**

- `frontend/src/router.tsx`：`recipesSearchSchema` に `tags: string[]` を足す
- `frontend/src/components/recipe/detail-origin.ts`：S03 の開いた元に `tags` を持たせ、戻るときに渡す
- `frontend/src/routes/recipes.tsx`：絞り込みの処理を `recipe-filter.ts` に移し、タグの選択を search param と同期する
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`
- `docs/product.md`：§4 S02 の「タグ・材料での絞り込みは将来 S10」を直し、タグの絞り込みの節を足す。§2 の S10 の扱いを直す
- `docs/design.md`：§9 のタグフィルタを実装済みに直す
- `docs/wireframes/S02_recipe_list.drawio.svg`：検索欄の下にタグのチップを足す

**生成物 (再生成のみ)**

- なし (`RecipeService.List` がタグを返しており、Go は変えない)

**migration**

- なし

**後続 proposal への引き継ぎ**

- S01 のレシピ選択ダイアログにタグの絞り込みを足すなら、`recipe-tag-filter.tsx` と `recipe-filter.ts` をそのまま使う
- タグが 30 を超えて 1 行に収まらなくなったら、チップの折りたたみか検索つきの選択に変える
