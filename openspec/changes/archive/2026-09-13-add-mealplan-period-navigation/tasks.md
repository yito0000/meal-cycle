## 1. サービス API 拡張 (ListPeriod)

- [x] 1.1 `internal/service/mealplan.go` に
      `ListPeriod(ctx context.Context, periodStart string, periodWeeks int) ([]mealplan.Entry, error)`
      を追加する。`periodWeeks` は 1 / 2 / 4 のいずれか、それ以外は新 sentinel `ErrInvalidPeriodWeeks`
      を返す。完了判定: `go build ./` が通る
- [x] 1.2 `internal/domain/mealplan/validate.go` に `ErrInvalidPeriodWeeks` sentinel を追加する。
      完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/storage/sqlite/mealplan_repo.go` に
      `ListPeriod(ctx, periodStart string, periodDays int) ([]mealplan.Entry, error)` を追加する。
      既存 `ListWeek` の SQL を流用し `< periodStart + periodDays` に一般化する。完了判定: 下記 1.5 の
      テストが pass する
- [x] 1.4 既存 `MealPlanService.ListWeek` は削除し、フロントの呼び出しを `ListPeriod(periodStart, 1)`
      に置き換える。E2E shim も同時更新。完了判定: `go build ./ && npm run type-check` が通る
- [x] 1.5 `internal/storage/sqlite/mealplan_repo_test.go` / `internal/service/mealplan_test.go` に
      `ListPeriod` の以下ケースを追加する
    - periodWeeks=1 で 7 日分だけ返す (既存 ListWeek 相当)
    - periodWeeks=2 で 14 日分返す
    - periodWeeks=4 で 28 日分返す
    - periodWeeks=3 (非対応) は ErrInvalidPeriodWeeks を返す
    - (完了判定: `go test ./internal/...` が pass する)

## 2. Wails bindings 再生成 + shim 更新

- [x] 2.1 `wails3 generate bindings` を実行し、`ListPeriod` が生成されたことを確認する。完了判定:
      `frontend/bindings/.../mealplanservice.ts` に `ListPeriod` がある
- [x] 2.2 `frontend/e2e/bindings-shim/mealplanservice.ts` を更新し、`ListWeek` を削除して `ListPeriod`
      を追加する。完了判定: `npm run type-check` が通る

## 3. フロント: 期間切替 + 前後ナビ UI

- [x] 3.1 `frontend/src/components/mealplan/period-nav.tsx` を新設する。props: `periodStart: Date`,
      `periodWeeks: 1 | 2 | 4`, `onPeriodChange(start: Date, weeks: 1|2|4)`, `onToday()`. UI は
      `＜` `今週` `＞` ボタン + 「1 週 / 2 週 / 4 週」のセグメント切替。完了判定: `npm run type-check`
      が通る
- [x] 3.2 `frontend/src/routes/*.tsx` (S01 の route) を更新し、`periodStart` と `periodWeeks` を
      `useState` で保持する。初期値は「今日を含む週の月曜日 / 1 週」。完了判定: 目視で S01 起動時に
      1 週表示になる
- [x] 3.3 `useQuery` の queryKey を `["mealplan", "period", periodStart, periodWeeks]` に変更し、
      `MealPlanService.ListPeriod(...)` を呼び出す。`invalidateQueries({ queryKey: ["mealplan"] })`
      に統一する。完了判定: `npm run type-check` が通り、目視で切替時に再取得される

## 4. WeekCalendar → 期間対応化

- [x] 4.1 `frontend/src/components/mealplan/week-calendar.tsx` の props に `periodWeeks: 1 | 2 | 4` を
      追加し、`weekStart` を `periodStart` に改名する。完了判定: `npm run type-check` が通る
- [x] 4.2 Grid の列数を `periodWeeks` に応じて `7 / 14 / 7 (4 行折り返し)` にする。4 週表示は 7 列 × 4
      行折り返し、2 週表示は 14 列で横スクロール。完了判定: 目視で切替がレイアウトに反映
- [x] 4.3 `resize-mealplan-cells` の `pickCellPreset(periodWeeks, visibleMeals.size)` に `periodWeeks`
      を正しく渡す (resize-mealplan-cells 実装完了後にこの task を再確認)。完了判定: 各 preset が
      正しく効く (2 週 → 小、4 週 → 最小)

## 5. i18n / ドキュメント

- [x] 5.1 `frontend/src/locales/{ja,en}.json` に以下 key を追加する
    - `mealplan.period.week_1` / `week_2` / `week_4`
    - `mealplan.period.prev` / `next` / `today`
    - `mealplan.errors.invalid_period_weeks`
    - (完了判定: `npm run type-check` が通り、`npm run build:dev` が成功する)
- [x] 5.2 `frontend/src/components/mealplan/error-messages.ts` に
      `mealplan: invalid period weeks` → `pages.meal_plan.errors.invalid_period_weeks` のマッピングを
      追加する。完了判定: `npm run type-check` が通る
- [x] 5.3 `docs/product.md` §S01 の説明を更新: 期間切替と前後ナビが実装済みである旨、および UI の
      配置 (画面右上) を明記する。完了判定: 目視で該当節が更新される
- [x] 5.4 `docs/scenarios/mealplan.feature` の「献立カレンダーの表示」機能に「期間切替」「前後ナビ」の
      シナリオを追記する (spec の Scenario 名と 1 対 1)。完了判定: feature と spec で Scenario 名が
      揃う

## 6. E2E

- [x] 6.1 `frontend/e2e/mealplan-assign.spec.ts` の「機能: 献立カレンダーの表示」の @planned / test.skip
      の期間切替 / 前後ナビシナリオを実際に動く test に置き換える (少なくとも「翌週に移動できる」の
      1 本)。完了判定: `task test:e2e` に新テストが追加され pass する

## 7. lint / format / テスト最終ゲート

- [x] 7.1 `task lint`。完了判定: exit code 0
- [x] 7.2 `task format`。完了判定: `git diff` が整形のみ
- [x] 7.3 `go test ./internal/... ./`。完了判定: exit code 0
- [x] 7.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev`。完了判定:
      exit code 0
- [x] 7.5 `task test:e2e`。完了判定: exit code 0
