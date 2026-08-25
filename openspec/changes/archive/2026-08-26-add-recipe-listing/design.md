## Context

meal-cycle は Wails v3 + Go + React 構成のローカルデスクトップアプリ。既存 proposal `add-mealplan-week-view` で
`mealplan` capability を「サービスが固定値を返す + UI が実装される」形で先に通し、後続 proposal で SQLite 永続化を
差し替える方式を採った。本 proposal は同じ戦略を `recipe` capability に適用し、S02 レシピ一覧の UI + Wails サービスの
境界 + ドメイン型を固定値ベースで確定させる。詳細は [proposal.md](proposal.md) を参照。

制約:

- SQLite 層 (repository / migration) は本 proposal 対象外。データは Wails サービス内の固定配列 (const) で提供する
- UI 文字列は i18n key 経由必須 (日本語ハードコード禁止)。既定値・閾値は `internal/config/defaults.go` に集約する
- Wails v3 beta.12 は `.js` + JSDoc をバインディングとして生成する。`frontend/bindings/` は git 管理外

## Goals / Non-Goals

**Goals:**

- `recipe` capability の観察可能な挙動 (一覧・検索・ソート・新規ボタンの配置) を固定値で通しで動く形にする
- `RecipeService.List` のシグネチャと戻り値型を、後続 `add-recipe-crud` が repo 差し替えだけで完了する形に確定させる
- ドメイン層 `internal/domain/recipe` に、後続 proposal で拡張しやすい `Recipe` 型を最小フィールドで定義する
- 検索・ソートはクライアント側で完結させ、サービス呼び出しは 1 度きり (`useQuery`) にする

**Non-Goals:**

- SQLite テーブル / migration / repository の実装 (`add-recipe-crud` 側)
- 行クリック → S03 レシピ詳細ドロワー (`add-recipe-detail-drawer` 側)
- 「＋ 新しいレシピ」の遷移先実装 (`add-recipe-edit` 側)
- タグ / 材料での絞り込み・全文検索 (S10, `add-recipe-search` 側)
- レシピの CRUD 操作

## Decisions

### D1. `Recipe` struct の最小フィールドを一覧用途に絞る

**選択肢:**

- (a) 一覧用に別型 `RecipeSummary` を切り、後で `Recipe` (詳細型) を追加する
  — Pros: 「一覧用」であることが型名から明確。詳細を持たせる際に別型として設計自由度が高い
  — Cons: 現時点で 2 型に分ける必然性が薄い。バインディング再生成で 2 型ぶんの JSDoc が出て見通しが下がる
- (b) 単一 `Recipe` 型に一覧に必要な最小フィールドだけを載せ、後続 proposal で `Ingredients` `Steps` `Notes` を追加する
  ← 採用
  — Pros: mealplan 側 (`Entry` を単一型で回している) と揃う。後続で足すフィールドが必要な proposal から
    順番に増やせる
  — Cons: 「一覧の projection」と「詳細エンティティ」を型で区別しないため、詳細画面が別 projection を取りたく
    なった時点で分割が必要になる可能性はある (`add-recipe-detail-drawer` で判断)

**採用: (b)**。理由: 既存 `internal/domain/mealplan.Entry` と方針を揃えるため、および後続 proposal (`add-recipe-crud`
`add-recipe-detail-drawer`) のどちらも「同じ struct を返すサービスの中身が変わる」形に収まるため。projection 分割が
必要になった時点で切り出せば十分。

### D2. 一覧テーブルを自前実装するか shadcn/ui の Table を使うか

**選択肢:**

- (a) `<table>` を素の JSX で書き、Tailwind でスタイリングする
  — Pros: 依存が増えない。行数が少ない (~50 行想定) ためテーブル制御は最小限で足りる
  — Cons: 見た目の揃えを次回以降のテーブル (冷蔵庫 / 買い物リスト) と手動で維持する必要がある
- (b) shadcn/ui の Table プリミティブ (`src/components/ui/table.tsx`) を追加してそれを使う ← 採用
  — Pros: 冷蔵庫 (S05) / 買い物リスト (S07) も同じテーブルを使う想定。今 shadcn Table を導入すれば以後の proposal
    でそのまま流用できる。デザイン統一が楽
  — Cons: 依存プリミティブが 1 個増える (手動コピペ運用のため npm 依存追加は無い)

**採用: (b)**。理由: 後続の S05 / S07 でも同型のテーブルを使うため、ここで shadcn Table を導入する方が
最終的な proposal 総数コストが低い。shadcn CLI は使わず、[shadcn.dev/docs/components/table]
から `src/components/ui/table.tsx` を手動コピーする運用は既存方針どおり ([CLAUDE.md](../../../CLAUDE.md))。

### D3. 検索・ソートをサーバ側 (RecipeService) に持たせるかクライアント側で完結させるか

**選択肢:**

- (a) `RecipeService.Search(query, sort)` を用意し、Go 側でフィルタ / 並び替えして返す
  — Pros: 将来大量データになった時に SQL の LIKE / ORDER BY に落とせる
  — Cons: 本 proposal ではデータが固定値。サーバ側フィルタを書いても意味が薄い。UI の反応速度も
    サービス呼出 1 回のクライアント処理より遅くなる
- (b) `RecipeService.List(ctx)` で全件返し、フロント側で `useMemo` によりフィルタ / ソートする ← 採用
  — Pros: サービス呼出は初回のみ。検索欄入力に対しレンダの `useMemo` だけで反応するため体感速い。
    Go 側 API を最小に保てる (後続 proposal で `List` の中身だけ差し替え可能)
  — Cons: レシピが数万件になれば全件フェッチが重くなる。想定ペルソナのレパートリー 20〜50 品では問題ない

**採用: (b)**。理由: ペルソナ想定 (20〜50 品, [product.md #1](../../../docs/product.md)) に対しクライアント側処理で十分。
将来 S10 `add-recipe-search` で本格検索が必要になった時点で `Search` メソッドを別途生やせばよい。

### D4. 「新しいレシピ」ボタンの押下時の挙動

**選択肢:**

- (a) ボタン自体を本 proposal では出さず、`add-recipe-edit` 完了後にまとめて出す
  — Pros: 「押しても何も起きないボタン」が UI 上に出ない
  — Cons: S02 のワイヤーからボタン枠が抜けるため、視覚的な完成度が落ちる。かつ次 proposal がボタン配置と
    導線の両方を扱うことになりスコープが膨らむ
- (b) ボタンは表示するが、`onClick` は no-op にしておく (spec でも「押下しても遷移しない」を明示) ← 採用
  — Pros: ワイヤーどおりの見た目にできる。ボタン配置とハンドラを分けて proposal できる
  — Cons: 「押しても反応しないボタン」を数 proposal ぶん残すことになる

**採用: (b)**。理由: mealplan proposal でも「＋」等の操作ボタン相当は表示せず後続に譲る方針を採ったが、
今回のボタンは常時表示のグローバル CTA で位置が S03/S04 で変わらないため、先に配置だけ確定させる方が
後続 proposal の diff が小さくなる。spec 側で「押下しても遷移しない」を明示することでユーザー期待とのズレを避ける。

### D5. 「タグ列」の型を string / []string / タグ Entity のどれにするか

**選択肢:**

- (a) 事前に整形した 1 本の文字列 (例: `"和食 / 簡単"`) をサービスから返す
  — Pros: フロント側は表示するだけで済む
  — Cons: 表示形式が Go 側に閉じ込められ、i18n の区切り文字や折返しがフロントで変えられない
- (b) `[]string` として返し、フロントで結合 / 折返しする ← 採用
  — Pros: 見た目 (区切り文字 / バッジ化 / 折返し) はフロント都合で変えられる。将来タグバッジ化する際に
    データ型を変える必要がない
  — Cons: フロント側で描画時に結合が必要 (1 行なので `.join(" / ")` 相当で足りる)
- (c) タグを別 struct (`Tag{ID, Name}`) にする
  — Pros: 将来タグの名前変更・並び替えを ID で追跡できる
  — Cons: 本 proposal のスコープでは過剰。タグマスタ (`Tag` テーブル) は S08 (MVP 外) の対象

**採用: (b)**。理由: 表示の自由度が上がり、S08 (タグマスタ管理) を導入する際に破壊的変更が要らない。
`(c)` は S08 着手時に検討する。

### D7. ソート切替 UI に素の `<select>` を使うか shadcn Select を使うか

**背景:** 実機 (WSLg) で目視確認したところ、素の `<select>` は閉じた状態は WebView 内 (Wails `Zoom: 1.5` 追従)
なのに、開いた時のオプション一覧が OS ネイティブ popup で描かれるため文字が小さく、
Wails Zoom が効かない食い違いが発生した。

**選択肢:**

- (a) 素の `<select>` のまま。option の `font-size` を CSS で無理やり上げる
  — Pros: 依存追加なし
  — Cons: Chromium 系 WebView は option の font-size を効かせない。GTK は挙動不定。
    プラットフォーム差でハマる
- (b) shadcn/ui の `Select` (Radix ベース) を導入し、メニューも WebView 内で描画する ← 採用
  — Pros: Wails Zoom がそのまま効く。デザインが他コンポーネントと揃う。
    後続 proposal (S05 冷蔵庫の絞り込み等) でも再利用できる
  — Cons: `@radix-ui/react-select` 1 個を npm 依存に追加する

**採用: (b)**。理由: プラットフォーム差の問題を根本解決でき、shadcn 手動コピペ運用 ([CLAUDE.md](../../../CLAUDE.md))
と一致する。Table を先に入れたのと同じ流儀で `frontend/src/components/ui/select.tsx` に手動コピーで新設する。

### D8. 「材料数」列を一覧に出すか

**背景:** 実機目視で「一覧の情報密度が下がる」「規模感の代理指標としては弱い」判断となり、
一覧の目的 (何を作るか思い出す) に対して名前・タグ・更新日の 3 列で十分と結論した。

**選択肢:**

- (a) 4 列 (名前 / タグ / 材料数 / 更新日) のまま。「材料数」を規模感の目安として残す
  — Pros: レシピの規模感が一目で分かる
  — Cons: 名前列の視認性が下がる。数値だけの列があると視線が右に流れる。
    材料数は詳細ドロワー (S03) で見れば十分
- (b) 3 列 (名前 / タグ / 更新日) に減らす ← 採用
  — Pros: 一覧の情報密度が上がる。名前・タグが視覚的に主役になる
  — Cons: 材料数を見たいユーザーは詳細ドロワーを開く必要がある (S03 で解決)
- (c) 表示項目を設定 (S09) で可変にする
  — Pros: ユーザーの好みに合わせられる
  — Cons: S09 (MVP 後) のスコープ。今の proposal に混ぜるとブレる

**採用: (b)**。理由: MVP 段階では情報密度を優先し、可変化は必要になった時点で S09 で扱う。
domain の `IngredientCount` フィールドは残す (詳細ドロワー / 献立割当で使うため)。

### D6. 固定値データの配置

固定値は `recipeservice.go` 内 (`var fixtureRecipes = []recipe.Recipe{...}`) にまとめて置く。理由: `add-mealplan-week-view`
と同じパターンで、次回の repo 差し替え時に「fixture を丸ごと消して repo 呼び出しに置換」が 1 ファイル修正で
済む。テスト側 (`recipeservice_test.go`) は fixture の内容を直接読まず、公開メソッド `List` 経由で検証する。

**採用パターン**: 5〜6 件程度、うち少なくとも 1 件はタグ 2 個以上を持つレシピ、少なくとも 1 件はタグ 0 個の
レシピを含める (タグ列の空状態を目視確認できるようにするため)。更新日は現在日 (`time.Now()`) からの相対値ではなく、
テスト再現性を優先して固定日付文字列 (例: `"2026-08-20"`) で持つ。

## Risks / Trade-offs

- **[Risk] 「押しても遷移しないボタン」への戸惑い** → Mitigation: spec で明示、i18n 文言に `(準備中)` 等は入れず
  ワイヤーどおりの見た目を保つ。次 proposal `add-recipe-edit` で遷移を有効化して即解消する
- **[Risk] 固定値と repo 実装の schema ズレ** → Mitigation: `Recipe` struct を先に確定させ、
  `add-recipe-crud` は同じ struct を返す SQL に落とすだけにする。fixture のフィールド構成が仕様の source of truth
- **[Risk] shadcn/ui Table の初回導入が本 proposal の作業量を膨らませる** → Mitigation: 手動コピー 1 ファイル
  (`src/components/ui/table.tsx`) で済む。後続 proposal (S05 / S07) で確実に再利用されるため損にならない
- **[Risk] クライアント側フィルタ / ソートがレシピ数増加でスケールしない** → Mitigation: ペルソナ想定
  レパートリー (20〜50 品) の範囲では `useMemo` で十分。S10 `add-recipe-search` 着手時に本格検索を
  サービス側に足せば済む (`List` は残す)

## Migration Plan

本 proposal は既存機能の変更ではなく新規追加のため、マイグレーションは不要。
ロールバックは次のいずれか:

- `main.go` から `RecipeService` の登録を外し、`frontend/src/routes/recipes.tsx` を placeholder に戻す
- git revert で proposal コミットを打ち消す (追加ファイルが多いので後者を推奨)
