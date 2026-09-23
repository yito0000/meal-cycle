## 1. 選択状態 → API 呼び出し分解ヘルパ (pure function)

- [x] 1.1 `frontend/src/lib/spread-batch-selection.ts` を新設する。以下の型と関数を定義する
    - `SpreadTarget = { date: string; meal: string }`
    - `SpreadSelection = { targets: SpreadTarget[]; shareBatch: boolean; conflictMode: "append" |
      "overwrite" }`
    - `planSpread(source: { date, meal, position }, selection): { sameMealDates: string[];
      crossMeal: SpreadTarget[] }` — 選択セルを「同 meal 日群」と「別 meal セル群」に分ける
    - `dateGap(a: string, b: string): number` — ISO 日付の差分 (日数)
- [x] 1.2 現時点で test runner が入っていないため、ヘルパは types を厳密にして手動確認する。
      Vitest 導入 (別 proposal) 後にユニットテストを足す

## 2. 統合ダイアログ

- [x] 2.1 `frontend/src/components/mealplan/spread-batch-dialog.tsx` を新設する。props: `open`,
      `source: { date: string; meal: string; position: number }`, `existingEntries: Entry[]`,
      `dayRange: number`, `presetDays: number`, `onOpenChange`, `onConfirm(selection)`
- [x] 2.2 対象日レンジ (起点日 ±dayRange 日) を行、時間帯 (朝 / 昼 / 夕) を列とする格子を描画する。
      起点セルはグレーアウト。既存割当セルは warning tone の背景色
- [x] 2.3 初期チェック状態は「起点日 +1..+presetDays 日ぶんの同 meal」。ダイアログを開き直すと
      毎回この既定に戻る
- [x] 2.4 下部に `作り置き / 別調理` セグメント (既定: 作り置き) と `衝突挙動: 追加 / 上書き`
      セグメント (既定: 追加) を配置する。衝突セルが 1 つも無いときも常時表示し、UX を統一する
- [x] 2.5 OK 押下で `onConfirm(selection)` を呼ぶ。キャンセルは選択破棄

## 3. `⋯` メニューと配線

- [x] 3.1 `frontend/src/components/mealplan/cell-item-menu.tsx` から `onCopyDays` サブメニューと
      `onCopyToMeal` 項目を削除する。代わりに `onSpread` prop と「作り置きを広げる」項目を追加する。
      表示条件は「常時」(spec `事後編集による作り置きの外し / まとめ` の記述に一致)
- [x] 3.2 `frontend/src/components/mealplan/meal-plan-cell.tsx` の `EntryAction` から `"copy-days"` と
      `"copy-to-meal"` を削除、`"spread"` を追加する
- [x] 3.3 `frontend/src/components/mealplan/week-calendar.tsx` の `pendingCopy` / `copyToMeal` state を
      削除し、`spreadState` (源セル + ダイアログ open/close) に統合する。`handleEntryAction` は
      `"spread"` 経路のみに
- [x] 3.4 mutation 呼び出し: `planSpread(...)` の結果を使って `CopyLeftover` (同 meal 分) と
      `CopyToOtherMeal` (別 meal 分) を並行実行する。両方成功で invalidate、片方失敗時は
      「N 件の反映に失敗しました」toast (既存 `errorI18nKey` で分類できないケースは
      `pages.meal_plan.errors.unknown`)

## 4. 旧ダイアログ / 旧設定の削除

- [x] 4.1 `frontend/src/components/mealplan/copy-to-meal-dialog.tsx` を削除する
- [x] 4.2 `frontend/src/components/mealplan/copy-mode-dialog.tsx` を削除する
      (時間帯全体コピーは「作り置きコピー (時間帯全体)」の inline 記述に沿って、統合ダイアログとは
      別に `CopyModeDialog` 相当を残す — このタスク削除後に時間帯全体経路が壊れる場合は 4.3 で復元)
      → 4.3 の判断 (a) により据え置き。詳細は design.md D6
- [x] 4.3 `セル右下 ⋯ → 時間帯全体を連続 N 日コピー` の経路を再確認する。CopyModeDialog を削除する
      場合、時間帯全体の作り置き / 別調理選択は失われるので、
      (a) CopyModeDialog を残す or
      (b) 時間帯全体側にも `spread-batch-dialog.tsx` を再利用する (Non-goal に触れる)
      から選ぶ。既定は (a)。判断結果を design.md に追記して確定する

## 5. i18n の刷新

- [x] 5.1 `frontend/src/locales/{ja,en}.json` に以下 key を追加する
    - `mealplan.item_menu.spread` — 「作り置きを広げる」
    - `mealplan.spread.title` — ダイアログ題
    - `mealplan.spread.origin_hint` — 「起点: {{date}} {{meal}} {{recipeName}}」
    - `mealplan.spread.share_batch_label` / `mealplan.spread.distinct_label`
    - `mealplan.spread.conflict_append_label` / `mealplan.spread.conflict_overwrite_label`
    - `mealplan.spread.confirm` — 「反映」
    - `mealplan.spread.partial_failure` — 「{{count}} 件の反映に失敗しました」
- [x] 5.2 `mealplan.item_menu.copy_days` / `mealplan.item_menu.copy_to_meal` /
      `mealplan.copy_to_meal.*` を削除する。時間帯全体側で使う `mealplan.copy.*` は残す

## 6. ドキュメント + E2E

- [x] 6.1 `docs/product.md` §S01 の「セル内の操作」を「作り置きを広げる」統合 UX に書き換える
- [x] 6.2 `docs/scenarios/mealplan.feature` に以下シナリオを追加する
    - 起点日 + 2 日ぶん同時間帯が既定でチェックされる
    - 対象日 × 時間帯を自由に組み合わせて広げる
    - 別調理を選ぶと各セルが独立した作り置きになる
    - 衝突セルは追加 / 上書きの指定に従って反映される
    - キャンセルすると割当は変わらない
- [x] 6.3 `docs/scenarios/mealplan.feature` から旧シナリオを削除する
    - 「1 品を連続 3 日に作り置きコピー」
    - 「前日の夕食の作り置きを翌日の昼食に足す」
    - 「追加先が既存割当と衝突すると衝突解決ダイアログが出る」
- [x] 6.4 `frontend/e2e/mealplan-batch.spec.ts` のテスト名を新シナリオに更新する (DropdownMenu
      Portal に依存するため実行は `test.skip` で可)

## 7. lint / format / テスト最終ゲート

- [x] 7.1 `task lint`。完了判定: exit code 0
- [x] 7.2 `task format`。完了判定: 整形差分のみ
- [x] 7.3 `go test ./internal/... ./`。完了判定: exit code 0
- [x] 7.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev`。完了判定:
      exit code 0
- [x] 7.5 `task test:e2e`。完了判定: exit code 0
