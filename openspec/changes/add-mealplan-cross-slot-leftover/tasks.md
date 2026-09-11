## 1. repo: ListMergeCandidates 改名 + MergeBatch の slot 制約解除

- [ ] 1.1 `internal/storage/sqlite/mealplan_repo.go` の `ListSlotBatches` を `ListMergeCandidates` に
      改名し、SQL の WHERE 句から `slot = ?` フィルタを除去する。呼び出し元の `slot` パラメータは
      SlotSummary 表示用に残す (BatchSummary の Slot 情報として活用)。完了判定: `go build ./internal/...`
      が通る
- [ ] 1.2 `MergeBatch` の防御チェックから「同 slot」条件を除去する。同一レシピ + ±dayRange の
      2 条件だけで判定する。完了判定: 下記 1.4 のテストが pass する
- [ ] 1.3 `MealPlanRepo.CopyToOtherSlot(ctx, sourceDate, sourceSlot string, sourcePosition int,
      targetDate, targetSlot string) error` を新規追加する。対象行の recipeID / batch_id を取得し、
      対象 slot に既存衝突があれば `ErrBatchMergeDenied` (単純版) を返す — 衝突解決は task 3 で拡張。
      完了判定: 下記 1.4 のテストが pass する
- [ ] 1.4 `internal/storage/sqlite/mealplan_repo_test.go` に以下ケースを追加する
    - ListMergeCandidates は別 slot の同レシピ候補も返す
    - MergeBatch は別 slot 間の同レシピ ±4 日以内なら成功する
    - MergeBatch は別 slot でもレシピ違いなら ErrBatchMergeDenied
    - CopyToOtherSlot: 「月曜・夕食 → 火曜・昼食」で対象セルに同一 batch_id + 同一 recipe の行が追加
    - CopyToOtherSlot: 対象日が ±5 日を超えると ErrBatchMergeDenied
    - CopyToOtherSlot: 元と同じ slot は ErrBatchMergeDenied
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)

## 2. service 層

- [ ] 2.1 `internal/service/mealplan.go` の `ListSlotBatches` を `ListMergeCandidates` に改名、内部で
      repo に委譲する。同 slot 判定は不要になる。完了判定: `go build ./` が通る
- [ ] 2.2 `MealPlanService.CopyToOtherSlot(ctx, sourceDate, sourceSlot string, sourcePosition int,
      targetDate, targetSlot string) error` を追加。入力バリデーション (date / slot / position) を通し、
      repo に委譲する。完了判定: 下記 2.3 のテストが pass する
- [ ] 2.3 `internal/service/mealplan_test.go` fakeRepo に `CopyToOtherSlot` を実装 + 上記ケースを service
      経由でも検証する。完了判定: `go test ./internal/service/...` が pass する

## 3. 衝突解決の統合 (CopyToOtherSlot にも 追加 / 上書き / キャンセル を提供)

- [ ] 3.1 `MealPlanRepo.CopyToOtherSlot` に mode パラメータ (`append` / `overwrite`) を追加し、
      衝突解決を CopyLeftover の repo ヘルパと共通化する。完了判定: 下記 3.3 のテストが pass する
- [ ] 3.2 `MealPlanService.CopyToOtherSlot(..., mode string)` を対応するシグネチャに拡張する。
      完了判定: `go build ./` が通る
- [ ] 3.3 `internal/service/mealplan_test.go` / `internal/storage/sqlite/mealplan_repo_test.go` に:
    - 衝突なしでも append で追加できる
    - 既存があるとき append は末尾に追加
    - 既存があるとき overwrite は既存を削除してから追加
    - (完了判定: `go test ./internal/...` が pass する)

## 4. Wails bindings 再生成 + shim 更新

- [ ] 4.1 `wails3 generate bindings` を実行する。完了判定: `ListMergeCandidates` と `CopyToOtherSlot`
      が生成される
- [ ] 4.2 `frontend/e2e/bindings-shim/mealplanservice.ts` の `ListSlotBatches` を
      `ListMergeCandidates` に改名し、`CopyToOtherSlot` を追加する。完了判定: `npm run type-check`
      が通る

## 5. i18n 追加

- [ ] 5.1 `frontend/src/locales/{ja,en}.json` に以下 key を追加する
    - `mealplan.item_menu.copy_to_slot` — 「別の時間帯にも作り置きを追加」
    - `mealplan.copy_to_slot.title` — 「別の時間帯にも作り置きを追加」ダイアログ題
    - `mealplan.copy_to_slot.target_date_label` — 「対象日」
    - `mealplan.copy_to_slot.target_slot_label` — 「対象時間帯」
    - `mealplan.copy_to_slot.confirm` — 「追加」
    - (完了判定: `npm run type-check` が通る)

## 6. フロント: メニュー項目 + ダイアログ

- [ ] 6.1 `frontend/src/components/mealplan/cell-item-menu.tsx` に `onCopyToSlot` prop と
      「別の時間帯にも作り置きを追加」項目を追加する。完了判定: `npm run type-check` が通る
- [ ] 6.2 `frontend/src/components/mealplan/copy-to-slot-dialog.tsx` を新設する。props: `open`,
      `sourceDate: string`, `sourceSlot: Slot`, `dayRange: number`, `onSelect(targetDate, targetSlot)`.
      日付ピッカーで ±dayRange 以内の日付を選ばせ、slot は元 slot 以外を選ばせる。完了判定:
      `npm run type-check` が通る
- [ ] 6.3 `frontend/src/components/mealplan/batch-merge-dialog.tsx` の候補行表示に slot 名 (朝食 / 昼食 /
      夕食) を追加する。既存の `mealplan.batch.merge_dialog.item_label` を slot 表示付きに拡張する
      (i18n key の args に slot を追加)。完了判定: 目視で候補行に slot 名が表示される
- [ ] 6.4 `frontend/src/components/mealplan/week-calendar.tsx` の `handleEntryAction` に
      `copy-to-slot` 経路を追加し、CopyToOtherSlot mutation を配線する。既存の
      `openMergeDialog` は `MealPlanService.ListMergeCandidates` を呼ぶよう更新。完了判定:
      `npm run type-check` が通る

## 7. ドキュメント + E2E

- [ ] 7.1 `docs/scenarios/mealplan.feature` に以下シナリオを追加する
    - 前日の夕食の作り置きを翌日の昼食に足す
    - 追加先が既存割当と衝突すると衝突解決ダイアログが出る
    - 別 slot の同一レシピ作り置きにもまとめられる
    - (完了判定: spec の Scenario 名と 1 対 1 で揃う)
- [ ] 7.2 `docs/product.md` §S01 の「セル内の操作」に「別の時間帯にも作り置きを追加」項目、および
    「同 slot」制約が外れた旨を追記する。完了判定: 目視で該当節が更新される
- [ ] 7.3 `frontend/e2e/mealplan-batch.spec.ts` に該当シナリオの test 名を追加 (実行は DropdownMenu
      Portal の都合上 test.skip でも可)。完了判定: `task test:e2e` に新 test 名が表示される

## 8. lint / format / テスト最終ゲート

- [ ] 8.1 `task lint`。完了判定: exit code 0
- [ ] 8.2 `task format`。完了判定: 整形差分のみ
- [ ] 8.3 `go test ./internal/... ./`。完了判定: exit code 0
- [ ] 8.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev`。完了判定:
      exit code 0
- [ ] 8.5 `task test:e2e`。完了判定: exit code 0
