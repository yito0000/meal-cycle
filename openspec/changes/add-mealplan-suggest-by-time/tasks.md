## 1. レシピの所要時間 (ドメイン層とリポジトリ層)

- [ ] 1.1 `internal/storage/sqlite/migrations/00012_recipes_cook_minutes.sql` を新設し、`recipes` に `cook_minutes INTEGER NOT NULL DEFAULT 0` を足す (Down で列を落とす)。
  完了判定: `go test ./internal/storage/sqlite/ -run TestMigrate` が pass する
- [ ] 1.2 `internal/domain/recipe/recipe.go` の `Recipe` に `CookMinutes int` を足し、検証に 0〜999 を足す。`ErrCookMinutesOutOfRange` を定義する。
  完了判定: `go build ./internal/...` が通る
- [ ] 1.3 `internal/domain/recipe/recipe_test.go` に検証のケースを足す (table-driven)。完了判定: `go test ./internal/domain/recipe/` が pass する
  - 0 (未入力) は通る
  - 999 は通る
  - 1000 は ErrCookMinutesOutOfRange
  - -1 は ErrCookMinutesOutOfRange
- [ ] 1.4 `internal/storage/sqlite/recipe_repo.go` の SELECT / INSERT / UPDATE に `cook_minutes` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 1.5 `internal/storage/sqlite/recipe_repo_test.go` にケースを足す。完了判定: `go test ./internal/storage/sqlite/ -run TestRecipeRepo` が pass する
  - 所要時間を保存して読み戻せる
  - migration 前からある行は所要時間 0 で読める

## 2. インポートとエクスポート

- [ ] 2.1 `internal/domain/recipe/importfile.go` の frontmatter に `CookMinutes *int \`yaml:"cookMinutes"\`` を足し、`ImportIssueCookMinutesOutOfRange` を定義する (design.md D4)。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.2 `internal/domain/recipe/importfile_test.go` にケースを足す。完了判定: `go test ./internal/domain/recipe/ -run TestParseImportFile` が pass する
  - cookMinutes: 15 を 15 として読む
  - cookMinutes を省略すると 0
  - cookMinutes: 0 は ImportIssueCookMinutesOutOfRange
  - cookTime は今までどおり無視する
- [ ] 2.3 `internal/domain/recipe/exportfile.go` で `CookMinutes` が 0 でなければ `cookMinutes` を書く。完了判定: `go build ./internal/...` が通る
- [ ] 2.4 `internal/domain/recipe/exportfile_test.go` にケースを足す。完了判定: `go test ./internal/domain/recipe/ -run TestExportFile` が pass する
  - 所要時間 30 のレシピは cookMinutes: 30 を書く
  - 所要時間 0 のレシピは cookMinutes を書かない
  - 書き出したファイルを読み戻すと同じ所要時間になる

## 3. 設定の値

- [ ] 3.1 `internal/config/defaults.go` に `QuickCookMaxMinutes int` (既定 20) を足す。完了判定: `go build ./internal/...` が通る
- [ ] 3.2 `internal/config/defaults_test.go` に既定値 20 を足す。完了判定: `go test ./internal/config/` が pass する
- [ ] 3.3 `internal/domain/settings/settings.go` の `Values` / `FromConfig` / `ApplyTo` に `QuickCookMaxMinutes` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 3.4 `internal/domain/settings/errors.go` に `ErrQuickCookMinutesOutOfRange` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 3.5 `internal/domain/settings/validate.go` で 5〜120 を検証する。完了判定: `go build ./internal/...` が通る
- [ ] 3.6 `internal/domain/settings/settings_test.go` に検証のケースを足す。完了判定: `go test ./internal/domain/settings/` が pass する
  - 5 は通る
  - 120 は通る
  - 4 は ErrQuickCookMinutesOutOfRange
  - 121 は ErrQuickCookMinutesOutOfRange
- [ ] 3.7 `internal/service/settings.go` の `settingKeys` に `quickCookMaxMinutes` を足す。完了判定: `go build ./internal/...` が通る

## 4. 時短の候補

- [ ] 4.1 `internal/domain/suggestion/suggestion.go` の `Candidate` に `CookMinutes int` を、`Result` に `ByTime []Candidate` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 4.2 `internal/domain/suggestion/by_time.go` を新設し、`ByTime(recipes, maxMinutes, limit)` を書く (design.md D3)。完了判定: `go build ./internal/...` が通る
- [ ] 4.3 `internal/domain/suggestion/by_time_test.go` を新設する (table-driven)。完了判定: `go test ./internal/domain/suggestion/ -run TestByTime` が pass する
  - maxMinutes 以下のレシピだけを返す
  - 所要時間 0 のレシピを返さない
  - 所要時間の昇順、同じならレシピ名の昇順に並べる
  - limit 件で切る
- [ ] 4.4 `internal/storage/sqlite/suggestion_repo.go` で削除済みでないレシピの所要時間を読む。完了判定: `go build ./internal/...` が通る
- [ ] 4.5 `internal/service/suggestion.go` の `List` で `cfg.QuickCookMaxMinutes` を使って `ByTime` を埋める。完了判定: `go build ./internal/...` が通る
- [ ] 4.6 `internal/service/suggestion_test.go` の `TestSuggestionService_List` にケースを足す。完了判定: `go test ./internal/service/ -run TestSuggestionService_List` が pass する
  - should return recipes within the quick cook minutes when listing by time
  - should not return archived recipes when listing by time
  - should use the saved quick cook minutes when settings are saved
- [ ] 4.7 `wails3 generate bindings -clean=true -ts -i` で bindings を作り直す。完了判定: `cd frontend && npm run type-check` が通る

## 5. 画面

- [ ] 5.1 `frontend/src/components/recipe/use-recipe-form.ts` と `recipe-edit-form.tsx` に所要時間の欄を足す (空は 0 として送る)。完了判定: `task dev` の S04 で目視
- [ ] 5.2 `frontend/src/components/recipe/recipe-detail.tsx` の基本情報に「約 N 分」を足す。完了判定: `task dev` の S03 で目視
- [ ] 5.3 `frontend/src/components/recipe/recipe-import-preview.tsx` に所要時間を足す。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 5.4 `frontend/src/components/mealplan/suggestion-section.tsx` と `recipe-picker-dialog.tsx` に「時短」の欄を足す。完了判定: `task dev` の S01 で目視
- [ ] 5.5 `frontend/src/components/settings/settings-form.tsx` の献立の節に「時短」とみなす所要時間の欄を足す。完了判定: `task dev` の S09 で目視
- [ ] 5.6 `frontend/src/components/settings/error-messages.ts` と recipe のエラーの文言の表に新しいエラーを足す。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 5.7 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に文言を足す。完了判定: `cd frontend && npm run type-check` が通る

## 6. E2E

- [ ] 6.1 `frontend/e2e/suggestion-by-time.spec.ts` を新設する。完了判定: `task test:e2e` で pass する
  - describe「時短の候補」
    - 所要時間の短いレシピが候補に出る
    - 所要時間を持たないレシピは候補に出ない
    - S09 の分数を変えると候補が変わる
  - describe「レシピの所要時間」
    - 所要時間を入れて保存すると詳細画面に表示される
    - 範囲外の所要時間では保存できない
- 「所要時間の短い順に並ぶ」と、インポート / エクスポートの Scenario は 2.2 / 2.4 / 4.3 の Go のテストで確かめる

## 7. ドキュメント

- [ ] 7.1 `docs/recipe-import.md` の frontmatter の項目に `cookMinutes` を足し、プロンプトのひな形を直す。完了判定: 目視
- [ ] 7.2 `docs/wireframes/S01a_recipe_picker_dialog.drawio.svg` / `S04_recipe_edit.drawio.svg` / `S09_settings.drawio.svg` を更新する (可視 SVG と編集 XML の両方)。完了判定: 目視
- [ ] 7.3 `docs/product.md` の §1 可変値一覧に「『時短』とみなす所要時間」の行を足し、§4 S03 / S04 / S09 と §5 の時短の行を直す。完了判定: 目視
- [ ] 7.4 `docs/design.md` の §1 と §9 の時短を実装済みに直し、§4 の Recipe に `cookMinutes` を足す。完了判定: 目視

## 8. 確認

- [ ] 8.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [ ] 8.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [ ] 8.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [ ] 8.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [ ] 8.5 `task test:e2e` を実行する。完了判定: 全 pass
- [ ] 8.6 `task lint` を実行する。完了判定: エラー 0
