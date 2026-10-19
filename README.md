# meal-cycle

1〜4 週間分の献立を計画し、レシピと冷蔵庫の在庫を一元管理するデスクトップアプリ。

すべてのデータはアプリを動かす PC 上の SQLite に保存され、外部サーバーやクラウドとは通信しません。
開発者本人の個人利用のために作ったもので、UI は日本語のみです。
Windows 版の実行ファイルは [Releases](https://github.com/yito0000/meal-cycle/releases) で配布しています。
macOS と Linux で使う場合は、ソースからビルドしてください。

## 主な機能

- **献立カレンダー**：1〜4 週間分の献立を日と時間帯ごとに計画する。1 食に主菜、副菜、汁物など複数の品を並べられる
- **レシピ管理**：レシピを登録、編集、検索する。手順は Markdown で書ける
- **レシピのインポート**：決まった形式の Markdown ファイルからレシピをまとめて登録する。形式は [docs/recipe-import.md](docs/recipe-import.md) を参照
- **冷蔵庫在庫**：食材の在庫量と賞味期限を記録し、期限が近いものを目立たせる
- **買い物リスト**：指定した期間の献立で使う材料を集計し、在庫の有無と並べて表示する

## インストール（Windows）

[Releases](https://github.com/yito0000/meal-cycle/releases) から `meal-cycle.exe` をダウンロードし、好きな場所に置いて起動します。
インストーラーはなく、exe 1 つで動きます。
画面の表示には Microsoft Edge WebView2 ランタイムを使います。
Windows 11 には標準で入っており、Windows 10 でも Edge の更新で入っていることがほとんどです。

exe にはコード署名をしていないので、初回起動時に「Windows によって PC が保護されました」という警告が出ます。
「詳細情報」を押すと「実行」ボタンが表示されるので、そこから起動してください。
ダウンロードしたファイルが改ざんされていないかは、同じリリースの `SHA256SUMS.txt` と PowerShell の `Get-FileHash meal-cycle.exe` の出力を見比べて確かめられます。

## ドキュメント

- **プロダクト設計**（利用シナリオ、画面、遷移）：[docs/product.md](docs/product.md)
- **技術設計**（アーキテクチャ、ストレージ、スタック、OpenSpec 運用）：[docs/design.md](docs/design.md)
- **ワイヤーフレーム**（drawio ファイル群）：[docs/wireframes/](docs/wireframes/)
- **振る舞いの仕様**（capability ごとの Requirement と Scenario）：[openspec/specs/](openspec/specs/)
- **開発規約**（テスト方針、コメント方針、コミット規約など）：[CLAUDE.md](CLAUDE.md)
- **リリース運用**（バージョン番号、ブランチ、リリース手順）：[docs/release.md](docs/release.md)

## 技術スタック

- **フレームワーク**：[Wails v3](https://v3.wails.io/)（Go と Web フロントエンドで単一バイナリのデスクトップアプリを作る）
- **バックエンド**：Go、SQLite（[modernc.org/sqlite](https://gitlab.com/cznic/sqlite)、Cgo 不要）、[goose](https://github.com/pressly/goose)（マイグレーション）
- **フロントエンド**：React、TypeScript、Vite、TanStack Router / Query、Tailwind CSS v4、shadcn/ui、i18next
- **テスト**：Go 標準 `testing`、Playwright（E2E）
- **lint / format**：golangci-lint、Biome
- **タスクランナー**：[Task](https://taskfile.dev/)

## セットアップ

次のツールが必要です。

- Go 1.27 以上
- Node.js（npm 同梱）
- [Wails CLI v3](https://v3.wails.io/getting-started/installation/)
- [Task](https://taskfile.dev/installation/)

初回のみ、依存関係を取得します。

```bash
go mod download
```

```bash
cd frontend && npm install
```

## よく使うコマンド

開発モード（ホットリロード）で起動します。

```bash
task dev
```

本番ビルドと配布用パッケージの作成は、それぞれ次のとおりです。

```bash
task build
```

```bash
task package
```

動作確認用のサンプルレシピを DB に投入できます（同名のレシピがあればスキップします）。

```bash
task seed
```

テストと lint は次のコマンドで走ります。
Go のテストを `./...` で走らせると、`build/ios` に `main` が無いため失敗します。

```bash
go test ./internal/... ./cmd/...
```

```bash
task test:e2e
```

```bash
task lint
```

その他のタスクは `task --list` で一覧できます。

## データの保存先

DB は OS 標準のデータディレクトリに作られます。

- **Windows**：`%APPDATA%\meal-cycle\meal-cycle.db`
- **macOS**：`~/Library/Application Support/meal-cycle/meal-cycle.db`
- **Linux**：`~/.local/share/meal-cycle/meal-cycle.db`

バックアップや移行は、このファイルをコピーすれば済みます。

## WSL2 で開発する場合の動作確認

開発は WSL2 (AlmaLinux 10) で行っています。
ただし WSLg と WebKit2GTK の IME 連携が不安定で、`task dev` の webview には日本語を入力できません。
そのため、動作確認を次のように分けています。

- **開発時**：WSL 上で `task dev` を使い、レイアウト、遷移、バリデーションなど日本語入力を伴わない観点を確かめる
- **日本語入力を含む確認**：Windows 向けにクロスビルドした `.exe` を Windows 側で起動し、IME、Markdown、日本語の保存を確かめる

Windows 向けのクロスビルドは WSL 側で実行します。
`WIN_DEST` を渡すと、ビルド後に指定先へコピーします。

```bash
task build:win WIN_DEST=/mnt/c/Users/<Windows のユーザー名>/Desktop/
```

WSL 側と Windows 側では DB が別ファイルになるので、片方で登録したデータはもう片方には出てきません。
詳細は [CLAUDE.md の「動作確認の 2 階建て構成」](CLAUDE.md#動作確認の-2-階建て構成) を参照してください。

## ディレクトリ構成

```
.
├── main.go                          # エントリポイント
├── cmd/
│   ├── e2e-server/                  # E2E 用に Go サービスを HTTP で公開するサーバ
│   └── seed/                        # サンプルレシピ投入ツール
├── internal/
│   ├── service/                     # Wails サービス (package service, 1 capability = 1 ファイル)
│   ├── domain/<capability>/         # ドメイン層 (recipe, mealplan, fridge, ...)
│   ├── storage/sqlite/              # SQLite Repository + migrations
│   ├── sampledata/                  # サンプルレシピ
│   └── config/                      # 可変値の集約 (defaults.go)
├── frontend/                        # React + Vite フロントエンド (TypeScript)
│   └── e2e/                         # Playwright E2E
├── build/                           # 各 OS 向けビルド設定
├── docs/                            # 設計ドキュメントとワイヤーフレーム
├── openspec/                        # OpenSpec の仕様、change proposal、archive
├── CLAUDE.md                        # プロジェクト規約 (Claude Code 向け、実装ワークフロー)
└── README.md
```

## 開発の進め方

機能の追加は [OpenSpec](https://github.com/Fission-AI/OpenSpec) の change proposal を経由し、[Claude Code](https://claude.com/claude-code) の支援を受けて実装しています。
AI の支援を受けたコミットには `Co-Authored-By` trailer を付けています。

個人で使うためのプロジェクトなので、Issue や Pull Request に対応できるとは限りません。
fork して自由に改変してください。

## ライセンス

[MIT License](LICENSE)
