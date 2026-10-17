## 1. 依存とスクリプト

- [x] 1.1 `frontend/` で `npm install -D vitest@^5 @testing-library/react@^16 @testing-library/dom@^10 happy-dom` を実行し、
      `frontend/package.json` の devDependencies と `frontend/package-lock.json` を更新する。
      完了判定: `npm ls vitest @testing-library/react @testing-library/dom happy-dom` が peer 警告なしで 4 つとも表示する
- [x] 1.2 `frontend/package.json` の `scripts` に `"test": "vitest run"` を追加する。
      完了判定: テストファイルが 0 本の状態で `npm run test` が「No test files found」で終了することを確認する (終了コードは問わない)
- [x] 1.3 `frontend/package.json` の `scripts` に `"test:watch": "vitest"` を追加する。
      完了判定: `npm run test:watch` が watch モードで起動し、`q` で終了できる

## 2. Vitest の設定

- [x] 2.1 `frontend/vitest.config.ts` を新設し、`vitest/config` の `defineConfig` で `react()` プラグインと `@/` alias だけを持つ設定を書く。
      `vite.config.ts` は読み込まない (design.md D1)。alias を変えるときは `vite.config.ts` と両方直す旨のコメントを 1 行付ける。
      完了判定: `npx vitest run --passWithNoTests` が設定エラーなく終了する
- [x] 2.2 `frontend/vitest.config.ts` の `test.include` を `["src/**/*.test.{ts,tsx}"]` にする。
      完了判定: `npx vitest list` の出力に `e2e/` 配下のファイルが含まれない
- [x] 2.3 `frontend/vitest.config.ts` の `test.environment` を `"happy-dom"` にする (design.md D2)。
      完了判定: 3.5 の localStorage を使うテストが通る
- [x] 2.4 `frontend/vitest.config.ts` の先頭で `process.env.TZ = "Asia/Tokyo"` を設定し、理由をコメント 1 行で残す (design.md D4)。
      完了判定: `TZ=UTC npm run test` と `TZ=America/Los_Angeles npm run test` の両方で 3.1 のテストが通る
- [x] 2.5 `frontend/tsconfig.json` の `include` に `vitest.config.ts` を追加する。
      完了判定: `npm run type-check` が通る
- [x] 2.6 `frontend/biome.json` の `files.includes` に `vitest.config.ts` を追加する。
      完了判定: `npm run lint` の対象に `vitest.config.ts` が含まれ、エラーなく通る

## 3. 最初のユニットテスト

各ファイルで `describe` / `it` / `expect` を `vitest` から明示 import する (design.md D5)。
純関数は `it.each` の table-driven、hook は `it("should ... when ...")` で書く (design.md D6)。

- [x] 3.1 `frontend/src/components/mealplan/week-utils.test.ts` を新設し、日付ヘルパーのテストを書く。
      完了判定: `npx vitest run src/components/mealplan/week-utils.test.ts` が通る
  - `formatDate` が 1 桁の月と日をゼロ埋めして `2026-01-05` の形で返す
  - `parseDate` が `undefined` に `null` を返す
  - `parseDate` が `2026/10/15` のように形式の違う文字列に `null` を返す
  - `parseDate` が存在しない日付 `2026-02-30` に `null` を返す (繰り上がりで 3/2 にならない)
  - `parseDate` が `2026-10-15` を 2026 年 10 月 15 日のローカル日付として返す
  - `weekStartOf` が月曜 `2026-10-12` をそのまま返す
  - `weekStartOf` が水曜 `2026-10-14` に月曜 `2026-10-12` を返す
  - `weekStartOf` が日曜 `2026-10-18` に前の月曜 `2026-10-12` を返す
  - `weekStartOf` が月をまたぐ日曜 `2026-11-01` に `2026-10-26` を返す
  - `periodDates` が開始日から 14 日分を連続した日付で返し、月末をまたいでも日が飛ばない
  - `weekDates` が 7 要素を返す
  - `isSameLocalDate` が同じ日の異なる時刻に `true` を返す
  - `isSameLocalDate` が異なる日に `false` を返す
- [x] 3.2 `frontend/src/components/recipe/recipe-sort.test.ts` を新設し、`compareRecipes` のテストを書く。
      フィクスチャは `name` と `updatedAt` だけを持つ値を `Recipe` として渡し、生成物の全フィールドを組み立てない (design.md Risks)。
      完了判定: `npx vitest run src/components/recipe/recipe-sort.test.ts` が通る
  - `updated-desc` で `updatedAt` が新しいほうが先に並ぶ
  - `updated-desc` で `updatedAt` が同じなら 0 を返す
  - `name-asc` で名前の昇順に並ぶ
- [x] 3.3 `frontend/src/components/shopping/stock-filter.test.ts` を新設し、在庫の絞り込みのテストを書く。
      完了判定: `npx vitest run src/components/shopping/stock-filter.test.ts` が通る
  - `all` は 3 種類の `stockStatus` すべてに `true` を返す
  - `needed` は `none` と `partial` に `true`、`covered` に `false` を返す
  - `covered` は `covered` にだけ `true` を返す
  - `countByStockFilter` が空配列に 3 項目とも 0 を返す
  - `countByStockFilter` が `none` 1 件 / `partial` 1 件 / `covered` 1 件に `{ all: 3, needed: 2, covered: 1 }` を返す
- [x] 3.4 `frontend/src/lib/format-quantity.test.ts` を新設し、`formatQuantity` のテストを書く。
      完了判定: `npx vitest run src/lib/format-quantity.test.ts` が通る
  - 量る単位 `大さじ` は単位を前に置いて `大さじ 1` を返す
  - 量る単位以外の `g` は量を前に置いて `200 g` を返す
  - 量が 0 なら単位だけを返す
  - 単位が空文字なら量だけを返す
  - 単位の前後の空白を取り除いてから判定する (` 大さじ ` でも `大さじ 1` になる)
  - 小数の量は小数第 2 位までに丸める (`0.333` と `個` で `0.33 個`)
  - 丸めた結果の末尾の 0 を出さない (`1.5` と `個` で `1.50 個` ではなく `1.5 個`)
- [x] 3.5 `frontend/src/lib/mealplan-preferences.test.ts` を新設し、設定の読み書きのテストを書く。
      各テストの前に `localStorage.clear()`、後に Testing Library の `cleanup` を呼ぶ (design.md D5)。
      完了判定: `npx vitest run src/lib/mealplan-preferences.test.ts` が通る
  - `mealPlanKey` が `mealcycle.mealplan.` を前置したキーを返す
  - should return default value when key does not exist
  - should return default value when stored value is not valid JSON
  - should return default value when validate rejects the stored value
  - should return stored value when validate accepts it
  - should store value as JSON when writePreference is called
  - should not throw when localStorage.setItem throws (`vi.spyOn(Storage.prototype, "setItem")` で例外を投げさせる)
  - should initialize hook state from stored value when useMealPlanPreference mounts (`renderHook`)
  - should persist value to localStorage when hook updater is called (`renderHook` + `act`)

## 4. Taskfile とドキュメント

- [x] 4.1 `Taskfile.yml` に `test:unit` タスクを追加する。`dir: frontend` で `npm run test` を実行し、
      `summary` は既存タスクと同じく英語 1 行で書く。
      完了判定: リポジトリルートで `task test:unit` が通り、`task --list` に表示される
- [x] 4.2 `CLAUDE.md` の「常用コマンド」のコードブロックに `task test:unit` と `cd frontend && npm run test:watch` を追記する。
      bindings が生成済みであることが前提である旨を、既存の `frontend/bindings/` の説明段落に 1 文足す。
      完了判定: 目視
- [x] 4.3 `CLAUDE.md` の「テスト方針」のフロントエンド (ユニット) 節を書き換え、Vitest + happy-dom、`*.test.ts` を対象と同じディレクトリに置くこと、
      純関数は `it.each`、hook は `should ... when ...` で書くことを記述する。
      完了判定: 目視。「テストランナーは入れていない」の記述が残っていない
- [x] 4.4 `CLAUDE.md` の「テスト実行ルール」の Frontend 項目を、`cd frontend && npm run test`、`npm run type-check`、`npm run build:dev` の
      3 つを実行する内容に書き換える。
      完了判定: `grep -n "test runner がまだ入っていない" CLAUDE.md` が何も返さない
- [x] 4.5 `openspec/config.yaml` の `rules.tasks` のテスト実行項目の Frontend 行を、`npm run test` を含む内容に書き換える。
      完了判定: `grep -n "test runner 未導入" openspec/config.yaml` が何も返さない
- [x] 4.6 `openspec/config.yaml` の `operations.apply.guidance` のテスト実行項目の Frontend 行を、`npm run test` を含む内容に書き換える。
      完了判定: `openspec validate add-frontend-unit-tests` が通り、4.5 と同じ grep が何も返さない

## 5. 全体の確認

- [x] 5.1 `cd frontend && npm run test` を実行し、3 章の全テストが通ることを確認する。
      完了判定: 5 ファイルすべて pass
- [x] 5.2 `cd frontend && npm run type-check` を実行する。
      完了判定: エラーなし
- [x] 5.3 `cd frontend && npm run build:dev` を実行し、`frontend/dist/` に `*.test` 由来のコードが含まれないことを確認する。
      完了判定: ビルド成功、かつ `grep -rl "vitest" frontend/dist` が何も返さない
- [x] 5.4 `go test ./internal/... ./cmd/...` を実行する。
      完了判定: 全 pass (Go 側は変更しないが、テスト実行ルールに従い確認する)
- [ ] 5.5 `task test:e2e` を実行し、Playwright が Vitest のテストファイルを拾わず、既存の E2E が通ることを確認する。
      完了判定: 全 pass、かつ実行対象に `*.test.ts` が含まれない
- [x] 5.6 `task lint` を実行する。
      完了判定: エラーなし
