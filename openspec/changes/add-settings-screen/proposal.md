## Why

期限アラートの日数や量る単位などは、作り置き中心の利用者に合わせた既定値のまま動いている。
`internal/config/defaults.go` に集めた値は、どれも利用者ごとに変えられる前提で選んだもので、
毎日別のレシピを作る世帯や、英語で使いたい利用者には合わないものがある。
変える手段がソースの書き換えしかないので、配布したバイナリの利用者は既定値を受け入れるしかない。

フロントには、Go の既定値を手で写した定数が 3 か所に残っている (作り置きの日数、買い物リストの日数、量る単位)。
値を変えるたびに両方を直す必要があり、片方だけ直すと、画面に出る選択肢をサービスが拒否する。
設定画面を作れば、フロントもサービスから値を受け取る形になり、この二重管理もなくなる。

## What Changes

- サイドバー下部の「設定」を押せるようにし、S09 設定画面を開けるようにする
- S09 で次の値を変えて保存できるようにする。保存した値は再起動後も残る
  - 表示言語 (日本語 / English)
  - 冷蔵庫の期限アラートの日数 (🔴 / 🟡)
  - 作り置きを広げる日数の候補
  - 買い物リストの日数の候補と、サイドバーから開いたときの日数
  - 量る単位
  - 量を持たない言葉 (レシピの取り込み用)
  - 取り込むファイルが人数を省略したときの人数
- S09 に「既定に戻す」を置き、すべての値を既定値に戻せるようにする
- 不正な値 (🔴 の日数が 🟡 以上、候補が空、同じ単位の重複など) は保存せず、該当する欄の下にエラーを出す
- 保存した値は、冷蔵庫 (S05)、献立 (S01)、買い物リスト (S07)、レシピ詳細 (S03)、レシピの取り込みにすぐ反映する
- サービスは起動時に保存済みの値を読み、既定値に重ねた値で動く。保存すると、再起動せずに新しい値で動く
- フロントの既定値の写し (作り置きの日数、買い物リストの日数、量る単位) をやめ、設定のサービスから受け取る

## Non-goals

- 未実装の機能に属する値 (「久しぶりに」の閾値、週プリセットの品数、献立を埋める曜日パターン、1 時間帯あたりの品数の目安)。
  画面に出しても変える効果が無いので、その機能を作る proposal で設定項目に加える
- S01 の初回表示の時間帯 (`DefaultVisibleMeals`)。S01 は前回の切替を覚えるので、変えても効くのは初回の起動だけになる
- S01 のセルサイズのプリセット表と、S02 の保存直後のハイライト時間。利用者が調整する値ではなく、見た目の調整用の値として残す
- S01 の表示設定 (時間帯 / 期間長) の保存先を localStorage から DB に移すこと。今の保存で困っていないので、別の proposal にする
- データの書き出しと取り込み、DB の控えからの復元。レシピの書き出しは進行中の `add-recipe-export` が扱う
- 項目ごとに既定値へ戻す操作。戻すのは全項目まとめてだけにする
- 未保存の変更があるまま S09 から離れたときの確認。離れると未保存の変更は捨てる
- ダークテーマとフォントサイズの設定。配色は明るいテーマに固定する方針 (shell の「配色の固定」) を変えない

## Capabilities

### New Capabilities

- `settings`: S09 設定画面。変えられる値の一覧、保存と検証、既定に戻す操作、保存した値の各画面への反映、表示言語の切替

### Modified Capabilities

- `shopping`: 「S07 上での期間の切替」の日数プリセットを固定の 3 / 4 / 7 / 14 / 28 日から、設定した候補 (既定は同じ) に変える

## Impact

**新設ファイル**

- `internal/domain/settings/settings.go`：変えられる値の型 `Values`、`config.Config` との相互変換、正規化 (前後の空白除去と数値の並べ替え)
- `internal/domain/settings/validate.go`：`Values` の検証と、上限下限の定数
- `internal/domain/settings/errors.go`：検証エラーの sentinel (`settings: ...`)
- `internal/domain/settings/settings_test.go`：正規化と検証の table-driven test
- `internal/storage/sqlite/settings_repo.go`：`settings` テーブルの読み書き (`Load` / `ReplaceAll` / `DeleteAll`)
- `internal/storage/sqlite/settings_repo_test.go`
- `internal/service/settings.go`：`SettingsService` (`Get` / `Save` / `Reset`) と、他サービスに現在値を渡す非公開メソッド
- `internal/service/settings_test.go`
- `frontend/src/routes/settings.tsx`：S09 のページ
- `frontend/src/components/settings/use-settings.ts`：設定の取得と保存の TanStack Query ラッパ
- `frontend/src/components/settings/settings-gate.tsx`：設定を読み終えてから各画面を描き、表示言語を当てる
- `frontend/src/components/settings/settings-form.tsx`：S09 のフォーム本体
- `frontend/src/components/settings/number-list-field.tsx`：日数の候補の編集欄
- `frontend/src/components/settings/word-list-field.tsx`：単位と言葉の編集欄
- `frontend/src/components/settings/error-messages.ts`：Go の検証エラーを i18n key と欄に対応付ける
- `frontend/e2e/bindings-shim/settingsservice.ts`
- `frontend/e2e/settings.spec.ts`
- `docs/wireframes/S09_settings.drawio.svg`

**更新ファイル**

- `internal/config/defaults.go`：`Config` に `Language` を足し、既定値 `"ja"` を返す
- `internal/config/defaults_test.go`：`Language` の既定値を検査する
- `internal/service/fridge.go` / `mealplan.go` / `recipe.go` / `shopping.go`：`config.Defaults()` を直接読まず、設定サービスから現在値を読む
- `internal/service/fridge_test.go` / `mealplan_test.go` / `recipe_test.go` / `shopping_test.go`：コンストラクタの変更に合わせる
- `main.go`：`SettingsService` を先に作り、他サービスに渡して `application.Options.Services` に登録する
- `cmd/e2e-server/main.go`：`newRegistry` に `SettingsService` を足す (6 サービス)
- `cmd/e2e-server/dispatch.go`：registry のコメントのサービス数を直す
- `cmd/e2e-server/dispatch_test.go`：登録数の突き合わせを 6 にする
- `frontend/e2e/bindings-shim/index.ts`：`SettingsService` を export する
- `frontend/src/router.tsx`：`/settings` ルートを足す
- `frontend/src/components/layout/root-layout.tsx`：`SettingsGate` で `Outlet` を包む
- `frontend/src/components/layout/sidebar.tsx`：「設定」を S09 へのリンクにし、選択中をハイライトする
- `frontend/src/i18n.ts`：初期言語を `ja` のままにし、切替は `SettingsGate` に任せる旨を残す
- `frontend/src/locales/ja.json` / `en.json`：S09 の文言、検証エラー、`nav.settings_disabled_hint` の削除
- `frontend/src/lib/format-quantity.ts`：`MEASURING_UNITS` を消し、量る単位を引数で受け取る
- `frontend/src/components/fridge/fridge-item-row.tsx` / `shopping/shopping-table.tsx` / `shopping/shopping-text.ts` /
  `recipe/recipe-detail.tsx`：`formatQuantity` に設定の量る単位を渡す
- `frontend/src/components/mealplan/mealplan-config.ts`：`MEALPLAN_DEFAULTS.leftoverCopyPresets` を消す
- `frontend/src/components/mealplan/meal-menu.tsx` / `week-calendar.tsx`：作り置きの日数を設定から読む
- `frontend/src/components/shopping/shopping-period.ts`：候補と既定日数を引数で受け取る
- `frontend/src/components/shopping/shopping-period-picker.tsx` / `routes/shopping.tsx`：候補と既定日数を設定から読む
- `docs/product.md`：§1 の可変値一覧に S09 で変えられるかの列を足す、§2 の S09 を MVP の画面に移す、§4 に S09 の節を足す、
  §4 共通シェルの「将来の設定枠」と S01 の「S09 で ConfigService に移管予定」を直す
- `docs/design.md`：§7 の付近に設定の保存と反映の節を足す、§8 の proposal 順序案に本 proposal を足す
- `docs/wireframes/S00_shell.drawio.svg`：サイドバーの「設定」を押せる見た目にする
- `CLAUDE.md`：「可変値は 1 箇所に集約」に、S09 で変えられる値は `internal/domain/settings` の検証も直す旨を足す

**削除ファイル**

- `frontend/src/components/shopping/shopping-config.ts`：`SHOPPING_DEFAULTS` の写しをやめる

**生成物 (再生成のみ)**

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)

**migration**

- `00010_settings.sql`：`settings` テーブル (`key` / `value` (JSON) / `updated_at`)。既定値から変えた項目だけを持つ

**後続 proposal への引き継ぎ**

- 未実装の機能の値を足すときは、`settings.Values` に項目を足し、検証と S09 の欄を同じ proposal で足す
- データの書き出しと取り込み、DB の控えからの復元は、S09 に節を足す形で別の proposal にする
- S01 の表示設定を DB に移すときは、`settings` テーブルに key を足す。`useMealPlanPreference` の中身だけを差し替える
- 進行中の `add-frontend-unit-tests` は `formatQuantity(amount, unit)` のテストを予定している。
  本 proposal を先に入れるなら、そちらのテストは第 3 引数に量る単位を渡す形にする
