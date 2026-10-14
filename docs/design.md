# meal-cycle 設計ドキュメント

このドキュメントは meal-cycle の設計判断・スコープ・未決事項を記録するもの。実装が進むにつれて更新していく。細かい機能仕様は [OpenSpec](#opencspec-の運用方針) の change proposal 側に寄せる方針。

---

## 1. プロダクト概要

### 目的

日々の食事の計画と食材管理を、ローカル完結の一つのアプリで済ませたい。

### 想定シナリオ

- 週末に「来週〜再来週の献立」をカレンダー上で組む
- 献立を作った時点で必要な食材が分かり、買い物リストとして自動集計される（MVP）
- 冷蔵庫にあるものから逆引きで「作れるレシピ」を確認できる（拡張）
- 献立を組む際、賞味期限が近い在庫や最近作っていないレシピを**根拠つきで候補提示**し、選択を楽にする（MVP 後拡張、[product.md #5](product.md#5-献立候補の提示mvp-後拡張)）
  - 「使い切り」「久しぶりに」「時短」の 3 軸を想定。全自動生成は AI 前提になるため採らない

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

### バックエンド

- **Wails v3 (beta)**: Go バックエンド + Web フロントエンドを 1 バイナリにする
- **Go 1.25**
- **[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)**: Cgo なしの SQLite ドライバ
- **Task**: ビルド / 開発コマンドのラッパー

### フロントエンド

| 要素 | 選定 |
|---|---|
| UI フレームワーク | **React 18 + TypeScript** |
| ビルドツール | Vite（Wails スキャフォールドのまま） |
| スタイリング | **Tailwind CSS** |
| コンポーネント | **[shadcn/ui](https://ui.shadcn.com/)**（コピペ導入、依存にせず自プロジェクト内に配置） |
| サーバー状態管理 | **[TanStack Query](https://tanstack.com/query)**（Wails サービス呼び出しのキャッシュ・再取得） |
| ルーティング | **[TanStack Router](https://tanstack.com/router)** または React Router（実装時に確定） |
| カレンダー | **自作**（CSS Grid ベース。既製カレンダーは機能過剰） |
| i18n | **[i18next](https://www.i18next.com/) + react-i18next** |

### 選定理由

- **React**: 開発者がフロントに深くないため、AI 支援と情報量が最大のスタックを選ぶ。バンドルサイズは Wails デスクトップ用途では問題にならない
- **Tailwind + shadcn/ui**: 2026 時点で React 系の実質標準。shadcn 方式は自分でカスタマイズしやすく、依存を最小化できる
- **TanStack Query**: Wails のサービス呼び出しは async。手動で state 管理するより Query に任せた方が同期・キャッシュ・エラー処理が楽

### 検討した他の選択肢

- **Svelte 5**: コード量が少なく軽量だが、教材・AI 支援量で React に劣る。開発者の習熟度を考えて見送り
- **Solid**: React 風の JSX + Signals で軽量。ただし情報量が少なく、詰まった時に自力解決が難しい

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
archivedAt   : timestamp | null // 論理削除の時刻。null = 未削除
```

- レシピの削除は論理削除にする。物理削除だと `meal_plans.recipe_id` の `ON DELETE CASCADE` で献立の履歴まで消えるため
  （`add-recipe-deletion` の design.md D1）
- 削除済みレシピは、レシピ一覧、献立のレシピ選択、献立への新たな割当と入替から除く。既存の献立、買い物リスト、食材名候補では従来どおり参照する

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
name         : string           // 食材名 (自由テキスト。実装済みの recipe_ingredients.name と同じく食材マスタは未導入)
amount       : number
unit         : string
expiresAt    : date | null      // 賞味期限
addedAt      : timestamp
updatedAt    : timestamp
```

- 食材マスタへの移行は `add-ingredient-master` で `recipe_ingredients.name` と同時に行う

### MealPlan（献立エントリ）

**1 行 = meal (時間帯) に割り当てられた 1 品**。同じ `(date, meal)` に対して 0..N 行が入る（1 食で主菜＋副菜＋汁物のような複数品構成を表現するため）。

```
id           : string
date         : date             // YYYY-MM-DD
meal         : "breakfast" | "lunch" | "dinner" | "other"
recipeId     : string           // NOT NULL。未定は「行が無い」ことで表現
position     : int              // 同一 (date, meal) 内での並び順（0 始まり、主菜→副菜→汁物 の順を想定）
```

- `(date, meal)` に一意制約はかけない。代わりに `(date, meal, position)` にユニークインデックスを張って表示順を安定化
- 「未定 = null」は現状の使い方（1 行必ずある + `recipeId` が null）から「行を作らない」に変える方が自然
- `memo` は MVP からは外す。将来時間帯単位のメモが欲しくなれば `MealMemo(date, meal, memo)` を別テーブルで足す

**作った印 `meal_plan_cooked`**（proposal: add-mealplan-cooked-mark）

```
batchId      : string           // meal_plans.batch_id
recipeId     : string           // meal_plans.recipe_id
cookedAt     : string           // RFC3339。画面には出さない
PRIMARY KEY (batchId, recipeId)
```

- 印は S07 が材料を 1 回分として数える単位、つまり `(batch_id, recipe_id)` の組に 1 行で持つ。同じ組の行をいくつ足しても動かしても印はそのまま効く
- `(batch_id, recipe_id)` は `meal_plans` 上で一意でないので外部キーは張らない。
  代わりに、行を消したり組を変えたりする操作の最後に、対応する行の無くなった印を消す (入れ替え後に元のレシピへ戻したときの印の復活を防ぐ)
- 作り置きから外す (`SplitBatch`) と他の作り置きにまとめる (`MergeBatch`) は `batch_id` を変えるので、印を新しい組へ写す

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

## 8. OpenSpec の運用方針

[OpenSpec](https://github.com/Fission-AI/OpenSpec) は「仕様変更を change proposal 単位で管理する」ツール。本プロジェクトでは以下の方針で運用する。

### 粒度: 垂直スライス（1〜2 日単位、ユーザー動詞で切る）

- **capability**（例: `recipe`, `mealplan`, `fridge`）は機能領域の単位。1 領域 = 1 capability
- **proposal** は capability に対する変更単位で、**ユーザー動詞ベースの垂直スライス**で切る
  - 例: `add-recipe-listing`, `add-recipe-creation`, `add-recipe-editing`, ...
- 1 proposal の目安: **1〜2 日で完了**、**エンドツーエンドで動作検証できる最小単位**、**ユーザーが「これができるようになった」と言える単位**
- 「画面単位」ではなく「ユーザー行動」で切る。1 proposal で複数レイヤ（domain / repository / service / UI）に触るのは前提

### 検討して見送った粒度

- **capability まるごと 1 proposal**（例: `add-recipe-management`）: 1 proposal が数日〜1週間かかりレビューが重い
- **画面単位 1 proposal**（例: `add-recipe-list-screen`, `add-recipe-create-screen`）: バックエンド全レイヤに触るので境界が曖昧、overhead も割高

### スキャフォールド系は proposal にしない

以下のような「ユーザー価値を伴わない基盤整備」は proposal ではなく通常の `chore:` コミットで扱う：

- SQLite の初期化・接続層
- マイグレーションランナーの導入
- 依存ライブラリの追加
- Go モジュール名の変更
- ディレクトリ構成の変更

OpenSpec は「capability に対する変更」を管理するツールなので、基盤整備を無理に proposal 化すると噛み合わない。

### ディレクトリ構成

```
openspec/
├── specs/              # 現在の仕様スナップショット（archive の合成結果）
│   ├── recipe/
│   ├── mealplan/
│   └── fridge/
├── changes/            # 進行中の proposal
│   └── add-recipe-listing/
│       ├── proposal.md
│       ├── tasks.md
│       └── specs/recipe/spec.md  # capability への delta
└── archive/            # 完了した proposal
```

### ワークフロー

1. proposal 作成（`.claude/commands/opsx/propose.md` の skill 経由）
2. proposal レビュー・調整
3. 実装（proposal 単位でブランチを切る想定）
4. archive（`opsx/archive.md` skill）→ specs スナップショットに反映

### 最初にやる proposal（順序案）

MVP:

```
add-recipe-listing        ← model + repo + service + 一覧 + 最小登録
add-recipe-details        ← 詳細ドロワー + 手順・メモ等の追加項目
add-recipe-editing
add-recipe-deletion
add-fridge-inventory      ← 在庫一覧 + 賞味期限順 + インライン追加 / 編集 / 削除 (当初の 3 分割を 1 本化)
add-mealplan-week-view    ← 1週間カレンダー表示
add-mealplan-assign-recipe ← 日付にレシピを割り当て
add-mealplan-multi-week   ← 2週/4週表示切替
add-shopping-list         ← 献立から不足食材を集計
add-shopping-purchase     ← 購入済チェックと確認ダイアログで在庫追加
```

MVP 後（[product.md #5](product.md#5-献立候補の提示mvp-後拡張) の候補提示 3 軸）:

```
add-mealplan-cooked-mark        ← S01 の品に「作った」印を付け、S07 の計上から外す（予定と実績の区別）
add-mealplan-cook-consume       ← 作った印に合わせて在庫を材料分だけ減らす（提案精度の下敷き）
add-mealplan-suggest-by-stock   ← 使い切り軸: 期限が近い在庫を使うレシピを提示
add-mealplan-suggest-by-recency ← 久しぶり軸: 最近作っていないレシピを提示
add-mealplan-suggest-week-preset ← 週プリセット: 5コマ分を貪欲生成した案を2〜3件提示
```

`add-mealplan-cook-consume` は候補提示の前提として先に入れる（「使い切り」軸が意味を持つのは在庫が減る仕組みがあってこそ）。3 軸は独立に足せるので、まず `suggest-by-stock` だけ入れて反応を見るのが良い。

### 運用中の見直し

粒度が実運用と合わないと感じたら途中で方針変更してよい（proposal が毎回大きすぎ / 小さすぎ / overhead が重い等）。その際は本セクションを更新する。

---

## 9. MVP スコープ

「最初にリリースする範囲」として以下の 4 機能を含める：

- **レシピ管理 (CRUD)**: 登録・編集・削除・一覧・材料リストの記述
- **献立カレンダー (1-4 週)**: カレンダーに日付ごとにレシピを割り当てる
- **冷蔵庫在庫管理**: 食材の在庫・賞味期限を登録して一覧できる
- **買い物リスト**: 献立に必要な食材のうち在庫に無いものを自動集計

「献立を作った時に在庫を自動で減らす」など料理実施との連動は MVP 後。買い物リストは「集計と表示のみ、購入済チェックと確認のダイアログで在庫に加える」までを含む想定。

### MVP 後の候補

- **献立候補の提示（3 軸）**: 使い切り / 久しぶりに / 時短。方針は [product.md #5](product.md#5-献立候補の提示mvp-後拡張)。全自動生成は AI 前提になるため採らず、根拠つきで候補を並べてユーザーが選ぶ形にする
- **料理実施と在庫消費の連動**: 「作った」マークで材料分の在庫を減らす。上記「使い切り」軸の精度向上に必要
- **設定 UI（S09）**: MVP 期に定数として持っている可変値（時間帯表示のデフォルト、「久しぶりに」の消費期間閾値、期限アラート日数、週プリセット品数など。全一覧は [product.md #1](product.md#1-想定利用シナリオ)）をユーザーが変更できる画面。データ import/export・表示言語切替もここに集約
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

- [x] `go.mod` の module 名を `changeme` から変更する（実装開始前に）→ `github.com/yito0000/meal-cycle` に確定済み

現時点で未決の項目なし。新しく発生したら追記する。

設計と UI/UX の見直しで挙がった指摘（未対応を含む）は [reviews/](reviews/) に日付ごとに残している。
最新は [2026-10-14 のレビュー](reviews/2026-10-14-design-ux-review.md)。
