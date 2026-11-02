## 1. 見直しの状態の純関数

- [x] 1.1 `frontend/src/components/fridge/stock-review.ts` を新設し、`nextUnreviewedId(rowIds, reviewed, currentId)` を書く (design.md D4)。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `frontend/src/components/fridge/stock-review.test.ts` を新設し、`it.each` の table-driven で書く。完了判定: `cd frontend && npm run test` が pass する
  - 現在の行の次に未見直しの行があれば、その ID を返す
  - 現在の行より後がすべて見直し済みなら、先頭から探した未見直しの行の ID を返す
  - すべて見直し済みなら null を返す
  - 現在の行が一覧に無い (削除済み) とき、元の位置の次の未見直しの行を返す

## 2. 見直しの行と操作

- [x] 2.1 `frontend/src/components/fridge/use-fridge.ts` を更新し、量だけを変える `useUpdateStockAmount` を足す (`FridgeService.Update` に既存の値と新しい量を渡す)。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/components/fridge/fridge-review-row.tsx` を新設し、「ある」「減った」「ない」と量の入力欄を描く (design.md D2)。
  変換確定の Enter では保存しない。完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.3 `frontend/src/components/fridge/fridge-review-bar.tsx` を新設し、「見直し済み N / M 件」と「見直しを終える」を描く。完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.4 `frontend/src/components/fridge/fridge-table.tsx` を更新し、見直しの状態のとき `fridge-review-row` を描き、見直し済みの行を薄く表示する。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.5 `frontend/src/routes/fridge.tsx` を更新し、「見直す」ボタン、見直し済みの ID の集合、フォーカスの移動を足す (design.md D3 / D4)。
  完了判定: `task dev` の S05 で目視
- [x] 2.6 `frontend/src/routes/fridge.tsx` で、「ない」の通知の「元に戻す」で戻した行を見直し済みの集合から外す。完了判定: `task dev` の S05 で目視
- [x] 2.7 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に見直しの文言を足す。完了判定: `cd frontend && npm run type-check` が通る

## 3. E2E

- [x] 3.1 `frontend/e2e/fridge-stock-review.spec.ts` を新設する。最外の describe は「在庫の見直し」にする。完了判定: `task test:e2e` で pass する
  - 見直すと各行に 3 つのボタンが出る
  - ある を押すと在庫を変えずに見直し済みになる
  - 減った で量を打ち直す
  - 減った で量を 0 にすると保存されない
  - ない を押すと行が消え、元に戻せる
  - 操作のあとは次の行にフォーカスが移る
  - 見直しを終えると通常の表示に戻る

## 4. ドキュメント

- [x] 4.1 `docs/wireframes/S05_fridge.v3.drawio.svg` を新設し、見直しの状態の S05 を描く (可視 SVG と `content=` の編集 XML の両方)。完了判定: 目視
- [x] 4.2 `docs/wireframes/README.md` に v3 を足す。完了判定: 目視
- [x] 4.3 `docs/product.md` の §4 S05 に見直しの状態の節を足し、§1「家族で使う場合」の在庫の見直しを実装済みに直す。完了判定: 目視
- [x] 4.4 `docs/design.md` の §9「在庫の見直しを速くする手段」を実装済みに直す。完了判定: 目視

## 5. 確認

- [x] 5.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全 pass
- [x] 5.2 `cd frontend && npm run test` を実行する。完了判定: 全 pass
- [x] 5.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [x] 5.4 `cd frontend && npm run build:dev` を実行する。完了判定: 成功
- [x] 5.5 `task test:e2e` を実行する。完了判定: 全 pass
- [x] 5.6 `task lint` を実行する。完了判定: エラー 0
