## 1. 開いた元と画面の状態の持ち回り

- [x] 1.1 `frontend/src/components/recipe/detail-origin.ts` を新設する (design.md D2)
    - `DetailOrigin` 型を `{ to: "recipes"; q?: string; sort?: RecipeSortKey } | { to: "mealplan"; start?: string }` で定義する
    - `detailOriginSchema` (zod) を定義し、欠けているか壊れているときは `.catch` で `{ to: "recipes" }` にする
    - `originNavigateOptions(origin)` を定義し、`router.navigate` に渡す `{ to, search }` を返す (`recipes` → `/recipes`、`mealplan` → `/`)
    - `originLabelKey(origin)` を定義し、`nav.recipes` または `nav.meal_plan` を返す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `frontend/src/router.tsx` を更新し、`/recipes` の search を `q` と `sort` に替え、`detail` を削除する (design.md D3)。完了判定: `npm run type-check` が通る
- [x] 1.3 `frontend/src/router.tsx` を更新し、`/` に search `start` (YYYY-MM-DD、省略可) を追加する (design.md D4)。完了判定: `npm run type-check` が通る
- [x] 1.4 `frontend/src/router.tsx` を更新し、`/recipes/$recipeId/edit` に search `back` (`detailOriginSchema`) を追加する。完了判定: `npm run type-check` が通る
- [x] 1.5 `frontend/src/routes/meal-plan.tsx` を更新し、開始日を search の `start` から作る (design.md D4)
    - `start` を含む週の月曜に丸め、`start` が無いか壊れているときは今週の月曜にする
    - `PeriodNav` の変更で `navigate({ search: { start }, replace: true })` を呼び、`useState` の `periodStart` を削除する
    - 完了判定: `npm run type-check` が通り、`task dev` で `＜` `＞` 「今週」が従来どおり動くことを目視で確認する
- [x] 1.6 `frontend/src/routes/recipes.tsx` を更新し、検索語と並び順を search の `q` と `sort` で持つ (design.md D3)
    - 検索欄の入力値は `useState` で持ち、初期値を `q` から取り、変わるたびに `replace: true` で `q` に書き写す
    - 並び順の変更は `replace: true` で `sort` に書き写す
    - 保存直後の `highlight` を消す処理は、`q` と `sort` を残したまま `highlight` だけを外す形を保つ
    - 完了判定: `npm run type-check` が通る

## 2. 画面

- [x] 2.1 `frontend/src/components/ui/breadcrumb.tsx` を新設し、shadcn の Breadcrumb を手動でコピーする (design.md D10)。完了判定: `npm run type-check` が通る
- [x] 2.2 `frontend/src/components/layout/breadcrumb.tsx` を新設する (design.md D10)
    - props は `items: { label: string; onSelect?: () => void }[]`
    - `onSelect` のある項目は `button` で描き、最後の項目は `aria-current="page"` の文字で描く
    - `nav` の `aria-label` は `breadcrumb.aria_label` を使う
    - 完了判定: `npm run type-check` が通る
- [x] 2.3 `frontend/src/components/recipe/recipe-detail.tsx` を新設し、`recipe-detail-drawer.tsx` の `RecipeDetailBody` / `Section` / `Empty` を移す
    - Sheet の部品 (`SheetHeader` / `SheetTitle` / `SheetFooter` など) を、見出し `h1` と通常の要素に置き換える
    - 上部にレシピ名、基本情報の行、「編集」「削除」ボタン (右寄せ) を置く (design.md D6)
    - 本文を `lg:grid-cols-[20rem_1fr]` の 2 列にし、左に材料、右に手順とメモを置き、右の列を `max-w-3xl` に抑える
    - 削除の確認ダイアログの開閉を props `confirmOpen` / `onConfirmOpenChange` で親に渡す (design.md D5)
    - 削除の成功時に `onDeleted` を呼ぶ。キャッシュの無効化と `refetchType: "none"` は今のまま残す (design.md D11)
    - 完了判定: `npm run type-check` が通る
- [x] 2.4 `frontend/src/routes/recipes/$recipeId.tsx` を新設する
    - `useParams` の `recipeId` で `useRecipeDetail` を呼び、読み込み中と読み込めないときの表示を出す
    - 上部にパンくず `[originLabelKey(back), レシピ名]` を置き、レシピを読み込めないときは先頭の項目だけを出す
    - パンくずの先頭で `originNavigateOptions(back)` へ移る
    - 削除の確認ダイアログが閉じているときだけ、`keydown` の `Escape` で `originNavigateOptions(back)` へ移る (design.md D5)
    - 「編集」で `/recipes/$recipeId/edit` に `back` を付けて移る
    - 削除の成功時に `originNavigateOptions(back)` へ移る
    - 完了判定: `npm run type-check` が通る
- [x] 2.5 `frontend/src/router.tsx` を更新し、`/recipes/$recipeId` を `RecipeDetailPage` と search `back` (`detailOriginSchema`) で登録する。完了判定: `npm run type-check` が通る
- [x] 2.6 `frontend/src/routes/recipes.tsx` を更新し、行の押下で `/recipes/$recipeId` に `back: { to: "recipes", q, sort }` を付けて移る。完了判定: `npm run type-check` が通る
- [x] 2.7 `frontend/src/routes/recipes.tsx` から `RecipeDetailDrawer` を外す。完了判定: `npm run type-check` が通る
- [x] 2.8 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、`view-detail` で `/recipes/$recipeId` に `back: { to: "mealplan", start }` を付けて移る
    - `start` は props の `periodStart` を `formatDate` した値にする
    - `detailRecipeId` の state と `RecipeDetailDrawer` を削除する
    - 完了判定: `npm run type-check` が通る
- [x] 2.9 `frontend/src/components/recipe/recipe-edit-form.tsx` を更新し、props `breadcrumb` を受け取ってフォームの上部にパンくずを描く (design.md D8)
    - `onSelect` のある各項目を、未保存なら確認ダイアログを開いて移動先を保留し、そうでなければ即座に移る処理で包む
    - 確認ダイアログの「破棄」で、保留した移動先 (キャンセルのときは `onCancel`) を実行する
    - 完了判定: `npm run type-check` が通る
- [x] 2.10 `frontend/src/routes/recipes/$recipeId.edit.tsx` を更新する
    - パンくず `[originLabelKey(back), レシピ名 (S03 へ), 編集]` を `RecipeEditForm` に渡す
    - キャンセルで `/recipes/$recipeId` に同じ `back` を付けて移る
    - 読み込めないときの「一覧に戻る」は今のまま S02 へ移す
    - 完了判定: `npm run type-check` が通る
- [x] 2.11 `frontend/src/routes/recipes/new.tsx` を更新し、パンくず `[レシピ (S02 へ), 新しいレシピ]` を `RecipeEditForm` に渡す。完了判定: `npm run type-check` が通る
- [x] 2.12 `frontend/src/components/recipe/recipe-detail-drawer.tsx` を削除する。完了判定: `npm run type-check` が通る
- [x] 2.13 `frontend/src/components/ui/sheet.tsx` を削除する。完了判定: `npm run type-check` と `npm run build:dev` が通る
- [x] 2.14 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する
    - `breadcrumb.aria_label`：「パンくずリスト」/ "Breadcrumb"
    - `pages.recipes.edit.breadcrumb_edit`：「編集」/ "Edit"
    - `pages.recipes.detail.close` を削除する
    - 完了判定: `npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 3. E2E

- [x] 3.1 `frontend/e2e/helpers.ts` を更新する
    - `openRecipeDetail` を、行を押したあと S03 のパンくず (`navigation` の「パンくずリスト」) が見えるまで待つ形に改める
    - `deleteRecipeFromDetail` を、S03 の「削除」を押し、確認ダイアログの「削除」を選び、ダイアログが 0 個になるまで待つ形に改める
    - `detailBreadcrumb(page)` を追加し、S03 と S04 のパンくずの `navigation` を返す
    - 完了判定: `npm run type-check` が通る
- [x] 3.2 `frontend/e2e/recipe-detail.spec.ts` の describe「S03 レシピ詳細ドロワーの表示」を「S03 レシピ詳細画面の表示」に改め、テストをそろえる
    - 詳細画面に基本情報と材料と手順とメモが表示される (改名。S02 の一覧の見出しが表示されないことも確かめる)
    - 空の欄には未登録である旨が表示される
    - 量が 0 の材料は量を出さずに表示される
    - 量る単位は量の前に表示される
    - `getByRole("dialog")` で絞っていた箇所を `getByRole("main")` に替える
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.3 `frontend/e2e/recipe-detail.spec.ts` の describe「S03 の開閉」を削除する。完了判定: `npm run check:scenarios` が通る
- [x] 3.4 `frontend/e2e/recipe-detail.spec.ts` の describe「手順とメモの Markdown 描画」で、`getByRole("dialog")` で絞っていた箇所を `getByRole("main")` に替える。完了判定: `task test:e2e` で同 describe が pass する
- [x] 3.5 `frontend/e2e/recipe-detail-navigation.spec.ts` を新設し、describe「S03 から開いた元の画面へ戻る操作」を書く
    - パンくずに開いた元とレシピ名が表示される
    - パンくずの先頭を押すと開いた元の画面に戻る
    - Esc キーで開いた元の画面に戻る
    - 確認ダイアログを開いているときの Esc キーはダイアログだけを閉じる
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.6 `frontend/e2e/recipe-detail-navigation.spec.ts` に describe「S01 献立カレンダーから S03 詳細画面へ移る」を書く
    - S01 から開いてパンくずで戻ると表示期間が保たれている
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.7 `frontend/e2e/recipe-listing.spec.ts` の describe「S02 一覧から S03 を開く」を「S02 一覧から S03 詳細画面へ移る」に改め、テストをそろえる
    - 行を押すと S03 詳細画面に移る (改名)
    - S02 に戻ると検索語と並び順が保たれている (改名。パンくずの「レシピ」で戻る)
    - サイドバーから開き直すと検索語と並び順は既定に戻る (新規)
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.8 `frontend/e2e/recipe-edit.spec.ts` の describe「S03 から S04 編集画面への遷移」をそろえる
    - 編集を押すと既存値がロードされた S04 に遷移する
    - 保存すると S02 に戻り行が強調される
    - キャンセルすると S03 に戻る (S03 のパンくずに「レシピ」「肉じゃが」が並ぶことを確かめる)
    - 保存した内容が S03 に反映される
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.9 `frontend/e2e/recipe-edit.spec.ts` の describe「未保存変更のキャンセル確認」の「編集モードで変更を破棄すると S03 に戻る」を、S03 詳細画面で人数を確かめる形に改める。完了判定: `task test:e2e` で同テストが pass する
- [x] 3.10 `frontend/e2e/recipe-edit.spec.ts` に describe「S04 のパンくず」を追加する
    - 編集画面のパンくずに開いた元とレシピ名が表示される
    - 新規登録画面のパンくず
    - パンくずで移るときも未保存の変更を確かめる
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.11 `frontend/e2e/recipe-delete.spec.ts` の describe「レシピの削除」を「S03 詳細画面からのレシピの削除」に改め、テストをそろえる
    - 確認してレシピを削除すると一覧から消える
    - 確認ダイアログでキャンセルすると削除しない (S03 に留まり、パンくずで戻ると行が残っている)
    - S01 から開いた詳細画面で削除すると S01 に戻る (改名)
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.12 `frontend/e2e/recipe-delete.spec.ts` の describe「S03 レシピ詳細ドロワーの表示」を「S03 レシピ詳細画面の表示」に改める
    - 削除済みレシピは内容と削除済みの旨を表示し、操作ボタンを出さない
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.13 `frontend/e2e/mealplan-assign.spec.ts` の「詳細を開くと S02 を経由せずに S01 に戻れる」を、パンくずの「献立」で S01 に戻る形に改める。完了判定: `task test:e2e` で同テストが pass する

## 4. ドキュメント

- [x] 4.1 `docs/wireframes/S03_recipe_detail.v2.drawio.svg` を新設する (design.md D12)
    - サイドバー、パンくず、レシピ名と基本情報、右寄せの「編集」「削除」、左に材料、右に手順とメモの 2 列を描く
    - 可視 SVG と `content=` 属性内の drawio XML の両方に同じ図形を入れる
    - 完了判定: ブラウザで SVG として表示でき、drawio で開いて同じ図形が編集できることを目視で確認する
- [x] 4.2 `docs/wireframes/README.md` のファイル一覧に `S03_recipe_detail.v2.drawio.svg` を追加する。完了判定: 差分を目視で確認する
- [x] 4.3 `docs/product.md` §3 を更新する
    - 画面遷移図の `S03<br/>レシピ詳細<br/>ドロワー` を独立した画面に改め、S01 から S03 への遷移を足す
    - 「S03 レシピ詳細はドロワー」と「S03 レシピ詳細だけメイン右側にドロワーが重なる」の説明を、全画面とパンくずの説明に改める
    - 完了判定: 差分を目視で確認する
- [x] 4.4 `docs/product.md` §4 を更新する
    - S01 の遷移の「S03 ドロワー」を S03 の画面に改める
    - S02 の「行クリック → S03 レシピ詳細ドロワーが右から展開」を S03 の画面への移動に改め、戻ったときに検索語と並び順が残ることを書く
    - S03 の節の見出しと本文を、全画面、パンくず、Esc キー、削除後の戻り先の説明に書き換え、ワイヤーに v2 を追加する
    - S04 の遷移にパンくずと、キャンセルの戻り先が S03 の画面であることを書く
    - 完了判定: 差分を目視で確認する
- [x] 4.5 `docs/product.md` の決定の記録の「S03 レシピ詳細：ドロワー展開」を、全画面にした理由 (手順を読む画面になったこと、S04 との一貫性) に改める。完了判定: 差分を目視で確認する

## 5. 検証

- [x] 5.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 5.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 5.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 5.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 5.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 5.6 `openspec validate revise-recipe-detail-page --strict` を実行する。完了判定: エラー 0 件
- [x] 5.7 `task build:win` で Windows ビルドを作り、Windows 実機で確かめる
    - S02 の検索欄に IME で「じゃが」を入力でき、変換中の文字が崩れない
    - S03 からパンくずで戻ると、検索欄に「じゃが」が残っている
    - 完了判定: 目視で確認する

## 6. archive 時

- [x] 6.1 archive のあと、`openspec/specs/recipe/spec.md` の Purpose の「S03 詳細ドロワー」を「S03 詳細画面」に改める。完了判定: `grep -n "ドロワー" openspec/specs/recipe/spec.md` が 0 件になる
