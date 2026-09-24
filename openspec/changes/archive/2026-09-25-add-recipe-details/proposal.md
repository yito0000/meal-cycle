## Why

レシピの中身を確かめたいだけの場面でも、今は S02 の行を押すと S04 編集画面に入ってしまう。
add-recipe-edit で置いたこの導線は、S03 レシピ詳細ドロワーが無い期間の暫定として spec に明記してある。
読むだけのつもりで編集画面に入ると、うっかり入力欄を触ったときに未保存確認が挟まり、一覧へ戻るのに一手間かかる。

S01 献立カレンダーの品行メニュー「この品のレシピ詳細を見る」も、押すとエラー表示が出るだけのプレースホルダのまま残っている。
献立を組みながら「この料理は材料に何を使うのだったか」を確かめたい場面は多い。
ここで S02 へ移ってしまうと、カレンダーのどこを組んでいたかを見失う。

手順とメモは Markdown 記法で書けるが、読み手側での描画は「S03 で追加する」と後回しにしてある。
S04 の textarea で `## 下ごしらえ` がそのまま見えている状態は、書き手には問題なくても、読むには向かない。

本 proposal では S03 をドロワーとして実装し、S02 と S01 の両方から開けるようにする。
design.md §8 の順序案どおり、削除は後続の `add-recipe-deletion` に分ける。

## What Changes

- S03 レシピ詳細ドロワーを追加する。画面右から、幅は画面の 40〜50% で重ねて開く。背後の画面は暗く見せる。
  - 表示項目は、基本情報 (名前 / タグ / 人数 / 更新日)、材料、手順、メモ、「編集」ボタン
  - 手順とメモは Markdown として描画する (見出し / 箇条書き / 番号付きリスト / 強調 / コード)。
    外部への通信を起こさないよう、画像記法は代替テキストだけを出し、リンクは遷移しない文字列として出す
  - 材料、手順、メモが空のときは、その欄に「未登録」の旨を出す
  - 閉じる操作は `×` ボタン、背景のクリック、Esc キーの 3 つ
  - 「編集」を押すと S04 編集画面に遷移する
- S02 レシピ一覧の行を押したときの遷移先を、S04 編集画面から S03 詳細ドロワーに差し替える。
  **BREAKING** (画面導線)：行を押しても直接 S04 には入らなくなる。
  - ドロワーを閉じると、S02 の検索語と並び順を保ったまま一覧に戻る
- S04 編集画面の「キャンセル」の戻り先を、S02 一覧から、そのレシピの S03 詳細ドロワーを開いた S02 に変える。
  保存後は従来どおり S02 に戻り、保存したレシピの行を強調する。新規登録モードの戻り先は変えない。
- S01 献立カレンダーの品行メニュー「この品のレシピ詳細を見る」で、S01 の上に S03 ドロワーを開く。閉じると S01 に留まる。
- 暫定要件「S02 一覧からのレシピ編集導線 (暫定)」を確かめていた E2E を削除し、S03 経由の導線を確かめる E2E に置き換える。
- Go 側の変更は無い。既存の `RecipeService.GetByID` で詳細を取得する。

### Non-goals

- **レシピの削除**：ドロワーに「削除」ボタンは置かない。削除確認と、献立の割当が連鎖して消える件の扱いは
  `add-recipe-deletion` で決める。
- **S04 保存後に S03 へ戻す挙動**：product.md §4 S04 は「呼び出し元 (S02 or S03) に戻る」と書くが、
  既存の「レシピの編集」要件は保存後に S02 へ戻ることを求めている。本 proposal では保存後の戻り先を S02 のまま据え置く。
- **S01 から開いたドロワーの編集後に S01 へ戻す挙動**：S01 から「編集」に進んだ場合も、保存後とキャンセル後は S02 側に戻る。
- **S04 での Markdown プレビュー**：書き手側はプレーンテキスト入力のまま。
- **Markdown の表 / チェックボックス / 取り消し線** (GFM 拡張)：MVP では標準の Markdown 記法だけを描画する。
- **ドロワー内での前後レシピ送り**：一覧の次 / 前のレシピへ切り替える操作は付けない。
- **ドロワー幅の設定化**：幅はレイアウト上の定数とし、`internal/config/defaults.go` には置かない (S09 の可変対象ではない)。
- **Frontend テスト runner の導入**：Vitest は未導入のまま。フロントは type-check / build:dev / E2E で検証する。

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：S03 レシピ詳細ドロワーの表示と開閉、S02 行押下の遷移先の差し替え (暫定要件の削除)、
  S03 から S04 への編集導線とキャンセル時の戻り先、手順とメモの Markdown 描画を追加する。

mealplan の「1 品の入替 / 削除 / 詳細表示」要件は、S03 を開いて閉じたら S01 に戻ることをすでに要求している。
本 proposal はその未実装部分を満たすだけで、mealplan の要件自体は変えない。

## Impact

### 新設ファイル

- `frontend/src/components/ui/sheet.tsx`：shadcn の Sheet プリミティブ (Radix Dialog ベース) を手動コピー
- `frontend/src/components/recipe/recipe-detail-drawer.tsx`：S03 本体。`recipeId` と `onClose` / `onEdit` を受け取る
- `frontend/src/components/recipe/recipe-markdown.tsx`：手順とメモの Markdown 描画。画像とリンクの差し替えを含む
- `frontend/src/components/recipe/use-recipe-detail.ts`：`["recipes", "detail", id]` の query フック (S03 と S04 で共有)
- `frontend/e2e/recipe-detail.spec.ts`：S03 の表示 / 開閉 / Markdown 描画の E2E (対象シナリオは tasks.md で列挙)

### 更新ファイル

- `frontend/package.json` / `frontend/package-lock.json`：`react-markdown` を dependencies に追加
- `frontend/src/router.tsx`：`recipesSearchSchema` に `detail` (開いているレシピ ID) を追加
- `frontend/src/routes/recipes.tsx`：行押下で `detail` を立ててドロワーを開く。強調の自動解除で `detail` を落とさない
- `frontend/src/routes/recipes/$recipeId.edit.tsx`：キャンセルの戻り先を `/recipes?detail=<id>` に変更。
  詳細取得を `use-recipe-detail.ts` に置き換え
- `frontend/src/components/mealplan/week-calendar.tsx`：`view-detail` でドロワーを開く。プレースホルダのエラー表示を削除
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.recipes.detail.*` を追加
- `frontend/e2e/helpers.ts`：`createRecipeDetailed` (タグ / 人数 / 材料 / 手順 / メモを指定して作り ID を返す)、
  `openRecipeDetail` (一覧の行からドロワーを開く)、`openItemMenu` (S01 の品行メニューを開く) を追加
- `frontend/e2e/recipe-listing.spec.ts`：「S02 一覧から S03 を開く」の 2 シナリオを実装。「S03 の開閉」の `test.skip` を削除
  (`recipe-detail.spec.ts` へ移す)
- `frontend/e2e/recipe-edit.spec.ts`：「S02 一覧からのレシピ編集導線 (暫定)」の describe を削除。
  「S03 から S04 編集画面への遷移」の 4 シナリオと、「未保存変更のキャンセル確認」の編集モードのシナリオを実装
- `frontend/e2e/mealplan-assign.spec.ts`：「詳細を開くと S02 を経由せずに S01 に戻れる」を実装
- `docs/product.md`：§4 S03 に Markdown 描画と外部通信を起こさない描画規則を追記し、削除は後続 proposal である旨を添える
- `docs/reviews/2026-09-25-design-ux-review.md`：R11 を対応済みにし、R5 に「詳細を見る」の部分対応を注記する

### 生成物 (再生成のみ)

- なし。Go サービスの公開メソッドは変わらない

### migration

- なし

### 後続 proposal への引き継ぎ

- `add-recipe-deletion`：ドロワーのフッタ (「編集」の隣) に「削除」を足す。削除後はドロワーを閉じ、
  `["recipes", "list"]` と `["recipes", "detail", id]` を無効化する。S01 から開いた場合は献立の query も無効化が要る
- S04 保存後の戻り先を S03 にするかは、利用してみてから判断する。変えるなら `$recipeId.edit.tsx` の `onSave` の遷移先だけで済む
- S10 (タグ / 材料での絞り込み)：ドロワーのタグを押して絞り込む導線を足すなら、S02 の search に項目を足す形になる
