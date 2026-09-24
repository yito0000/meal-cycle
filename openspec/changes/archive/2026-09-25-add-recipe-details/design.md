## Context

動機と範囲は proposal.md の Why / What Changes を参照。ここでは実装に効く現状だけを書く。

- 詳細取得の API はすでにある。`RecipeService.GetByID` は材料 / 手順 / メモ込みの `Recipe` を返し、該当なしは
  `ErrRecipeNotFound` になる。S04 編集画面が `["recipes", "detail", id]` の query key で使っている。
  このため本 proposal に Go 側の変更は無い
- S02 (`routes/recipes.tsx`) は検索語と並び順をコンポーネントの `useState` で持ち、保存直後の強調対象だけを
  search param `highlight` で受け取る。強調は 2 秒後に `router.navigate({ to: "/recipes", search: {} })` で消している
- S01 の品行メニュー「この品のレシピ詳細を見る」は `week-calendar.tsx` の `handleEntryAction` で `view-detail` として届き、
  現状はエラー表示に落としている
- ダイアログは `components/ui/dialog.tsx` (Radix Dialog) を使っている。Sheet プリミティブはまだ無い
- Markdown の描画ライブラリは入っていない。アプリは完全ローカルで、webview から外部へ通信しないことが前提である
- ワイヤー `S03_recipe_detail.drawio.svg` は「背後に S02、ドロワー分だけ暗く」「基本情報 1 行 (タグ / 人数 / 最終更新)」
  「材料 → 手順 → メモ」「下部に編集 / 削除」の構成を示している

## Goals / Non-Goals

**Goals:**

- S02 と S01 の両方から同じドロワー部品を開けるようにし、開閉の状態の持ち方だけを呼び出し側に任せる
- Markdown を描画しても、webview が外部 URL を読み込んだり遷移したりしない状態を保つ
- S04 からキャンセルで戻ったときに、同じレシピのドロワーが開いた状態を再現する

**Non-Goals:**

- ドロワーを開いたまま別の行を押して中身を切り替える操作の最適化 (背景クリックで閉じる仕様なので、行は押せない)
- S01 のドロワー状態を URL (search param) に載せること。S01 は閉じたら元に戻るだけで、外から復元する導線が無い

## Decisions

### D1. ドロワーの部品

**選択肢:**

- (a) shadcn の Sheet を `components/ui/sheet.tsx` に手動コピーする ← 採用 — Pros: 依存済みの `@radix-ui/react-dialog` の上に
  乗るので追加依存が無い。背景クリック / Esc / フォーカストラップ / `role="dialog"` が Radix 側で揃う。
  shadcn 手動コピー運用と一致する / Cons: shadcn の Sheet は `side` バリアントを持ち、右以外は使わない
- (b) `div` と `position: fixed` で自前実装する — Pros: 依存も抽象も最小 / Cons: フォーカス管理、Esc、スクロールロック、
  背景クリック判定を自前で書くことになり、既存ダイアログと挙動がずれる
- (c) `vaul` などドロワー専用ライブラリ — Pros: スワイプなどの操作が付く / Cons: デスクトップでは不要な機能のために依存が増える

**採用: (a)**。理由: spec の開閉 3 操作 (`×` / 背景 / Esc) が Radix Dialog の既定挙動そのものであり、自前実装する理由が無い。
`side` バリアントは右だけ残し、使わないバリアントは削る。幅は `w-[45vw] min-w-[28rem] max-w-[48rem]` とし、
product.md §4 の「画面の 40〜50%」に収める。最小幅は Wails の最小ウィンドウでも材料行が折り返しすぎない値にする。

### D2. ドロワーの開閉状態の持ち方

**選択肢:**

- (a) S02 は search param `detail`、S01 はローカル state ← 採用 — Pros: S04 のキャンセルで `/recipes?detail=<id>` へ戻れば
  ドロワーが開いた状態を再現できる。S01 は URL に余計な項目を増やさない / Cons: 呼び出し元で持ち方が 2 通りになる
- (b) 両方ローカル state — Pros: 一貫している / Cons: S04 から「S03 を開いた S02」へ戻す手段が無くなる。
  navigate の state で渡す方法もあるが、memory history 上で state を読む処理が別に要る
- (c) S03 を独立ルート (`/recipes/$recipeId`) にし、S02 をその親レイアウトにする — Pros: URL と画面が 1 対 1 になる /
  Cons: S01 の上に開く要件と両立しない (S01 を親にした別ルートが要る)。ルート構成の変更量が大きい

**採用: (a)**。理由: 戻り先の再現が必要なのは S02 だけで、S01 は閉じたら元に戻るだけで足りる。
ドロワー部品 `RecipeDetailDrawer` は `recipeId: string | null` / `onClose` / `onEdit` を props に取り、
状態を自分で持たない。S02 は `detail` を読み書きし、S01 は `useState<string | null>` を持つ。

S02 の検索語と並び順は `useState` のままでよい。`/recipes` から `/recipes?detail=x` への navigate は同じルート内の
search 変更なので `RecipesPage` は再マウントされず、ドロワーを閉じても検索語と並び順が残る。

### D3. 強調の自動解除とドロワーの干渉

既存の強調解除は `search: {}` で search を丸ごと空にしている。このままだと、保存直後の 2 秒以内に行を押して
ドロワーを開くと、タイマーで `detail` まで消えてドロワーが閉じる。
解除処理を `search: (prev) => ({ ...prev, highlight: undefined })` に変え、`highlight` だけを落とす。
他案として「ドロワーを開いたら強調タイマーを止める」も考えたが、強調とドロワーは独立した状態なので、
片方の解除がもう片方に触れない形のほうが後で項目を足しても壊れにくい。

### D4. S04 からの戻り先

- 保存：従来どおり `/recipes?highlight=<id>`。ドロワーは開かない (spec「S03 から S04 編集画面への遷移」)
- キャンセル (未保存変更の破棄を含む)：`/recipes?detail=<id>` に変える
- 読み込み失敗時の「一覧に戻る」：`/recipes` のまま (レシピが無い可能性が高く、ドロワーを開いてもエラーになる)

編集モードの S04 に入る導線は、本 proposal 以降 S03 の「編集」ボタンだけになる。
このため「編集モードのキャンセルは常に S03 へ戻す」と決め打ちでき、呼び出し元を search param で持ち回る必要は無い。
S01 から開いたドロワー経由で S04 に入った場合も S02 側に戻るが、これは proposal の Non-goals に書いたとおり許容する。
他案の「`from` search param で S01 / S02 を持ち回る」は、S01 の表示期間まで復元しないと戻った意味が薄く、
その復元は本 proposal の範囲を超えるため採らない。

### D5. 詳細取得の query

S03 と S04 で同じ `["recipes", "detail", id]` を使う。取得処理を `components/recipe/use-recipe-detail.ts` の
`useRecipeDetail(id: string | null)` にまとめ、`id` が null の間は `enabled: false` にする。
S04 の保存時は既存どおり `["recipes", "detail", id]` を invalidate しているので、保存後に S03 を開き直すと新しい値が出る。
他案として、S02 の一覧データ (`List`) から詳細を引く方法も考えた。
しかし `List` は材料 / 手順 / メモを返さない projection なので使えない。

ドロワーのアニメーション中に `recipeId` が null に戻ると中身が空になってちらつく。
閉じ始めた時点の最後の `recipeId` を部品内で保持し、閉じ切るまで同じ中身を描く。

### D6. Markdown の描画

**選択肢:**

- (a) `react-markdown` を追加し、`components` prop で `img` と `a` を差し替える ← 採用 — Pros: 既定で生 HTML を
  解釈しない (rehype-raw を入れない限り HTML はテキストになる) ので、`dangerouslySetInnerHTML` を使わずに済む。
  要素ごとの差し替えが React コンポーネントで書ける / Cons: 依存が増える (micromark / unified 系の推移的依存を含む)
- (b) `marked` + `DOMPurify` で HTML 文字列を作り `dangerouslySetInnerHTML` で差し込む — Pros: 軽い /
  Cons: サニタイズを自前で正しく設定する責任を負う。画像とリンクの差し替えも HTML 文字列の後処理になる
- (c) 描画せず `whitespace-pre-wrap` でテキストのまま表示する — Pros: 依存ゼロ / Cons: spec の Markdown 描画要件を満たさない

**採用: (a)**。理由: 外部通信の禁止と HTML の無害化を、ライブラリの既定挙動と 2 要素の差し替えだけで満たせる。
差し替えは次のとおり。

- `img`：`<span>` に `alt` のテキストだけを出す。`src` は描画しないので webview は画像を取りに行かない
- `a`：`<span>` に子要素だけを出す。`href` を持たないのでクリックしても遷移しない

GFM (`remark-gfm`) は入れない。表やチェックボックスは MVP の手順 / メモで使う場面が薄く、依存をさらに増やす理由にならない。
Tailwind の typography プラグインも入れず、見出し / リスト / コード / 引用に最小限のクラスを `components` で当てる。
`index.css` に `prose` 相当の汎用クラスを増やさないのは、Markdown を描く場所が S03 の 2 欄だけだからである。

### D7. 表示の整形

- 更新日：`updatedAt` (RFC3339 UTC) をローカル日付にし、`lib/format-date.ts` の `formatDisplayDate` で `yyyy/mm/dd` にして出す。
  ワイヤーが日付だけを示しており、表示用の日付表記は S07 と揃えるため。
  S02 の更新日列 (RFC3339 をそのまま表示) は本 proposal では変えない
- 人数：`pages.recipes.detail.servings` (`{{count}} 人分`) で出す
- 材料の量：`lib/format-amount.ts` の `formatAmount` を使う。量が 0 のときは量を出さず「名前 単位」にする
- タグ：S02 と同じく ` / ` 区切りで 1 行に出す。0 件ならタグの表示自体を省く (基本情報の行は崩さない)
- 空の欄：`pages.recipes.detail.empty` (「未登録」) を薄い文字で出す

### D8. S01 との結線

`week-calendar.tsx` に `const [detailRecipeId, setDetailRecipeId] = useState<string | null>(null)` を置き、
`view-detail` で `entry.recipeId` を入れる。ドロワーの `onEdit` は `/recipes/$recipeId/edit` へ navigate する。
S01 の表示期間は `meal-plan.tsx` の state と localStorage 設定に載っており、ドロワーの開閉では触らないので保たれる。

### D9. E2E

E2E のテスト名は spec.md に揃える (ファイル名の先頭語 = `recipe` / `mealplan`、最外の describe = Requirement 名、
test 名 = Scenario 名)。どのシナリオを E2E で確かめるかは tasks.md のグループ 4 で列挙する。

**ファイルの割り振り:**

- (a) S03 のシナリオを新規 `recipe-detail.spec.ts` に集め、S02 / S04 側の導線は既存ファイルに置く ← 採用 —
  Pros: 既存ファイルの describe (`S02 一覧から S03 を開く` は `recipe-listing`、`S03 から S04 編集画面への遷移` は `recipe-edit`) を
  そのまま活かせる。ドロワー単体の表示と Markdown のシナリオが 1 ファイルで読める / Cons: S03 関連が 3 ファイルに分かれる
- (b) S03 に関わるシナリオをすべて `recipe-detail.spec.ts` に集める — Pros: S03 だけを回しやすい /
  Cons: 既存の `test.skip` の置き場所を動かす差分が増え、S04 の保存 / キャンセルの手順ヘルパも重複する

**採用: (a)**。`recipe-listing.spec.ts` にある「S03 の開閉」の `test.skip` は `recipe-detail.spec.ts` へ移す
(同じ Scenario のテストが 2 つあると `check:scenarios` が落ちる)。

**E2E に含めないシナリオ:**

- 「レシピを読み込めない場合はエラーを表示する」：UI から存在しないレシピのドロワーを開く手段が無い。
  ドロワーを開いたまま API でレシピを消す手もあるが、削除 API は `add-recipe-deletion` まで存在しない。表示は `task dev` の目視で確かめる
- recipe の「S01 から開いて閉じると S01 に留まる」：mealplan の「詳細を開くと S02 を経由せずに S01 に戻れる」と同じ操作と観察になる。
  mealplan 側に `test.skip` がすでにあるので、そちらを実装し、recipe 側は重ねない

**その他:**

- 暫定要件「S02 一覧からのレシピ編集導線 (暫定)」の E2E は削除する。行を押すと S04 に入る前提が崩れ、残すと落ちるため。
  REMOVED の要件は archive まで `openspec/specs/` に残るので、`check:scenarios` はテストの有無どちらでも通る
- ドロワーの特定は `getByRole("dialog")` で行う。Radix の Sheet は `role="dialog"` を付ける
- S01 の品行メニュー (Radix DropdownMenu) を E2E で操作するのは初めてになる。`mealplan-batch.spec.ts` などが
  「Portal の操作が扱いづらい」として skip している。トリガをクリックしたあと、Portal 内の項目を `page.getByRole("menuitem")` で
  ページ全体から探せば届く (Portal は `body` 直下に出るが、`page` 起点のロケータなら親子関係に縛られない)。
  これを `helpers.ts` の `openItemMenu` にまとめ、既存の skip を後で有効化するときにも使えるようにする
- Markdown の描画は E2E で確かめる。前提のレシピは API で作るので、IME を経由せずに `## 見出し` などの文字列を入れられる。
  日本語の入力 (S04 で `##` を打つ操作) は Windows 実機で確かめる

## Risks / Trade-offs

- [Risk] `react-markdown` の推移的依存でバンドルが 100KB 前後増える → デスクトップ配布の単一バイナリなので
  ダウンロード時間の問題にならない。起動時間への影響は `task build` 後に体感で確認し、問題があれば S03 を lazy import にする
- [Risk] 将来 `rehype-raw` などを足すと、HTML が素通しになり外部画像の読み込みが復活しうる →
  `recipe-markdown.tsx` に「生 HTML を有効にしない」理由をコメントで残す
- [Risk] S01 から開いたドロワーで「編集」を押すと、保存 / キャンセル後に S01 へ戻らない → proposal の Non-goals で明示済み。
  使ってみて不満が出たら、S01 の表示期間を search param に載せる proposal と一緒に扱う
- [Risk] S04 キャンセルで S03 に戻るとき、S02 は再マウントされて検索語と並び順が初期値に戻る → spec が保証するのは
  ドロワーを閉じたときの保持だけである。S04 往復での保持が必要になったら、検索語と並び順を search param に移す
- [Risk] 保存直後に行を押すと強調解除がドロワーを閉じる → D3 で `highlight` だけを落とすようにして防ぐ
- [Trade-off] 「削除」ボタンをワイヤーどおりに置かない → `add-recipe-deletion` までの間、ワイヤーとの差分になる。
  ワイヤーは最終形として残し、差分は proposal の Non-goals と product.md §4 の注記で示す

## Migration Plan

データ移行は無い。フロントのみの変更なので、ロールバックは該当コミットの revert で済む。
