## Context

動機は proposal.md の Why を参照。
ここでは実現方法を決めるうえで効く現状の制約だけを挙げる。

- `frontend/vite.config.ts` は `({ mode }) => ({...})` の関数形式で、`react()` / `tailwindcss()` / `wails("./bindings")` に加え、
  `mode === "e2e"` のときだけ bindings を fetch 版 shim に差し替えるプラグインを持つ。
- `frontend/playwright.config.ts` は `testDir: "./e2e"` で、E2E のテストファイル名は `<capability>[-<topic>].spec.ts`。
- テスト対象の純関数の一部は、生成 bindings から型だけでなく値も import している
  (`stock-filter.ts` の `StockStatus` は `internal/domain/shopping/models` の enum)。
  `frontend/bindings/` は `.gitignore` 対象で、`task dev` / `task build` / `wails3 generate bindings` で生成される。
- `week-utils.ts` はローカルタイムゾーンで日付を組み立てる (`new Date(y, m, d)` と `getDay()`)。
  実行環境の TZ によって `weekStartOf` などの結果が変わりうる。
- `mealplan-preferences.ts` は `window.localStorage` を読み書きし、例外を握り潰して既定値に倒す。
- 依存の版：Vite 8、React 19、TypeScript 5.6。Vitest 5 系は peer に `vite ^8.0.0` を含み、
  `@testing-library/react` 16 系は React 19 と `@testing-library/dom ^10` を peer に取る。

## Goals / Non-Goals

**Goals:**

- `cd frontend && npm run test` 1 本で、`src/` 配下のユニットテストが実行環境に依存せず同じ結果になる。
- 既定のビルド (`task dev` / `task build`) と E2E (`task test:e2e`) の経路に影響を与えない。
- Playwright の spec と Vitest のテストが互いのファイルを拾わない。

**Non-Goals:**

- テストから Wails の IPC や Go サービスを呼ぶこと。ユニットテストの対象は Go を介さない関数と hook に限る。
- i18next を初期化した状態でのテスト。今回の対象モジュールは `t()` を使わない。

## Decisions

### D1. Vitest の設定をどこに置くか

**選択肢:**

- (a) `vite.config.ts` に `test` ブロックを足す — Pros: 設定ファイルが 1 つで alias が二重にならない /
  Cons: `wails()` と `tailwindcss()` がテスト実行時にも読み込まれる。`defineConfig` を `vitest/config` 由来に替える必要があり、
  本番ビルドの設定ファイルがテストランナーの型に依存する
- (b) `vitest.config.ts` で `mergeConfig` を使い `vite.config.ts` を継承する — Pros: alias を共有できる /
  Cons: 関数形式の設定を `mode` 付きで呼び出す必要があり、(a) と同じくテストに不要なプラグインを持ち込む
- (c) `vitest.config.ts` を独立させ、`react()` と `@/` alias だけを持たせる ← 採用 — Pros: テストに必要なものだけを読む。
  `vite.config.ts` は無変更で済む / Cons: `@/` alias の定義が 2 箇所になる

**採用: (c)**。
理由: (a) と (b) は、テストに関係しない `wails()` プラグイン (bindings の仮想モジュール解決) をテスト経路に入れる。
これが Vitest のモジュール解決と干渉したとき、原因の切り分けにテストランナーと Wails の両方を調べることになる。
alias の重複は 1 行で、ずれた場合は import 解決エラーとしてすぐに表面化する。

### D2. テスト環境 (DOM 実装) をどう選ぶか

**選択肢:**

- (a) 既定を `node` にし、DOM が要るファイルだけ `// @vitest-environment happy-dom` を書く — Pros: 純関数のテストが最速 /
  Cons: 書き忘れると `window is not defined` で落ち、ファイルごとに判断が要る
- (b) 全体を happy-dom にする ← 採用 — Pros: ファイル先頭の指定が不要。localStorage や hook のテストもそのまま書ける /
  Cons: 純関数のテストでも DOM の初期化コストを払う
- (c) 全体を jsdom にする — Pros: ブラウザ互換性の実績が長い / Cons: happy-dom より起動が重く、本 proposal の対象では差が出る機能を使わない

**採用: (b)**。
理由: テストファイルが数十本規模のうちは、環境初期化のコストより「書くたびに環境を選ぶ」判断の手間のほうが大きい。
実行時間が問題になった時点で、純関数の多いディレクトリだけ `node` に切り替えられる (Vitest の `projects` 設定)。

### D3. テストファイルの置き場所と命名

**選択肢:**

- (a) 対象と同じディレクトリに `<name>.test.ts` を置く ← 採用 — Pros: 対象との対応が一目でわかり、移動や削除のときに一緒に動く /
  Cons: `src/` 配下にテストファイルが混ざる
- (b) `frontend/test/` に `src/` と同じ構造でミラーする — Pros: `src/` が本番コードだけになる /
  Cons: ディレクトリ構造の二重管理になり、リネームで対応が切れる
- (c) 各ディレクトリに `__tests__/` を切る — Pros: (a) と (b) の中間 / Cons: ファイル数の少ないディレクトリでは階層が増えるだけ

**採用: (a)**。
拡張子は `.test.ts(x)` に固定し、Playwright の `.spec.ts` と名前で区別する。
`vitest.config.ts` の `include` を `src/**/*.test.{ts,tsx}` に絞るので、`e2e/` 配下は対象にならない。
Vite のビルドはエントリから辿れるモジュールだけを含めるため、`*.test.ts` が `dist/` に入ることはない。

### D4. タイムゾーンの固定方法

**選択肢:**

- (a) `vitest.config.ts` の先頭で `process.env.TZ = "Asia/Tokyo"` を設定する ← 採用 — Pros: 実行方法 (npm script / IDE / `npx vitest`) を問わず効く /
  Cons: 設定ファイルに副作用のある 1 行が入る
- (b) npm script を `TZ=Asia/Tokyo vitest` にする — Pros: 設定ファイルが純粋なまま /
  Cons: Windows の cmd / PowerShell では環境変数の前置構文が通らない。`npx vitest` で直接叩くと効かない
- (c) テスト側で TZ に依存しない書き方に揃える — Pros: 固定が要らない /
  Cons: `weekStartOf` の「月曜に揃う」はローカル日付の性質そのものであり、TZ 非依存に書き直すと検証したい内容が抜ける

**採用: (a)**。
Vitest は設定ファイルを読んだ後にワーカーを起動するので、ワーカーはこの環境変数を引き継ぐ。
`Asia/Tokyo` を選ぶのは、アプリの利用者の環境と一致させるためで、夏時間が無いことは選定理由ではない。
夏時間を跨ぐ `periodDates` の挙動は、必要になった時点でテスト内で個別に TZ を切り替えて確かめる。

### D5. `describe` / `it` / `expect` を明示 import するか

**選択肢:**

- (a) `globals: true` にして import を省く — Pros: テストコードが短い /
  Cons: `tsconfig.json` の `types` に `vitest/globals` を足す必要があり、`src/` の本番コードからもグローバルの型が見えてしまう
- (b) 各テストファイルで `import { describe, expect, it } from "vitest"` と書く ← 採用 — Pros: 型の影響範囲がテストファイルに閉じる。
  Biome にグローバル宣言を足す必要も無い / Cons: 各ファイルに import 1 行が増える

**採用: (b)**。
Testing Library の自動 cleanup は `afterEach` がグローバルにあるときだけ登録されるので、hook のテストでは
`afterEach(cleanup)` を明示的に書く。

### D6. テストの書き方

CLAUDE.md の層ごとの方針に合わせる。
純関数は Go のドメイン層と同じ位置づけとして、`it.each` による table-driven で書く。
hook (`useMealPlanPreference`) はサービス層と同じく、`it("should <期待動作> when <前提>")` の BDD 風で書く。
`npx vitest -t "<名前>"` で部分実行しやすくするためで、Go 側の `go test -run` と同じ考え方である。

## Risks / Trade-offs

- [Risk] `frontend/bindings/` が未生成の状態で `npm run test` を走らせると、`stock-filter.test.ts` などが import 解決で落ちる
  → `npm run type-check` と同じ前提 (bindings が生成済み) として扱い、CLAUDE.md の「常用コマンド」に前提を書く。
  落ちたときのエラーは bindings のパスを含むので、原因はすぐに特定できる。
- [Risk] `vite.config.ts` と `vitest.config.ts` の `@/` alias がずれる
  → ずれると import 解決エラーで全テストが落ちるので、気付かないまま進むことはない。`vitest.config.ts` に、
  alias を変えるときは両方を直す旨のコメントを 1 行残す。
- [Risk] `.ts` 生成と `.js` 生成で bindings の型の付き方が違い、テストの型チェックが片方でだけ落ちる
  → テスト内のフィクスチャは `Pick<Item, "stockStatus">` のように対象関数の引数型に合わせて最小限で組み、
  生成物の完全な型を組み立てない。E2E shim と同じく、狭い型なら両方の生成形式に代入できる。
- [Trade-off] happy-dom を全体に効かせるので、純関数だけのテストにも DOM 初期化のコストがかかる
  → D2 のとおり、実行時間が問題になってから `projects` で分割する。

## Migration Plan

開発ツーリングの追加だけなので、データの移行も段階的なロールアウトも無い。
ロールバックは、追加した devDependencies とファイルを取り除き、CLAUDE.md と config.yaml の記述を戻せば済む。

## Open Questions

- コンポーネントの描画テストを始めるとき、i18next をどう初期化するか。
  `src/i18n.ts` をそのまま読み込む setup ファイルを置くか、テスト用にキーをそのまま返す `t` を渡すかは、
  最初に描画テストを書く proposal で決める。本 proposal の対象モジュールは `t()` を使わないので、決めなくても進められる。
