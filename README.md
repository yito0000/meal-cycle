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

## ディレクトリ構成

```
.
├── main.go             # エントリポイント
├── greetservice.go     # Wails サービス（現在はテンプレのサンプル）
├── frontend/           # TypeScript + Vite フロントエンド
├── build/              # 各 OS 向けビルド設定
├── docs/               # 設計ドキュメント
│   ├── product.md
│   ├── design.md
│   └── wireframes/     # 各画面の .drawio.svg
├── openspec/           # OpenSpec の change proposal / archive
└── README.md
```

## ライセンス

未定（個人利用のため当面は非公開）。
