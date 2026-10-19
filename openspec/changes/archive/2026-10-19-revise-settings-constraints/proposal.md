## Why

S09 の導入で、量る単位などの既定値のフロント側の写しは消えた。
それでも、設定に結びついた規則のうち 2 つが、まだ Go とフロントの両方に書かれている
(docs/reviews/2026-10-19-design-ux-review.md R48、R17 の残り)。

- S09 の「買い物リストの日数の候補」で消せない日数 (7 / 14 / 28) と、既定の日数を消したときに選び直す日数 (7)
- 「作り置きを広げる」の格子の行数 (作り置きの日数の候補の最大値 − 1)

どちらもコメントで出典を指しているので、今すぐ壊れるものではない。
ただ、片方だけを直すと、Go が拒否する値をフロントが送ったり、送れるはずの値をフロントが止めたりする。
さらに、統合ダイアログは格子の判定に 7 日分の献立だけを取っている。
作り置きの日数の上限 (7 日) と取得範囲が偶然一致しているだけで、上限を広げると 8 日目以降のセルの判定が抜ける。

## What Changes

- S09 の制約を Go から返すメソッドを `SettingsService` に足し、S09 はその値で「消せない日数」と「選び直す日数」を決める
- 「作り置きを広げる」の格子に必要な値 (起点から何日後まで選べるか、その範囲の献立) を、1 回の呼び出しで返すメソッドを `MealPlanService` に足す
- 統合ダイアログは、格子の行数と既存の品の判定を、そのメソッドの戻り値だけで決める
- S01 の献立カレンダーは、作り置きの日数の候補から格子の行数を計算しなくなる

### Non-goals

- **画面に見える挙動の変更**：消せない日数、格子の行数、既存の品の色分けは今と同じ値になる。spec は変えない
- **S09 の範囲や件数の検証をフロントでも行うこと**：従来どおり保存時に Go が判定する (add-settings-screen design.md D5)
- **期間の週数 (1 / 2 / 4) の写し**：S01 の期間の選択肢は設定で変わらないので、この change では扱わない
- **統合ダイアログの「もっと表示」の削除** (R51)：`fix-mealplan-dialog-polish` で扱う

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

(なし。画面と操作の振る舞いは変わらないので、`.openspec.yaml` に `skip_specs: true` を置く)

## Impact

### 新設ファイル

- `internal/domain/settings/constraints.go`：`Constraints` 型 (消せない日数、選び直す日数) と、それを組み立てる関数
- `internal/domain/mealplan/spreadgrid.go`：`SpreadGrid` 型 (起点から選べる日数、その範囲の献立)

### 更新ファイル

- `internal/domain/settings/validate.go`：`ShoppingRequiredDays` を `constraints.go` の値と同じ出典にする
- `internal/service/settings.go`：`Constraints` メソッドを足す
- `internal/service/settings_test.go`：`Constraints` のテストを足す
- `internal/service/mealplan.go`：`SpreadGrid` メソッドを足す
- `internal/service/mealplan_test.go`：`SpreadGrid` のテストを足す
- `frontend/src/components/settings/use-settings.ts`：`useSettingsConstraints` を足す
- `frontend/src/routes/settings.tsx`：制約を読み終えてから `SettingsForm` を描く
- `frontend/src/components/settings/settings-form.tsx`：`SHOPPING_REQUIRED_DAYS` と `SHOPPING_FALLBACK_DAYS` を消し、props の制約を使う
- `frontend/src/components/mealplan/spread-batch-dialog.tsx`：格子の取得を `SpreadGrid` に置き換え、`initialFutureDays` の prop を消す
- `frontend/src/components/mealplan/week-calendar.tsx`：`spreadInitialFutureDays` の計算と、`SpreadBatchDialog` への受け渡しを消す
- `frontend/e2e/bindings-shim/settingsservice.ts`：`Constraints` を足す
- `frontend/e2e/bindings-shim/mealplanservice.ts`：`SpreadGrid` を足す
- `docs/design.md`：§5「設定の保存と反映」に、設定に結びついた規則は Go から返すことを書き足す
- `docs/reviews/2026-10-19-design-ux-review.md`：R48 にチェックを入れ、R17 の残りが消えたことを書き足す

### 生成物 (再生成のみ)

- `frontend/bindings/`：`wails3 generate bindings -clean=true -ts -i` で `Constraints`、`SpreadGrid` と型を出す

### migration

- なし

### 後続 proposal への引き継ぎ

- `fix-mealplan-dialog-polish` と `spread-batch-dialog.tsx` の props が重なる。先に `fix-mealplan-dialog-polish` を入れ、この change はその後に入れる
- S09 に欄を足す proposal で、新しい欄に「フロントで固定する値」が出たら、`Constraints` に足す
