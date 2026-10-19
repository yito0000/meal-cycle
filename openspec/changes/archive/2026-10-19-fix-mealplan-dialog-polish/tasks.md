## 1. 日付の表記を 1 つの関数にまとめる

- [x] 1.1 `frontend/src/lib/format-meal-date.test.ts` を新設し、`formatMealDate` の table-driven test を `it.each` で書く。
  足した時点で import 先が無く落ちることを `cd frontend && npm run test` で確かめる
  - `ja` で `2026-10-20` を渡すと `10/20(火)` になる
  - `en` で `2026-10-20` を渡すと `Tue, 10/20` になる
  - 月末をまたぐ `2026-10-31` と `2026-11-01` がローカル日付のまま `10/31(土)` と `11/1(日)` になる
  - 同じ言語で 2 回呼んでも同じ文字列を返す
- [x] 1.2 `frontend/src/lib/format-meal-date.ts` を新設し、`formatMealDate(isoDate: string, language: string): string` を定義する。
  `Intl.DateTimeFormat` (`month: "numeric"`、`day: "numeric"`、`weekday: "short"`) を言語ごとに `Map` に保持して使い回す。1.1 のテストが通ることで確かめる
- [x] 1.3 `frontend/src/components/mealplan/spread-batch-dialog.tsx` を更新し、`dateFmt` と `formatRow` を消して `formatMealDate(date, i18n.language)` を使う。
  既存の E2E「起点の日付は格子の行見出しと同じ表記で示される」が通ることを `task test:e2e` で確かめる
- [x] 1.4 `frontend/src/components/mealplan/meal-plan-cell.tsx` を更新し、`dateFmt` を消して `formatMealDate` を使う。
  `task dev` で S01 のセルの日付の表記が変わらないことを目視で確かめる

## 2. まとめるダイアログの日付を整える

- [x] 2.1 `frontend/src/components/mealplan/batch-merge-dialog.tsx` を更新し、候補の `item_label` に渡す `date` を `formatMealDate(c.date, i18n.language)` にする。
  `npm run type-check` で確かめる

## 3. E2E

- [x] 3.1 `frontend/e2e/mealplan-batch.spec.ts` に describe「まとめ候補の日付の表記」を足す (更新)。
  期待値は `../src/lib/format-meal-date` の `formatMealDate(MON, "ja")` で作る。`task test:e2e` で通ることを確かめる
  - まとめ候補の日付は月/日と曜日で示される
- [x] 3.2 `frontend/e2e/mealplan-batch.spec.ts` の describe「まとめ候補の一意性」で、候補の日付を `toContainText(MON)` で確かめている箇所を
  `toContainText(formatMealDate(MON, "ja"))` に置き換える (更新)。`fix-mealplan-merge-range` が足したテストも同じく置き換える。`task test:e2e` で通ることを確かめる

## 4. 「さらに N 日表示」の経路を消す

- [x] 4.1 `frontend/src/components/mealplan/spread-batch-dialog.tsx` を更新し、`expandStepDays` の prop、「さらに N 日表示」のボタン、`setFutureDays` の加算を消す。
  `futureDays` の state を消し、`initialFutureDays` から行を作る。関数のコメントの「もっと表示」の説明も消す。`npm run type-check` で確かめる
- [x] 4.2 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、`SPREAD_EXPAND_STEP` とそのコメント、`expandStepDays` の受け渡しを消す。`npm run type-check` で確かめる
- [x] 4.3 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` から `pages.meal_plan.spread.show_more` を消す (更新)。
  `grep -rn show_more frontend/src` が何も返さないことで確かめる

## 5. ドキュメント

- [x] 5.1 `docs/reviews/2026-10-19-design-ux-review.md` の R49 と R51 にチェックを入れ、proposal 名 `fix-mealplan-dialog-polish` を添える。目視で確かめる

## 6. 確認

- [x] 6.1 `go test ./internal/... ./cmd/...` を実行し、すべて通ることを確かめる
- [x] 6.2 `cd frontend && npm run test` を実行し、すべて通ることを確かめる
- [x] 6.3 `cd frontend && npm run type-check` を実行し、エラーが無いことを確かめる
- [x] 6.4 `cd frontend && npm run build:dev` を実行し、ビルドが通ることを確かめる
- [x] 6.5 `task test:e2e` を実行し、すべて通ることを確かめる (`check:scenarios` を含む)
- [x] 6.6 `task lint` を実行し、エラーが無いことを確かめる
