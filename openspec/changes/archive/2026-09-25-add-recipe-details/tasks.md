## 1. 依存と UI プリミティブ

- [x] 1.1 `frontend/package.json` を更新し、`cd frontend && npm install react-markdown@^10` で dependencies に追加する
      (`frontend/package-lock.json` も更新される)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `frontend/src/components/ui/sheet.tsx` を新設し、shadcn の Sheet を手動コピーする (design.md D1)
    - `@radix-ui/react-dialog` をベースに `Sheet` / `SheetContent` / `SheetHeader` / `SheetTitle` / `SheetDescription` / `SheetFooter` / `SheetClose` を export する
    - `side` バリアントは右 (`right`) だけ残し、他のバリアントは削る
    - `SheetContent` の幅は `w-[45vw] min-w-[28rem] max-w-[48rem]`、中身は縦スクロールできるようにする
    - `×` ボタンの aria-label は呼び出し側から i18n 文字列で渡せるよう props で受け取る
    - 完了判定: `cd frontend && npm run type-check` が通る

## 2. 詳細ドロワー部品

- [x] 2.1 `frontend/src/components/recipe/use-recipe-detail.ts` を新設し、`useRecipeDetail(id: string | null)` を実装する。
      query key は `["recipes", "detail", id]`、`id` が null の間は `enabled: false` にする (design.md D5)。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/routes/recipes/$recipeId.edit.tsx` を更新し、詳細取得の `useQuery` を `useRecipeDetail(recipeId)` に置き換える。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.3 `frontend/src/components/recipe/recipe-markdown.tsx` を新設し、`react-markdown` で `text: string` を描画する (design.md D6)
    - `components.img` を `alt` だけを出す `<span>` に差し替える
    - `components.a` を子要素だけを出す `<span>` に差し替える (`href` を持たせない)
    - 見出し (h1〜h3) / `ul` / `ol` / `li` / `code` / `pre` / `blockquote` / `p` に最小限の Tailwind クラスを当てる
    - `rehype-raw` を入れない理由 (生 HTML を通すと外部画像の読み込みが復活する) を 1 行コメントで残す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.4 `frontend/src/components/recipe/recipe-detail-drawer.tsx` を新設し、`RecipeDetailDrawer` を実装する
      (props: `recipeId: string | null` / `onClose: () => void` / `onEdit: (id: string) => void`、design.md D2)
    - `recipeId` が非 null のとき `Sheet` を open にし、`onOpenChange(false)` で `onClose` を呼ぶ
    - 閉じるアニメーション中に中身が消えないよう、最後に表示した `recipeId` を内部で保持する (design.md D5)
    - 読み込み中は `pages.recipes.detail.loading`、失敗時は `pages.recipes.detail.load_error` を出す
    - 基本情報の行に、タグ (` / ` 区切り、0 件なら省略)、人数 (`pages.recipes.detail.servings`)、更新日 (ローカル日付を `formatDisplayDate` で `yyyy/mm/dd`) を出す (design.md D7)
    - 材料欄は 1 行 1 材料で、量が 0 なら「名前 単位」、それ以外は `formatAmount` を使って「名前 量 単位」を出す
    - 手順欄とメモ欄は `RecipeMarkdown` で描画する
    - 材料 0 件 / 手順が空 / メモが空の欄には `pages.recipes.detail.empty` を出す
    - フッタに「編集」ボタンだけを置き、押したら `onEdit(recipeId)` を呼ぶ。「削除」ボタンは置かない
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.5 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.recipes.detail.*` を追加する
    - `close` / `loading` / `load_error` / `servings` (`{{count}} 人分`) / `updated_at` / `section.ingredients` / `section.steps` /
      `section.notes` / `empty` / `edit`
    - 完了判定: 2.4 のコンポーネントにハードコードの日本語が無く、`cd frontend && npm run type-check` が通る

## 3. 呼び出し元の結線

- [x] 3.1 `frontend/src/router.tsx` を更新し、`recipesSearchSchema` に `detail: z.string().optional()` を追加する。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/routes/recipes.tsx` を更新し、行押下で `search: (prev) => ({ ...prev, detail: id })` へ navigate する
      (S04 への直接遷移を削除)。完了判定: `task dev` で行を押すとドロワーが開くことを目視
- [x] 3.3 `frontend/src/routes/recipes.tsx` に `RecipeDetailDrawer` を置き、`recipeId={detail ?? null}` を渡す。
      `onClose` は `detail` を落とす navigate、`onEdit` は `/recipes/$recipeId/edit` への navigate にする。
      完了判定: `task dev` で `×` / 背景クリック / Esc の 3 通りで閉じ、検索語と並び順が残ることを目視
- [x] 3.4 `frontend/src/routes/recipes.tsx` の強調の自動解除を `search: (prev) => ({ ...prev, highlight: undefined })` に変える (design.md D3)。
      完了判定: `task dev` で保存直後 2 秒以内に行を押しても、ドロワーが勝手に閉じないことを目視
- [x] 3.5 `frontend/src/routes/recipes/$recipeId.edit.tsx` の `onCancel` を `/recipes` + `search: { detail: recipeId }` への navigate に変える (design.md D4)。
      保存時の遷移と読み込み失敗時の「一覧に戻る」は変えない。完了判定: `task dev` で S04 のキャンセル後に同じレシピのドロワーが開いていることを目視
- [x] 3.6 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、`detailRecipeId` の `useState` を追加して
      `view-detail` で `entry.recipeId` を入れる (design.md D8)。プレースホルダのエラー表示とコメントを削除する。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.7 `frontend/src/components/mealplan/week-calendar.tsx` に `RecipeDetailDrawer` を置き、`onClose` で `detailRecipeId` を null に戻し、
      `onEdit` で `/recipes/$recipeId/edit` へ navigate する。完了判定: `task dev` で S01 から開いて閉じると表示期間が変わらないことを目視

## 4. E2E

E2E で確かめるシナリオは本グループで列挙したものに限る (design.md D9)。test 名は delta spec の `#### Scenario:`、
最外の describe 名は `### Requirement:` と一字一句揃える。E2E に含めないシナリオと理由は design.md D9 に書いてある。
各タスクの `npx playwright test <file>` は、`cd frontend && npx vite build --mode e2e` で e2e ビルドを作ってから実行する。

- [x] 4.1 `frontend/e2e/helpers.ts` を更新し、E2E 用のヘルパを 3 つ追加する
    - `createRecipeDetailed(page, { name, tags?, servings?, ingredients?, steps?, notes? })`：API で 1 件作り ID を返す。
      省略した項目は `createRecipe` と同じ既定値 (人数 2、材料「玉ねぎ 1 個」、手順 / メモは空) にし、`ingredients: []` を明示すれば 0 件にできる
    - `openRecipeDetail(page, name)`：S02 を開き、`name` の行を押して `getByRole("dialog")` が見えるまで待つ
    - `openItemMenu(page, recipeName)`：S01 の品行の `⋯` を押し、`page.getByRole("menu")` が見えるまで待つ (design.md D9)
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/e2e/recipe-edit.spec.ts` から describe「S02 一覧からのレシピ編集導線 (暫定)」を削除する (REMOVED 要件)。
      完了判定: `grep -n "暫定" frontend/e2e/recipe-edit.spec.ts` が 0 件になる
- [x] 4.3 `frontend/e2e/recipe-listing.spec.ts` の describe「S02 一覧から S03 を開く」に次の 2 本を実装する (既存の `test.skip` を置き換える)
    - 「行を押すと S03 詳細ドロワーが開く」：行を押すと `dialog` に「肉じゃが」が見え、一覧の行も見えていて、`#recipe-name` (S04) は無い
    - 「閉じると検索語と並び順が保たれている」：検索欄に「じゃが」、並び順「名前 昇順」にしてから行を押し、`×` で閉じると
      検索欄の値と並び順の表示が変わっていない
    - 完了判定: `cd frontend && npx playwright test recipe-listing` で 2 本が pass する
- [x] 4.4 `frontend/e2e/recipe-listing.spec.ts` から describe「S03 の開閉」(`test.skip("背景クリックで閉じる")`) を削除する (4.5 へ移す)。
      完了判定: `grep -rn "背景クリックで閉じる" frontend/e2e/` が `recipe-detail.spec.ts` の 1 件だけになる (4.5 完了後)
- [x] 4.5 `frontend/e2e/recipe-detail.spec.ts` を新設し、describe「S03 レシピ詳細ドロワーの表示」に次の 3 本を実装する
    - 「詳細ドロワーに基本情報と材料と手順とメモが表示される」：タグ「和食」/ 人数 2 / 材料「玉ねぎ 1 個」/ 手順 / メモ付きのレシピで、
      `dialog` に「肉じゃが」「和食」「2 人分」「玉ねぎ」、手順とメモの文言、「編集」ボタンが見え、「削除」ボタンが無い
    - 「空の欄には未登録である旨が表示される」：材料 0 件 / 手順とメモが空のレシピで、`pages.recipes.detail.empty` が 3 か所に出る
    - 「量が 0 の材料は量を出さずに表示される」：材料「塩」(量 0、単位「少々」) で、材料欄に「塩」「少々」が見え、「0」を含まない
    - 完了判定: `cd frontend && npx playwright test recipe-detail` で 3 本が pass する
- [x] 4.6 `frontend/e2e/recipe-detail.spec.ts` に describe「S03 の開閉」を追加し、次の 3 本を実装する
    - 「`×` ボタンで閉じる」：`pages.recipes.detail.close` のボタンを押すと `dialog` が消え、S02 の見出しが見えている
    - 「背景クリックで閉じる」：ドロワー外 (画面左端付近の座標) をクリックすると `dialog` が消える
    - 「Esc キーで閉じる」：`page.keyboard.press("Escape")` で `dialog` が消える
    - 完了判定: `cd frontend && npx playwright test recipe-detail` で 3 本が pass する
- [x] 4.7 `frontend/e2e/recipe-detail.spec.ts` に describe「手順とメモの Markdown 描画」を追加し、次の 4 本を実装する
    - 「見出しと箇条書きが描画される」：手順 `## 下ごしらえ\n- 玉ねぎを切る\n- 豚肉を漬ける` で、`dialog` 内に
      `getByRole("heading", { name: "下ごしらえ" })` と `getByRole("listitem")` 2 件が見え、「##」の文字列が無い
    - 「画像記法は代替テキストだけが表示される」：メモ `![完成写真](https://example.com/a.jpg)` で、「完成写真」が見え、`dialog` 内に `img` が 0 件
    - 「リンク記法は遷移しない文字列として表示される」：メモ `[参考](https://example.com)` で、`dialog` 内に `a` が 0 件、
      「参考」を押しても `dialog` が開いたまま
    - 「HTML タグは文字列として表示される」：メモ `<b>強火</b>` で、`dialog` 内に `b` 要素が 0 件、「<b>強火</b>」の文字列が見える
    - 完了判定: `cd frontend && npx playwright test recipe-detail` で 4 本が pass する
- [x] 4.8 `frontend/e2e/recipe-edit.spec.ts` の describe「S03 から S04 編集画面への遷移」に次の 4 本を実装する (既存の `test.skip` を置き換える)
    - 「編集を押すと既存値がロードされた S04 に遷移する」：`openRecipeDetail` → 「編集」で `#recipe-name` が「肉じゃが」
    - 「保存すると S02 に戻り行が強調される」：人数を 3 にして保存すると S02 の見出しが見え、`dialog` が無く、「肉じゃが」の行に強調のクラスが付いている
    - 「キャンセルすると S03 に戻る」：何も変えずに「キャンセル」を押すと S02 に戻り、`dialog` に「肉じゃが」が見える
    - 「保存した内容が S03 に反映される」：人数を 3 にして保存したあと、行を押すと `dialog` に「3 人分」が見える
    - 完了判定: `cd frontend && npx playwright test recipe-edit` で 4 本が pass する
- [x] 4.9 `frontend/e2e/recipe-edit.spec.ts` の describe「未保存変更のキャンセル確認」に次の 1 本を追加する
    - 「編集モードで変更を破棄すると S03 に戻る」：S03 → 「編集」で人数を 3 に変え、「キャンセル」→「破棄」を押すと
      `dialog` に「肉じゃが」と「2 人分」が見える
    - 完了判定: `cd frontend && npx playwright test recipe-edit` でこのテストが pass する
- [x] 4.10 `frontend/e2e/mealplan-assign.spec.ts` の describe「1 品の入替 / 削除 / 詳細表示」の
      「詳細を開くと S02 を経由せずに S01 に戻れる」を実装する (既存の `test.skip` を置き換える)
    - `createRecipeWithIngredients` と `assignRecipe` で今日の夕食に「肉じゃが」を割り当て、S01 を開く
    - `openItemMenu(page, "肉じゃが")` のあと `page.getByRole("menuitem", { name: 「この品のレシピ詳細を見る」 })` を押すと `dialog` に「肉じゃが」が見える
    - Esc で閉じると `dialog` が消え、献立カレンダーの見出しと「肉じゃが」のセルが見えている
    - 完了判定: `cd frontend && npx playwright test mealplan-assign` でこのテストが pass する
- [x] 4.11 シナリオ名の対応を検査する。完了判定: `cd frontend && npm run check:scenarios` がエラー無しで終わり、
      recipe の E2E 件数が 4.2〜4.9 の増減どおりになっている

## 5. ドキュメント

- [x] 5.1 `docs/product.md` §4 S03 を更新する
    - 手順とメモを Markdown として描画し、画像は代替テキスト、リンクは遷移しない文字列にする旨を追記する
    - Esc キーでも閉じる旨を追記する
    - 「削除」は `add-recipe-deletion` で追加する旨を注記する
    - 完了判定: 目視で §4 S03 に 3 点が入っていることを確認
- [x] 5.2 `docs/reviews/2026-09-25-design-ux-review.md` の R5 (「この品のレシピ詳細を見る」の行き止まり部分) と R11 に、
      本 proposal で対応した旨と design.md D2 / D8 への参照を追記する。R5 の「並び替え」は対象外のため、R5 のチェックは付けない。
      完了判定: 目視で R11 が「対応済み」にチェックされ、R5 に部分対応の注記がある

## 6. 検証

- [x] 6.1 Go のテストを実行する。完了判定: `go test ./internal/... ./cmd/...` が全 pass する
- [x] 6.2 フロントの型チェックを実行する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.3 フロントのビルドを実行する。完了判定: `cd frontend && npm run build:dev` が通る
- [x] 6.4 E2E を実行する。完了判定: `task test:e2e` が全 pass する (`check:scenarios` を含む。skip は本 proposal 対象外のもののみ)
- [x] 6.5 lint を実行する。完了判定: `task lint` が通る
- [x] 6.6 `openspec validate add-recipe-details` を実行する。完了判定: エラーが出ない
- [x] 6.7 `task build:win` で Windows ビルドを作り、日本語の Markdown を含むレシピを S04 で入力して保存し、S03 で確認する。
      完了判定: IME で入力した `## 下ごしらえ` / `- 玉ねぎを切る` が見出しと箇条書きで描画されることを Windows 実機で目視
