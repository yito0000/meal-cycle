## 1. 履歴の読み出し

- [x] 1.1 `internal/domain/suggestion/suggestion.go` を更新し、`RecipeHistory` (`RecipeID` / `LastCookedDate` / `HasUpcomingPlan`) を追加する。
  `Candidate` に `DaysSince int`、`Result` に `ByRecency []Candidate` を JSON タグ付きで足す。完了判定: `go build ./internal/...` が通る
- [x] 1.2 `internal/storage/sqlite/suggestion_repo.go` を新設し、`SuggestionRepo.ListRecipeHistory(ctx, today string)` を design.md D2 のクエリで実装する。完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/storage/sqlite/suggestion_repo_test.go` を新設し、実 SQLite でテストする。完了判定: `go test ./internal/storage/sqlite/ -run TestSuggestionRepo` が pass する
    - 作った印の付いた行の最も新しい日付が LastCookedDate になる
    - 作った印の無い行は LastCookedDate に数えない
    - 作り置きの印は同じ組の全行の日付に効く
    - 今日以降に印の無い行があると HasUpcomingPlan が true になる
    - 今日以降の行が印付きなら HasUpcomingPlan は false のまま
    - 一度も印の無いレシピは LastCookedDate が空になる
    - 削除済みのレシピは返らない

## 2. 候補の算出とサービス

- [x] 2.1 `internal/domain/suggestion/by_recency.go` を新設し、`ByRecency(recipes []recipe.Recipe, history []RecipeHistory, today time.Time, thresholdDays, limit int) []Candidate` を実装する (design.md D1 / D3 / D4)。
  完了判定: `go build ./internal/...` が通る
- [x] 2.2 `internal/domain/suggestion/by_recency_test.go` を新設し、table-driven でテストする。完了判定: `go test ./internal/domain/suggestion/` が pass する
    - 経過日数がちょうど閾値のレシピは候補になる
    - 閾値に 1 日足りないレシピは候補にならない
    - 最終調理日が未来 (経過日数が負) のレシピは候補にならない
    - 一度も作っていないレシピは候補にならない
    - HasUpcomingPlan のレシピは候補にならない
    - 経過日数の降順、レシピ名の昇順に並ぶ
    - limit を超える候補は切られる
- [x] 2.3 `internal/service/suggestion.go` を更新し、`SuggestionService` に `SuggestionRepo` を持たせ、`List` で `ByRecency` を `cfg.RecencyThresholdDays` と `cfg.SuggestionLimit` で埋める。
  `ByRecency` は空でも空スライスで返す。完了判定: `go build .` が通る
- [x] 2.4 `internal/service/suggestion_test.go` に BDD 風のテストを追加する。完了判定: `go test ./internal/service/ -run TestSuggestionService_List` が pass する
    - should return recipes cooked long ago when threshold has passed
    - should follow saved recency threshold when settings change

## 3. S09 の欄

- [x] 3.1 `internal/domain/settings/settings.go` の `Values` に `RecencyThresholdDays int` を足し、`FromConfig` と `ApplyTo` に通す。完了判定: `go build ./internal/...` が通る
- [x] 3.2 `internal/domain/settings/validate.go` に `RecencyDaysMin = 1` と `RecencyDaysMax = 60`、`ValidateRecencyThreshold(days int) error` を足し、`Values.Validate` の列に加える。
  `internal/domain/settings/errors.go` に `ErrRecencyDaysOutOfRange` (`settings: recency days out of range`) を足す。完了判定: `go build ./internal/...` が通る
- [x] 3.3 `internal/domain/settings/settings_test.go` に table-driven のケースを追加する。完了判定: `go test ./internal/domain/settings/` が pass する
    - 1 と 60 は通る
    - 0 と 61 は ErrRecencyDaysOutOfRange になる
- [x] 3.4 `internal/service/settings.go` の `settingKeys` に `recencyThresholdDays` を足す (encode / decode / validate)。完了判定: `go test ./internal/service/ -run TestSettingsService` が pass する
- [x] 3.5 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.6 `frontend/src/components/settings/settings-form.tsx` の献立の節に、作り置きの日数の候補の下へ「久しぶりに」とみなす日数の整数入力欄を足す (design.md D5)。
  `frontend/src/components/settings/error-messages.ts` に `settings: recency days out of range` を足す。完了判定: `cd frontend && npm run type-check` が通る

## 4. ダイアログ

- [x] 4.1 `frontend/src/components/mealplan/use-suggestions.ts` で `byRecency` の null を空配列にそろえる。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/src/components/mealplan/recipe-picker-dialog.tsx` で、「使い切り」の欄の下に「久しぶりに」の `SuggestionSection` を置き、根拠を `by_recency_reason` で組む。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する
    - `pages.meal_plan.picker.suggestion.by_recency_title`：「久しぶりに ({{days}} 日以上作っていない)」
    - `pages.meal_plan.picker.suggestion.by_recency_reason`：「{{days}} 日ぶり」
    - S09 の欄の見出し、説明、エラー (`1 以上 60 以下の日数を入れてください`)
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 5. E2E

- [x] 5.1 `frontend/e2e/suggestion.spec.ts` に describe「久しぶりの候補」を足す。日付を過去にした行は `assignRecipe` と `setCooked` で作る。完了判定: `task test:e2e` で以下が pass する
    - しばらく作っていないレシピが候補に出る
    - 最近作ったレシピは候補に出ない
    - 予定のあるレシピは候補に出ない
    - 一度も作っていないレシピは候補に出ない
- [x] 5.2 `frontend/e2e/settings.spec.ts` にテストを足す。完了判定: `task test:e2e` で以下が pass する
    - describe「設定値の検証」：久しぶりとみなす日数が範囲外だと保存されない
    - describe「保存した値の各画面への反映」：久しぶりとみなす日数を変えると候補が変わる

ドメイン層とリポジトリ層で確かめるシナリオ (「作った印の無い行は最終調理日に数えない」「間が空いたレシピほど上に並ぶ」) は E2E に含めない。

## 6. ドキュメントと検証

- [x] 6.1 `docs/product.md` を更新する。完了判定: 差分を目視で確認する
    - §1 の可変値の表で、「久しぶりに」の消費期間閾値を S09 (`RecencyThresholdDays`) で変更可にする
    - §4 S09 に献立の節の欄を足す
    - §5 の「久しぶりに」の行に、一度も作っていないレシピと予定のあるレシピを除くことを足す
- [x] 6.2 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 6.3 `cd frontend && npm run test` と `npm run type-check` と `npm run build:dev` を実行する。完了判定: 3 つとも成功する
- [x] 6.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 6.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 6.6 `openspec validate add-mealplan-suggest-by-recency --strict` を実行する。完了判定: エラー 0 件
