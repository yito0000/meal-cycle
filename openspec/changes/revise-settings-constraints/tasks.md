## 1. S09 の制約を Go から返す

- [ ] 1.1 `internal/domain/settings/constraints.go` を新設し、`Constraints` struct (`ShoppingRequiredDays []int` `json:"shoppingRequiredDays"`、
  `ShoppingFallbackDays int` `json:"shoppingFallbackDays"`) と、それを組み立てる `CurrentConstraints()` を定義する。
  `ShoppingRequiredDays` は `validate.go` の変数の複製、`ShoppingFallbackDays` はその先頭にする。`go build ./internal/...` が通ることで確かめる
- [ ] 1.2 `internal/domain/settings/settings_test.go` に `CurrentConstraints` の table-driven test を足す (更新)。
  `go test ./internal/domain/settings/` が通ることで確かめる
  - 消せない日数が 7 / 14 / 28 になる
  - 選び直す日数が消せない日数に含まれる
  - 戻り値の配列を書き換えても `ShoppingRequiredDays` は変わらない
- [ ] 1.3 `internal/service/settings.go` に `Constraints(ctx) (settings.Constraints, error)` を足す。1.4 のテストで確かめる
- [ ] 1.4 `internal/service/settings_test.go` に `TestSettingsService_Constraints` を足す (更新)。`go test ./internal/service/ -run TestSettingsService_Constraints` が通ることで確かめる
  - should return required shopping days 7, 14 and 28 when nothing is saved
  - should return the same constraints when shopping presets are saved with extra days

## 2. 統合ダイアログの格子を Go から返す

- [ ] 2.1 `internal/domain/mealplan/spreadgrid.go` を新設し、`SpreadGrid` struct (`MaxOffsetDays int` `json:"maxOffsetDays"`、`Entries []Entry` `json:"entries"`) を定義する。
  `go build ./internal/...` が通ることで確かめる
- [ ] 2.2 `internal/service/mealplan.go` に `SpreadGrid(ctx, entryID string) (mealplan.SpreadGrid, error)` を足す。
  `entryID` が空なら `ErrEntryNotFound`、`s.repo.Entry` で起点の日付を読み、`s.repo.ListPeriod(ctx, src.Date, maxOffsetDays+1)` の結果を返す。
  `Entries` は空でも非 nil にする。2.3 のテストで確かめる
- [ ] 2.3 `internal/service/mealplan_test.go` に `TestMealPlanService_SpreadGrid` を足す (更新)。`go test ./internal/service/ -run TestMealPlanService_SpreadGrid` が通ることで確かめる
  - should return max offset 3 and entries for 4 days from the source date when presets are the defaults
  - should return max offset 6 when leftover presets are 2 and 7
  - should include an entry on the last selectable day and exclude one on the day after
  - should return ErrEntryNotFound when entryID is empty
  - should return an empty non-nil entries slice when nothing else is planned
- [ ] 2.4 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/bindings/` に `Constraints` と `SpreadGrid` が出ることを確かめる (生成物)

## 3. e2e の bindings shim

- [ ] 3.1 `frontend/e2e/bindings-shim/settingsservice.ts` に `Constraints()` を足す (更新)。`npm run type-check` が通ることで確かめる
- [ ] 3.2 `frontend/e2e/bindings-shim/mealplanservice.ts` に `SpreadGrid(entryID: string)` を足す (更新)。`npm run type-check` が通ることで確かめる

## 4. S09 の制約をフロントで使う

- [ ] 4.1 `frontend/src/components/settings/use-settings.ts` に `useSettingsConstraints()` を足す (更新)。
  query key は `["settings", "constraints"]`、`staleTime: Number.POSITIVE_INFINITY`、配列は `?? []` で非 null にそろえる。`npm run type-check` で確かめる
- [ ] 4.2 `frontend/src/routes/settings.tsx` を更新し、`useSettingsConstraints()` を読み終えてから `SettingsForm` に `constraints` を渡して描く。
  読み込みに失敗したときは `settings_gate.load_error` と同じ文言と再試行ボタンを出す。`task dev` で S09 を開いて目視で確かめる
- [ ] 4.3 `frontend/src/components/settings/settings-form.tsx` を更新し、`SHOPPING_REQUIRED_DAYS` と `SHOPPING_FALLBACK_DAYS` を消して、props の `constraints` を使う。
  `npm run type-check` と、既存の E2E `frontend/e2e/settings.spec.ts` が通ることで確かめる

## 5. 統合ダイアログの格子を `SpreadGrid` で作る

- [ ] 5.1 `frontend/src/components/mealplan/spread-batch-dialog.tsx` を更新し、`ListPeriod(source.date, 1)` を `SpreadGrid(source.id)` に置き換える。
  query key は `["mealplan", "spread-grid", source.id]` にする。`npm run type-check` で確かめる
- [ ] 5.2 `frontend/src/components/mealplan/spread-batch-dialog.tsx` の格子の行を `SpreadGrid` の `maxOffsetDays` から作り、`initialFutureDays` の prop と `futureDays` の state を消す (更新)。
  読み込み中は起点日の 1 行だけを描く。`npm run type-check` で確かめる
- [ ] 5.3 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、`spreadInitialFutureDays` の計算と、`SpreadBatchDialog` への `initialFutureDays` の受け渡しを消す。
  `spreadPresetDays` は残す。`npm run type-check` で確かめる
- [ ] 5.4 既存の E2E (`frontend/e2e/mealplan-batch.spec.ts` の describe「作り置きを広げる操作」「作り置きを広げる操作の取り消し」) が
  変更なしで通ることを `task test:e2e` で確かめる。spec の振る舞いは変わらないので、E2E の Scenario は足さない

## 6. ドキュメント

- [ ] 6.1 `docs/design.md` の §5「設定の保存と反映」に、「設定に結びついた規則 (消せない値、設定から導く範囲) はフロントに写さず Go から返す」を 1 段落足す。
  `git diff docs/design.md` で 1 か所だけの変更であることを確かめる
- [ ] 6.2 `docs/reviews/2026-10-19-design-ux-review.md` の R48 にチェックを入れ、proposal 名 `revise-settings-constraints` を添える。
  末尾の表の R17 の行に、R48 で写しが消えたことを書き足す。目視で確かめる

## 7. 確認

- [ ] 7.1 `go test ./internal/... ./cmd/...` を実行し、すべて通ることを確かめる
- [ ] 7.2 `cd frontend && npm run test` を実行し、すべて通ることを確かめる
- [ ] 7.3 `cd frontend && npm run type-check` を実行し、エラーが無いことを確かめる
- [ ] 7.4 `cd frontend && npm run build:dev` を実行し、ビルドが通ることを確かめる
- [ ] 7.5 `task test:e2e` を実行し、すべて通ることを確かめる
- [ ] 7.6 `task lint` を実行し、エラーが無いことを確かめる
