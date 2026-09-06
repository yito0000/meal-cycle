## Context

動機は proposal.md の Why を参照。
ここでは実現方法を決めるうえで効く現状の制約だけを挙げる。

- フロントから Go への呼び出しは、生成 bindings (`frontend/bindings/.../internal/service/*.ts`) が
  `@wailsio/runtime` の `Call.ByID(<uint32 のメソッド ID>, ...args)` を呼ぶ形に統一されている。
  ID は Wails が生成時に振るハッシュで、Go 側のメソッド名との対応表はフロントには存在しない。
- `@wailsio/runtime` の既定トランスポートは `${window.location.origin}/wails/runtime` への `fetch` (`dist/runtime.js`)。
  同ファイルは `setTransport(transport)` を公開しており、`call(objectID, method, windowName, args)` を差し替えられる。
- 生成 bindings は `$CancellablePromise` を返す。
  `CancellablePromise` は `Promise` を継承したクラスで、`new CancellablePromise(executor, oncancelled)` として構築できる。
- 公開サービスのメソッドは大半が `func (s *T) M(ctx context.Context, args...) (R, error)` の形だが、揃ってはいない。
  `HealthService.Ping() Health` は `ctx` を取らず `error` も返さない。
  ディスパッチャは第 1 引数の `context.Context` と戻り値末尾の `error` を、いずれも省略可として扱う必要がある。
- コンポーネントは bindings を相対パス (`../../bindings/github.com/...`) で import している。
  Vite の `resolve.alias` は import 指定子の文字列に対して働くため、相対パスの import には効かない。
- `frontend/bindings/` は `.gitignore` 対象で、`task dev` / `task build` が再生成する。
  手で編集してはいけない。
- 開発環境は WSL2 上の AlmaLinux 10。
  Playwright の `install-deps` は Debian 系専用で、この環境では使えない。

## Goals / Non-Goals

**Goals:**

- React、Go サービス、SQLite を本物のまま通した経路を、WSL2 上のヘッドレス実行で検証できるようにする。
- shim と生成 bindings のずれを、実行時ではなく `npm run type-check` で検出できる構造にする。
- 既定のビルドとホットリロードに影響を与えない。
  E2E 用の分岐は `mode === "e2e"` の中に閉じる。
- テストごとに DB を使い捨て、実行順序に依存しないようにする。

**Non-Goals:**

- Wails 既定トランスポートとの完全な等価性。
  chunked 転送 (32KB 超のボディ) とイベント配信は再現しない。
- 本番バイナリ (`main.go`) の構成変更。
  E2E 用の経路を本番側に混ぜない。

## Decisions

### D1. IPC の縫い目をどこに置くか

**選択肢:**

- (a) 実 webview を外部ドライバから駆動する — Pros: 本物のアプリそのものを検証できる /
  Cons: Linux 側は WebKit2GTK で CDP を持たず、接続する口が無い。実現不能
- (b) `setTransport()` で既定トランスポートを差し替える — Pros: 公開 API で、bindings を触らない /
  Cons: `call` に渡るのはメソッド ID (uint32) なので、ID から Go のメソッド名を引く対応表を自前で維持する必要がある。
  bindings 再生成で ID が変わると静かに壊れる
- (c) 生成 bindings モジュールを shim モジュールに解決し直す ← 採用 — Pros: 縫い目が「関数名」なので型で検査できる。
  ID ハッシュに依存しない / Cons: bindings の export 集合を shim 側で追う手間が要る
- (d) JS でバックエンドを模す — Pros: サーバプロセスが不要 /
  Cons: バリデーションと永続化を TypeScript で二重実装することになり、検証対象が本物でなくなる

**採用: (c)**。
理由: (b) の ID 対応表は bindings 再生成のたびに破綻し、しかも破綻がテスト失敗としてしか現れない。
(c) は shim が生成 bindings と同じ export 名と同じシグネチャを持つことを型検査で強制でき、ずれが `type-check` で落ちる。

### D2. shim への差し替え方法

**選択肢:**

- (a) `resolve.alias` — Pros: 設定 1 行 / Cons: 相対 import には効かない
- (b) Vite プラグインの `resolveId` フック ← 採用 — Pros: 解決後の絶対パスで判定できるので相対 import でも捕まえられる /
  Cons: 20 行程度のプラグインを書いて保守する必要がある
- (c) E2E 実行前に shim を `frontend/bindings/` へコピーする — Pros: プラグイン不要 /
  Cons: 生成物ディレクトリを手で汚す。`task dev` の再生成と競合し、失敗時に開発環境が壊れた状態で残る

**採用: (b)**。
`resolveId` の `order: "pre"` で、解決結果が `/bindings/github.com/yito0000/meal-cycle/internal/service` 配下になる
import を `frontend/e2e/bindings-shim/` に振り替える。
`internal/domain/*/models` (型のみ) は振り替えない。
プラグインは `mode === "e2e"` のときだけ `plugins` に入れるので、通常のビルド経路には存在しない。

### D3. E2E サーバをどう構成するか

**選択肢:**

- (a) `main.go` にフラグを足して再利用する — Pros: 配線が 1 箇所 /
  Cons: 本番バイナリにテスト専用の HTTP 経路が入る。完全ローカルを謳う製品として持ちたくない
- (b) 独立した `cmd/e2e-server` ← 採用 — Pros: 本番バイナリに影響しない。`cmd/seed` と同じ扱いで配布物に含めない /
  Cons: サービス登録が `main.go` と 2 箇所になる
- (c) Go の `httptest` をテスト側から起動する — Pros: Go のテストとして完結 /
  Cons: ドライバは Node 側の Playwright なので、結局プロセスを跨いで起こす必要があり利点が消える

**採用: (b)**。
(c) の懸念どおり Playwright が主導するため、独立プロセスとして起動できることが要件になる。
サービス登録が 2 箇所になる点は、`cmd/e2e-server` が起動時に `main.go` と同じ 3 サービスを登録しているかを
`dispatch_test.go` で確認して抑える。

### D4. ディスパッチの実装

**選択肢:**

- (a) メソッドごとに手書きハンドラを並べる — Pros: 明示的で読みやすい /
  Cons: サービス追加のたびに書き漏らす。漏れは E2E の失敗としてしか現れない
- (b) リフレクションによる汎用ディスパッチ ← 採用 — Pros: メソッドが増えても変更不要。
  引数は `reflect.Type` から `json.Unmarshal` で復元できる / Cons: 型の取り違えが実行時にしか出ない
- (c) Wails のバインディング生成器を再利用する — Pros: 本物と同じ経路 /
  Cons: `wails/v3/internal/` 配下で、外部パッケージから import できない

**採用: (b)**。
(b) の Cons は `cmd/e2e-server/dispatch_test.go` で潰す。
`POST /api/{Service}/{Method}` のボディを `[]json.RawMessage` として受ける。
メソッドの第 1 引数が `context.Context` ならそこに `r.Context()` を渡し、残りの引数の型に個別に unmarshal する。
戻り値は末尾が `error` なら、非 nil で 500、nil なら残りの戻り値を JSON で返す。
`error` を返さないメソッド (`HealthService.Ping`) は戻り値をそのまま JSON で返す。

### D5. shim と生成 bindings のずれをどう検出するか

shim は戻り値を `CancellablePromise` にする。
`@wailsio/runtime` からクラスをそのまま import して `new CancellablePromise((resolve, reject) => { ... })` で構築すれば、
生成 bindings と同じ型になる。
そのうえで `frontend/e2e/bindings-shim/service.ts` の末尾に構造互換の検査を置く。

```ts
import type * as Generated from "../../bindings/github.com/yito0000/meal-cycle/internal/service";
// 生成 bindings に無い関数、シグネチャ違い、export 漏れは type-check で落ちる
const _typeCheck: typeof Generated = { Create, GetByID, List, Update, ListWeek, Get } as const;
void _typeCheck;
```

`npm run type-check` は `frontend/e2e/` も対象に入るので、bindings 再生成で signature が変わった時点で落ちる。
検査を実行時アサーションではなく型で行うのは、E2E を回さない普通の開発サイクルでも検出させるため。

**他案**: shim を生成する codegen を書く案も検討した。
生成器が Wails の生成器と二重になり、維持対象が増えるだけで検出力は型検査と変わらないため見送った。

### D6. テストごとのデータ分離とシード

- Playwright の worker 単位 fixture で `cmd/e2e-server` を 1 プロセス起動する。
  DB は `os.tmpdir()` 下の worker 固有ディレクトリに作り、worker 終了時に削除する。
- 「レシピが 1 件も登録されていない」シナリオが必要なので、シードは既定で行わない。
  `-seed` フラグを付けたときだけ `internal/sampledata` のサンプルを投入する。
- worker 内でテストが状態を共有しないよう、spec ファイルごとに `fullyParallel: false` とし、spec 単位で fixture を作り直す。
  worker 間は DB が別なので並列でよい。

**他案**: 各テストの冒頭に「全レシピ削除」の API を叩く案もある。
削除 API はまだ実装が無く (`add-recipe-deletion` は別 proposal)、テストのためだけに本番サービスへ経路を足すことになるため採らない。

### D7. ポートの決め方

`cmd/e2e-server` は `-port 0` を既定とし、実際に割り当てられたポートを起動直後に stdout へ 1 行で出力する。
fixture はその行を読んでから `page.goto` する。
固定ポートにしないのは、worker 並列と WSL 側の他プロセスとの衝突を避けるため。
Playwright の `webServer` は固定 URL を要求するので使わず、fixture 内で子プロセスを起こす。

### D8. 画面遷移はクリックで行う

ルータは memory history なので、`page.goto("/recipes")` では遷移せず常に `/` が描画される。
テストはサイドバーやボタンのクリックで遷移させる。
`cmd/e2e-server` の SPA fallback は、直接 URL を叩いたときに 404 を返さないための保険として残す。

### D9. テスト名を `.feature` に揃える

`test()` の第 1 引数を `docs/scenarios/*.feature` の `シナリオ:` 行と一字一句同じにする。
`describe()` は `機能:` 行に揃える。
これで grep 一発でシナリオと自動テストを突き合わせられる。
まだ実装していないシナリオは `test.skip` で名前だけ置き、`.feature` 側の `@planned` と対応させる。

## Risks / Trade-offs

- [AlmaLinux 10 で `npx playwright install --with-deps` が使えない] → ブラウザ本体は `npx playwright install chromium`
  で入る。
  実装時に検証したところ、AlmaLinux 10 では追加の `dnf` パッケージなしで headless Chromium が起動した。
  他環境で不足が出た場合に備え、確認手順を `frontend/e2e/README.md` に残す。
- [shim が生成 bindings とずれる] → D5 の型検査で `npm run type-check` が落ちる。
  `task lint` の経路には type-check が入っていないため、tasks で `test:e2e` の前段に type-check を挟む。
- [リフレクションのディスパッチが引数型を取り違える] → `dispatch_test.go` で引数 0 個 / 1 個 / 2 個、構造体引数、
  エラー返却、不正 JSON、未知のメソッド名を検証する。
- [Wails の IPC 層を通らないため、本番でだけ壊れる経路が残る] → E2E が緑でも Windows 実機確認は省かない。
  CLAUDE.md の 2 階建て構成はそのまま維持し、E2E は下段の自動化として位置づける。
- [E2E が増えて実行時間が伸びる] → E2E は画面を跨ぐ経路だけに使う。
  入力バリデーションの分岐網羅などは Vitest 側 (別 proposal) に寄せる。
  本 proposal では spec 2 ファイル、テスト 10 本程度に抑える。
- [`frontend/e2e/` を Biome が既存ルールで弾く] → shim は生成物ではなく手書きコードなので lint 対象に残す。
  Playwright の `test` / `expect` グローバルは import して使い、除外設定を増やさない。

## Migration Plan

既存の動作を変えないため、移行もロールバックも段階を持たない。
`frontend/vite.config.ts` の変更が既定ビルドに影響していないことは、`npm run build:dev` が従来どおり通ることで確認する。
問題が起きた場合は、追加したファイル群と `vite.config.ts` の差分を落とすだけで元に戻る。

## Open Questions

- worker 並列数をいくつにするか。
  まずは Playwright の既定に任せ、WSL2 上での実行時間を見てから `playwright.config.ts` で固定するか決める。
  これは後から変えても spec もタスク分割も動かない。
