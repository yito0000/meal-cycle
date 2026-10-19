# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト

meal-cycle は **完全ローカル**（外部通信なし）のデスクトップアプリ。週次の献立計画・冷蔵庫在庫・買い物リストを扱う。Wails v3 の単一バイナリで配布する。

背景・意図は [docs/product.md](docs/product.md)（UX / 画面 / Gherkin）と [docs/design.md](docs/design.md)（アーキテクチャ / スタック / 設計判断）に集約している。**大きめの変更を入れる前に両方に目を通すこと**。コードからは読めない意思決定の履歴がここにある。

## 常用コマンド

すべてリポジトリルートから実行する。

```bash
task dev                    # Wails ホットリロード開発モード（Go 再ビルド + Vite dev）
task run                    # 1 回ビルドして起動
task build                  # プロダクションビルド
task package                # 配布用パッケージ
task --list                 # 全タスク一覧

go test ./internal/...      # Go の単体テスト（./... は避ける — build/ios に main がなく落ちる）
go build .                  # Go 側だけコンパイル（軽い健全性チェック）
wails3 generate bindings    # Go サービスを追加/変更したら frontend/bindings/ を再生成

cd frontend && npm run type-check   # TypeScript 型チェック
cd frontend && npm run build:dev    # Vite ビルド（import 解決の総合確認）

task test:unit              # Vitest ユニットテスト（フロントの純関数 / hook）
cd frontend && npm run test:watch   # Vitest を watch モードで起動
task test:e2e               # Playwright E2E（実 React + 実 Go サービス + 実 SQLite）

task lint                   # Go (golangci-lint) + Frontend (Biome) をまとめて実行
task lint:go                # Go 側だけ
task lint:frontend          # Frontend 側だけ
task format                 # Go + Frontend を自動整形（Biome fix + golangci-lint fmt）
```

`frontend/bindings/` は `.gitignore` 対象、`task dev` / `task build` で再生成される。生成形式は呼び方で変わる: `task dev` / `task build` は `-ts -i` 付きで **`.ts`**、素の `wails3 generate bindings` は **`.js`＋JSDoc** を出す。どちらでも型チェックが通るよう `tsconfig.json` で `allowJs` + `checkJs` を有効化している。`npm run type-check` と `npm run test` は bindings が生成済みであることを前提にする（未生成なら `wails3 generate bindings -clean=true -ts -i`）。

## アーキテクチャ

Go 側 4 レイヤ + フロントエンドを、Wails 自動生成のバインディング経由で繋いでいる:

```
Frontend (React + Vite)  ──┐
                           │  Wails IPC (bindings/)
Service (internal/service/) ─┤  ← 単一 package service に 1 capability = 1 ファイル (recipe.go / mealplan.go / fridge.go / health.go)
                           │
Domain (internal/domain/)  │  ← capability 単位: recipe/, mealplan/, fridge/
                           │
Repository (internal/storage/sqlite/)  ← Open / Migrate + capability ごとに 1 repo
```

Wails サービスは `internal/service/`（`package service`）に **1 capability = 1 ファイル** で置き、`main.go` の `application.Options.Services` で `service.NewRecipeService(db)` の形で登録する。**1 proposal ≒ 1 capability** で、通常はサービス 1 ファイル / ドメインパッケージ 1 個 / リポジトリ 1 個 / マイグレーション 1 個を同時に足す。単一パッケージにまとめる理由は、サブパッケージ (`internal/service/recipe/`) にすると `internal/domain/recipe/` と名前が衝突して `main.go` で import alias が必要になるため。

**SQLite 周り**:
- ドライバ: `modernc.org/sqlite`（Cgo なし — クロスコンパイルが素直）
- マイグレーション: `pressly/goose`、SQL ファイルを `//go:embed migrations/*.sql` でバイナリ同梱
- 配置先: `github.com/adrg/xdg` 経由で OS 標準の場所
  - Windows: `%APPDATA%\meal-cycle\meal-cycle.db`
  - macOS: `~/Library/Application Support/meal-cycle/meal-cycle.db`
  - Linux: `~/.local/share/meal-cycle/meal-cycle.db`
- `sqlite.Open` で WAL / foreign_keys / busy_timeout(5s) を有効化

**フロントエンド**:
- **TanStack Router**、履歴は **memory history**（Wails webview の URL は `/` 固定なのでブラウザ履歴モデルは合わない）。ルート定義は `frontend/src/router.tsx`、ページ本体は `frontend/src/routes/`
- **TanStack Query** で全 Wails サービス呼び出しをラップ。async の Go バインディングに自前 state 管理を書かない
- **shadcn/ui** は CLI ではなく **手動で `src/components/ui/` にコピー**する運用。新規プリミティブは shadcn の docs からコピペ
- **Tailwind v4** の CSS-first `@theme`（`src/index.css`）を使う。**`tailwind.config.js` は無い**
- **i18n は i18next**: ユーザーに見せる文字列は必ず `t()` 経由。**コンポーネントに日本語をハードコードしない**、`src/locales/{ja,en}.json` に追加する。Go 側はエラーコード / 型付き値を返し、日本語メッセージは持たない（[design.md §7](docs/design.md)）

## テスト方針

Go 標準 `testing` のみ使う。BDD ライブラリ（Ginkgo 等）は入れない。層ごとにスタイルを変える:

**サービス層（Wails 公開サービス）**: **BDD 風の記述**。`t.Run("should <期待動作> when <前提>", ...)` を入れ子にして、テスト名だけで意図が読めるようにする。

```go
func TestRecipeService_List(t *testing.T) {
    t.Run("should return empty slice when no recipes exist", func(t *testing.T) {
        svc, _ := setup(t)
        got, err := svc.List(ctx)
        if err != nil { t.Fatal(err) }
        if len(got) != 0 { t.Errorf("want empty, got %d", len(got)) }
    })

    t.Run("should return recipes ordered by updatedAt desc when multiple exist", func(t *testing.T) {
        svc, _ := setup(t)
        // seed …
        got, _ := svc.List(ctx)
        // assert ordering …
    })
}
```

`go test -run TestRecipeService_List/should_return_empty` のように部分実行しやすい命名にする。

**ドメイン層**: ビジネスルール単体を **table-driven test** で検証する（`cases := []struct{ name string; ... }{...}` + `for _, tc := range cases { t.Run(tc.name, ...) }`）。BDD 風にしなくてよい。

**リポジトリ層（SQLite）**: **モックしない**。`t.TempDir()` で毎テスト新しい DB を作り、実 SQLite で検証する（既存 `internal/storage/sqlite/sqlite_test.go` 参照）。SQLite 自体が高速なのでこれで十分。

**ファイル配置**: `_test.go` は対象と同じパッケージ内（`package sqlite`）に置き、ホワイトボックスで書く。外部公開 API の観点だけで書きたい場合は `package sqlite_test` を切る。

**フロントエンド（ユニット）**: Vitest + happy-dom（設定は `frontend/vitest.config.ts`、TZ は `Asia/Tokyo` に固定）。`task test:unit` で実行する。

- テストは対象と同じディレクトリに `<name>.test.ts(x)` で置く。Playwright の `.spec.ts` と拡張子で分け、Vitest は `src/**/*.test.{ts,tsx}` だけを拾う
- `describe` / `it` / `expect` は `vitest` から明示 import する（`globals` は使わない）。Testing Library を使うファイルでは `afterEach(cleanup)` を書く
- 純関数はドメイン層と同じく `it.each` の table-driven、hook はサービス層と同じく `it("should <期待動作> when <前提>")` で書く
- フィクスチャは bindings の完全な型を組み立てず、対象関数が読むフィールドだけを持たせる（`.ts` / `.js` 生成の型差を避けるため）

**E2E（`frontend/e2e/`）**: Playwright + headless Chromium。Wails の IPC 層だけを HTTP に差し替え、React / Go サービス / SQLite は本物のまま通す。`task test:e2e` で実行する。

- 生成 bindings への import は、`--mode e2e` のときだけ `frontend/e2e/bindings-shim/` に差し替わる（`vite-plugin-e2e-bindings.ts`）。`task dev` / `task build` には影響しない
- shim の戻り値型は生成物より狭く書く（例: `StockItem[] | null` ではなく `StockItem[]`）。`.ts` 生成と `.js` 生成で nullable の付き方が違い、狭い型なら両方に代入できる
- サーバは `cmd/e2e-server`。`frontend/dist` を配信し、`POST /api/{Service}/{Method}` を reflect で `internal/service` に転送する。DB はテストごとに一時ディレクトリへ作り直す
- **テスト名は `openspec/specs/<capability>/spec.md` の `#### Scenario:` と一字一句揃える**。最外の `describe` 名は `### Requirement:` に揃え、ファイル名は `<capability>[-<topic>].spec.ts` にする。`task test:e2e` の最初に走る `check:scenarios` が、spec.md（と進行中 change の delta）に実在しない名前を落とす
- spec.md の全シナリオに E2E は要らない（サービス層で確かめるシナリオもある）。どのシナリオを E2E で確かめるかは proposal の tasks.md で決める
- ルータは memory history なので `page.goto("/recipes")` では遷移しない。サイドバーをクリックする
- godog / playwright-bdd などの Gherkin 実行系は入れない（判断の理由は [frontend/e2e/README.md](frontend/e2e/README.md)）
- 詳細は [frontend/e2e/README.md](frontend/e2e/README.md)

## コメント方針

ソースコードのコメントは **必要最低限、日本語で書く**。

- **書かないのが既定**。関数名 / 変数名で伝わる what はコメントに書かない (「保存する」「エラーを返す」等)。
- **書くのは WHY だけ**：隠れた制約、非自明な不変条件、特定バグへの回避、直感に反する挙動など、コード自体から読み取れない意図を残す。
- **日本語で統一**。プロジェクト全体で言語を揃え、grep とレビューコストを下げる。Wails / shadcn などの外部由来の名詞はそのまま英字で書く。
- **1〜2 行に収める**。長い解説が要る場合は、対応する proposal.md / design.md / spec.md に書き、コメントからその節へリンクする。
- **書き手側の予告 (「〜する」)** や **進行実況 (「次に〜」)** を書かない。コードの構造で表現する。
- **400 字または 20 行を超える説明** を書く場合、グローバル CLAUDE.md の指示どおり `japanese-tech-writing` / `cognitive-rhythm-writing` スキルを適用する。

```go
// 悪い例：what を書いている (関数名から自明)
// レシピを保存する
func (s *RecipeService) Create(ctx context.Context, input recipe.Recipe) (recipe.Recipe, error) { ... }

// 良い例：WHY (非自明な制約) を書いている
// updated_at を必ず進めるため、time.Now が既存 updated_at と同一秒に落ちたときは +1s する (RFC3339 秒精度)。
if now == existing.UpdatedAt { ... }
```

## 可変値は 1 箇所に集約

設定画面（S09）で変更可能にする値は、既定値を **すべて `internal/config/defaults.go`** に置く。**call site にリテラル閾値（`10 日` `3 日` など）を書かない**。サービスはコンストラクタで受け取る `configSource` の `current()` から読み（S09 で保存した値が重なる）、フロントは `useSettingsValues()` から読む。可変対象の一覧は [product.md §1](docs/product.md) が正、数値の source of truth は `defaults_test.go`（デフォルト値を変えるならこのテストと product.md の両方を更新する）。

S09 で変えられる値を足すときは、`internal/domain/settings` の `Values` と検証、`internal/service/settings.go` の key もあわせて直す（[design.md §5「設定の保存と反映」](docs/design.md)）。

## データモデルの癖

- `MealPlan` は **1 行 = 1 品**。`(date, meal)` は意図的に非一意で、複数行が主菜＋副菜＋汁物を表現する。`position` で meal (時間帯) 内の並び順、`recipeId` は NOT NULL（「未定」= 行が無い状態）。詳細は [design.md §4](docs/design.md)
- 献立の行は **`id` で指す**。`position` は並び順だけを表し、削除や並べ替えのたびに振り直されるので行の指定に使わない
- データモデルの変更は **それを必要とする proposal と一緒に**入れる。スキーマだけの proposal は作らない

## ドキュメントは load-bearing

- `docs/wireframes/*.drawio.svg` — **ワイヤーは視覚レイアウトの正**。[product.md §4](docs/product.md) は絵から読めない相互作用・遷移だけを言葉で書く。`.drawio.svg` は **可視 SVG と drawio 用の編集 XML の二重構造**（`content=` 属性内）— 片方だけ触ると乖離する。詳細は [docs/wireframes/README.md](docs/wireframes/README.md)
- `openspec/specs/<capability>/spec.md` — 振る舞いの正。受け入れシナリオもここにだけ書き、Playwright E2E のテスト名と一字一句対応させる（上記「テスト方針」の E2E 節）。proposal 前の構想段階の観点は product.md に文章で置く
- ワイヤーと product.md は節番号で相互参照している。画面の挙動を変えたら両方を更新する必要が大抵ある

## OpenSpec ワークフロー

機能開発は OpenSpec proposal 経由。**スキャフォールドは対象外** — 境界は [design.md §8](docs/design.md) 参照（chore-01 SQLite 基盤 等は proposal ではなく通常コミット）。

このリポジトリで使えるスキル（`Skill` ツールから呼ぶ、名前は憶測しない）:
- `openspec-explore` — 変更に入る前のアイデア整理・要件明確化
- `openspec-propose` — proposal + design + spec デルタ + tasks を一括生成
- `openspec-update-change` — 進行中の change を各アーティファクトの整合を取りつつ更新
- `openspec-apply-change` — proposal の tasks を実装
- `openspec-sync-specs` — archive せずに delta を `openspec/specs/` に反映
- `openspec-archive-change` — 実装完了後の確定処理

proposal は `openspec/changes/<name>/`、アーカイブは `openspec/changes/archive/`、現行仕様スナップショットは `openspec/specs/<capability>/`。proposal の順序案は [design.md §8](docs/design.md)。

**OpenSpec を通さずに実装を変えるとき**（不具合修正、小さな改善、リファクタなど）は、変更が `openspec/specs/` の Requirement / Scenario に書かれた振る舞いを変えるかを必ず判断する。変えるなら同じコミットで spec.md も直す（E2E のテスト名を変えたらシナリオ名も合わせる）。変えないなら spec.md は触らない。archive の流れを通らない変更は spec.md に反映される経路が無く、放置すると仕様と実装がずれるため。

**運用フロー（重要）**: `/opsx:propose`（または `openspec-propose`）で planning artifacts（proposal.md / specs/…/spec.md / design.md / tasks.md）を生成した**あとは、そのまま tasks.md に沿って実装を開始しない**。必ず**人間のレビューを挟み、明示的な指示（例: 「apply して」「実装始めて」「/opsx:apply」）を受けてから** `openspec-apply-change` に進む。propose は planning のみで停止するのが既定 — 続けて実装したくなっても、レビュー承認を待つこと。

**artifact 記述ルール**（詳細は [openspec/config.yaml](openspec/config.yaml) `rules` セクションが正）:
- proposal.md / design.md / spec.md は **1 行あたり 200 文字以内**（超えたら改行）
- design.md で複数実現方法がある場合は **選択肢を列挙し、各案の Pros/Cons と選定理由を書く**
- tasks.md は **1 行 = 1 指示**。複数作業を含む場合は箇条書きで分割
- tasks.md のテスト作成タスクは **各テストケースをネストした箇条書き**で列挙する

## 動作確認の 2 階建て構成

meal-cycle の開発は WSL2 (AlmaLinux 10) で行うが、WSLg + WebKit2GTK は IME 連携が不安定で、`task dev` の webview に日本語 (IME) 入力ができない。ibus / fcitx5 のセットアップを WSL 側に入れる方針は詰まりやすく、費用対効果が悪かったため、動作確認を **開発時 (WSL) と最終確認 (Windows)** の 2 階建てにする。

- **自動化された下段**：`task test:e2e` が headless Chromium で画面を跨ぐ経路を検証する。Windows 実機確認を置き換えるものではない（Wails の IPC 層と IME は通らない）。
- **開発時の軽い動作確認**：WSL2 側で `task dev` を叩き、ホットリロードで反復する。日本語入力を伴わない観点 (レイアウト、遷移、ボタン挙動、バリデーションの発火、Tab キー移動、CSS の見た目) はここで済ませる。
- **日本語入力を含む動作確認**：Windows ネイティブビルドを作って Windows 側から実行する。IME 経由の入力、日本語文字を含む保存、Markdown 記法 (`##`、`- `) など、日本語が絡む観点はここでまとめて確認する。

Windows ビルドはビルド時間と `.exe` サイズのコストがあるので、**毎コミットで作る必要はない**。以下のタイミングで走らせる:

- capability を通す proposal を実装完了したとき (アーカイブ前の最終確認)
- 日本語文字列や Markdown 入力に関わる変更を入れたとき
- リリース前

### Windows ビルド手順

WSL2 側で `task build:win` を実行する。内部では Wails 標準の `task windows:build` を呼び、次を順に走らせる:

1. フロントエンドのビルドと bindings 生成（`-ts -i`）
2. `wails3 generate syso` でアイコン・バージョン情報（`build/windows/info.json`）・manifest を埋め込む
3. `GOOS=windows go build -tags production -trimpath -ldflags="-w -s -H windowsgui" -o bin/meal-cycle.exe`

`WIN_DEST` を指定すると Windows 側のパスへ自動コピーする:

```bash
task build:win WIN_DEST=/mnt/c/Users/<Windows のユーザー名>/Desktop/
```

`WIN_DEST` 未指定なら `bin/meal-cycle.exe` (`.gitignore` 対象) が出力されるので手動で Windows 側へコピーする。Windows 側の PowerShell またはエクスプローラーからデスクトップの `meal-cycle.exe` を起動する。`modernc.org/sqlite` は Pure Go で CGO 不要のためクロスコンパイルが素直に通る。

### データベースが 2 系統になる点に注意

WSL 側で `task dev` を叩いた DB は `~/.local/share/meal-cycle/meal-cycle.db`、Windows で `.exe` を起動した DB は `%APPDATA%\meal-cycle\meal-cycle.db` に作られる。物理的に別ファイルなので、WSL で作ったレシピは Windows 側の起動には出てこない (逆も同じ)。Windows 側は空 DB から始める前提で確認する。

## テスト実行ルール

機能追加・改善・不具合修正のいずれでも、**実装の最後に必ずテストを走らせて全て pass することを確認**してから完了報告する。テストが落ちたまま「完了」と報告しない。

- Go: `go test ./internal/... ./cmd/...`（`./...` は build/ios に main が無く落ちるので避ける。`./cmd/...` は E2E サーバの dispatch / サービス登録数のテストを含む）
- Frontend: `cd frontend && npm run test`、`cd frontend && npm run type-check`、`cd frontend && npm run build:dev` の 3 つを実行する
- 画面の挙動に触れた変更では `task test:e2e` も走らせる
- OpenSpec 経由の実装では [openspec/config.yaml](openspec/config.yaml) の `operations.apply.guidance` にも同ルールを載せてある

## lint / format

Go とフロントの両方で lint/format を有効化している。実装後は **`task lint` が通ることを確認**、書き味を統一したいときは **`task format` で自動修正**する。

- **Go**: [`golangci-lint` v2](https://golangci-lint.run/)。設定は [.golangci.yml](.golangci.yml)。有効なリンタは errcheck / govet / ineffassign / staticcheck / unused、formatter は gofmt / goimports（`goimports.local-prefixes` で `github.com/yito0000/meal-cycle` を分離セクションに寄せる）
  - **バイナリは `go tool` 経由で go.mod にピン留めしている**。追加 install 不要、`go tool golangci-lint ...` で走る。バージョン更新は `go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- **Frontend**: [Biome](https://biomejs.dev/) v2 系。設定は [frontend/biome.json](frontend/biome.json)。linter + formatter が 1 ツールで完結する（ESLint / Prettier は入れていない）
  - Biome CLI は `frontend/` の devDependency に入っているので `cd frontend && npm run lint`（= `biome check`）で走る
  - `frontend/bindings/` は Wails 生成物なので lint 対象外（`biome.json` の `files.includes` で除外）
- 走らせるコマンド: 個別なら `task lint:go` / `task lint:frontend` / `task format:go` / `task format:frontend`

## コミット規約

- Conventional Commits: `docs:` / `chore:` / `feat:` / `fix:`。capability に閉じる変更は scope を付ける（例: `feat(fridge):` / `chore(openspec):`）
- スキャフォールドは `chore:`、ユーザー価値のあるスライスは `feat:`（通常 proposal 内）
- **メッセージは 1 行程度で簡潔に**。長い説明が要るときだけ空行を挟んで本文を続ける
- **AI 支援コミットには `Co-Authored-By: <実際に使ったモデル名> <noreply@anthropic.com>` trailer を付ける**（例: `Claude Opus 4.7` / `Claude Opus 5.5`）。モデルはセッションごとに変わるので固定しない。将来 OSS 公開したときに AI 関与と使ったモデルを後から特定できるようにするため

## Wails 固有の注意点

- `main.go` の初期ウィンドウサイズは `app.Screen.GetPrimary().WorkArea` の 85% (幅 2400 / 高さ 1500 で cap、失敗時は 1440×900) を使い、OS の DPI 設定にそのまま合わせる。`Zoom: 1.5` は高 DPI (4K 等) 向けの内容拡大で、Wails の WebView zoom で拡大することで dev / preview / packaged の見た目を揃える。CSS 側の基準の文字サイズは `index.css` の `html { font-size: 18px }`
- `build/ios` と `build/android` は Wails 標準のモバイル雛形、**MVP 対象外**。`go build ./...` はここで `main undeclared` で落ちるので、Go コマンドは `./internal/...` か `./` に絞ること
- Wails サービスにメソッドを足したら、`internal/service/<capability>.go` の struct にメソッドを定義 → `wails3 generate bindings -clean=true -ts -i`（`task dev` と同じ形式）→ TS 側は `frontend/bindings/github.com/yito0000/meal-cycle/internal/service` から import (`import { RecipeService } from "…/internal/service"`)
