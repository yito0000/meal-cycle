## 1. 絞り込みの純関数

- [ ] 1.1 `frontend/src/components/recipe/recipe-filter.ts` を新設し、`collectTags(recipes)` と `filterRecipes(recipes, { query, tags })` を書く (design.md D2 / D3 / D5)。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 1.2 `frontend/src/components/recipe/recipe-filter.test.ts` を新設し、`it.each` の table-driven で書く。完了判定: `cd frontend && npm run test` が pass する
  - collectTags は重複を除いて名前の昇順に返す
  - collectTags は削除済みのレシピのタグを含めない
  - filterRecipes はタグが空なら名前の検索だけで絞る
  - filterRecipes は選んだタグをすべて持つレシピだけを返す
  - filterRecipes はタグを完全一致で比べる (「和食」は「和食・洋食」に一致しない)
  - filterRecipes は名前の検索とタグの両方を満たすレシピだけを返す

## 2. 画面

- [ ] 2.1 `frontend/src/router.tsx` の `recipesSearchSchema` に `tags` を足す (design.md D4)。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 2.2 `frontend/src/components/recipe/detail-origin.ts` の S02 の開いた元に `tags` を足し、戻る先の search に渡す。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 2.3 `frontend/src/components/recipe/recipe-tag-filter.tsx` を新設し、チップの並びと「タグの選択を解除」を描く。チップは `aria-pressed` で選択状態を示す。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 2.4 `frontend/src/routes/recipes.tsx` を更新し、検索欄の下に `recipe-tag-filter` を置き、`filterRecipes` で絞り込む。
  選択を search param の `tags` と同期し、一覧に無くなったタグを選択から外す。完了判定: `task dev` の S02 で目視
- [ ] 2.5 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に文言を足す。完了判定: `cd frontend && npm run type-check` が通る

## 3. E2E

- [ ] 3.1 `frontend/e2e/recipe-tag-filter.spec.ts` を新設する。完了判定: `task test:e2e` で pass する
  - describe「タグでの絞り込み」
    - タグを選ぶとそのタグを持つレシピだけが残る
    - 複数のタグを選ぶとすべてを持つレシピだけが残る
    - 選んだタグをもう一度押すと解除される
    - タグと名前の検索を同時に使う
    - 該当するレシピが無いと一致しない旨が出る
  - describe「S02 一覧から S03 詳細画面へ移る」
    - S02 に戻ると選んでいたタグが保たれている
    - サイドバーから開き直すとタグの選択は外れる

## 4. ドキュメント

- [ ] 4.1 `docs/wireframes/S02_recipe_list.drawio.svg` の検索欄の下にタグのチップを足す (可視 SVG と編集 XML の両方)。完了判定: 目視
- [ ] 4.2 `docs/product.md` の §4 S02 にタグの絞り込みの節を足し、「タグ・材料での絞り込みは将来 S10」を直す。完了判定: 目視
- [ ] 4.3 `docs/design.md` の §9 のタグフィルタを実装済みに直す。完了判定: 目視

## 5. 確認

- [ ] 5.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [ ] 5.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [ ] 5.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [ ] 5.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [ ] 5.5 `task test:e2e` を実行する。完了判定: 全 pass
- [ ] 5.6 `task lint` を実行する。完了判定: エラー 0
