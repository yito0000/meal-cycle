## 1. テキストを組む純関数

- [x] 1.1 `frontend/src/components/mealplan/mealplan-text.ts` を新設し、`buildMealPlanText(entries, period, t)` を書く (design.md D1 / D3)。
  戻り値は `{ text, dayCount }`。完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `frontend/src/components/mealplan/mealplan-text.test.ts` を新設し、`it.each` の table-driven で書く。完了判定: `cd frontend && npm run test` が pass する
  - 1 行目に期間の見出しを置く
  - 同じ時間帯の品をセル内の順に「、」で区切る
  - 同じ日の行を朝、昼、夕の順に並べる
  - 献立の無い日付と時間帯の行を出さない
  - dayCount は行のある日付の数を返す
  - 献立が 0 件なら dayCount は 0

## 2. ボタン

- [x] 2.1 `frontend/src/components/mealplan/copy-mealplan-text-button.tsx` を新設し、`navigator.clipboard.writeText` で書き込み、成功と失敗を通知で知らせる。
  `frontend/src/components/shopping/copy-shopping-text-button.tsx` と同じ作法にする。完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/routes/meal-plan.tsx` の下部の CTA の並びにボタンを足し、表示中の期間の献立が 0 件なら押せなくする。完了判定: `task dev` の S01 で目視
- [x] 2.3 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.mealPlan.copy.*` の文言を足す。完了判定: `cd frontend && npm run type-check` が通る

## 3. E2E

- [x] 3.1 `frontend/e2e/mealplan-copy-text.spec.ts` を新設する。最外の describe は「献立のテキストコピー」にする。
  クリップボードは `frontend/e2e/shopping-copy.spec.ts` と同じ方法で読む。完了判定: `task test:e2e` で pass する
  - 表示中の期間の献立をコピーする
  - 夕食のみの表示でも昼食の献立を含める
  - 献立の無い期間ではコピーできない

## 4. ドキュメント

- [x] 4.1 `docs/wireframes/S01_meal_calendar.v2.drawio.svg` の下部に「献立をテキストでコピー」を足す (可視 SVG と編集 XML の両方)。完了判定: 目視
- [x] 4.2 `docs/product.md` の §4 S01 にテキストでコピーの節を足し、§1「家族で使う場合」を実装済みに直す。完了判定: 目視
- [x] 4.3 `docs/design.md` の §9 の献立のテキストでのコピーを実装済みに直す。完了判定: 目視

## 5. 確認

- [x] 5.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [x] 5.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [x] 5.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [x] 5.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [x] 5.5 `task test:e2e` を実行する。完了判定: 全 pass
- [x] 5.6 `task lint` を実行する。完了判定: エラー 0
