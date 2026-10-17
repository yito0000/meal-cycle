## 1. 既定値とドメイン

- [x] 1.1 `internal/config/defaults.go` (更新) の `Config` に `Language string` (`json:"language"`) を足し、`Defaults()` で `"ja"` を返す (design.md D7)。
  完了判定: `go build .` が通る
- [x] 1.2 `internal/config/defaults_test.go` (更新) の `cases` に `{"Language", c.Language, "ja"}` を足す。完了判定: `go test ./internal/config/` が pass する
- [x] 1.3 `internal/domain/settings/settings.go` (新設) に `Values` 構造体を書く (design.md D4)
    - フィールド: `Language` / `ExpiryAlertRedDays` / `ExpiryAlertYellowDays` / `LeftoverCopyPresets` / `ShoppingPeriodDayPresets` /
      `ShoppingDefaultPeriodDays` / `MeasuringUnits` / `UnquantifiedWords` / `RecipeImportDefaultServings`
    - JSON タグは `config.Config` の同名フィールドと同じ camelCase にする
    - 完了判定: `go build .` が通る
- [x] 1.4 `internal/domain/settings/settings.go` (更新) に `FromConfig(config.Config) Values` と `(Values) ApplyTo(config.Config) config.Config` を書く。
  完了判定: `go build .` が通る
- [x] 1.5 `internal/domain/settings/settings.go` (更新) に `(Values) Normalize() Values` を書く
    - 単位と言葉は前後の空白を除く
    - 日数の候補は小さい順に並べる
    - 完了判定: `go build .` が通る
- [x] 1.6 `internal/domain/settings/errors.go` (新設) に検証エラーの sentinel を書く (`settings: ` で始まるメッセージ、design.md D5)
    - `ErrInvalidLanguage` / `ErrRedDaysOutOfRange` / `ErrYellowDaysOutOfRange` / `ErrRedNotBelowYellow`
    - `ErrLeftoverPresetsCount` / `ErrLeftoverPresetOutOfRange` / `ErrLeftoverPresetDuplicate`
    - `ErrShoppingPresetsCount` / `ErrShoppingPresetOutOfRange` / `ErrShoppingPresetDuplicate` / `ErrShoppingPresetRequiredMissing` /
      `ErrShoppingDefaultNotInPresets`
    - `ErrMeasuringUnitsCount` / `ErrMeasuringUnitLength` / `ErrMeasuringUnitDuplicate`
    - `ErrUnquantifiedWordsCount` / `ErrUnquantifiedWordLength` / `ErrUnquantifiedWordDuplicate`
    - `ErrImportServingsOutOfRange`
    - 完了判定: `go build .` が通る
- [x] 1.7 `internal/domain/settings/validate.go` (新設) に上限下限の定数を書く (🟡 60 日、作り置き 2〜7 日で 1〜5 個、買い物 1〜28 日で 8 個以下、
  必須の 7 / 14 / 28、単位 10 個、言葉 20 個、各 1〜20 文字、人数 1〜99)。完了判定: `go build .` が通る
- [x] 1.8 `internal/domain/settings/validate.go` (更新) に key 単位の検証関数を書く (design.md D2)
    - `ValidateLanguage` / `ValidateExpiryAlert` / `ValidateLeftoverCopyPresets` / `ValidateShoppingPeriod` /
      `ValidateMeasuringUnits` / `ValidateUnquantifiedWords` / `ValidateImportServings`
    - 文字数は rune で数える
    - 完了判定: `go build .` が通る
- [x] 1.9 `internal/domain/settings/validate.go` (更新) に、1.8 の関数を順に呼んで最初のエラーを返す `(Values) Validate() error` を書く。
  完了判定: `go build .` が通る
- [x] 1.10 `internal/domain/settings/settings_test.go` (新設) に table-driven test を書く
    - `FromConfig(config.Defaults())` を `ApplyTo(config.Defaults())` に戻すと元の `Config` と一致する
    - `Normalize` は単位の前後の空白を除く
    - `Normalize` は日数の候補を小さい順に並べる
    - 既定値の `Values` は `Validate` を通る
    - 言語が `"fr"` なら `ErrInvalidLanguage`
    - 🔴 が -1 なら `ErrRedDaysOutOfRange`
    - 🟡 が 61 なら `ErrYellowDaysOutOfRange`
    - 🔴 7 / 🟡 7 なら `ErrRedNotBelowYellow`
    - 🔴 0 / 🟡 1 なら通る
    - 作り置きの候補が空なら `ErrLeftoverPresetsCount`
    - 作り置きの候補が 6 個なら `ErrLeftoverPresetsCount`
    - 作り置きの候補に 1 または 8 があれば `ErrLeftoverPresetOutOfRange`
    - 作り置きの候補に 3 が 2 個あれば `ErrLeftoverPresetDuplicate`
    - 買い物の候補に 14 が無ければ `ErrShoppingPresetRequiredMissing`
    - 買い物の候補に 29 があれば `ErrShoppingPresetOutOfRange`
    - 買い物の候補が 9 個なら `ErrShoppingPresetsCount`
    - 買い物の既定日数 5 が候補に無ければ `ErrShoppingDefaultNotInPresets`
    - 量る単位に 21 文字の単位があれば `ErrMeasuringUnitLength`
    - 量る単位に空文字があれば `ErrMeasuringUnitLength`
    - 量る単位に「大さじ」が 2 個あれば `ErrMeasuringUnitDuplicate`
    - 量る単位が 11 個なら `ErrMeasuringUnitsCount`
    - 量る単位が 0 個なら通る
    - 量を持たない言葉が 21 個なら `ErrUnquantifiedWordsCount`
    - 人数が 0 または 100 なら `ErrImportServingsOutOfRange`
    - 完了判定: `go test ./internal/domain/settings/` が pass する

## 2. 保存先 (リポジトリ層)

- [x] 2.1 `internal/storage/sqlite/migrations/00010_settings.sql` (新設) で `settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TEXT NOT NULL)` を作り、
  Down で消す (design.md D1)。完了判定: `go test ./internal/storage/sqlite/ -run Migrate` が pass する
- [x] 2.2 `internal/storage/sqlite/settings_repo.go` (新設) に `SettingsRepo` と `NewSettingsRepo(db *sql.DB)` を書く。
  既存の `stock_repo.go` を読んで規約をそろえる。完了判定: `go build .` が通る
- [x] 2.3 `internal/storage/sqlite/settings_repo.go` (更新) に `Load(ctx) (map[string]json.RawMessage, error)` を書く。完了判定: `go build .` が通る
- [x] 2.4 `internal/storage/sqlite/settings_repo.go` (更新) に `ReplaceAll(ctx, rows map[string]json.RawMessage, now time.Time) error` を書く。
  1 トランザクションで全行を消してから入れ直す。完了判定: `go build .` が通る
- [x] 2.5 `internal/storage/sqlite/settings_repo.go` (更新) に `DeleteAll(ctx) error` を書く。完了判定: `go build .` が通る
- [x] 2.6 `internal/storage/sqlite/settings_repo_test.go` (新設) に実 SQLite のテストを書く (`t.TempDir()`)
    - 空の DB では `Load` が空の map を返す
    - `ReplaceAll` で入れた行を `Load` で読める
    - `ReplaceAll` を 2 回呼ぶと、2 回目に含まれない key は消える
    - `DeleteAll` のあとは `Load` が空の map を返す
    - 完了判定: `go test ./internal/storage/sqlite/ -run Settings` が pass する

## 3. サービス層

- [x] 3.1 `internal/service/settings.go` (新設) に非公開 interface `configSource { current() config.Config }` を書く (design.md D3)。
  完了判定: `go build .` が通る
- [x] 3.2 `internal/service/settings.go` (更新) に、`map[string]json.RawMessage` を key ごとに解釈と検証をして `config.Config` に重ねる `loadConfig` を書く
    - 解釈と検証に失敗した key と未知の key は捨て、その項目は既定値にする (design.md D6)
    - 完了判定: `go build .` が通る
- [x] 3.3 `internal/service/settings.go` (更新) に、`Values` と `Defaults()` を比べて既定値から変わった key だけを JSON にする `diffRows` を書く (design.md D2)。
  完了判定: `go build .` が通る
- [x] 3.4 `internal/service/settings.go` (更新) に `SettingsService` (`repo` / `mu sync.RWMutex` / `cfg config.Config` / `now`) と `NewSettingsService(db *sql.DB)` を書く
    - 作成時に `Load` → `loadConfig` で現在値を作る
    - `Load` が失敗したら既定値で動かし、`log.Printf` で残す
    - `current()` を実装する
    - 完了判定: `go build .` が通る
- [x] 3.5 `internal/service/settings.go` (更新) に `Get(ctx) (settings.Values, error)` を書く。完了判定: `go build .` が通る
- [x] 3.6 `internal/service/settings.go` (更新) に `Save(ctx, values settings.Values) (settings.Values, error)` を書く
    - `Normalize` → `Validate` → `diffRows` → `ReplaceAll` の順に呼ぶ
    - 書き込みに成功してから現在値を差し替える
    - 完了判定: `go build .` が通る
- [x] 3.7 `internal/service/settings.go` (更新) に `Reset(ctx) (settings.Values, error)` を書く。`DeleteAll` に成功してから現在値を既定値に戻す。
  完了判定: `go build .` が通る
- [x] 3.8 `internal/service/fridge.go` (更新) の `NewFridgeService` に `cfg configSource` 引数を足し、`s.cfg.X` を `s.cfg.current().X` に置き換える。
  完了判定: `go build ./internal/...` が通る
- [x] 3.9 `internal/service/mealplan.go` (更新) の `NewMealPlanService` と `newMealPlanServiceWithRepo` に `cfg configSource` 引数を足し、参照を `current()` 経由にする。
  完了判定: `go build ./internal/...` が通る
- [x] 3.10 `internal/service/recipe.go` (更新) の `NewRecipeService` に `cfg configSource` 引数を足し、参照を `current()` 経由にする。完了判定: `go build ./internal/...` が通る
- [x] 3.11 `internal/service/shopping.go` (更新) の `NewShoppingService` に `cfg configSource` 引数を足し、参照を `current()` 経由にする。完了判定: `go build ./internal/...` が通る
- [x] 3.12 `internal/service/settings_test.go` (新設) に、`config.Config` をそのまま返すテスト用の `staticConfig` 型を書く。完了判定: `go vet ./internal/service/` が通る
- [x] 3.13 `internal/service/fridge_test.go` / `mealplan_test.go` / `recipe_test.go` / `shopping_test.go` (更新) のコンストラクタ呼び出しに
  `staticConfig(config.Defaults())` を渡す。完了判定: `go test ./internal/service/` が pass する
- [x] 3.14 `internal/service/settings_test.go` (更新) に BDD 風のテスト `TestSettingsService_Get` を書く
    - should return default values when nothing is saved
    - should fall back to the default only for a key whose saved value cannot be decoded
    - should fall back to the default for a key whose saved value fails validation
    - should ignore unknown keys
    - 完了判定: `go test ./internal/service/ -run TestSettingsService_Get` が pass する
- [x] 3.15 `internal/service/settings_test.go` (更新) に `TestSettingsService_Save` を書く
    - should persist values so that a new service on the same database returns them
    - should store only keys that differ from the defaults
    - should reject empty leftover copy presets and keep the previous values
    - should reject red days not below yellow days and keep the previous values
    - should return normalized values with sorted presets and trimmed units
    - 完了判定: `go test ./internal/service/ -run TestSettingsService_Save` が pass する
- [x] 3.16 `internal/service/settings_test.go` (更新) に `TestSettingsService_Reset` を書く
    - should return default values and remove all saved rows
    - 完了判定: `go test ./internal/service/ -run TestSettingsService_Reset` が pass する
- [x] 3.17 `internal/service/settings_test.go` (更新) に、保存した値が他サービスへ反映されることのテスト `TestSettingsService_Propagation` を書く
    - should evaluate fridge expiry with the saved red days without restarting
    - should accept a leftover copy of days added to the presets and reject removed days
    - should accept a shopping period of days added to the presets
    - should import an unquantified word added to the settings as amount 0 with the word as unit
    - should import a recipe without servings using the saved default servings
    - should treat a measuring unit added to the settings as unquantified in the shopping list
    - 完了判定: `go test ./internal/service/ -run TestSettingsService_Propagation` が pass する

## 4. 組み立てと E2E サーバ

- [x] 4.1 `main.go` (更新) で `settingsSvc := service.NewSettingsService(db)` を先に作り、他の 4 サービスに渡す。
  `application.Options.Services` に `SettingsService` を足す。完了判定: `go build .` が通る
- [x] 4.2 `cmd/e2e-server/main.go` (更新) の `newRegistry` で `SettingsService` を作って他サービスに渡し、`"SettingsService"` を足す (6 サービス)。
  `main` の `recipeSvc` の作り方も合わせる。完了判定: `go build ./cmd/...` が通る
- [x] 4.3 `cmd/e2e-server/dispatch.go` (更新) の `serviceRegistry` のコメントのサービス数を直す。完了判定: 差分を目視で確認する
- [x] 4.4 `cmd/e2e-server/dispatch_test.go` (更新) の登録数の突き合わせとコンストラクタ呼び出しを 6 サービスに合わせる。
  完了判定: `go test ./cmd/...` が pass する
- [x] 4.5 `wails3 generate bindings -clean=true -ts -i` で `frontend/bindings/` を再生成する。
  完了判定: `frontend/bindings/github.com/yito0000/meal-cycle/internal/service/` に `settingsservice.ts` がある
- [x] 4.6 `frontend/e2e/bindings-shim/settingsservice.ts` (新設) に `Get` / `Save` / `Reset` の shim を書く。既存の `fridgeservice.ts` にそろえる。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.7 `frontend/e2e/bindings-shim/index.ts` (更新) で `SettingsService` を export する。完了判定: `cd frontend && npm run type-check` が通る

## 5. フロントの設定の受け取り

- [x] 5.1 `frontend/src/components/settings/use-settings.ts` (新設) に `useSettings()` を書く (query key `["settings"]`、`staleTime: Infinity`、design.md D8)。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.2 `frontend/src/components/settings/use-settings.ts` (更新) に `useSaveSettings()` と `useResetSettings()` を書く
    - 成功時に `["settings"]` を `setQueryData` する
    - 成功時に `i18n.changeLanguage` を呼ぶ
    - 成功時に `["settings"]` 以外の query を無効にする
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.3 `frontend/src/components/settings/settings-gate.tsx` (新設) に `SettingsGate` と `useSettingsValues()` を書く
    - 取得前は空の背景 (`min-h-screen bg-neutral-50`) を出す
    - 取得後に `i18n.changeLanguage` を待ってから子を描く
    - 取得に失敗したら再試行ボタン付きのメッセージを出す
    - `useSettingsValues()` は Context から `settings.Values` を返し、`undefined` を返さない
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.4 `frontend/src/components/layout/root-layout.tsx` (更新) で、サイドバーとメインエリアを `SettingsGate` で包む。
  起動時のエラー画面は `SettingsGate` の外に残す。完了判定: `task dev` で S01 が表示される (目視)
- [x] 5.5 `frontend/src/i18n.ts` (更新) に、表示言語の切替は `SettingsGate` が行う旨のコメントを 1 行足す。完了判定: 差分を目視で確認する
- [x] 5.6 `frontend/src/locales/ja.json` / `en.json` (更新) に、`SettingsGate` の読み込み失敗のメッセージと再試行ボタンの文言を足す。
  完了判定: `cd frontend && npm run type-check` が通る

## 6. フロントの既定値の写しの置き換え

- [x] 6.1 `frontend/src/lib/format-quantity.ts` (更新) から `MEASURING_UNITS` を消し、`formatQuantity(amount, unit, measuringUnits)` にする (design.md D9)。
  完了判定: 呼び出し側の型エラーが 6.2〜6.5 の 4 ファイルだけに出る
- [x] 6.2 `frontend/src/components/fridge/fridge-item-row.tsx` (更新) で `useSettingsValues().measuringUnits` を `formatQuantity` に渡す。
  完了判定: このファイルの型エラーが消える
- [x] 6.3 `frontend/src/components/shopping/shopping-table.tsx` (更新) で `useSettingsValues().measuringUnits` を `formatQuantity` に渡す。
  完了判定: このファイルの型エラーが消える
- [x] 6.4 `frontend/src/components/shopping/shopping-text.ts` (更新) の文字列生成関数に `measuringUnits` 引数を足し、呼び出し側
  (`frontend/src/components/shopping/copy-shopping-text-button.tsx`) から渡す。完了判定: このファイルの型エラーが消える
- [x] 6.5 `frontend/src/components/recipe/recipe-detail.tsx` (更新) で `useSettingsValues().measuringUnits` を `formatQuantity` に渡す。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.6 `frontend/src/components/shopping/shopping-period.ts` (更新) の `isPresetDays` と `resolveShoppingPeriod` に、候補と既定日数の引数を足す。
  `SHOPPING_DEFAULTS` の import を消す。完了判定: 型エラーが `routes/shopping.tsx` だけに出る
- [x] 6.7 `frontend/src/routes/shopping.tsx` (更新) で `useSettingsValues()` の候補と既定日数を `resolveShoppingPeriod` に渡す。
  完了判定: このファイルの型エラーが消える
- [x] 6.8 `frontend/src/components/shopping/shopping-period-picker.tsx` (更新) で、日数プリセットを `useSettingsValues().shoppingPeriodDayPresets` から描く。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.9 `frontend/src/components/shopping/shopping-config.ts` を削除する。完了判定: `grep -rn shopping-config frontend/src` が何も返さない
- [x] 6.10 `frontend/src/components/mealplan/mealplan-config.ts` (更新) から `MEALPLAN_DEFAULTS.leftoverCopyPresets` を消す。
  完了判定: 型エラーが `meal-menu.tsx` と `week-calendar.tsx` だけに出る
- [x] 6.11 `frontend/src/components/mealplan/meal-menu.tsx` (更新) で、連続 N 日コピーの日数を `useSettingsValues().leftoverCopyPresets` から描く。
  完了判定: このファイルの型エラーが消える
- [x] 6.12 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `SPREAD_INITIAL_FUTURE_DAYS` / `SPREAD_PRESET_DAYS` をモジュールの定数から
  部品内の計算に移し、`useSettingsValues().leftoverCopyPresets` から求める。完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.13 `task test:e2e` を走らせ、既存のテストが写しの置き換えで壊れていないことを確かめる。完了判定: 全テストが pass する

## 7. S09 設定画面

- [x] 7.1 `frontend/src/locales/ja.json` / `en.json` (更新) に S09 の文言を足す
    - 見出し、各節の名前、各欄のラベル、「保存」「既定に戻す」「追加」、確認ダイアログ、保存と既定に戻すの通知
    - 1.6 の sentinel ごとのエラー文言と、追加時の重複、空白、文字数のエラー文言
    - `nav.settings_disabled_hint` を消す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 7.2 `frontend/src/components/settings/error-messages.ts` (新設) に、Go のエラーメッセージから i18n key と欄の名前を引く関数を書く。
  既存の `frontend/src/components/fridge/error-messages.ts` にそろえる。完了判定: `cd frontend && npm run type-check` が通る
- [x] 7.3 `frontend/src/components/settings/number-list-field.tsx` (新設) に日数の候補の編集欄を書く (design.md D10)
    - 値ごとのチップと「×」、末尾の数値の入力欄と「追加」
    - 消せない値 (`lockedValues`) のチップには「×」を出さない
    - 追加時に重複と整数でない値を弾き、欄の下にエラーを出す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 7.4 `frontend/src/components/settings/word-list-field.tsx` (新設) に単位と言葉の編集欄を書く
    - 値ごとのチップと「×」、末尾の文字の入力欄と「追加」
    - Enter で追加し、IME の変換確定の Enter は `frontend/src/lib/ime.ts` で除外する
    - 追加時に前後の空白を除き、空、20 文字超え、重複を弾いて欄の下にエラーを出す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 7.5 `frontend/src/components/settings/settings-form.tsx` (新設) に S09 のフォームを書く
    - 節は 表示 / 献立 / 冷蔵庫 / 買い物リスト / レシピ の順
    - 表示言語の select は「日本語」「English」を i18n を通さずに出す
    - サイドバーから開いたときの日数は、フォーム内の買い物リストの候補を選択肢にする。選択中の候補を消したら 7 にする
    - 正規化した値が保存済みの値と同じ間は「保存」を押せない
    - 保存の失敗は 7.2 で該当する欄の下に出す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 7.6 `frontend/src/components/settings/settings-form.tsx` (更新) に「既定に戻す」と、`frontend/src/components/ui/dialog.tsx` を使った確認を足す。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 7.7 `frontend/src/routes/settings.tsx` (新設) に `SettingsPage` を書き、見出しと `SettingsForm` を置く。既存の `frontend/src/routes/fridge.tsx` にそろえる。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 7.8 `frontend/src/router.tsx` (更新) に `/settings` ルートを足す。完了判定: `cd frontend && npm run build:dev` が通る
- [x] 7.9 `frontend/src/components/layout/sidebar.tsx` (更新) の「設定」を `disabled` のボタンから `/settings` へのリンクにし、他の項目と同じく選択中をハイライトする。
  完了判定: `task dev` でサイドバーの「設定」から S09 が開く (目視)

## 8. ドキュメント

- [x] 8.1 `docs/wireframes/S09_settings.drawio.svg` (新設) に S09 のワイヤーを描く。可視 SVG と `content=` 内の編集 XML の両方に描く
  (`docs/wireframes/README.md` の手順)。完了判定: ブラウザと drawio の両方で同じ絵が開く
- [x] 8.2 `docs/wireframes/S00_shell.drawio.svg` (更新) のサイドバーの「設定」を、他の項目と同じ押せる見た目にする。可視 SVG と編集 XML の両方を直す。
  完了判定: ブラウザと drawio の両方で同じ絵が開く
    - 実装時の確認: S00 の「設定」は可視 SVG と編集 XML の両方で、既に他の項目と同じ白地と枠線で描かれていたので、差分は無い
- [x] 8.3 `docs/product.md` (更新) §1 の可変値一覧の「変更 UI」列を、本 proposal で S09 から変えられる項目とそうでない項目が分かるように直す。完了判定: 差分を目視で確認する
- [x] 8.4 `docs/product.md` (更新) §2 の S09 を MVP 後の表から外し、画面一覧に足す。完了判定: 差分を目視で確認する
- [x] 8.5 `docs/product.md` (更新) §4 に「S09 設定」の節を足す (保存の可否、7 / 14 / 28 を消せない理由、既定に戻すの確認、表示言語は保存時に切り替わること)。
  完了判定: 差分を目視で確認する
- [x] 8.6 `docs/product.md` (更新) §4 共通シェルの「下部に将来の『設定』枠」と、S01 の「S09 で ConfigService に移管予定」を現状に合わせて直す。
  完了判定: 差分を目視で確認する
- [x] 8.7 `docs/design.md` (更新) に設定の保存と反映の節を足す (既定値から変えた key だけを `settings` テーブルに持つこと、`SettingsService` の現在値を他サービスが読むこと)。
  §402 付近の S09 の記述と §8 の proposal 順序案も直す。完了判定: 差分を目視で確認する
- [x] 8.8 `CLAUDE.md` (更新) の「可変値は 1 箇所に集約」に、S09 で変えられる値を足すときは `internal/domain/settings` の `Values` と検証も直す旨を 1〜2 行で足す。
  完了判定: 差分を目視で確認する

## 9. E2E

- [x] 9.1 `frontend/e2e/settings.spec.ts` (新設) に describe「S09 設定画面の表示」を書く
    - サイドバーから設定画面を開く
    - 完了判定: `cd frontend && npx playwright test settings` で pass する
- [x] 9.2 `frontend/e2e/settings.spec.ts` (更新) に describe「設定の保存」を書く (「開き直す」は `page.reload()` で再現する)
    - 保存した値は開き直しても残る
    - 保存せずに離れると変更は捨てられる
    - 値を変えるまで保存は押せない
    - 完了判定: `cd frontend && npx playwright test settings` で pass する
- [x] 9.3 `frontend/e2e/settings.spec.ts` (更新) に describe「設定値の検証」を書く
    - 🔴 の日数が 🟡 の日数以上だと保存されない
    - 同じ量る単位は追加できない
    - 買い物リストの 7 / 14 / 28 日は消せない
    - 完了判定: `cd frontend && npx playwright test settings` で pass する
- [x] 9.4 `frontend/e2e/settings.spec.ts` (更新) に describe「既定に戻す」を書く
    - 既定に戻すとすべての値が既定値になる
    - 確認で取り消すと値は変わらない
    - 完了判定: `cd frontend && npx playwright test settings` で pass する
- [x] 9.5 `frontend/e2e/settings.spec.ts` (更新) に describe「保存した値の各画面への反映」を書く
    - 期限アラートの日数を変えると S05 の色が変わる
    - 買い物リストの日数の候補を足すと S07 に出る
    - サイドバーから開いたときの日数を変えると S07 の対象期間が変わる
    - 作り置きの日数の候補を変えるとコピーの選択肢が変わる
    - 量る単位に足した単位は量の前に表示される
    - 完了判定: `cd frontend && npx playwright test settings` で pass する
- [x] 9.6 `frontend/e2e/settings.spec.ts` (更新) に describe「表示言語の切替」を書く
    - English を保存すると画面が英語になる
    - 保存した表示言語で起動する
    - 完了判定: `cd frontend && npx playwright test settings` で pass する
- [x] 9.7 「作り置きの日数の候補を空にすると保存されない」「量を持たない言葉に足した言葉は取り込みで量 0 になる」
  「人数を省略したときの人数を変えると取り込みに使われる」「壊れた項目だけ既定値になる」は 3.14〜3.17 のサービス層のテストで確かめ、E2E には足さない。
  完了判定: 3.14〜3.17 が pass している

## 10. 最終確認

- [x] 10.1 `go test ./internal/... ./cmd/...` を走らせる。完了判定: 全テストが pass する
- [x] 10.2 `cd frontend && npm run type-check` を走らせる。完了判定: エラーが出ない
- [x] 10.3 `cd frontend && npm run build:dev` を走らせる。完了判定: ビルドが通る
- [ ] 10.4 `task test:e2e` を走らせる。完了判定: `check:scenarios` を含む全テストが pass する
- [x] 10.5 `task lint` を走らせる。完了判定: Go と Frontend の両方でエラーが出ない
- [x] 10.6 `openspec validate add-settings-screen --strict` を走らせる。完了判定: エラーが出ない
- [ ] 10.7 `task build:win` で Windows ビルドを作り、Windows 側で S09 の量る単位と量を持たない言葉に日本語を IME で入力して保存できることを確かめる。
  完了判定: Windows 実機で目視確認する
