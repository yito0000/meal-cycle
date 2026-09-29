## Why

S03 レシピ詳細は、S02 または S01 の右側に重ねるドロワーで、幅は画面の 40〜50% である。
`add-recipe-details` で手順とメモを Markdown で書けるようになり、S03 は材料の一覧を確かめるだけでなく、手順を読みながら調理する画面になった。
手順が長いレシピでは、ドロワーの幅だと 1 行が短く折り返され、材料と手順を行き来するたびにスクロールすることになる。

ドロワーを選んだのは、一覧を左に残したまま中身を覗けるからだった (product.md §3)。
だが S03 を開いたあとに一覧を操作することは無く、開いている間は背景が暗くなって一覧は触れない。
左に残る一覧は、戻り先を示す以上の役に立っていない。
S04 レシピ編集はすでに全画面の独立した画面なので、S03 も同じ形にそろえ、戻り先はパンくずで示す。

## What Changes

- S03 をドロワーから、メインエリア全体に表示する独立した画面に変える。サイドバーはそのまま残る
  - 幅の広い画面では、材料を左、手順とメモを右に並べる。狭い画面では縦に積む
  - 「編集」と「削除」のボタンは、下端ではなく基本情報の横 (画面上部) に置く
- S03 の上部にパンくずを置き、どこから開いたかを示す
  - S02 から開いたとき：`レシピ > <レシピ名>`
  - S01 から開いたとき：`献立 > <レシピ名>`
  - 先頭の項目を押すと、開いた元の画面に戻る。Esc キーでも同じ画面に戻る
- 開いた元の画面に戻ったとき、開く前の状態を保つ
  - S02：検索語と並び順
  - S01：表示期間の開始日 (期間長と時間帯表示は、これまでどおり localStorage で保たれる)
- S04 編集画面の上部にもパンくずを置く：`<開いた元> > <レシピ名> > 編集`。新規登録では `レシピ > 新しいレシピ`
  - パンくずの項目を押したときも、キャンセルと同じく未保存の変更を確かめる
- S04 でキャンセルすると、S03 の画面に戻る。パンくずの開いた元は、S04 に入る前と同じまま引き継ぐ
- S04 で保存すると、これまでどおり S02 に戻って行を強調する (開いた元にかかわらない)
- S03 で削除すると、開いた元の画面に戻り、削除した旨をトーストで知らせる
- **BREAKING (振る舞い)**：S03 を背景のクリックと `×` ボタンで閉じる操作は無くなる。戻る操作はパンくずと Esc キーになる

### Non-goals

- **スクロール位置の復元**：S02 と S01 に戻ったとき、スクロール位置は先頭に戻る。
  S01 は最長 4 週でスクロールが浅く、S02 は検索語が残るので目的の行に戻りやすい。不満が出たら後続で検討する
- **ブラウザ型の戻る / 進む**：履歴をさかのぼる操作は持たない。戻り先は、パンくずが示す開いた元に固定する
- **S07 など他の画面へのパンくずの導入**：サイドバーの直下にある画面にはパンくずを置かない
- **S04 のレイアウト変更**：パンくずを足すだけで、入力欄の配置は変えない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：S03 をドロワーから独立した画面に変え、パンくずと Esc キーで開いた元に戻す。S02 の検索語と並び順を戻ったときに保つ。
  S04 にパンくずを置き、キャンセルの戻り先を S03 の画面にする。削除後は開いた元の画面に戻る。
  ドロワーを前提にした名前のシナリオを消すため、次の要件を REMOVED にして新しい名前で ADDED する (design.md D7)。
  - 「S03 レシピ詳細ドロワーの表示」→「S03 レシピ詳細画面の表示」
  - 「S03 の開閉」→「S03 から開いた元の画面へ戻る操作」
  - 「S02 一覧から S03 を開く」→「S02 一覧から S03 詳細画面へ移る」
  - 「S01 献立カレンダーから S03 を開く」→「S01 献立カレンダーから S03 詳細画面へ移る」
  - 「レシピの削除」→「S03 詳細画面からのレシピの削除」
  S04 のパンくずは、新しい要件「S04 のパンくず」として ADDED する。
  「S03 から S04 編集画面への遷移」と「未保存変更のキャンセル確認」は、戻り先の文言を MODIFIED で改める。
- `mealplan`：「1 品の入替 / 削除 / 詳細表示」の詳細表示を、ドロワーではなく S03 の画面への移動に改める。
  「S01 表示設定の永続化」に、S03 から戻ったときは表示期間の開始日を保つ例外を書き足す。

## Impact

### 新設ファイル

- `frontend/src/routes/recipes/$recipeId.tsx`：S03 レシピ詳細画面
- `frontend/src/components/recipe/recipe-detail.tsx`：S03 の本文 (基本情報、材料、手順、メモ、編集と削除のボタン)。ドロワーから移す
- `frontend/src/components/layout/breadcrumb.tsx`：パンくず
- `frontend/src/components/recipe/detail-origin.ts`：開いた元 (S02 / S01 とその状態) の型、zod スキーマ、戻り先への変換
- `frontend/e2e/recipe-detail-navigation.spec.ts`：パンくずと Esc キーで戻る E2E
- `docs/wireframes/S03_recipe_detail.v2.drawio.svg`：全画面にした S03 のワイヤー

### 更新ファイル

- `frontend/src/router.tsx`：`/recipes/$recipeId` を追加。`/` に `start`、`/recipes` に `q` と `sort` の search を追加。
  `/recipes` の `detail` を削除。`/recipes/$recipeId` と `/recipes/$recipeId/edit` に開いた元の search を追加
- `frontend/src/routes/recipes.tsx`：検索語と並び順を URL の search で持つ。行を押したら S03 の画面へ移る。ドロワーを外す
- `frontend/src/routes/meal-plan.tsx`：表示期間の開始日を URL の search で持つ
- `frontend/src/components/mealplan/week-calendar.tsx`：「この品のレシピ詳細を見る」で S03 の画面へ移る。ドロワーを外す
- `frontend/src/routes/recipes/$recipeId.edit.tsx`：パンくずを置く。キャンセルで S03 の画面に戻る
- `frontend/src/routes/recipes/new.tsx`：パンくずを置く
- `frontend/src/components/recipe/recipe-edit-form.tsx`：パンくずからの移動にも未保存の確認をかける
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：パンくずの文言を追加、`pages.recipes.detail.close` を削除
- `frontend/e2e/recipe-detail.spec.ts`：describe「S03 レシピ詳細ドロワーの表示」「S03 の開閉」を新しい要件名に改める
- `frontend/e2e/recipe-listing.spec.ts`：describe「S02 一覧から S03 を開く」を新しい要件名に改める
- `frontend/e2e/recipe-edit.spec.ts`：S04 のキャンセルと破棄の戻り先を S03 の画面に改める
- `frontend/e2e/recipe-delete.spec.ts`：describe「レシピの削除」を新しい要件名に改め、戻り先を確かめ直す
- `frontend/e2e/helpers.ts`：S03 を開いて待つ補助関数をドロワー前提から画面前提に改める
- `docs/product.md`：§3 の画面遷移図と画面構成の説明、§4 S02 と S03 と S04、決定の記録の S03 を更新
- `docs/wireframes/README.md`：ファイル一覧に `S03_recipe_detail.v2.drawio.svg` を追加
- `openspec/specs/recipe/spec.md` / `openspec/specs/mealplan/spec.md`：archive 時に delta を合成 (手で直接は触らない)

### 削除ファイル

- `frontend/src/components/recipe/recipe-detail-drawer.tsx`：本文は `recipe-detail.tsx` に移す
- `frontend/src/components/ui/sheet.tsx`：使うのは S03 のドロワーだけなので削除する

### 生成物 (再生成のみ)

- なし。Go 側とバインディングは変えない

### migration

- なし

### 前提と順序

- `add-shopping-pantry-section` とはファイルが重ならないので、どちらを先に archive してもよい

### 後続 proposal への引き継ぎ

- S02 と S01 のスクロール位置の復元 (Non-goals)
- `openspec/specs/recipe/spec.md` の Purpose は「S03 詳細ドロワー」と書いている。delta では Purpose を変えられないので、archive 時に手で直す (tasks.md)
