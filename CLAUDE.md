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
```

`frontend/bindings/` は `.gitignore` 対象、`task dev` / `task build` で再生成される。Wails v3 beta.12 は **`.ts` ではなく `.js`＋JSDoc** を出力するので、`tsconfig.json` で `allowJs` + `checkJs` を有効化している。

## アーキテクチャ

Go 側 4 レイヤ + フロントエンドを、Wails 自動生成のバインディング経由で繋いでいる:

```
Frontend (React + Vite)  ──┐
                           │  Wails IPC (bindings/)
Service (Go, ルート pkg)   ─┤  ← 現状: healthservice.go
                           │
Domain (internal/domain/)  │  ← capability 単位: recipe/, mealplan/, fridge/, stock/
                           │
Repository (internal/storage/sqlite/)  ← Open / Migrate + capability ごとに 1 repo
```

Wails サービスは `package main`（リポジトリ直下、例: `healthservice.go`）に置き、`main.go` の `application.Options.Services` で登録する。**1 proposal ≒ 1 capability** で、通常はサービス 1 個 / ドメインパッケージ 1 個 / リポジトリ 1 個 / マイグレーション 1 個を同時に足す。

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

**フロントエンド**: 現状テストランナーは入れていない。ユニット的な検証が欲しくなったら Vitest を追加する（本格的な E2E は [scenarios/](docs/scenarios/) を godog で走らせる方向で別途検討 — 現状は仕様ドキュメント扱い）。

## 可変値は 1 箇所に集約

設定画面（S09、MVP 後）で将来変更可能にする値は **すべて `internal/config/defaults.go`** に置く。**call site にリテラル閾値（`10 日` `3 日` など）を書かない**、必ず `config.Defaults()` から読む。可変対象の一覧は [product.md §1](docs/product.md) が正、数値の source of truth は `defaults_test.go`（デフォルト値を変えるならこのテストと product.md の両方を更新する）。

## データモデルの癖

- `MealPlan` は **1 行 = 1 品**。`(date, slot)` は意図的に非一意で、複数行が主菜＋副菜＋汁物を表現する。`position` で slot 内の並び順、`recipeId` は NOT NULL（「未定」= 行が無い状態）。詳細は [design.md §4](docs/design.md)
- データモデルの変更は **それを必要とする proposal と一緒に**入れる。スキーマだけの proposal は作らない

## ドキュメントは load-bearing

- `docs/wireframes/*.drawio.svg` — **ワイヤーは視覚レイアウトの正**。[product.md §4](docs/product.md) は絵から読めない相互作用・遷移だけを言葉で書く。`.drawio.svg` は **可視 SVG と drawio 用の編集 XML の二重構造**（`content=` 属性内）— 片方だけ触ると乖離する。詳細は [docs/wireframes/README.md](docs/wireframes/README.md)
- `docs/scenarios/*.feature` — capability ごとの Gherkin シナリオ（日本語 Gherkin、`# language: ja`）。**現状はドキュメント扱い、実行はしていない** — E2E が実際に必要になったタイミングで godog 導入を検討する。proposal を切るとき対応する `.feature` にシナリオを追加する
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

proposal は `openspec/changes/<name>/`、アーカイブは `openspec/changes/archive/`、現行仕様スナップショットは `openspec/specs/<capability>/`。proposal の順序案は [design.md §8](docs/design.md)、最初にやるのは `add-recipe-listing` の想定。

## コミット規約

- Conventional Commits: `docs:` / `chore:` / `feat:` / `fix:`
- スキャフォールドは `chore:`、ユーザー価値のあるスライスは `feat:`（通常 proposal 内）
- **メッセージは 1 行程度で簡潔に**。長い説明が要るときだけ空行を挟んで本文を続ける
- **AI 支援コミットには `Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>` trailer を付ける**（既存履歴に揃えるため、および将来 OSS 公開したときに AI 関与を後から特定できるようにするため）

## Wails 固有の注意点

- `main.go` の `Zoom: 1.5` + `1500×927` ウィンドウは高 DPI（4K 等）向けデフォルト。CSS の `font-size` は意図的に触っていない — Wails の WebView zoom で拡大することで dev / preview / packaged の見た目を揃える
- `build/ios` と `build/android` は Wails 標準のモバイル雛形、**MVP 対象外**。`go build ./...` はここで `main undeclared` で落ちるので、Go コマンドは `./internal/...` か `./` に絞ること
- Wails サービスにメソッドを足したら、サービス struct にメソッドを定義 → `wails3 generate bindings` → TS 側は `frontend/bindings/github.com/yito0000/meal-cycle` から import
