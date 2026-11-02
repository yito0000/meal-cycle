## 1. スキーマとドメイン層

- [x] 1.1 `internal/storage/sqlite/migrations/00013_meal_plan_templates.sql` を新設し、design.md D1 の 2 表を作る (Down で 2 表を落とす)。
  完了判定: `go test ./internal/storage/sqlite/ -run TestMigrate` が pass する
- [x] 1.2 `internal/domain/mealplan/errors.go` に `ErrTemplateNameEmpty` / `ErrTemplateNameTooLong` / `ErrTemplateNameDuplicate` / `ErrTemplateEmptyWeek` /
  `ErrTemplateNotFound` / `ErrTemplateAllArchived` を足す。完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/domain/mealplan/template.go` を新設し、`Template` / `TemplateItem` / `TemplateApplyResult` と `ValidateTemplateName` を書く。
  完了判定: `go build ./internal/...` が通る
- [x] 1.4 `internal/domain/mealplan/template.go` に、週の行からテンプレートの行を作る `TemplateItemsFromWeek(weekStart, entries)` を書く (batch_id を出現順の group_no に振り直す)。
  完了判定: `go build ./internal/...` が通る
- [x] 1.5 `internal/domain/mealplan/template.go` に、テンプレートの行から `PlanItem` を作る `PlanItemsFromTemplate(weekStart, items, archived)` を書く。
  削除済みの行を除き、除いた数を返す。完了判定: `go build ./internal/...` が通る
- [x] 1.6 `internal/domain/mealplan/template_test.go` を新設する (table-driven)。完了判定: `go test ./internal/domain/mealplan/ -run 'TestValidateTemplateName|TestTemplateItems|TestPlanItemsFromTemplate'` が pass する
  - 空白だけの名前は ErrTemplateNameEmpty
  - 51 文字の名前は ErrTemplateNameTooLong
  - 同じ batch_id の行に同じ group_no を振る
  - 週の外に同じ batch_id の行があっても週の中の行だけを返す
  - day_offset は週の最初の日からの日数
  - PlanItem は (day_offset, meal, position) の順に並ぶ
  - 削除済みのレシピの行を除いて除いた数を返す

## 2. リポジトリ層

- [x] 2.1 `internal/storage/sqlite/template_repo.go` を新設し、`Create` / `List` / `Get` / `Delete` / `ExistsName` を書く。完了判定: `go build ./internal/...` が通る
- [x] 2.2 `internal/storage/sqlite/template_repo_test.go` を新設する。完了判定: `go test ./internal/storage/sqlite/ -run TestTemplateRepo` が pass する
  - 保存したテンプレートを行ごと読み戻せる
  - List は作成日時の新しい順に返し、各テンプレートの行を含む
  - Delete で行も消える
  - ExistsName は前後の空白を除いて比べる

## 3. サービス層

- [x] 3.1 `internal/service/mealplan.go` の `MealPlanService` にテンプレートのリポジトリを持たせ、`NewMealPlanService` の引数に足す。完了判定: `go build ./internal/...` が通る
- [x] 3.2 `internal/service/mealplan.go` に `SaveTemplate(ctx, weekStart, name)` を書く。完了判定: `go build ./internal/...` が通る
- [x] 3.3 `internal/service/mealplan.go` に `ListTemplates(ctx)` を書き、各行にレシピ名と削除済みかを添える。完了判定: `go build ./internal/...` が通る
- [x] 3.4 `internal/service/mealplan.go` に `DeleteTemplate(ctx, id)` を書く。完了判定: `go build ./internal/...` が通る
- [x] 3.5 `internal/service/mealplan.go` に `ApplyTemplate(ctx, id, weekStart)` を design.md D2 のとおり書く。完了判定: `go build ./internal/...` が通る
- [x] 3.6 `main.go` と `cmd/e2e-server/main.go` の `NewMealPlanService` にテンプレートのリポジトリを渡す。完了判定: `go build . ./cmd/...` が通る
- [x] 3.7 `internal/service/mealplan_test.go` にテストを足す。完了判定: `go test ./internal/service/ -run 'TestMealPlanService_SaveTemplate|TestMealPlanService_ApplyTemplate'` が pass する
  - should save all meals of the week with batch groups when saving a template
  - should fail with ErrTemplateEmptyWeek when the week has no entries
  - should fail with ErrTemplateNameDuplicate when the name already exists
  - should append entries to the end of each cell when applying a template
  - should keep batch groups when applying a template
  - should skip archived recipes and report the count when applying a template
  - should fail with ErrTemplateAllArchived when every recipe is archived
  - should restore the previous plan when the apply result is undone
  - should keep applied entries when the template is deleted
- [x] 3.8 `wails3 generate bindings -clean=true -ts -i` で bindings を作り直す。完了判定: `cd frontend && npm run type-check` が通る

## 4. 画面

- [x] 4.1 `frontend/e2e/bindings-shim/mealplanservice.ts` に 4 つのメソッドを足す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/src/components/mealplan/use-templates.ts` を新設する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/components/mealplan/save-template-dialog.tsx` を新設する。変換確定の Enter では保存しない。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.4 `frontend/src/components/mealplan/template-list-dialog.tsx` を新設し、一覧、適用、削除の確認を描く。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.5 `frontend/src/components/mealplan/template-menu.tsx` を新設する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.6 `frontend/src/routes/meal-plan.tsx` の下部にテンプレートのメニューを置き、適用の通知に「元に戻す」を付ける (`UndoSpread` を呼ぶ)。完了判定: `task dev` の S01 で目視
- [x] 4.7 `frontend/src/components/mealplan/error-messages.ts` にテンプレートのエラーを足す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.8 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に文言を足す。完了判定: `cd frontend && npm run type-check` が通る

## 5. E2E

- [x] 5.1 `frontend/e2e/mealplan-template.spec.ts` を新設する。完了判定: `task test:e2e` で pass する
  - describe「献立テンプレートの保存」
    - 週の献立をテンプレートとして保存する
    - 品の無い週は保存できない
    - 同じ名前のテンプレートは保存できない
  - describe「献立テンプレートの適用」
    - テンプレートを別の週に適用する
    - 適用を元に戻すと割り当てた品が消える
    - 削除済みのレシピの品は除いて適用する
  - describe「献立テンプレートの削除」
    - テンプレートを削除する
- 「既存の品は残して末尾に足す」は 3.7 のサービス層のテストで確かめる

## 6. ドキュメント

- [x] 6.1 `docs/wireframes/S01c_template_dialog.drawio.svg` を新設する。完了判定: 目視
- [x] 6.2 `docs/wireframes/S01_meal_calendar.v2.drawio.svg` の下部にテンプレートのメニューを足し、`docs/wireframes/README.md` に S01c を足す。完了判定: 目視
- [x] 6.3 `docs/product.md` の §4 S01 にテンプレートの節を足す。完了判定: 目視
- [x] 6.4 `docs/design.md` の §4 に MealPlanTemplate を足し、§6 のディレクトリ構成に `template_repo.go` を足し、§9 の献立テンプレートを実装済みに直す。完了判定: 目視

## 7. 確認

- [x] 7.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [x] 7.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [x] 7.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [x] 7.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [x] 7.5 `task test:e2e` を実行する。完了判定: 全 pass
- [x] 7.6 `task lint` を実行する。完了判定: エラー 0
