## 1. WeekCalendar のレイアウト分岐変更

- [x] 1.1 `frontend/src/components/mealplan/week-calendar.tsx` の `isWrapped` を
      `periodWeeks === 4` から `periodWeeks >= 2` に変更する。完了判定: `npm run type-check` が通る
- [x] 1.2 同ファイルの `blockCount` を `isWrapped ? 4 : 1` から `isWrapped ? periodWeeks : 1` に変更し、
      2 週表示で 2 block、4 週表示で 4 block が生成されることを確認する。完了判定: 目視で 2 週表示が
      2 行、4 週表示が 4 行に折り返す
- [x] 1.3 同ファイルの `overflow-x-auto` class 適用条件を削除する (常に折り返しなので横スクロール不要)。
      `minWidth` style の分岐も削除する。完了判定: 2 週表示に横スクロール bar が出ない

## 2. E2E テスト更新

- [x] 2.1 `frontend/e2e/mealplan-assign.spec.ts` のテスト名「2 週を選ぶと 14 日分が表示される」は
      spec / feature の Scenario 名と揃えるため変更しない (design.md D3 (a))。
      完了判定: テスト名が spec と一致する
- [x] 2.2 上記テストの検証を折り返しレイアウト前提に更新する
    - `data-period-weeks="2"` の確認は継続
    - 横スクロールが発生しないこと (`overflow-x` が `auto` でないこと) の確認を追加、
      または block 要素が 2 個描画されることを確認 (`data-testid` を必要に応じて追加)
    - (完了判定: `task test:e2e` が pass する)

## 3. ドキュメント更新

- [x] 3.1 `docs/product.md` §S01 の「1 週なら 7 列、2 週なら 14 列（横スクロール想定）、4 週は行方向に
      折り返す」の記述を「1 週 = 7 列 × 1 行、2 週 = 7 列 × 2 行折り返し、4 週 = 7 列 × 4 行折り返し。
      横スクロールは発生しない」に更新する。完了判定: 目視で該当節が更新される
- [x] 3.2 `docs/scenarios/mealplan.feature` の「2 週を選ぶと 14 日分が表示される」シナリオの
      本文を新レイアウトに合わせる (2 行折り返し / 横スクロール無し)。シナリオ名は spec と
      揃えるため変更しない (design.md D3 (a))。完了判定: feature と spec で Scenario 名が揃う

## 4. lint / format / テスト最終ゲート

- [x] 4.1 `task lint`。完了判定: exit code 0
- [x] 4.2 `task format`。完了判定: `git diff` が整形のみ
- [x] 4.3 `go test ./internal/... ./`。完了判定: exit code 0 (本 proposal で Go 側は無変更だが回帰確認)
- [x] 4.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev`。完了判定:
      exit code 0
- [x] 4.5 `task test:e2e`。完了判定: exit code 0
