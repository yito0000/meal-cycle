# meal-cycle 設計ドキュメント

このドキュメントは meal-cycle の設計判断・スコープ・未決事項を記録するもの。実装が進むにつれて更新していく。細かい機能仕様は [OpenSpec](#opencspec-の運用方針) の change proposal 側に寄せる方針。

---

## 1. プロダクト概要

### 目的

日々の食事の計画と食材管理を、ローカル完結の一つのアプリで済ませたい。

### 想定シナリオ

- 週末に「来週〜再来週の献立」をカレンダー上で組む
- 献立を作った時点で必要な食材が分かり、買い物リストの参考になる（MVP 後の拡張）
- 冷蔵庫にあるものから逆引きで「作れるレシピ」を確認できる（拡張）
- 自身が作れるメニューから週ごとの献立予定、買い物リストを組める（拡張）
- 賞味期限が近い食材を優先して使う献立を組める（拡張）

### スコープ外（当面）

- 複数ユーザーでのデータ共有
- クラウド同期
- モバイルアプリ
- 栄養計算（将来的な検討候補）

---

## 2. 非機能要件

| 項目 | 方針 |
|---|---|
| デプロイ形態 | デスクトップアプリ単体（バックエンドサーバー不要） |
| データ保存先 | アプリを動作させる PC 内のローカルストレージのみ |
| ネットワーク通信 | 原則なし。将来オンライン機能を足す場合も opt-in |
| ユーザー数 | 1 台につき 1 ユーザー（マルチアカウント考慮なし） |
| 対応 OS | Windows を第一優先。macOS / Linux は Wails の対応範囲で追随 |
| 言語 | 日本語がデフォルト。i18n 基盤を最初から入れて英語追加に備える |

---

## 3. 技術スタック

- **Wails v3 (beta)**: Go バックエンド + Web フロントエンドを 1 バイナリにする
- **Go 1.25**
- **TypeScript + Vite**（フロントエンド）
- **Task**（ビルド / 開発コマンドのラッパー）

### 未決: フロントエンド UI フレームワーク

現時点で `frontend/` は素の TypeScript + Vite のみ。カレンダー UI や CRUD 画面を書くと素のままはつらいので、いずれ選定が必要。

- 候補: React / Svelte / Vue / Solid
- 推奨: **Svelte または Solid**（バンドルサイズが軽くデスクトップアプリと相性が良い）
- MVP 前に決めて、最初の画面を作る前に導入する

---

## 4. データモデル（初期案）

エンティティは 4 つに整理する。ID は UUID v7（時系列ソート可能）を想定。

### Recipe（レシピ）

```
id           : string (UUID)
name         : string
servings     : int              // 何人分
ingredients  : RecipeIngredient[]
steps        : string[]         // 手順
tags         : string[]         // "和食", "簡単" など任意
notes        : string           // 自由メモ
createdAt    : timestamp
updatedAt    : timestamp
```

### RecipeIngredient（レシピ内の食材）

```
ingredientId : string           // Ingredient への参照
amount       : number
unit         : string           // "g", "個", "本" など
```

### Ingredient（食材マスタ）

レシピと在庫で共通に参照する食材の名寄せ辞書。

```
id           : string
name         : string           // "玉ねぎ"
defaultUnit  : string           // "個"
category     : string           // "野菜", "肉" など
```

### StockItem（冷蔵庫在庫）

```
id           : string
ingredientId : string
amount       : number
unit         : string
expiresAt    : date | null      // 賞味期限
addedAt      : timestamp
```

### MealPlan（献立エントリ）

```
id           : string
date         : date             // YYYY-MM-DD
slot         : "breakfast" | "lunch" | "dinner" | "other"
recipeId     : string | null    // 未定の場合 null
memo         : string
```

### 議論ポイント

- **食材マスタを分けるか、レシピ内に文字列で持つか**: 分けたほうが在庫連動や名寄せが効くが初期実装は重い。→ **最初から分ける方針**（後で統合し直すコストのほうが高い）
- 単位変換（g ⇔ 個）は最初は扱わない。両方 `Ingredient.defaultUnit` に合わせて記録する運用でよい

---

## 5. データ保存方式

**決定: SQLite（`modernc.org/sqlite` ドライバ、Cgo なし）**

### 選定理由

- レシピ ↔ 食材 ↔ 献立 ↔ 在庫のリレーションが最初から多く、SQL で書けるメリットが大きい
- 「賞味期限が近い在庫を使うレシピを探す」など、後で欲しくなる機能が SQL 前提だと自然
- `modernc.org/sqlite` は Cgo 不要 → Wails のクロスコンパイル（Windows / macOS / Linux）が素直

### 検討した他の選択肢

いずれも今回は採用せず。

- **JSON ファイル**: 実装は最小だがリレーション・検索を全部自前になる。データ量が増えると全読み書きが辛い
- **libSQL 系（Turso）**: 将来のクラウド同期を見据えて検討したが、embedded 用の `go-libsql` は Cgo 必須でクロスコンパイルが面倒。**SQLite ファイル形式は libSQL と互換なので、将来同期が欲しくなった時点でドライバ差し替えで対応可能** → 今 Cgo コストを払う必要はなし
- **bbolt / BadgerDB などの KV ストア**: クエリなしで検索を全部 Go 側で書く必要があり、リレーション主体の本アプリには不向き
- **Markdown-per-recipe（Obsidian 型）**: 面白い候補だが MVP には過剰。将来「レシピを Git 管理したい」ニーズが出れば proposal で追加

### マイグレーション

- スキーマ変更は連番マイグレーション方式で管理
- ライブラリ候補: [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) または [`pressly/goose`](https://github.com/pressly/goose)（実装時に確定）
- マイグレーションファイルは `internal/storage/sqlite/migrations/` に置く

### データ配置先

いずれの案でも [`github.com/adrg/xdg`](https://github.com/adrg/xdg)（既に依存に入っている）を使って OS 標準の場所に置く：

- Windows: `%APPDATA%\meal-cycle\`
- macOS: `~/Library/Application Support/meal-cycle/`
- Linux: `~/.local/share/meal-cycle/`

---

## 6. アーキテクチャ

Wails v3 の構造に沿って、以下のレイヤに分ける。

```
┌─────────────────────────────────────────┐
│  Frontend (TypeScript + Vite)            │
│  - 画面 / ユーザー操作                    │
│  - Wails 生成のバインディング経由でサービス呼び出し │
└──────────────┬──────────────────────────┘
               │  Wails IPC (自動生成される bindings)
┌──────────────▼──────────────────────────┐
│  Services (Go)                           │
│  - RecipeService / MealPlanService / ...  │
│  - フロントに公開するユースケース単位のAPI  │
└──────────────┬──────────────────────────┘
┌──────────────▼──────────────────────────┐
│  Domain (Go)                             │
│  - エンティティ / ビジネスロジック         │
└──────────────┬──────────────────────────┘
┌──────────────▼──────────────────────────┐
│  Repository (Go)                         │
│  - 永続化の抽象。実装は JSON or SQLite    │
└─────────────────────────────────────────┘
```

Wails のサービス層を DDD で言う「アプリケーションサービス」相当と捉える。ドメインは framework 非依存で書く。

### ディレクトリ構成（案）

```
.
├── main.go
├── internal/
│   ├── domain/           # エンティティとビジネスルール
│   │   ├── recipe/
│   │   ├── mealplan/
│   │   ├── ingredient/
│   │   └── stock/
│   ├── service/          # Wails に公開するサービス
│   │   ├── recipe_service.go
│   │   ├── mealplan_service.go
│   │   └── ...
│   └── storage/          # 永続化実装
│       └── sqlite/
│           ├── migrations/
│           └── *.go
├── frontend/
└── openspec/             # OpenSpec の change proposal / archive
```

### 未対応: モジュール名

`go.mod` が `module changeme` のままなので `module github.com/yito0000/meal-cycle` に変更する必要あり。**最初の内部パッケージを切る前に対応する。**

---

## 7. 国際化（i18n）

デフォルト日本語、後から英語追加可能な形にする。

### フロントエンド

- ライブラリ候補: [`i18next`](https://www.i18next.com/) + フレームワーク別アダプタ（framework 選定後に確定）
- 翻訳リソース: `frontend/src/locales/ja.json`, `en.json` の JSON ベース
- **原則**: 画面に出す文字列は必ず翻訳キー経由。日本語のハードコード禁止

### バックエンド（Go）

- サービスから返すエラーメッセージなど、UI に出る文字列はキーで返してフロント側で翻訳する
- Go 側でメッセージそのものを日本語で持たない（`ErrRecipeNotFound` のような型 / コードで返す）

---

## 8. OpenSpec の運用方針（要相談）

[OpenSpec](https://github.com/Fission-AI/OpenSpec) は「仕様変更を change proposal 単位で管理する」ツール。以下の 2 スタイルが考えられる。

### スタイル 1: 機能ごとに proposal を都度作る（推奨）

```
openspec/
├── specs/              # 現在の仕様スナップショット（archive の合成結果）
├── changes/            # 進行中の proposal
│   └── add-recipe-crud/
└── archive/            # 完了した proposal
```

- 1 機能 = 1 proposal → 実装 → archive のサイクル
- **Pros**: 変更履歴が読みやすい、AI 支援と相性が良い、1 人開発でも「今何やってるか」が明確
- **Cons**: 最初の 1 個目を作るときに「土台となる spec」が空なので違和感

### スタイル 2: 最初にベース spec を一気に書いてから proposal で差分管理

- 「レシピ」「献立」「在庫」の 3 つの capability を最初に spec として書く
- 以降、追加機能は change proposal で差分定義

**Pros**: 全体像が最初から見える
**Cons**: 最初の spec を書ききるまで実装に入れない、書いた仕様に実装が追いつかず腐りやすい

### 推奨

**スタイル 1**（機能ごとに proposal）で進める。理由：

- OpenSpec のワークフローに素直に乗れる
- 1 機能ずつ開発したいという方針と一致する
- 「レシピ管理」を最初の proposal として扱えば、そこで capability の骨格も定義される

### 最初にやること（OpenSpec 導入後）

1. `openspec init` でディレクトリを作る
2. 最初の proposal `add-recipe-crud` を作成
3. 実装 → archive
4. 次の proposal `add-meal-plan-calendar` へ

---

## 9. MVP スコープ

「最初にリリースする範囲」として以下の 3 機能を含める：

- **レシピ管理 (CRUD)**: 登録・編集・削除・一覧・材料リストの記述
- **献立カレンダー (1-4 週)**: カレンダーに日付ごとにレシピを割り当てる
- **冷蔵庫在庫管理**: 食材の在庫・賞味期限を登録して一覧できる

MVP では「献立を作ると在庫が減る」などの**連動機能は含めない**。それぞれの CRUD が独立して動く状態で一度リリースする。連動系は MVP 後に proposal で追加。

### MVP 後の候補

- 献立から必要食材を集計 → 買い物リスト自動生成
- 賞味期限が近い食材を優先するレシピ提案
- レシピの検索・タグフィルタ
- 献立テンプレート（お気に入りの 1 週間パターンを保存）
- 栄養計算

---

## 10. GitHub の使い方

- リポジトリは個人アカウント `yito0000/meal-cycle` 配下（会社組織とは分離される）
- **Issues は当面使わない**方針。OpenSpec の proposal がタスクの粒度になるため
- 必要が出てきたら Issue を追加する（大きめのリファクタや不具合トラッキング用など）

---

## 11. 未決事項サマリ

このドキュメント内で「要相談」「未決」となっているもの一覧：

- [ ] **フロントエンド UI フレームワーク**: Svelte / Solid / React などから選定
- [ ] **OpenSpec 運用スタイル**: スタイル 1（推奨）で確定するか
- [ ] `go.mod` の module 名を `changeme` から変更する（実装開始前に）

これらが決まったら本ドキュメントを更新する。
