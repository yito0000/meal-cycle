## Why

フロントエンドの純関数を、関数単体で検証する手段が無い。
`week-utils.ts` の日付計算、`stock-filter.ts` の絞り込み、`format-quantity.ts` の単位表記などは React 側だけにあり、
Go のサービス層テストでは届かない。
現状これらは `task test:e2e` で画面を通して間接的に確かめるしかなく、境界値 (月またぎ、存在しない日付、空文字の単位) を
1 件ずつ E2E に足すと、ビルドとサーバ起動を伴う重いテストが増えていく。

`add-e2e-harness` の Non-goals で「純関数とコンポーネント単体の層は別 proposal `add-frontend-unit-tests` で扱う」と予告しており、
本 proposal がその回収にあたる。

## What Changes

- `frontend/` の devDependencies に Vitest、`@testing-library/react` (と peer の `@testing-library/dom`)、happy-dom を追加する。
- `frontend/vitest.config.ts` を新設し、テスト対象を `src/**/*.test.{ts,tsx}` に限定する。
  Playwright の `e2e/*.spec.ts` は拾わない。
- テスト実行時のタイムゾーンを `Asia/Tokyo` に固定し、日付ヘルパーのテスト結果が実行環境の TZ に左右されないようにする。
- `frontend/package.json` に `test` (1 回実行) と `test:watch` スクリプトを追加する。
- `Taskfile.yml` に `test:unit` タスクを追加する。
- 土台が機能することを示す最初のテストとして、既存の純関数 4 モジュールと、localStorage を使う設定読み書き 1 モジュールに
  ユニットテストを書く。最後の 1 つは happy-dom と Testing Library の `renderHook` が動くことの確認を兼ねる。
- `CLAUDE.md` の「常用コマンド」「テスト方針」「テスト実行ルール」と、`openspec/config.yaml` の tasks ルールと apply guidance から
  「test runner 未導入のため type-check と build:dev で代替」の記述を外し、`npm run test` を実行対象に加える。

### Non-goals

- **既存純関数の網羅**：最初に書くのは 5 モジュールだけにする。残りは、各モジュールを触る proposal の tasks で足していく。
- **コンポーネントの描画テスト**：Testing Library は入れるが、本 proposal で書くのは `renderHook` を使う hook のテスト 1 件だけにする。
  shadcn/ui や Radix を含むコンポーネントの描画テストは、必要になった proposal で追加する。
- **カバレッジ計測と閾値**：`@vitest/coverage-v8` は入れない。閾値を設けると、テストを足す動機が数値合わせに寄るため。
- **スナップショットテスト**：使わない。DOM 構造の変化で落ちるだけのテストが増え、レビューで中身を確かめにくい。
- **`task test:e2e` への組み込み**：E2E の前にユニットテストを走らせる連結はしない。両者は独立に実行する。
- **CI の追加**：`add-e2e-harness` と同じく、ローカルで完走するところまでを扱う。
- **Go 側のテスト方針の変更**：対象外。

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

(なし)

本 proposal は開発ツーリングの追加であり、プロダクトの観察可能な挙動は変えない。
`.openspec.yaml` に `skip_specs: true` を設定し、spec delta は作らない。
受け入れ条件は tasks.md 側に完了判定として書く。

docs/design.md §8 は「スキャフォールド系は proposal にしない」としている。
`add-e2e-harness` と同じく、設定の置き場所やテスト環境に複数の選択肢があり、選定理由を残す価値があるため例外として proposal 化する。

## Impact

### 新設ファイル

- `frontend/vitest.config.ts`：テスト対象の glob、happy-dom 環境、`@/` alias、TZ 固定
- `frontend/src/components/mealplan/week-utils.test.ts`：`formatDate` / `parseDate` / `weekStartOf` / `periodDates` / `isSameLocalDate`
- `frontend/src/components/recipe/recipe-sort.test.ts`：`compareRecipes`
- `frontend/src/components/shopping/stock-filter.test.ts`：`matchesStockFilter` / `countByStockFilter`
- `frontend/src/lib/format-quantity.test.ts`：`formatQuantity`
- `frontend/src/lib/mealplan-preferences.test.ts`：`readPreference` / `writePreference` / `useMealPlanPreference`

### 更新ファイル

- `frontend/package.json`：`vitest` / `@testing-library/react` / `@testing-library/dom` / `happy-dom` を devDependencies に追加、`test` / `test:watch` スクリプトを追加
- `frontend/package-lock.json`：上記の追加に伴う更新 (`npm install` の結果)
- `frontend/tsconfig.json`：`include` に `vitest.config.ts` を追加する
- `frontend/biome.json`：`files.includes` に `vitest.config.ts` を追加する (`src/**/*.test.ts` は既存の `src/**/*.ts` で対象になる)
- `Taskfile.yml`：`test:unit` タスクを追加する
- `CLAUDE.md`：「常用コマンド」に `task test:unit`、「テスト方針」のフロントエンド (ユニット) 節を書き換え、
  「テスト実行ルール」の Frontend 項目を書き換える
- `openspec/config.yaml`：`rules.tasks` のテスト実行項目と `operations.apply.guidance` の Frontend 項目を書き換える

### 生成物 (再生成のみ)

- `frontend/bindings/`：テスト対象モジュールが `StockStatus` などの値を bindings から import するため、実行前に存在している必要がある。
  `wails3 generate bindings -clean=true -ts -i` で再生成する (中身は変わらない)

### migration

なし。

### 後続 proposal への引き継ぎ

- フロントの純関数を新設または変更する proposal では、tasks に対応する `*.test.ts` の追加を含める。
  テストケースは config.yaml の tasks ルールどおりネストした箇条書きで列挙する。
- 純関数のテスト層ができたので、E2E に置いている境界値の確認のうち画面を通す必要のないものは、ユニットテストへ移す余地がある。
  移すかどうかは、各 capability の proposal で判断する。
- コンポーネントの描画テストを始める proposal では、i18next の初期化 (`src/i18n.ts`) をテスト用 setup ファイルで行う必要がある
  (design.md の Open Questions 参照)。
