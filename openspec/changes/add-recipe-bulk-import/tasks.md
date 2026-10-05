## 1. 可変値と依存

- [ ] 1.1 `internal/config/defaults.go` を更新し、`Config` に `UnquantifiedWords []string` と `RecipeImportDefaultServings int` を追加する (design.md D5, D9)
    - 「レシピ」の区切りに置き、それぞれ WHY のコメントを 1〜2 行で書く
    - `Defaults()` で `UnquantifiedWords` に `[]string{"少々", "適量", "適宜", "少量", "ひとつまみ", "お好みで"}` を返す
    - `Defaults()` で `RecipeImportDefaultServings` に `2` を返す
    - 完了判定: `go build ./internal/...` が通る
- [ ] 1.2 `internal/config/defaults_test.go` の `TestDefaults` に検証を追加する
    - `UnquantifiedWords` が `[少々 適量 適宜 少量 ひとつまみ お好みで]` と一致する
    - `RecipeImportDefaultServings` が 2 である
    - 完了判定: `go test ./internal/config/...` が pass する
- [ ] 1.3 `go get gopkg.in/yaml.v3 golang.org/x/text` を実行し、`go.mod` の 2 つを direct の `require` に移す。完了判定: `go mod tidy` のあとも `go.mod` の direct に 2 つが残る

## 2. ドメイン層 (量とファイルの読み取り)

- [ ] 2.1 `internal/domain/recipe/amount.go` を新設し、`ParseAmount(raw any, unit string, measuringUnits, unquantifiedWords []string) (float64, string, bool)` を実装する (design.md D5)
    - `raw` が `int` / `float64` ならそのまま量にし、負なら `false` を返す
    - `raw` が `nil` か空文字なら量 0 を返す
    - 文字列は `width.Narrow` で半角にし、「〜」「～」を `~` に揃えてから読む
    - 読む順は、量を持たない言葉、先頭の量る単位、帯分数、分数、範囲、小数、末尾の単位とする
    - 量は小数第 2 位で丸める
    - 完了判定: 2.2 のテストが pass する
- [ ] 2.2 `internal/domain/recipe/amount_test.go` を新設し、`ParseAmount` の table-driven test を書く
    - int `2` は量 2、単位は `unit` のまま
    - float `0.5` は量 0.5
    - 負の数 `-1` は `false`
    - `nil` と `""` は量 0
    - `"1/2"` は量 0.5
    - `"1/3"` は量 0.33
    - `"1と1/2"` と `"1 1/2"` は量 1.5
    - `"１．５"` と `"１／２"` は量 1.5 と 0.5
    - `"2〜3"` と `"2~3"` と `"2-3"` は量 3
    - `"大さじ2"` (unit 空) は量 2、単位「大さじ」
    - `"3個"` (unit 空) は量 3、単位「個」
    - `"3個"` (unit「個」) は量 3、単位「個」
    - `"3個"` (unit「本」) は `false`
    - `"大さじ2杯"` は `false` (前後の両方に単位の候補がある)
    - `"少々"` (unit 空) は量 0、単位「少々」
    - `"適量"` (unit「g」) は量 0、単位「g」
    - `"たくさん"` は `false`
    - 完了判定: `go test ./internal/domain/recipe/...` が pass する
- [ ] 2.3 `internal/domain/recipe/importfile.go` を新設し、`ImportFile` / `ImportCandidate` / `ImportIssue` と問題コードの定数を定義する (design.md D3, D4)。完了判定: `go build ./internal/...` が通る
- [ ] 2.4 `internal/domain/recipe/importfile.go` に `ParseImportFile(content string, defaultServings int, measuringUnits, unquantifiedWords []string) (Recipe, ImportIssue, string)` を実装する (design.md D1)
    - 先頭の `---` と閉じる `---` の間を frontmatter として `yaml.v3` で読む。無ければ `no_frontmatter`、読めなければ `invalid_yaml`
    - `servings` が無ければ `defaultServings` を使う
    - `ingredients` の各要素の `amount` を `ParseAmount` に渡す。失敗したら `amount_invalid` と材料名を返す
    - 本文をレベル 2 の見出しで区切り、「手順」「Steps」を手順に、「メモ」「Notes」をメモにする (英字は大文字小文字を区別しない)
    - どちらの見出しも無ければ本文全体を手順にする
    - 手順とメモの前後の空行を除く
    - 完了判定: 2.5 のテストが pass する
- [ ] 2.5 `internal/domain/recipe/importfile_test.go` を新設し、`ParseImportFile` の table-driven test を書く
    - frontmatter の name / servings / tags / ingredients がレシピの各欄になる
    - `servings` が無ければ既定値になる
    - 「## 手順」と「## メモ」の中身が手順とメモになり、見出しの行は含まない
    - 「## Steps」と「## notes」も手順とメモになる
    - 手順の中の「### コツ」は手順の中身として残る
    - 見出しが無い本文は全体が手順になる
    - 見出しより前の本文は、見出しがあるときは捨てられる
    - メモに書いた URL がそのまま残る
    - `---` で始まらないと `no_frontmatter`
    - 閉じる `---` が無いと `no_frontmatter`
    - YAML が壊れていると `invalid_yaml`
    - `amount` が読めない材料があると `amount_invalid` と材料名を返す
    - frontmatter の未知の項目は無視される
    - 完了判定: `go test ./internal/domain/recipe/...` が pass する

## 3. リポジトリ層

- [ ] 3.1 `internal/storage/sqlite/recipe_repo.go` に `CreateMany(ctx, inputs []recipe.Recipe) ([]recipe.Recipe, error)` を追加する (design.md D7)
    - 1 トランザクションで `insertRecipeRow` と `insertRecipeIngredients` を件数分呼ぶ
    - ID の発行と `created_at` / `updated_at` / `IngredientCount` の設定は `Create` と同じ処理を共有する
    - 名前が空の要素があれば何も書かずに `ErrRecipeNameEmpty` を返す
    - 完了判定: 3.2 のテストが pass する
- [ ] 3.2 `internal/storage/sqlite/recipe_repo_test.go` に `CreateMany` のテストを追加する
    - 2 件を渡すと 2 件とも材料つきで `GetByID` で読める
    - 空の配列を渡すと何も書かずに空を返す
    - 2 件目の名前が空なら 1 件目も登録されない
    - 完了判定: `go test ./internal/storage/...` が pass する

## 4. サービス層

- [ ] 4.1 `internal/service/recipe.go` を更新し、`RecipeService` に `cfg config.Config` を持たせ、`NewRecipeService` で `config.Defaults()` を入れる。完了判定: `go build ./internal/...` が通る
- [ ] 4.2 `internal/service/recipe.go` に定数 `recipeImportMaxFiles = 100` と `recipeImportMaxBytes = 64 * 1024`、エラー `ErrRecipeImportTooManyFiles` を追加する (design.md D2)。完了判定: `go build ./internal/...` が通る
- [ ] 4.3 `internal/service/recipe.go` に `validateRecipeInput` のエラーから `recipe.ImportIssue` への対応表を追加する (design.md D4)。完了判定: 4.6 のテストが pass する
- [ ] 4.4 `internal/service/recipe.go` に `PreviewImport(ctx, files []recipe.ImportFile) ([]recipe.ImportCandidate, error)` を追加する (design.md D3)
    - 件数が上限を超えたら `ErrRecipeImportTooManyFiles` を返す
    - 大きさが上限を超えたファイルは `too_large` にする
    - `ParseImportFile` のあとに `validateRecipeInput` を当て、正規化した値を候補に入れる
    - 問題の無い候補の名前を、登録済みのレシピと、ほかの問題の無い候補の名前と比べて `Duplicate` を立てる
    - ファイル名の昇順に並べて返す
    - 完了判定: 4.6 のテストが pass する
- [ ] 4.5 `internal/service/recipe.go` に `Import(ctx, inputs []recipe.Recipe) ([]recipe.Recipe, error)` を追加する
    - 全件に `validateRecipeInput` を当て、1 件でも違反があれば何も書かずにそのエラーを返す
    - `repo.CreateMany` で登録する
    - 完了判定: 4.6 のテストが pass する
- [ ] 4.6 `internal/service/recipe_test.go` に `TestRecipeService_PreviewImport` と `TestRecipeService_Import` を追加する
    - should return candidates sorted by file name when multiple files are given
    - should fill servings with the default when servings is omitted
    - should return ErrRecipeImportTooManyFiles when more than 100 files are given
    - should mark too_large when a file exceeds 64 KB
    - should mark name_empty when the name is missing
    - should mark steps_too_long when the steps exceed 4000 characters
    - should mark ingredient_name_empty with the ingredient position when an ingredient has no name
    - should mark duplicate when the name matches an existing recipe
    - should not mark duplicate when the name matches only an archived recipe
    - should mark both candidates duplicate when two files share a name
    - should not mark duplicate against a candidate that has an issue
    - should not write anything to the database when previewing
    - should create all recipes when every input is valid
    - should create a separate recipe when the name matches an existing recipe
    - should create nothing when one input violates a constraint
    - 完了判定: `go test ./internal/service/...` が pass する
- [ ] 4.7 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/bindings/` に `PreviewImport` / `Import` と `ImportFile` / `ImportCandidate` が出ることを確認する。完了判定: 生成物を目視で確認する
- [ ] 4.8 `frontend/e2e/bindings-shim/recipeservice.ts` に `PreviewImport` と `Import` を追加する。完了判定: `cd frontend && npm run type-check` が通る

## 5. フロントエンド

- [ ] 5.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.recipes.import.*` を追加する
    - ボタンとメニュー (インポート、ファイルを選ぶ、フォルダを選ぶ)
    - ダイアログの見出し、取り込む (件数つき)、キャンセル、すべて選択、すべて解除
    - 一覧の列 (ファイル名、レシピ名、材料数、状態) と状態 (取り込める、同名あり)
    - `issue.<code>` を design.md D4 の全コード分
    - 通知 (取り込んだ件数)、読み込めるファイルが無い、件数の上限超え、登録の失敗
    - 完了判定: 両方のファイルでキーの集合が一致する
- [ ] 5.2 `frontend/src/components/recipe/recipe-detail.tsx` を更新し、材料、手順、メモの欄を `RecipeContentSections` として切り出す (design.md D8)。完了判定: `npm run type-check` が通り、既存の `recipe-detail.spec.ts` が pass する
- [ ] 5.3 `frontend/src/components/recipe/read-import-files.ts` を新設し、`FileList` から拡張子 `.md` のものだけを `{ name, content }` の配列にする
    - 名前は `webkitRelativePath` が空でなければそれを、空なら `name` を使う
    - 100 件を超えたら読まずに件数超えを返す
    - 完了判定: `npm run type-check` が通る
- [ ] 5.4 `frontend/src/components/recipe/recipe-import-button.tsx` を新設し、`dropdown-menu` のメニューと、ファイル用 (`multiple`、`accept=".md"`) とフォルダ用 (`webkitdirectory`) の隠し `input` を置く (design.md D8)
    - 読み取りのあとで `input.value` を空にする
    - `.md` が 0 件なら通知を出してダイアログを開かない
    - 件数超えなら通知を出してダイアログを開かない
    - 完了判定: `npm run type-check` が通る
- [ ] 5.5 `frontend/src/components/recipe/recipe-import-preview.tsx` を新設し、名前、人数、タグと `RecipeContentSections` を表示する。同名ありなら警告、問題があれば理由を表示する。完了判定: `npm run type-check` が通る
- [ ] 5.6 `frontend/src/components/recipe/recipe-import-list.tsx` を新設し、1 候補 1 行の一覧を表示する
    - 列はチェック、ファイル名、レシピ名、材料数、状態
    - 問題のある行はチェックを `disabled` にし、状態に理由を出す
    - 一覧の上に「すべて選択」と「すべて解除」を置く
    - 完了判定: `npm run type-check` が通る
- [ ] 5.7 `frontend/src/components/recipe/recipe-import-dialog.tsx` を新設し、S02a を組み立てる
    - `PreviewImport` を TanStack Query の `useQuery` で呼ぶ
    - 候補が 1 件ならプレビュー、2 件以上なら一覧を出す
    - 選択の初期値は、問題も重複も無い候補にする
    - 「取り込む」で `Import` を `useMutation` で呼び、成功したら `["recipes", "list"]` を無効化し、件数の通知を出して閉じる
    - 失敗したら閉じずに失敗の旨を出す
    - 完了判定: `npm run type-check` が通る
- [ ] 5.8 `frontend/src/routes/recipes.tsx` を更新し、ヘッダの「新しいレシピ」の左に `RecipeImportButton` を置き、読み取った結果で `RecipeImportDialog` を開く。完了判定: `task dev` で S02 から開けることを目視で確認する

## 6. E2E

- [ ] 6.1 `frontend/e2e/fixtures/recipe-import/` を新設し、読み込ませるファイルを置く
    - `nikujaga.md` (肉じゃが、人数 3、タグ、材料、手順、URL を含むメモ)
    - `curry.md` (カレー)
    - `curry-2.md` (カレー、同名)
    - `no-name.md` (name なし)
    - `no-servings.md` (servings なし)
    - `no-frontmatter.md`
    - `amounts.md` (`1/2`、`1と1/2`、`大さじ2`、`3個`、`少々`)
    - `bad-amount.md` (`たくさん`)
    - `folder/` (`a.md`、`b.md`、`photo.png`)
    - `empty-folder/` (`photo.png` のみ)
    - 完了判定: ファイル一式が揃っている
- [ ] 6.2 `frontend/e2e/recipe-import.spec.ts` を新設し、次の Scenario を describe (Requirement 名) と test (Scenario 名) で書く
    - S02 からのレシピのインポート
      - 選んだファイルがインポートダイアログに表示される
      - フォルダを選ぶと中の Markdown ファイルだけが読み込まれる
      - Markdown ファイルが 1 つも無いと知らせる
    - インポートするファイルの形式
      - frontmatter と本文の節がレシピの各欄になる
      - 人数を省略すると 2 人分になる
      - 名前の無いファイルはエラーになる
      - frontmatter の無いファイルはエラーになる
    - 材料の量の表記ゆれの吸収
      - 分数と帯分数は小数になる
      - 単位が付いた量は量と単位に分かれる
      - 量を持たない言葉は量 0 になる
      - 読み取れない量はエラーになる
    - 1 件のインポートのプレビュー
      - 1 件ならプレビューを見てから取り込める
      - キャンセルすると何も登録されない
      - 1 件がエラーなら取り込めない
    - 複数件のインポートの選択
      - エラーの無いファイルは既定で選ばれている
      - チェックを外したファイルは取り込まれない
      - すべて解除すると取り込めない
    - 同名のレシピの警告
      - 登録済みのレシピと同じ名前なら警告が出る
      - 同時に読み込んだファイルどうしで名前が同じなら警告が出る
      - 1 件でも同名なら警告を添えてプレビューする
      - 警告を承知で取り込むと別のレシピとして追加される
    - インポートの確定
      - 取り込んだ件数が通知される
    - 完了判定: `task test:e2e` が pass する

## 7. ドキュメント

- [ ] 7.1 `docs/recipe-import.md` を新設し、インポート形式の説明と AI 向けプロンプトのひな形を書く
    - frontmatter の項目と本文の節、量の書き方として受け付けるもの
    - 1 ファイルの完全な例
    - プロンプトのひな形 (元レシピの URL をメモの節に書く、文字列の値は引用符で囲む、量と単位は分けて書く、1 レシピ 1 ファイルで出す)
    - `japanese-tech-writing` と `cognitive-rhythm-writing` のスキルを適用する
    - 完了判定: 例のファイルを実際に取り込めることを `task dev` で確認する
- [ ] 7.2 `docs/wireframes/S02a_recipe_import_dialog.drawio.svg` を新設し、プレビューの状態と一覧の状態の 2 つを描く。完了判定: ブラウザで SVG を開いて目視で確認する
- [ ] 7.3 `docs/wireframes/README.md` のファイル一覧に S02a を追加する。完了判定: 差分を目視で確認する
- [ ] 7.4 `docs/product.md` を更新する
    - §1 の可変値の表に `UnquantifiedWords` と `RecipeImportDefaultServings` の行を追加する
    - §2 の画面一覧に S02a インポートダイアログを追加する
    - §4 S02 にインポートの項と S02a のワイヤーへのリンク、`docs/recipe-import.md` へのリンクを追加する
    - 完了判定: 差分を目視で確認する

## 8. 検証

- [ ] 8.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [ ] 8.2 `cd frontend && npm run type-check` を実行する。完了判定: エラーが 0 件
- [ ] 8.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [ ] 8.4 `task test:e2e` を実行する。完了判定: 全テストが pass する
- [ ] 8.5 `task lint` を実行する。完了判定: 指摘が 0 件
- [ ] 8.6 `task build:win` で Windows ビルドを作り、実機で確認する
    - フォルダ選択でフォルダ内の `.md` が読み込まれる
    - 日本語のファイル名と日本語の中身が崩れずに取り込まれる
    - 完了判定: Windows 実機で目視で確認する
