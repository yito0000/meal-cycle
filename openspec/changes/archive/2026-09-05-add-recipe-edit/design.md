## Context

前 proposal `add-recipe-listing` は S02 レシピ一覧を「Wails サービスが固定値を返す + UI が実装される」形で通し、SQLite 永続化と S04 (登録/編集) は
本 proposal に持ち越された。同時期に `add-mealplan-week-view` も同じ「固定値先行」パターンで通っている。

本 proposal は「登録は保存されて初めて機能」という UX 制約から、fixture の後回しを終わらせて SQLite 実データに差し替える必要がある。
材料は初回案では「食材マスタ + Combobox」で扱う設計だったが、UX レビューの結果、本 proposal では **食材マスタを持たない自由テキスト入力** に
方針転換した。手順とメモは単一 `textarea` で Markdown 記法として入力し、レンダリングは S03 詳細ドロワー proposal に持ち越す。

制約:

- SQLite 基盤 (`Open` / `Migrate`) は既に走っている ([internal/storage/sqlite/db.go](../../../internal/storage/sqlite/db.go))。
  goose のマイグレーションランナーも `00001_init.sql` 分だけ実績がある
- Wails v3 beta.12 の JSDoc バインディング生成は `Recipe` struct 変更に追随できる (生成物は git 管理外)。TS モードで生成した場合、
  モデルは class ではなく `interface` として export されるので、フロントで `new Recipe(...)` は使えない (リテラルを型注釈で組む)
- UI 文字列は i18n key 経由必須。Go 側は型付き値 / エラーコードで返す
- 実装/テスト方針: サービス層は BDD 風、ドメイン層は table-driven、SQLite repo は `t.TempDir()` の実 DB
  ([CLAUDE.md](../../../CLAUDE.md) テスト方針セクション)

詳細は [proposal.md](proposal.md) を参照。

## Goals / Non-Goals

**Goals:**

- レシピの Create / Update / GetByID / List を SQLite 実装で通しに動かし、`RecipeService` の fixture を撤去する
- S04 で自由テキスト入力の材料行 (名前 / 量 / 単位) を保存できるようにする (食材マスタは持たない)
- S02 → S04 の遷移導線 2 種類 (「＋新しいレシピ」ボタン / 行クリック) を有効化する。行クリック導線は S03 完了時に置換予定であることを design 上で明示する
- S04 の未保存キャンセル確認と保存後ハイライトを含む UX 要件を、`add-recipe-detail-drawer` / `add-recipe-deletion` に持ち越さず本 proposal で完結させる
- 入力バリデーション (レシピ名 1〜200 字、人数 1〜99、材料名必須、手順 4000 字、メモ 2000 字、タグ 30 字) をフロントと Go 側の両方で担保する
- ワイヤー ([S04_recipe_edit.drawio.svg](../../../docs/wireframes/S04_recipe_edit.drawio.svg)) のレイアウトと操作を素直に落とす
  (ただし手順は書き手 UI では単一 textarea、読み手 UI (S03) で番号付き表示を再現する)

**Non-Goals:**

- S03 詳細ドロワー / 削除機能 (別 proposal)
- **食材マスタ (`ingredient` capability)**：本 proposal では立てない。冷蔵庫在庫との名寄せが必要になった段階で
  別 proposal `add-ingredient-master` を立てる (名寄せ辞書 / autocomplete / 単位変換もそこで扱う)
- **手順 / メモの Markdown レンダリング**：書き手側は素の textarea のみ。S03 で `react-markdown` などを検討する
- 献立との参照整合 (MealPlan.RecipeName → RecipeID 切替は `add-mealplan-recipe-reference` の担当)
- 素材 / タグ / 手順のドラッグ並び替え (MVP は上下ボタン or 単一テキストで足りる)
- 既存タグからの候補提示 (別 proposal `add-recipe-tags-autocomplete`)

## Decisions

### D1. `recipe` の Domain 型: 単一 `Recipe` 型を拡張 vs 一覧 projection 用 `RecipeSummary` を分離

前 proposal `add-recipe-listing` D1 では「一覧に必要な最小フィールドで単一 `Recipe` 型」を採用済み。本 proposal はそこに詳細フィールド
(`Servings` / `Ingredients` / `Steps` / `Notes`) を足す時点。

**選択肢:**

- (a) 単一 `Recipe` 型に詳細フィールドを追加し、一覧 List も同じ struct を返す (詳細だけ Repository 側で n+1 を避ける実装) ← 採用
  - Pros：型が 1 個。JSDoc バインディングも 1 個。前 proposal D1 の判断を継続。Create / Update / GetByID / List の返り値型が揃う
  - Cons：一覧で `Ingredients` `Steps` `Notes` を毎回引くと重い可能性。ペルソナ規模 20〜50 品では問題にならない
- (b) `RecipeSummary` (一覧用、既存フィールドのみ) と `Recipe` (詳細用、全フィールド) の 2 型に分割
  - Pros：一覧クエリで JOIN しなくて済む。フロントも「一覧に手順は要らない」ことが型で分かる
  - Cons：バインディング 2 型。CreateInput / UpdateInput / GetByID / List で返す型が混在する。前 proposal と方針が変わる
- (c) `Recipe` に詳細フィールドを追加しつつ、List 時は詳細フィールドを空にした Recipe を返す (実質同型で content だけ差)
  - Pros：型は 1 個
  - Cons：フロントが `recipe.Steps === ""` の意味 (「無いレシピ」なのか「一覧射影で欠けている」のか) を判別できない。ハマる

**採用: (a)**。理由: ペルソナ規模で JOIN コストが問題にならない範囲であり、型が 1 個で済むメリットが大きい。将来一覧側で
本気にパフォーマンスを詰めたくなった時点で (b) に分割する (S10 `add-recipe-search` あたりが自然な切り出し場所)。
List は「recipes 行 + recipe_ingredients の COUNT を JOIN」で `IngredientCount` を維持し、GetByID の時のみ recipe_ingredients を全部引く。

### D2. スキーマ設計: 材料を別テーブル vs recipes に JSON カラムで持つ

**選択肢:**

- (a) `recipes` + `recipe_ingredients` の 2 テーブル、`recipe_ingredients.name` に材料名を直持ちする (FK は張らない) ← 採用
  - Pros：将来「この食材を使うレシピ」検索が SQL の LIKE / = で書ける (in-app 名寄せは実行時の文字列一致で済ませる方針)。
    ON DELETE CASCADE で recipe 削除時の後片付けが宣言的。将来食材マスタを導入するときは、`name` を FK 列に置き換える in-place migration で移行できる
  - Cons：recipe 保存が 2 テーブル書き込みになる (トランザクションで囲めばよい)。名寄せは実行時マッチなので「玉ねぎ」「たまねぎ」を別扱いになる問題は残る
    (本 proposal では対象外、`add-ingredient-master` で解決する)
- (b) `recipes` に `ingredients_json TEXT` を持ち、`[{name, amount, unit}, ...]` を JSON 保存
  - Pros：レシピ 1 件の Read/Write が 1 テーブルで済む。Repository コードが短い
  - Cons：「特定食材を使うレシピ」の逆引きが JSON 探索になる。SQLite JSON 関数を使うにしても index が効かない。将来食材マスタを
    後付けするときの migration コストが跳ねる (JSON 分解が要る)
- (c) `recipes` テーブルに material カラムを CSV で持つ (量 / 単位を区切り文字で埋め込む)
  - Pros：実装最短
  - Cons：区切り文字を材料名に含められない。将来食材マスタ化で崩れる

**採用: (a)**。理由: 本 proposal では食材マスタを持たないので FK も張らないが、将来 `add-ingredient-master` proposal で
「name TEXT を ingredient_id TEXT REFERENCES ingredients(id) に置き換える」migration が素直に書ける形にしておく。JSON 化 (b) は
将来のマスタ化時に分解 migration が要るため避ける。

`tags` / `steps` / `notes` はこれと逆で **TEXT カラムで recipes に inline**。理由:

- タグ Entity 化は [product.md §2](../../../docs/product.md) S08 相当で MVP 外
- `steps` は Markdown 記法の単一テキスト (`textarea` の内容そのまま)。順序ありの配列で持たなくてよい (次節 D12 参照)
- 別テーブル化は将来「手順単位でチェックリスト化」等のニーズが出た時点で分離すれば足りる

### D3. Recipe の ID 発行: UUID v7 vs アプリ生成 vs SQLite AUTOINCREMENT

[design.md #4](../../../docs/design.md) は UUID v7 を想定と書いてある。

**選択肢:**

- (a) `google/uuid` の `NewV7()` で Go 側発行 ← 採用
  - Pros：時系列ソート可能で、後から SQLite 外にデータを引き剥がしても一意性が保てる。design.md の想定を踏襲
  - Cons：依存追加 (1 個)。ただし小さいライブラリ
- (b) `crypto/rand` で UUID v4 を Go 側発行
  - Pros：標準ライブラリで済む
  - Cons：時系列ソート不可。design.md の方針と食い違う
- (c) SQLite `INTEGER PRIMARY KEY AUTOINCREMENT`
  - Pros：依存追加なし
  - Cons：フロントに渡す ID が数値になる。将来「別 DB からインポート」等で衝突する。既存 mealplan `Entry` は文字列 ID を期待

**採用: (a)**。UUID v7 は design.md の方針と一致し、リポジトリ既存の Wails サービス群 (recipeservice の `"r-001"` fixture ID
含む) も文字列 ID を返している。`google/uuid` v1.6.0+ が v7 に対応済み。

### D4. S02 → S04 編集への行クリック導線を暫定でどう実装するか

S03 詳細ドロワー未実装のため、行クリック → S04 直行の暫定を張るが、S03 完了時に上書きされる。

**選択肢:**

- (a) `RecipeTable` に `onRowClick` prop を渡し、S02 route が `router.navigate("/recipes/:id/edit")` を実装する ← 採用
  - Pros：`RecipeTable` は責務を持たず (行クリックの動作は呼び出し元次第)、S03 実装時に S02 route が
    `navigate` を「ドロワーを開く」に差し替えるだけで済む。`RecipeTable` は無傷
  - Cons：なし
- (b) `RecipeTable` 内に navigate を持たせる
  - Pros：呼び出し元が短くなる
  - Cons：S03 完了時に `RecipeTable` を書き換える (or 挙動を条件分岐する) 必要が出る
- (c) 行内に「編集」アイコンボタンを別に置き、行クリック自体は無効のまま
  - Pros：誤クリックがない
  - Cons：ワイヤーには編集アイコン列は無い。S03 完了時に消す UI を今追加する分の diff がもったいない

**採用: (a)**。理由: 暫定導線の後始末が呼び出し元 (S02 route) 1 ファイルで完結する。`RecipeTable` は
汎用な行クリック機構を持ち続け、S03 完了時にドロワーを開くコールバックに差し替わるだけ。仕様上「行押下 → 編集」に
なるのはこの proposal 適用期間中のみで、spec の該当要件も次 proposal で MODIFIED される (proposal.md 「後続 proposal
への引き継ぎ」参照)。

### D5. 新規作成後のハイライト対象: URL search param vs Router state vs Query cache

保存後 S02 に戻ってハイライトする方法。

**選択肢:**

- (a) `router.navigate({ to: "/recipes", search: { highlight: id } })` で URL search param に載せる ← 採用
  - Pros：TanStack Router の memory history と自然に噛み合う。S02 route の props で受け取れる。ページリロード相当の操作でも
    見た目が復元できる
  - Cons：URL に一時的な UI ヒントが載る (Wails webview 内なので実害はない)
- (b) React Context / Zustand 等の store に `justSavedId` を積み、S02 route が起動時に読む
  - Pros：URL を汚さない
  - Cons：新規に store を用意する (現状 store 依存を持ち込んでいない)。router.navigate 引数だけで済む問題に対して重い
- (c) TanStack Query cache に「最後に保存された ID」を書き込んで S02 が読む
  - Pros：既に Query を使っている
  - Cons：Query cache を UI 状態のバケツとして流用するのは責務分離が崩れる

**採用: (a)**。理由: TanStack Router には `useSearch()` があり、search schema は route 側で `validateSearch` に
zod スキーマを添えれば型が付く。ハイライトは一定時間 (数秒) or 別行選択 / 検索操作でクリアする。
実装は `useEffect` で `setTimeout` の素朴パターンで足りる (setTimeout の閾値は `internal/config/defaults.go` に
`RecipeListHighlightMs int` として置く。可変値集約ルールに従うため)。

### D6. dirty 判定の実装粒度

キャンセル確認ダイアログの発火条件。

**選択肢:**

- (a) フォームの各フィールドを初期値スナップショットと比較して dirty を計算する ← 採用
  - Pros：「入力してすぐ削除して元に戻した」ケースを dirty 扱いしない (UX として自然)
  - Cons：材料配列や手順配列の deep compare が要る (json-stable-stringify or lodash isEqual)
- (b) 任意の onChange 発火で dirty フラグを立て、以降は保存 or ページ離脱まで倒さない
  - Pros：実装最短
  - Cons：「入力して元に戻した」でも確認が出る (UX 上の摩擦)
- (c) 初期値と現在値の JSON.stringify を比較する
  - Pros：実装が短い
  - Cons：プロパティ順序が保証されないケースで false positive のリスク (React state で自作 struct を積む限りは順序は安定するが、
    将来のリファクタで壊れやすい)

**採用: (a)**。理由: dirty 判定は UX の要なので false positive を避けたい。deep compare は React 標準の `useMemo` +
`fast-equals` (小さい依存、6KB 程度) で実装可能。追加依存 1 個は許容する。

### D7. S04 のフォーム管理: react-hook-form 導入 vs 自前 useState

**選択肢:**

- (a) 自前 `useState` + カスタムフック `useRecipeForm` に集約 ← 採用
  - Pros：依存追加なし。フィールド数が高々 6 種類 (name / servings / tags / ingredients[] / steps / notes) で
    react-hook-form のメリット (再レンダ最適化) が薄い
  - Cons：検証ロジック (name 必須等) を自前で書く
- (b) react-hook-form + zod schema 導入
  - Pros：検証とエラー表示が宣言的
  - Cons：依存 2 個追加。学習コスト。バリデーションが増える S06 (冷蔵庫追加行) では過剰かも
- (c) TanStack Form
  - Pros：TanStack Router / Query と揃う
  - Cons：2025 時点でまだ発展途上、事例が少ない

**採用: (a)**。理由: フィールド数が少なく、検証項目も長さ / 範囲チェック程度なので自前で十分。将来 S05 (冷蔵庫追加行) や
S06 で似たフォームが増えたら再検討する (2 個以上似たフォームができた時点で共通化候補として (b) を検討)。

### D8. 保存の失敗時ハンドリング

**選択肢:**

- (a) 保存失敗時は S04 に留まり、上部にエラーバナー表示。入力内容は破棄しない ← 採用
  - Pros：ユーザーの入力労力を失わない
  - Cons：エラー原因の UI 表示ロジックが増える
- (b) 失敗時はトーストで通知し、画面遷移する
  - Cons：入力が消えるので絶対避ける

**採用: (a)**。理由: レシピの入力量が多く、失敗して入力が消えるのは UX 破綻。エラーバナーは Tailwind のプレーンな div で扱う
(shadcn Alert プリミティブは後続の proposal で必要になったときに導入する)。エラーメッセージは Go 側で型付きエラーコード
(`ErrRecipeNameEmpty` / `ErrRecipeNameTooLong` / `ErrRecipeServingsOutOfRange` / `ErrRecipeIngredientNameEmpty` /
`ErrRecipeStepTooLong` / `ErrRecipeNotesTooLong` / `ErrRecipeTagTooLong`) を返し、フロント側で i18n key に対応付ける
([design.md #7](../../../docs/design.md) の方針)。

### D9. 材料入力 UI: 純テキスト入力 vs autocomplete vs 食材マスタ Combobox

初回案は「食材マスタから Combobox で選択、未登録なら新規作成」だったが、UX レビューで自由入力に方針転換した。

**選択肢:**

- (a) 純テキスト入力 3 列 (名前 / 量 / 単位) + 削除ボタン ← 採用
  - Pros：入力が最も軽い。IME 変換の途中で候補ダイアログが割り込まない。食材マスタが不要になり Popover / Command / cmdk 依存が丸ごと落ちる
  - Cons：「玉ねぎ」「たまねぎ」「タマネギ」を別物として扱う (在庫との突合で問題になる)。ただし本 proposal では在庫連動を扱わないため
    影響は将来の `add-ingredient-master` に閉じる
- (b) 名前欄を autocomplete input (見た目はテキスト、既存名にマッチしたら候補を出す)
  - Pros：入力コストは (a) に近く、名寄せは残せる
  - Cons：食材マスタが必要になる (本 proposal では持たない方針と衝突する)。マスタが無ければ候補が空で意味が無い
- (c) 食材マスタ + Combobox (初回案)
  - Pros：名寄せが完全に効き、単位を自動セットできる
  - Cons：入力ステップが増える (フォーカス → タイプ → 候補選択 → 単位確認)。IME との相性が悪い。マスタ設計・migration・
    Popover / Command / cmdk の shadcn 導入が要る

**採用: (a)**。理由: 本 proposal では食材マスタを持たない方針 (proposal.md Non-goals) と一致し、shadcn 依存 (`cmdk` /
`@radix-ui/react-popover`) も削減できる。名寄せは将来 `add-ingredient-master` で導入するときに (b) / (c) へ移行する
in-place migration (`recipe_ingredients.name TEXT` を `ingredient_id TEXT REFERENCES ingredients(id)` に置き換え、既存 name の
FindOrCreate による backfill) で実装できる。

### D10. 手順入力 UI: 単一 textarea (Markdown) vs 1 手順 1 レコードの並び替えエディタ

**選択肢:**

- (a) 単一 `textarea` + `maxLength=4000`、書き手側は Markdown 記法として素で入力 (レンダリングは S03 で追加) ← 採用
  - Pros：執筆コストが最も低い。改行 / 見出し / リスト / 強調をユーザーが自由に書ける。ドメインは `Steps string` の 1 フィールドで済む
  - Cons：書き手側でプレビューが出ない (S03 実装まで見た目確認ができない)
- (b) 1 手順 = 1 行の `textarea` + 上下ボタン + 削除ボタン + 追加ボタン (初回案)
  - Pros：ワイヤーの読み手表示 (番号付きリスト) と DB 保存形式が対応する
  - Cons：入力ステップが多い (行を追加 → タイプ → 次の行を追加)。順序入替の需要は執筆時点ではほぼ無い。手順内の改行 (Markdown 段落) を
    表現できない
- (c) 単一 `textarea` に加えて右横に Markdown プレビュー
  - Pros：書きながら見た目確認できる
  - Cons：`react-markdown` などのレンダラー依存を本 proposal で導入することになる (S03 で導入したいのに前倒しになる)。UI が窮屈

**採用: (a)**。理由: ワイヤーは番号付きの読み手表示を示しているが、書き手 UI までは規定していない。単一 `textarea` にすると入力が軽く、
S03 で `react-markdown` を導入した段階でプレビューと読み手表示の両方が動く。書き手側のプレビュー導入は S03 完了後に必要性を再評価する
(そもそも書きながらプレビューが要らない可能性がある)。

### D11. タグ入力 UI: chip 入力 vs カンマ区切りテキスト

**選択肢:**

- (a) chip 入力 (テキストボックスに文字入力 → Enter または `＋` ボタンで確定、`×` で削除) ← 採用
  - Pros：`,` を含むタグを扱える。1 タグの視覚が明快。将来「既存タグの候補提示」を chip 入力に足すのが素直
  - Cons：実装量が (b) より少し多い (state と onKeyDown を持つ)
- (b) 単一 text input に「和食, 煮物」のようにカンマ区切りテキストで入力 (初回案)
  - Pros：実装最短
  - Cons：`,` を含むタグを扱えない。タグ境界の視覚がない
- (c) chip 入力 + 既存タグの候補提示 (autocomplete)
  - Pros：入力補助が効く
  - Cons：既存タグ集約クエリ / UI が要る。本 proposal では過剰

**採用: (a)**。理由: `,` を含むタグ (「うどん,そば」等) を扱えないのは仕様として弱い。実装量の差は小さい。既存タグ候補提示は
別 proposal `add-recipe-tags-autocomplete` で追加する。

### D12. バリデーションの配置: フロントのみ vs Go 側のみ vs 両方

**選択肢:**

- (a) フロント (`RecipeEditForm`) と Go 側 (`RecipeService`) の両方で検証 ← 採用
  - Pros：ユーザー入力の即時フィードバック (フロント) と、バインディング / 手動 IPC 呼び出しからの防御 (Go) を両立できる。
    Go 側エラーは型付きコード (`ErrRecipe*`) で返し、フロントは i18n key に対応付ける
  - Cons：ルールを 2 箇所に書く (人数上限 99 / 名前 200 字 等)。ルール変更時は両方を更新する必要がある
- (b) フロントのみ
  - Pros：実装コストが半分
  - Cons：Wails バインディング直呼びや将来の外部連携で防御できない。DB に不正データが入り得る
- (c) Go 側のみ
  - Pros：一元管理
  - Cons：入力の都度 IPC を往復することになり、UX が重い

**採用: (a)**。理由: 「フロントは UX、Go は最終防衛線」の二層防御が標準的で、SQLite 側の CHECK 制約に降ろすと migration 変更が
面倒になる。両方に書くルール一覧は本節末尾の表にまとめ、更新時のチェックリストにする。

**バリデーションルール一覧:**

| フィールド     | 制約                                              | Go 側エラーコード                       | i18n key                                                |
|----------------|---------------------------------------------------|-----------------------------------------|---------------------------------------------------------|
| name           | 1〜200 字 (trim 後)                               | `ErrRecipeNameEmpty` / `NameTooLong`    | `edit.validation.name_required` / `name_too_long`       |
| servings       | 1〜99 の整数                                      | `ErrRecipeServingsOutOfRange`           | `edit.validation.servings_out_of_range`                 |
| ingredient.name| 空不可 (trim 後、量 / 単位は空可)                 | `ErrRecipeIngredientNameEmpty`          | `edit.validation.ingredient_name_required`              |
| steps          | 4000 字以内                                       | `ErrRecipeStepsTooLong`                 | `edit.validation.steps_too_long`                        |
| notes          | 2000 字以内                                       | `ErrRecipeNotesTooLong`                 | `edit.validation.notes_too_long`                        |
| tag (1 個)     | 30 字以内 (trim 後、空タグは黙って捨てる)         | `ErrRecipeTagTooLong`                   | `edit.validation.tag_too_long`                          |

上限値は `internal/config/defaults.go` に集約する候補だが、本 proposal では call site (`recipeservice.go` / `RecipeEditForm`) の
const に留める。理由: 設定画面 (S09) 以降にユーザーが変えたい値ではなく、実装上の上限だから (可変値集約ルールの適用対象外)。

## Risks / Trade-offs

- [Risk] SQLite 移行に伴い前 proposal のテストが全滅する (fixture 前提の recipeservice_test.go) → Mitigation: 本 proposal 内で
  同じ観点のテストを `t.TempDir()` の実 SQLite ベースに書き換える。テスト観点自体 (List が空/複数/複数タグ 等) は流用する
- [Risk] `Recipe` struct 拡張で `frontend/bindings/` の型変更が S02 のコンパイルを壊す (Ingredients が undefined 参照) →
  Mitigation: `Recipe` の新規フィールドは Go 側で必ず zero value を返し、フロント側 `RecipeTable` は既存フィールドのみ参照するので影響なし。
  ただしバインディング再生成 → npm run type-check → 目視の順で確認する
- [Risk] Wails バインディング生成モードが `task dev` (JSDoc) と `task build` (TS `-ts`) で異なり、TS モードでは models が `interface`
  として export されるため `new Recipe(...)` が build 時にエラーになる → Mitigation: フロントの `Recipe` インスタンス化は `import type { Recipe }`
  + `const draft: Recipe = { ... }` のリテラルで統一する (design.md Context 節にも記述)
- [Risk] 行クリック → S04 直行の暫定仕様が S03 実装まで残る間、ユーザーが「詳細を見るつもりが編集に飛ぶ」 → Mitigation:
  S03 実装 (`add-recipe-detail-drawer`) を次に着手する proposal 順序で担保する。i18n / ワイヤー上の説明は変えず、
  暫定であることは spec 側に明示する
- [Risk] `MealPlan.Entry` が現状 `RecipeName` を直接持っている ([entry.go](../../../internal/domain/mealplan/entry.go)) ため、
  レシピ名の変更が S01 側に反映されない → Mitigation: 本 proposal の対象外。add-mealplan-recipe-reference で
  `RecipeID` 参照に切り替える (proposal.md 「後続 proposal への引き継ぎ」参照)
- [Risk] `Recipe.IngredientCount` が Repository と Service で二重管理される (DB は `len(recipe_ingredients)`、Domain は int フィールド) →
  Mitigation: Repository の Get / List で `IngredientCount = len(Ingredients)` を都度セットする unit test で担保する。
  将来 (D1 の a → b) 分割時に Summary 型側に閉じる
- [Risk] 材料を自由テキストにしたため「玉ねぎ」「たまねぎ」の名寄せが崩れる → Mitigation: 冷蔵庫在庫との突合を扱う proposal
  (`add-mealplan-suggest-by-stock` / `add-shopping-list`) が着手される段階で `add-ingredient-master` を先に立て、`recipe_ingredients.name`
  を `ingredient_id` に置き換える migration を書く
- [Risk] 手順の Markdown 書き手プレビューが無く、書きながら見た目を確認できない → Mitigation: S03 詳細ドロワー proposal で
  `react-markdown` を導入する。書き手側プレビューの必要性はそこで再評価する

## Migration Plan

- migration `00002_recipes.sql` は既存の goose ランナーに乗るため、`sqlite.Migrate(db)` が呼ばれる起動時に自動適用される
- 前 proposal の fixture (`fixtureRecipes` の 6 件) は削除する。既存ユーザーの DB は当該テーブルが無い状態から作られるので、
  空 DB でアプリを起動するとレシピ 0 件の空状態が表示される (spec 「レシピが 0 件の場合は空状態を示す」で担保)
- ロールバック: goose の Down で 2 テーブルを DROP。ただし本 proposal の変更は追加のみで既存データを壊さないため、
  実運用では git revert で proposal コミットを打ち消す方が安全 (前 proposal 同様)
- 将来 `add-ingredient-master` で `recipe_ingredients.name TEXT` を `ingredient_id TEXT REFERENCES ingredients(id)` に置き換える migration を
  書く。手順は「(1) ingredients テーブル作成 (2) DISTINCT name で backfill (3) ingredient_id 列追加 → FindOrCreate で埋める
  (4) name 列削除」の 4 ステップ。設計は当該 proposal で確定する
