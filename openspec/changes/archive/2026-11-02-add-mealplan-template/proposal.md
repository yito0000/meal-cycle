## Why

献立がうまく回った週の組み方 (月〜水は作り置きの煮物、木金はカレー、など) を、次に使いたいときに覚えておく手段が無い。
週の候補 (add-mealplan-suggest-week-preset) は在庫と履歴から毎回作り直す案で、利用者が気に入った組み方を残すものではない。
同じ週を再現するには、S01 で 1 品ずつ割り当てて作り置きを広げ直す必要がある。

[design.md §9](../../../docs/design.md) は MVP 後の候補に「献立テンプレート (お気に入りの 1 週間パターンを保存)」を挙げている。
本 proposal は、S01 の 1 週間を名前を付けて保存し、別の週にまとめて割り当てられるようにする。

## What Changes

- S01 の下部に「テンプレート」メニューを置き、「この週をテンプレートとして保存」と「テンプレートから組む」を出す
  - 対象の週は、週の候補と同じく表示中の期間の最初の 1 週間
- 「この週をテンプレートとして保存」：名前 (1〜50 文字、ほかのテンプレートと重複不可) を入れて保存する
  - 週の全時間帯の品を、曜日、時間帯、セル内の順、レシピ、作り置きのつながりごと保存する
  - 週に品が 1 つも無いときは保存できない
- 「テンプレートから組む」：テンプレートの一覧のダイアログを開く
  - 各テンプレートに名前、品の数、曜日 × 時間帯の中身の一覧を出す
  - 「このテンプレートを適用」で、対象の週の各セルの末尾に品を足す。既存の品は消さない
  - 作り置きのつながりは保つ (テンプレートで同じ作り置きだった行は、適用後も同じ作り置きになる)
  - 削除済みのレシピの品は除いて適用し、除いた品の数を通知で知らせる。すべて削除済みなら適用できない
  - 適用したら、追加した品の数と「元に戻す」を含む通知を出す (週の候補の適用と同じ)
  - 「削除」で、確認のうえテンプレートを消す

### Non-goals

- **テンプレートの名前の変更と中身の編集**：中身を変えたいときは、S01 で週を組み直して別の名前で保存し、古いものを消す
- **1 週間以外の長さ (2 週、1 日) のテンプレート**
- **適用の前の上書きの選択 (既存の品を消して置き換える)**：既存の品を消すと「元に戻す」の範囲が広がる。消したいときは時間帯のクリアを先に行う
- **作った印の保存**：テンプレートは予定の組み方で、実績は含めない
- **テンプレートのファイルへの書き出しと取り込み**：DB の書き出し (add-settings-database-file) に含まれる

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：S01 に献立テンプレートの保存、適用、削除を足す

## Impact

**新設ファイル**

- `internal/storage/sqlite/migrations/00013_meal_plan_templates.sql`：`meal_plan_templates` と `meal_plan_template_items`
- `internal/domain/mealplan/template.go`：`Template` / `TemplateItem` / `TemplateApplyResult`、名前の検証、週の行からテンプレートの行を作る関数と、その逆の関数
- `internal/domain/mealplan/template_test.go`
- `internal/storage/sqlite/template_repo.go`
- `internal/storage/sqlite/template_repo_test.go`
- `frontend/src/components/mealplan/template-menu.tsx`
- `frontend/src/components/mealplan/save-template-dialog.tsx`
- `frontend/src/components/mealplan/template-list-dialog.tsx`
- `frontend/src/components/mealplan/use-templates.ts`
- `frontend/e2e/mealplan-template.spec.ts`
- `docs/wireframes/S01c_template_dialog.drawio.svg`

**更新ファイル**

- `internal/domain/mealplan/errors.go`：`ErrTemplateNameEmpty` / `ErrTemplateNameTooLong` / `ErrTemplateNameDuplicate` / `ErrTemplateEmptyWeek` / `ErrTemplateNotFound` / `ErrTemplateAllArchived`
- `internal/service/mealplan.go`：`SaveTemplate` / `ListTemplates` / `DeleteTemplate` / `ApplyTemplate` を足す
- `internal/service/mealplan_test.go`
- `main.go`：`NewMealPlanService` にテンプレートのリポジトリを渡す
- `cmd/e2e-server/main.go`：同上
- `frontend/e2e/bindings-shim/mealplanservice.ts`：4 つのメソッドを足す
- `frontend/src/routes/meal-plan.tsx`：下部にテンプレートのメニューを置く
- `frontend/src/components/mealplan/error-messages.ts`：テンプレートのエラーの文言を引く
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`
- `docs/product.md`：§4 S01 にテンプレートの節を足す
- `docs/design.md`：§4 に MealPlanTemplate を足す。§9 の献立テンプレートを実装済みに直す
- `docs/wireframes/S01_meal_calendar.v2.drawio.svg`：下部にテンプレートのメニューを足す
- `docs/wireframes/README.md`

**生成物 (再生成のみ)**

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`MealPlanService` のメソッドと `mealplan.Template` などの型が増える

**migration**

- `00013_meal_plan_templates.sql`：テンプレートと行の 2 表。`add-mealplan-suggest-by-time` の 00012 より後に実装する想定の番号で、
  先に実装するなら 00012 にし、後の change の番号を直す

**後続 proposal への引き継ぎ**

- 名前の変更と中身の編集が欲しくなったら、テンプレートの一覧のダイアログに操作を足す
- 週の候補のダイアログから案をテンプレートとして保存する導線を足すかは、両方の使われ方を見てから決める
- `add-mcp-server` にテンプレートの参照と適用のツールを足すかは、MCP の削除と書き換えの方針 (取り消しの仕組み) と一緒に決める
