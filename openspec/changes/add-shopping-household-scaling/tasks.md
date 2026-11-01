## 1. 設定の値

- [ ] 1.1 `internal/config/defaults.go` の `Config` に `HouseholdServings int` を足し、`Defaults()` で 0 にする。完了判定: `go build ./internal/...` が通る
- [ ] 1.2 `internal/config/defaults_test.go` に `HouseholdServings` の既定値 0 を足す。完了判定: `go test ./internal/config/` が pass する
- [ ] 1.3 `internal/domain/settings/settings.go` の `Values` / `FromConfig` / `ApplyTo` に `HouseholdServings` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 1.4 `internal/domain/settings/errors.go` に `ErrHouseholdServingsOutOfRange` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 1.5 `internal/domain/settings/validate.go` で 0〜10 を検証する。完了判定: `go build ./internal/...` が通る
- [ ] 1.6 `internal/domain/settings/settings_test.go` に検証のケースを足す (table-driven)。完了判定: `go test ./internal/domain/settings/` が pass する
  - 0 は通る
  - 10 は通る
  - -1 は ErrHouseholdServingsOutOfRange
  - 11 は ErrHouseholdServingsOutOfRange
- [ ] 1.7 `internal/service/settings.go` の `settingKeys` に `householdServings` を足す。完了判定: `go build ./internal/...` が通る

## 2. 換算 (ドメイン層)

- [ ] 2.1 `internal/domain/shopping/list.go` の `PlannedBatch` に `Servings int` と `PairRowCount int` を、`List` に `ScaledServings int` を足す。完了判定: `go build ./internal/...` が通る
- [ ] 2.2 `internal/domain/shopping/scale.go` を新設し、`ScaledAmount(amount float64, servings, household, pairRowCount int) (float64, bool)` を書く (design.md D1 / D2)。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.3 `internal/domain/shopping/scale_test.go` を新設する (table-driven)。完了判定: `go test ./internal/domain/shopping/ -run TestScaledAmount` が pass する
  - household が 0 なら量をそのまま返し、換算していないことを返す
  - pairRowCount が 1 なら amount × household / servings を返す
  - pairRowCount が 2 なら量をそのまま返す
  - amount が 0 なら 0 を返し、換算していないことを返す
- [ ] 2.4 `internal/domain/shopping/aggregate.go` の `Aggregate` に `household int` を足し、量る単位の判定のあと合算の前に `ScaledAmount` を通す。
  換算した材料があったかも返す。完了判定: `go build ./internal/...` が通る
- [ ] 2.5 `internal/domain/shopping/aggregate_test.go` を更新し、既存の呼び出しに household 0 を渡し、換算のケースを足す。完了判定: `go test ./internal/domain/shopping/` が pass する
  - 1 回きりの組と作り置きの組の同じ材料を合算すると、1 回きりの分だけ換算される
  - 量る単位の材料は換算されない

## 3. リポジトリ層とサービス層

- [ ] 3.1 `internal/storage/sqlite/shopping_repo.go` の `loadPlannedGroups` で `recipes.servings` と、同じ `(batch_id, recipe_id)` の `meal_plans` 全体の行数を読む。
  完了判定: `go build ./internal/...` が通る
- [ ] 3.2 `internal/storage/sqlite/shopping_repo_test.go` にケースを足す。完了判定: `go test ./internal/storage/sqlite/ -run TestShoppingRepo` が pass する
  - 期間外に同じ組の行があるとき PairRowCount はそれを含む
  - 同じ batch_id に別のレシピの行があっても PairRowCount に含まない
- [ ] 3.3 `internal/service/shopping.go` で `cfg.HouseholdServings` を `Aggregate` に渡し、`List.ScaledServings` を埋める。完了判定: `go build ./internal/...` が通る
- [ ] 3.4 `internal/service/shopping_test.go` に `TestShoppingService_List` のケースを足す。完了判定: `go test ./internal/service/ -run TestShoppingService_List` が pass する
  - should scale a single-row dish by household servings when household servings is saved
  - should not scale a batch with multiple rows when household servings is saved
  - should set scaledServings to zero when nothing is scaled
- [ ] 3.5 `wails3 generate bindings -clean=true -ts -i` で bindings を作り直す。完了判定: `cd frontend && npm run type-check` が通る

## 4. 画面

- [ ] 4.1 `frontend/src/components/settings/settings-form.tsx` の買い物リストの節に世帯人数の select (換算しない / 1〜10 人) を足す。完了判定: `task dev` の S09 で目視
- [ ] 4.2 `frontend/src/components/settings/error-messages.ts` に `household servings out of range` の文言を足す。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.3 `frontend/src/routes/shopping.tsx` で `scaledServings` が 0 でないとき、対象期間の下に注記を出す。完了判定: `task dev` の S07 で目視
- [ ] 4.4 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に文言を足す。完了判定: `cd frontend && npm run type-check` が通る

## 5. E2E

- [ ] 5.1 `frontend/e2e/shopping-household-scaling.spec.ts` を新設する。最外の describe は「世帯人数による必要量の換算」にする。完了判定: `task test:e2e` で pass する
  - 1 回きりの献立を世帯人数で換算する
  - 作り置きは換算しない
  - 換算しないときはレシピの量のまま
- [ ] 5.2 `frontend/e2e/settings.spec.ts` に describe「S09 設定画面の表示」のテストを足す。完了判定: `task test:e2e` で pass する
  - 世帯人数の既定は換算しない

## 6. ドキュメント

- [ ] 6.1 `docs/wireframes/S09_settings.drawio.svg` の買い物リストの節に世帯人数の欄を足す (可視 SVG と編集 XML の両方)。完了判定: 目視
- [ ] 6.2 `docs/product.md` の §1 可変値一覧の世帯人数の行を S09 設定 (`HouseholdServings`) と ✓ に直す。完了判定: 目視
- [ ] 6.3 `docs/product.md` の §4 S07 に換算の規則と注記を、S09 に世帯人数の欄を足す。完了判定: 目視
- [ ] 6.4 `docs/design.md` の §9 の世帯人数による換算を実装済みに直す。完了判定: 目視

## 7. 確認

- [ ] 7.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [ ] 7.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [ ] 7.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [ ] 7.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [ ] 7.5 `task test:e2e` を実行する。完了判定: 全 pass
- [ ] 7.6 `task lint` を実行する。完了判定: エラー 0
