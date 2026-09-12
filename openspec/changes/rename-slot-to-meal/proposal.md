## Why

`slot` は「時間の帯」を指す engineering 語彙で、日本語 UI では既に「時間帯」(Step 1: rename-slot-to-japanese)
に置き換えたが、英語話者にとっても "3 meals a day / evening meal" が自然で、Slot は「レシピを差し込む
枠」以上の意味を持たない。朝食 / 昼食 / 夕食 を包む上位概念としては **meal** が最も近い。engineering
語彙 (Go 型 / DB カラム / Wails bindings / i18n key / en.json value) を meal に統一することで、UI と
エンジニアの語彙のずれを解消する。

## What Changes

- **Go**: `mealplan.Slot` 型を `Meal` に。`SlotBreakfast/SlotLunch/SlotDinner` → `MealBreakfast/…`。
  `ErrInvalidSlot` → `ErrInvalidMeal`、ドメインメソッドの引数名も追随
- **SQLite**: `meal_plans.slot` カラム → `meal_plans.meal`。既存インデックス `(date, slot, position)` の
  張り直し。goose マイグレーション 1 本追加 (未リリースなので down は不要、up のみ)
- **Wails サービス**: メソッドシグネチャの `slot string` 引数 → `meal string`。bindings 再生成
- **Frontend TS**: bindings 追随、`SlotVisibilityCheckboxes` → `MealVisibilityCheckboxes` 等の変数 /
  コンポーネント名も meal 系に。E2E shim も同時に
- **i18n key**: `pages.meal_plan.slot*` → `pages.meal_plan.meal*` (親子で meal が 2 回並ぶ違和感を
  design.md で `mealtime` 案と比較し決める)
- **en.json value**: "Slot menu" → "Meal menu"、"Clear this slot" → "Clear this meal"、
  "Invalid slot" → "Invalid meal" 等
- **ja.json value**: 「時間帯」のまま維持 (Step 1 で確定)
- **`openspec/specs/mealplan/spec.md`**: 全体を meal 語彙 (spec 本文の英字) / 時間帯 (spec 本文の日本語)
  に更新。Step 1 の delta が 1 Requirement しか触っていない残余もここで一括 cover
- **ドキュメント**: `docs/product.md` / `docs/design.md` / `docs/scenarios/mealplan.feature` の残 "slot"
  (バッククォート囲みのコード引用含む) を meal に。ワイヤーフレーム内 label は再生成コストのため触らない
- **BREAKING** (対内部): 既存の DB (`meal_plans.slot` カラム) は goose マイグレーションで自動リネームされる。
  WSL / Windows の開発 DB は自動で追随。未リリース段階なので下位互換は取らない

## Non-goals

- 日本語 UI 側の再翻訳 (時間帯 のまま維持)
- 「作り置き / 別調理」等、他の語彙の見直し
- ドメインの意味変更 (「1 (date, meal) = 0..N 品」の構造・不変条件は同じ)
- 未リリース段階での旧カラム保持や 2 段階マイグレーション — 単純 rename column 1 本で完結させる

## Capabilities

### Modified Capabilities

- `mealplan`: 「slot」語彙を含む Requirement 群を「時間帯」(日本語文) と「meal」(コード / SQL 引用) に統一。
  Requirement のふるまいは不変、Scenario の中身も既存動作と同じ (テキスト書き換えのみ)

## Impact

- **新設ファイル**:
  - `internal/storage/sqlite/migrations/0002_rename_slot_to_meal.sql` (goose マイグレーション、up のみ)
- **更新ファイル**:
  - `internal/domain/mealplan/*.go` (型 `Slot` → `Meal`、定数、`ErrInvalidSlot` → `ErrInvalidMeal`、
    バリデーション、`entry.go` / `slot.go`(→`meal.go` にファイル名変更) 等)
  - `internal/storage/sqlite/mealplan_repo.go` (`slot` カラム参照を `meal` に)
  - `internal/service/mealplan.go` (メソッドシグネチャ `slot string` → `meal string`)
  - `frontend/bindings/**` (`wails3 generate bindings` で再生成)
  - `frontend/src/components/mealplan/*.tsx` / `*.ts` (bindings 呼び出し、変数 / コンポーネント名の
    `Slot*` → `Meal*` リネーム)
  - `frontend/src/locales/en.json` (key: `slot*` → `meal*` (or `mealtime*`)、value: "Slot" → "Meal")
  - `frontend/src/locales/ja.json` (key リネームのみ、value は「時間帯」を維持)
  - `frontend/e2e/bindings-shim/mealplanservice.ts` / `frontend/e2e/mealplan-assign.spec.ts` (追随)
  - `openspec/specs/mealplan/spec.md` (Step 1 未 sync 分含む全体更新)
  - `docs/product.md` (`(date, slot)` などのバッククォート引用を `(date, meal)` に)
  - `docs/design.md` (§4 MealPlan の "slot" 記述を全面改訂)
  - `docs/scenarios/mealplan.feature` (バッククォート引用や技術文脈での残 "slot")
- **生成物 (再生成のみ)**: `frontend/bindings/` (`wails3 generate bindings`)
- **migration**: `0002_rename_slot_to_meal.sql` を追加。up は `ALTER TABLE meal_plans RENAME COLUMN slot TO meal`
  + 関連インデックス張り直し。既存の一意でない `(date, slot, position)` 系インデックスを再作成する
- **開発者への影響**:
  - `task dev` を叩き直すと goose が自動 migrate。既存の献立データはカラム名変更のみで内容は保持される
  - Windows 側 `%APPDATA%\meal-cycle\meal-cycle.db` も同様 (未リリースなので自動 migration を許容)

### 後続 proposal への引き継ぎ

- なし。以降の proposal では meal / 時間帯 を統一語彙として使う。この change 以降、「slot」は
  ソース中に一切出現させない (grep で `slot` が 0 件になることが最終ゲート)
