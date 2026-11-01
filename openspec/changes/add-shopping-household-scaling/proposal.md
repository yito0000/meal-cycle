## Why

S07 の必要量は、レシピに書いた量をそのまま合算している。
レシピの人数 (`servings`) と食べる人数が違うと、買う量の目安がずれる。
2 人分で登録したレシピを 4 人家族で作るとき、利用者は S07 の量を頭の中で 2 倍してから買い物に出ることになる。

[product.md §1](../../../docs/product.md) の可変値一覧は「世帯人数（買い物リストの量の換算）」を「人数換算の proposal で S09 に足す」としており、
[design.md §9](../../../docs/design.md) は「作り置きでは換算しないほうが合う場面もあり、換算の規則は proposal で決める」としている。
本 proposal で換算の規則を決め、S09 に世帯人数の欄を足す。

## What Changes

- S09 の買い物リストの節に「世帯人数」を足す。「換算しない」(既定) か 1〜10 人を選ぶ
- 世帯人数を選んでいるとき、S07 は 1 回きりの献立の材料を「世帯人数 ÷ レシピの人数」倍して合算する
  - 1 回きりの献立：同じ作り置きの同じレシピの行が、献立全体で 1 行だけのもの
  - 作り置き (同じ作り置きの同じレシピが 2 行以上) は換算しない。レシピの量で何日分かを作る前提で、人数で掛けると日数分の量を見誤るため
  - 量を持たない材料 (量 0 と量る単位の材料) は換算しない
- 換算した材料が 1 件でもあるとき、S07 の対象期間の下に「1 回きりの献立は N 人分に換算しています」と出す
- 換算した量は、材料テーブル、在庫の状態の判定、テキストでコピー、冷蔵庫への一括登録の初期値に同じく使う

### Non-goals

- **作り置きの換算**：作り置きの量の決め方 (何人で何日) は、レシピの人数だけでは決まらない。必要になったら作り置きごとに人数を持たせる proposal にする
- **作った印での在庫の消費の換算**：「使った在庫を減らす」ダイアログの初期値はレシピの量のまま。量を直してから減らせるので、まずは買い物リストだけに入れる
- **S03 / S04 での人数の換算表示**：レシピの表示はレシピに書いた量のまま
- **単位ごとの丸め (個は整数に切り上げる等)**：単位は自由テキストで、丸めの規則を一般化できない。既存の合算と同じ桁で丸める

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `shopping`：1 回きりの献立の材料を世帯人数で換算する
- `settings`：S09 の買い物リストの節に世帯人数を足し、検証の規則に範囲を足す

## Impact

**新設ファイル**

- `internal/domain/shopping/scale.go`：`(batch, recipe)` 1 組の材料を世帯人数で換算する関数
- `internal/domain/shopping/scale_test.go`
- `frontend/e2e/shopping-household-scaling.spec.ts`

**更新ファイル**

- `internal/config/defaults.go`：`HouseholdServings int` (既定 0 = 換算しない) を足す
- `internal/config/defaults_test.go`：既定値 0 を足す
- `internal/domain/settings/settings.go` (Values / FromConfig / ApplyTo)：`HouseholdServings` を足す
- `internal/domain/settings/validate.go` と `errors.go`：`ErrHouseholdServingsOutOfRange` (0〜10) を足す
- `internal/service/settings.go`：`settingKeys` に `householdServings` を足す
- `internal/domain/shopping/list.go`：`PlannedBatch` に `Servings int` と `PairRowCount int` を、`List` に `ScaledServings int` を足す
- `internal/domain/shopping/aggregate.go`：`Aggregate` の引数に世帯人数を足し、合算の前に `scale.go` で換算する
- `internal/storage/sqlite/shopping_repo.go`：`loadPlannedGroups` で `recipes.servings` と、同じ `(batch_id, recipe_id)` の献立全体の行数を読む
- `internal/service/shopping.go`：`cfg.HouseholdServings` を `Aggregate` に渡し、`List.ScaledServings` を埋める
- `internal/service/shopping_test.go` と `internal/domain/shopping/aggregate_test.go`：換算のテストを足す
- `frontend/src/components/settings/settings-form.tsx`：買い物リストの節に世帯人数の select を足す
- `frontend/src/components/settings/error-messages.ts`：新しいエラーの文言を引く
- `frontend/src/routes/shopping.tsx`：対象期間の下に換算の注記を出す
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`
- `docs/product.md`：§1 可変値一覧の世帯人数の行を ✓ にする。§4 S07 と S09 に換算の節を足す
- `docs/design.md`：§9 の世帯人数による換算を実装済みに直す
- `docs/wireframes/S09_settings.drawio.svg`：買い物リストの節に世帯人数の欄を足す

**生成物 (再生成のみ)**

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`settings.Values` と `shopping.List` の型が変わる

**migration**

- なし (`settings` テーブルは key / value なので列は増えない。レシピの `servings` は既存の列)

**後続 proposal への引き継ぎ**

- 作り置きを換算するなら、作り置きごとに「何人分 × 何日」を持たせる (meal_plans への列の追加) かを決める
- 作った印での在庫の消費 (`FridgeService.PlanCookConsumption`) に同じ換算を入れるかを、使ってみてから決める
- `add-mcp-server` の `get_shopping_list` は `List` をそのまま返すので、換算後の量と `scaledServings` がエージェントにも見える。説明文に一言足す
