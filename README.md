# meal-cycle

1〜4 週間分の献立を計画し、レシピと冷蔵庫の在庫を一元管理するデスクトップアプリ。

すべてのデータはアプリを動かす PC 上に保存され、外部サーバーやクラウドとは通信しません。当面は開発者本人の個人利用を想定しています。

## 主な機能（想定）

- **レシピ管理**: 自分が作れるレシピの登録・編集・検索
- **献立カレンダー**: 1〜4 週間先までの献立を日単位で計画
- **冷蔵庫在庫管理**: 食材の在庫・賞味期限を記録
- **買い物リスト**: 献立に必要な食材のうち在庫に無いものを自動集計

- **プロダクト設計**（ユーザーシナリオ、画面、Gherkin）: [docs/product.md](docs/product.md)
- **技術設計**（アーキテクチャ、ストレージ、スタック、OpenSpec 運用）: [docs/design.md](docs/design.md)
- **ワイヤーフレーム**（drawio ファイル群）: [docs/wireframes/](docs/wireframes/)

## 技術スタック

- **フレームワーク**: [Wails v3](https://v3.wails.io/)（Go + Web フロントエンドでデスクトップアプリを作る）
- **バックエンド**: Go
- **フロントエンド**: TypeScript + Vite
- **タスクランナー**: [Task](https://taskfile.dev/)

## 開発環境セットアップ

前提として以下がインストールされていること：

- Go 1.25 以上
- Node.js（npm 同梱）
- [Wails CLI v3](https://v3.wails.io/getting-started/installation/)
- [Task](https://taskfile.dev/installation/)

初回のみ依存関係を取得：

```bash
go mod download
```

```bash
cd frontend && npm install
```

## よく使うコマンド

開発モード（ホットリロード）で起動：

```bash
task dev
```

本番ビルド：

```bash
task build
```

配布用パッケージ作成：

```bash
task package
```

その他のタスク一覧：

```bash
task --list
```

## 動作確認の 2 階建て構成

開発は WSL2 (AlmaLinux 10) を想定していますが、WSLg + WebKit2GTK の IME 連携が不安定なため、`task dev` の webview では日本語入力ができません。動作確認を次のように分けています。

- **開発時**: WSL 上で `task dev`。ホットリロード。日本語入力を伴わない観点 (レイアウト、遷移、バリデーション発火など) の確認。
- **日本語入力を含む確認**: Windows 向けにクロスビルドして Windows ホストから `.exe` を起動。IME・Markdown・日本語保存の確認。

Windows 向けクロスビルド (WSL 側で実行):

```bash
cd frontend && npm run build && cd ..
wails3 generate bindings
GOOS=windows GOARCH=amd64 go build -tags production -trimpath -ldflags="-w -s" -o meal-cycle.exe ./
cp meal-cycle.exe /mnt/c/Users/<Windows のユーザー名>/Desktop/
```

生成された `meal-cycle.exe` を Windows のデスクトップから起動します。DB は WSL 側 (`~/.local/share/meal-cycle/`) と Windows 側 (`%APPDATA%\meal-cycle\`) で別ファイルになる点に注意してください。詳細は [CLAUDE.md § 動作確認の 2 階建て構成](CLAUDE.md#動作確認の-2-階建て構成) を参照。

## ディレクトリ構成

```
.
├── main.go                          # エントリポイント
├── internal/
│   ├── service/                     # Wails サービス (package service, 1 capability = 1 ファイル)
│   ├── domain/<capability>/         # ドメイン層 (recipe, mealplan, ...)
│   ├── storage/sqlite/              # SQLite Repository + migrations
│   └── config/                      # 可変値の集約 (defaults.go)
├── frontend/                        # React + Vite フロントエンド (TypeScript)
├── build/                           # 各 OS 向けビルド設定
├── docs/                            # 設計ドキュメント
│   ├── product.md
│   ├── design.md
│   └── wireframes/                  # 各画面の .drawio.svg
├── openspec/                        # OpenSpec の change proposal / archive
├── CLAUDE.md                        # プロジェクト規約 (Claude 向け・実装ワークフロー)
└── README.md
```

## ライセンス

未定（個人利用のため当面は非公開）。
