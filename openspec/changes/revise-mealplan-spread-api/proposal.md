## Why

S01 の「作り置きを広げる」は、対象日と時間帯の格子にチェックを付ける UI だが、選んだとおりには反映されない
(docs/reviews/2026-10-14-design-ux-review.md R22)。

- 月曜を起点に火曜と木曜だけを選ぶと、選んでいない水曜にも入る
- 「別調理」を選んでも、別の時間帯のセルは起点と同じ作り置きになる
- セルごとに別々の要求を順に投げるので、途中で失敗すると一部だけ反映された状態で止まる

反映の中身が、統合ダイアログより前からある 2 つの操作 (同じ時間帯の連続 N 日コピー、別の時間帯への 1 セルの持ち越し) の
組み合わせだからである。
格子のチェックを「起点からいちばん遠い日までの連続 N 日」に読み替えているので、あいだの日が埋まる。
別の時間帯への持ち越しは、作り置きと別調理の選択を受け取らない。

あわせて、同じ日の昼食と夕食にまたがる作り置きが、「他の作り置きにまとめる」の候補に 2 行出る (R23)。
別の時間帯へ広げる操作を直すと、この状態を作りやすくなるので、同じ change で直す。

## What Changes

- 「作り置きを広げる」の反映を、選んだセルの一覧をそのまま受け取る 1 回の操作にする
  - チェックしたセルだけに追加する。あいだの日は埋めない
  - 「別調理」は、同じ時間帯のセルにも別の時間帯のセルにも効く
  - 全部のセルが反映されるか、1 つも反映されないかのどちらかになる
- 統合ダイアログで、起点と同じ作り置きの同じレシピがすでに入っているセルを選べないようにする (spec にあって未実装だった)
- spec の食い違いを直す (R34 のうち、この要件にかかる分)
  - 既定でチェックされるのは「起点日の翌日の同じ時間帯」の 1 セルである (シナリオが 2 セルと書いていた)
- **BREAKING** (内部 API)：別の時間帯へ 1 セルずつ持ち越すサービスのメソッドを廃止する。画面からは統合ダイアログ経由でしか使っていない
- 「他の作り置きにまとめる」の候補は、1 つの作り置きにつき 1 行だけ出す

### Non-goals

- **対象日の範囲の拡張**：格子は起点日から未来 3 日ぶんの 4 行のままにする。範囲を広げるのは別の proposal で扱う
- **時間帯全体の「連続 N 日コピー」の統合**：時間帯の `⋯` メニューの経路 (R6) は変えない。こちらは連続した日にしか使わないので、今回の不具合は起きない
- **献立の行の ID**：`add-mealplan-entry-id` で扱う (R25)。この change では、起点の品を従来どおり (日付, 時間帯, 並び順) で指す
- **作り置きの枠色の重なり** (R8)：扱わない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：
  - 「作り置きを広げる (統合ダイアログ)」を「作り置きを広げる操作」に差し替え、選んだセルだけに一括で反映する形に改める。
    シナリオ名の変更を伴うので、要件ごと差し替える (MODIFIED ではシナリオ名を変えられない)
  - 「まとめ候補の一意性」を足す

## Impact

### 新設ファイル

- `internal/domain/mealplan/spread.go`：`SpreadTarget` と `ValidateSpreadTargets`
- `internal/domain/mealplan/spread_test.go`

### 更新ファイル

- `internal/domain/mealplan/validate.go`：`ErrInvalidSpreadTarget` を足す
- `internal/storage/sqlite/mealplan_repo.go`
  - `Spread` を足す
  - `CopyToOtherMeal` を消す
  - `ListMergeCandidates` の代表行の選び方を直す
- `internal/storage/sqlite/mealplan_repo_test.go`：`Spread` のテストを足し、`CopyToOtherMeal` のテストを消し、まとめ候補のテストを足す
- `internal/service/mealplan.go`：`SpreadEntry` を足し、`CopyToOtherMeal` と `mealPlanRepo` の同名メソッドを消す
- `internal/service/mealplan_test.go`：フェイクのリポジトリに `Spread` を足し、`TestMealPlanService_SpreadEntry` を足し、`TestMealPlanService_CopyToOtherMeal` を消す
- `frontend/src/components/mealplan/week-calendar.tsx`：`submitSpread` を 1 回の `SpreadEntry` の呼び出しにし、`computeCopyDays` を消す
- `frontend/src/components/mealplan/spread-batch-dialog.tsx`：起点と同じ作り置きの同じレシピが入っているセルを選べなくする
- `frontend/src/lib/spread-batch-selection.ts`：`planSpread` と `dateGap` を消し、型だけを残す
- `frontend/src/components/mealplan/error-messages.ts`：`mealplan: invalid spread target` の対応を足す
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.meal_plan.spread.partial_failure` を消し、
  `pages.meal_plan.errors.invalid_spread_target` と `pages.meal_plan.spread.same_batch_cell_label` を足す
- `frontend/e2e/bindings-shim/mealplanservice.ts`：`SpreadEntry` を足し、`CopyToOtherMeal` を消す
- `frontend/e2e/helpers.ts`：統合ダイアログを開いてセルにチェックを付けるヘルパーを足す
- `frontend/e2e/mealplan-batch.spec.ts`：describe「作り置きを広げる (統合ダイアログ)」を「作り置きを広げる操作」に改名し、`test.skip` を外して実装する。
  describe「まとめ候補の一意性」を足す
- `docs/product.md`：§4 S01 の「作り置きを広げる」の説明を直す (対象日は起点日から未来 3 日、選んだセルだけに入る)
- `docs/reviews/2026-10-14-design-ux-review.md`：R22、R23 にチェックを入れる
- `openspec/specs/mealplan/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`SpreadEntry` と `SpreadTarget` が増え、`CopyToOtherMeal` が消える

### migration

- なし

### 前提と順序

- `fix-mealplan-renumber-after-reorder` を先に apply する (上書きでセルを消したあとの詰め直しに影響しないが、同じファイルを触る)
- `revise-mealplan-destructive-actions` と `add-mealplan-cell-overflow-list` より先に apply する。
  どちらも `week-calendar.tsx` を触るので、この change で `submitSpread` を短くしてから入れる

### 後続 proposal への引き継ぎ

- R6：時間帯全体の「連続 N 日コピー」を統合ダイアログに寄せる。`SpreadEntry` を時間帯の全品ぶん呼べば済む
- 対象日の範囲の拡張：`ValidateSpreadTargets` の上限を `LeftoverCopyPresets` から切り離す
- `add-mealplan-entry-id`：`SpreadEntry` の起点の指定を行の ID に変える
