## Why

「作り置きを広げる」の上書きは、S01 の表示期間の外にあるセルの品を、警告なしに消す (docs/reviews/2026-10-15-design-ux-review.md R37)。
統合ダイアログは既存の品があるセルを薄い黄色で示すが、その判定は S01 が持つ表示期間の行しか見ていない。
1 週表示で土曜から広げると、格子の月曜と火曜は翌週に当たり、品があっても黄色にならない。
しかも上書きで消えた品は戻せない。
`revise-mealplan-destructive-actions` で品の削除と時間帯全体のクリアには「元に戻す」が付いたが、上書きはその対象から外れたままである。

統合ダイアログには、ほかに 3 つの食い違いがある。

- 起点と同じ日の、起点より前の時間帯 (夕食が起点のときの同じ日の朝食と昼食) にも広げられる (R39)。
  spec は「作り置きは翌日以降に伸ばす操作なので過去方向の対象日は載せない」としており、利用者の判断で禁止すると決めた
- 削除済みレシピの品を広げると、そのレシピの行が新しく増える (R40)。時間帯全体の「連続 N 日コピー」も同じである。
  recipe の spec は、削除済みレシピを「献立への新たな割当」から除くとしており、利用者の判断でこの 2 つも新たな割当として拒否すると決めた
- 説明文とチェックボックスの読み上げ名が、日付を `2026-10-19` の形で出している (R42)

あわせて、品行の「作った」チェックボックスの読み上げ名がどの品でも「作った」で、どの品の印か区別できない (R43)。

## What Changes

- 統合ダイアログの格子で、既存の品があるかどうかと「起点と同じ作り置きの同じレシピ」の判定を、S01 の表示期間ではなく格子の範囲の献立で行う
- 「作り置きを広げる」を反映したあと、通知に `元に戻す` を出す。追加でも上書きでも出す
  - `元に戻す` を押すと、広げて追加した行が消え、上書きで消えた品が元のセルの元の並び順に戻る。作り置きへの属し方と「作った」印も戻る
  - 通知には、広げたセルの数と、上書きで置き換えた品の数を示す
- 起点と同じ日の、起点より前の時間帯のセルを、格子で選べなくし、画面を経由しない要求も拒否する
- 削除済みレシピの品を「作り置きを広げる」で広げる要求と、削除済みレシピを含む時間帯の「連続 N 日コピー」を拒否し、理由を通知で示す
- 統合ダイアログの説明文とチェックボックスの読み上げ名の日付を、格子の行見出しと同じ表記 (`10/19(月)`) にする
- 「作った」チェックボックスの読み上げ名に、レシピ名を入れる (「肉じゃがを作った」)
- **BREAKING** (内部 API)：「作り置きを広げる」のサービスのメソッドが、取り消しに使う結果を返すようになる。画面は統合ダイアログからしか呼んでいない

### Non-goals

- **期間をまたぐ作り置きの件数** (R36) と **作り置きをやめたときに残る行** (R38)：`fix-mealplan-batch-boundaries` で扱う
- **時間帯全体の「連続 N 日コピー」の取り消しと統合ダイアログへの統合** (R6)：扱わない。
  この経路の上書きは、衝突するセルを確認のダイアログで示してから実行しており、今回の不具合 (警告なしに消える) は起きにくい。
  ただし衝突の判定は同じく表示期間の行しか見ていないので、後続の proposal への引き継ぎに残す
- **対象日の範囲の拡張**：格子は起点日から未来 3 日ぶんの 4 行のままにする
- **削除済みレシピであることの S01 での表示**：扱わない。広げようとしたときの通知で知らせる

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：
  - 「作り置きを広げる操作」を改める。選べないセルに「起点と同じ日の、起点より前の時間帯」を足し、
    既存の品の判定を格子の範囲の献立で行い、説明文の日付の表記を決める
  - 「作り置きを広げる操作の取り消し」を足す
  - 「削除済みレシピの品は広げられない」を足す (統合ダイアログと時間帯全体の連続 N 日コピーの両方)
  - 「作った印のチェックボックスの読み上げ名」を足す
- `recipe`：
  - 「削除済みレシピの除外範囲」の除外する操作に、既存の品を広げる操作と時間帯全体のコピーを足す
  - 「削除済みレシピを使った献立の保持」の「できる操作」から、別のセルへ増やす操作を外す (上の除外範囲と食い違わないようにする)

## Impact

### 新設ファイル

- なし

### 更新ファイル

- `internal/domain/mealplan/entry.go`：`MealRank` を足す (時間帯を朝、昼、夕の順に並べる順位)
- `internal/domain/mealplan/spread.go`
  - `SpreadResult` を足す (追加した行の ID と、上書きで消した行)
  - `ValidateSpreadTargets` で、起点と同じ日の起点より前の時間帯を拒否する
- `internal/domain/mealplan/spread_test.go`：前の時間帯のケースを足す
- `internal/domain/mealplan/validate.go`：`ErrArchivedRecipe` を足す
- `internal/storage/sqlite/mealplan_repo.go`
  - `Spread`：起点のレシピが削除済みなら拒否し、追加した行の ID と上書きで消した行を返す
  - `UndoSpread` を足す (追加した行を消し、消した行を戻す。1 トランザクション)
  - `CopyLeftover`：起点の時間帯に削除済みレシピがあれば拒否する
  - `mealRank` を `mealplan.MealRank` に寄せる
- `internal/storage/sqlite/mealplan_repo_test.go`：`Spread`、`UndoSpread`、`CopyLeftover` のテストを足す
- `internal/service/mealplan.go`：`SpreadEntry` の戻り値を `mealplan.SpreadResult` にし、`UndoSpread` を足す。`mealPlanRepo` も同じく変える
- `internal/service/mealplan_test.go`：フェイクのリポジトリを合わせ、`TestMealPlanService_SpreadEntry` と `TestMealPlanService_UndoSpread` を足す
- `cmd/e2e-server/dispatch_test.go`：`SpreadEntry` の応答の形に合わせる
- `frontend/src/components/mealplan/spread-batch-dialog.tsx`
  - 格子の範囲の献立を自分で取得し、既存の品と同じ作り置きのセルの判定に使う
  - 起点と同じ日の、起点より前の時間帯のセルを選べなくする
  - 説明文とチェックボックスの読み上げ名の日付を行見出しと同じ表記にする
- `frontend/src/components/mealplan/week-calendar.tsx`：`spreadMut` の成功時に `元に戻す` の通知を出し、取り消しの mutation を足す。`existingEntries` を渡すのをやめる
- `frontend/src/components/mealplan/meal-item-row.tsx`：「作った」チェックボックスの読み上げ名にレシピ名を入れる
- `frontend/src/components/mealplan/error-messages.ts`：`mealplan: archived recipe` の対応を足す
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`
  - `pages.meal_plan.toast.spread`、`pages.meal_plan.toast.spread_overwritten` を足す
  - `pages.meal_plan.errors.archived_recipe` を足す
  - `pages.meal_plan.spread.earlier_meal_cell_label` を足す
  - `pages.meal_plan.cell.cooked` を「{{name}}を作った」に改める
- `frontend/e2e/bindings-shim/mealplanservice.ts`：`SpreadEntry` の戻り値を `SpreadResult` にし、`UndoSpread` を足す
- `frontend/e2e/mealplan-batch.spec.ts`：describe「作り置きを広げる操作」にシナリオを足し、describe「作り置きを広げる操作の取り消し」と
  「削除済みレシピの品は広げられない」を足す
- `frontend/e2e/mealplan-cooked.spec.ts`：describe「作った印のチェックボックスの読み上げ名」を足す
- `docs/product.md`：§4 S01 の「作り置きを広げる」の説明に、選べないセル、取り消し、削除済みレシピの扱いを書き足す
- `docs/reviews/2026-10-15-design-ux-review.md`：R37、R39、R40、R42、R43 にチェックを入れる
- `openspec/specs/mealplan/spec.md` / `openspec/specs/recipe/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`SpreadEntry` の戻り値と `SpreadResult` 型、`UndoSpread` が増える

### migration

- なし

### 前提と順序

- `fix-mealplan-batch-boundaries` を先に apply する。同じ `mealplan_repo.go` を触り、
  「起点と同じ作り置き」の判定の E2E は、期間をまたぐ作り置きが正しく識別されることを前提にする

### 後続 proposal への引き継ぎ

- R6：時間帯全体の「連続 N 日コピー」を統合ダイアログに寄せる。そのとき取り消しも `SpreadResult` と `UndoSpread` に乗せられる
- 時間帯全体の連続 N 日コピーの衝突判定も、表示期間の行しか見ていない。R6 で統合ダイアログに寄せれば、格子の範囲の取得で同時に直る
