## Context

- 動機は [proposal.md](proposal.md) 参照。前提として `add-mealplan-assign-recipe` が archive されて `meal_plans`
  テーブル / `CopyLeftover` サービス経路が main spec に載っていることを想定する
- 参照ドキュメント: [docs/product.md #1](../../../docs/product.md) の作り置きペルソナ、
  [docs/design.md §4](../../../docs/design.md) の MealPlan、archive/`add-mealplan-assign-recipe`
- 既存の関連ファイル: [internal/domain/mealplan/entry.go](../../../internal/domain/mealplan/entry.go)、
  [internal/storage/sqlite/mealplan_repo.go](../../../internal/storage/sqlite/mealplan_repo.go)、
  [frontend/src/components/mealplan/week-calendar.tsx](../../../frontend/src/components/mealplan/week-calendar.tsx)

## Goals / Non-Goals

**Goals:**

- 「作り置きで 1 バッチ共有」と「別調理で N バッチ別々」を DB レベルで区別する
- 連続 N 日コピー時に毎回どちらかを選ばせる (ペルソナ既定は「作り置き」)
- 事後編集 (「バッチを分ける / 合流する」) で「あとで気が変わった」に対応する
- 視覚識別 (共通枠色 + 🔗 アイコン) をつけて、S01 でバッチ関係が一目でわかる

**Non-Goals:**

- 買い物リストの材料計上ロジックそのもの (`add-shopping-list` 側で扱う)
- 「作った」マークによる在庫連動 (`add-mealplan-cook-consume`)
- ドラッグ&ドロップでのバッチ変更 (メニュー経由のみ)
- 「常にこの選択」設定 (`S09` で扱う)
- バッチのメタ (調理日、賞味期限など) の追加

## Decisions

### D1. batch_id の型と生成戦略

**選択肢:**

- (a) `batch_id INTEGER AUTOINCREMENT` (グローバル連番) — Pros: 短い、単調増加。 Cons: 生成に SQL round trip が
  要る、UUID より衝突予測に有用な性質が少ない
- (b) `batch_id TEXT` (UUID v7、Recipe と同じ) ← 採用 — Pros: サーバ side / repo side / テストの fakeRepo でも
  同じ関数で生成できる、時系列順にソート可能、既存 `recipe.ID` と同じ扱い。 Cons: 文字列で 36 文字を消費

**採用: (b)**。理由: 一貫性 (recipes.id も UUID)。既存の `google/uuid` パッケージがすでに import 済み。

### D2. 既存行への batch_id 発行 (migration)

**選択肢:**

- (a) migration Up で全既存行に **同一の** batch_id を割り振る (「全部 1 バッチ」) — Cons: 買い物リスト計算が
  意味不明になる (1 セットしか計上しない)
- (b) migration Up で **行ごとにユニークな** batch_id を割り振る (「全部別調理」) ← 採用 — Pros: 直感的、
  material 計上時に「出現回数 = 別調理数」となり、既存機能の意味を壊さない。 Cons: 過去に連続コピーで作った行も
  「別調理」扱いになる (実際は作り置きだったかもしれない) が、ユーザーは事後 UI で `⋯ → バッチに合流する` で
  合流できる

**採用: (b)**。理由: MVP 期はまだ実データが少なく、既存の連続コピー結果を「別調理」に寄せてもユーザー体感は
軽微。逆に「全部作り置き扱い」にしてしまうと、単独割当した行同士まで 1 バッチに巻き込まれるので選べない。

SQLite (modernc.org/sqlite) の `randomblob(16)` + `hex()` で UUID 相当を生成できる。goose Up で:

```sql
UPDATE meal_plans SET batch_id = lower(hex(randomblob(16))) WHERE batch_id IS NULL OR batch_id = '';
```

を流し、その後 `ALTER TABLE meal_plans` で `NOT NULL` 制約を追加する (SQLite は ALTER で NOT NULL を後付けできる)。

### D3. サービス API 拡張の粒度

**選択肢:**

- (a) `CopyLeftover(..., shareBatch bool)` のシグネチャ拡張 — Pros: 呼び出し 1 本のまま、フロントの分岐も 1 引数
  で済む。 Cons: Wails bindings 再生成が要る、既存呼び出し (add-mealplan-assign-recipe 実装) を全部更新する
- (b) 別メソッド `CopyLeftoverSharedBatch` / `CopyLeftoverDistinctBatch` に分ける — Pros: 意味が明快。 Cons:
  メソッド数増、フロントで分岐が要る

**採用: (a)**。理由: 呼び出し側 (フロント + テスト) が既に 1 箇所にまとまっており、boolean 追加のコストが
小さい。将来 shareBatch に加えて他の flag が要るケースは想定しにくいので enum 化は先取りしない。

### D4. 分割 / 合流 API

**選択肢:**

- (a) `SplitBatch(ctx, date, slot, position)` / `MergeBatch(ctx, date, slot, position, targetBatchID)` /
  `ListSlotBatches(ctx, date, slot, recipeID)` の 3 本 ← 採用
- (b) 汎用 `SetBatchID(ctx, date, slot, position, batchID string)` 1 本 — Pros: シンプル。 Cons: 呼び出し側で
  UUID 発行のロジックを持つ必要が出る (「分ける」は新 UUID、「合流」は既存 UUID)、UI 側で 3 条件チェックを
  書くことになり整合性が保てない

**採用: (a)**。理由: 意図が API に載り、フロントは UUID を知らなくてよい。制約 (「同一レシピ / ±4 日以内 /
同 slot」) はサーバで一元的にかける。

#### D4a. SplitBatch の semantics: 「起点日を残して他を削除」

**選択肢:**

- (a) 対象 1 行だけ新 UUID に付け替える (バッチ関係だけを解消、行は残る) — Pros: 実装が単純。 Cons: UI
  ラベル「作り置きから外す」は「作り置き行為を取り消す」と読めるので意図と乖離。「1 行だけバッチから
  抜きたい」のユースケースは MVP では未確認
- (b) 対象行が属するバッチ全体を対象に、起点日 (バッチ内で最も古い日付の行) を残して他行を全削除、
  起点行に新 UUID を発行 ← 採用 — Pros: 「連続 N 日コピー」の逆演算として自然、買い物リストの計上と
  UI 表示の整合性が保てる、ユーザーの直感 (「作り置きしたのをやめる」) と一致

**採用: (b)**。理由: MVP は「作り置きを取り消す」ユースケース優先。「1 行だけ抜きたい」細かな編集は、
削除 + 再割当で代替可能なので proposal スコープ外。起点日の判定は SQL の `MIN(date)`、同日複数行 (未発生)
は `MIN(position)` で決定的にする。

シグネチャは `SplitBatch(ctx, date, slot, position) (newBatchID string, err error)` を維持し、呼び出し側は
「バッチ内のどれかの行」を指すだけでよい (起点行を指す必要はない)。バッチが元々 1 行だけの場合は no-op で
`(currentBatchID, nil)` を返す。

#### D4b. MergeBatch の制約: 3 条件

**選択肢:**

- (a) 「同 slot」だけをチェック (現状案の初期実装) — Cons: 別レシピにまとめる / 過去何週間前とまとめる /
  遠い日付とまとめる、が全部通ってしまう。「1 回の調理」の意味を破壊し、後続の買い物リスト計算が
  現実と乖離する
- (b) 「同 slot / 同レシピ / 対象行から ±max(`LeftoverCopyPresets`) 日以内」の 3 条件を AND でチェック ← 採用

**採用: (b)**。理由: 「作り置き = 1 回の調理でできる同一レシピ」の意味を守る不変条件。日付レンジは
CopyLeftover が「連続 N 日」でしか作り置きを生成しないので、事後まとめも同じレンジで揃える。「1 日飛ばし」
(月火・金) は現実にあるので条件からは外す (連続日制約は入れない)。

チェックは 2 段構え:

1. **`ListSlotBatches(ctx, date, slot, recipeID)`** — 候補を返す段階で 3 条件でフィルタ。呼び出し元は
   起点行の recipe_id を渡すだけ (`Entry` から自明)
2. **`MergeBatch(ctx, date, slot, position, targetBatchID)`** — 実行時に防御的に再チェック。targetBatchID
   の他の行のうち少なくとも 1 つが 3 条件を満たすかを検証、違反時は `ErrBatchMergeDenied` を返す

`ErrBatchMergeDenied` は `ErrBatchTargetNotFound` (指定 batch_id が全く存在しない) と別 sentinel で
分ける。フロントは i18n key `mealplan.errors.batch_merge_denied` (追加) と `batch_target_not_found`
(既存) を別文言で表示できる。

### D5. UI 色ローテーション

**選択肢:**

- (a) HSL の hue を batch_id からハッシュ生成 — Pros: 無限に区別できる。 Cons: 色数が多すぎて識別できない、
  Tailwind class に落としづらい
- (b) 事前定義した Tailwind class の配列 (2〜3 色) を batch_id ハッシュで index する ← 採用 — Pros: 少数の
  区別しやすい色で「同じかどうか」判別だけできればよい、Tailwind の JIT と相性がいい。 Cons: 4 バッチ以上が同時に
  存在すると同じ色が回るが、位置が離れていれば混同されない

**採用: (b)**。理由: MVP 期の 1 週間には多くて 3〜4 バッチしか同時に存在しないので、3 色で十分。

`frontend/src/components/mealplan/mealplan-config.ts` に:

```ts
export const BATCH_COLORS = [
  "border-orange-400 bg-orange-50",
  "border-emerald-400 bg-emerald-50",
  "border-violet-400 bg-violet-50",
] as const;
```

を置き、`batch-color.ts` で `hash(batchID) % BATCH_COLORS.length` を返す。1 行しか含まないバッチは色を返さない
(呼び出し側で null 判定して枠色をつけない)。

### D6. バッチメンバー数の判定コスト

**選択肢:**

- (a) フロントで週全体の `batch_id → count` を毎レンダー計算 — Pros: DB スキーマが軽い。 Cons: 週跨ぎの計算が
  必要な合流候補ダイアログのときに、その週の外にも同じ batch_id の行があるかの判定が要る
- (b) サーバの `ListWeek` 応答に `batchMemberCount int` (このバッチ全体の行数) を追加 ← 採用 — Pros: 「1 行だけの
  バッチは色を付けない」判定が一発で済む、フロントの計算コストなし、合流候補ダイアログにもそのまま使える。 Cons:
  DTO が広がる

**採用: (b)**。理由: バッチメンバー数はサーバ (SQLite) 側でウィンドウ関数 (`COUNT(*) OVER (PARTITION BY batch_id)`)
で 1 クエリで返せる。フロントに集計ロジックを持ち込まない方が「同じデータで判断が変わる」バグを防げる。

### D7. 合流候補 UI

**選択肢:**

- (a) `⋯ → バッチに合流する` を選ぶと **同じ slot 内の他の複数行バッチ** を一覧するサブメニュー — Pros: 意図が
  明快、選択肢が絞れる。 Cons: 1 週表示のときに 3 バッチ以上あると縦に伸びる
- (b) 直接クリック UX (「隣接する同名レシピの帯をクリックすると合流する」) — Pros: 極めて直感的。 Cons: 実装が
  複雑、DnD 相当の当たり判定が要る

**採用: (a)**。理由: MVP は明示的な選択で通す。バッチ数が多い場合の折り畳みは将来課題。

## Risks / Trade-offs

- [migration Up で全既存行を「別調理」に寄せると、過去の連続コピーは事後合流が必要] → 実データが少ない MVP 段階
  では影響軽微。文言で「既存の連続コピー結果は『別調理』扱いになりました」と初回表示で告知するか、README で触れる
- [color rotation は 3 色で回るので、4 バッチ以上あると色が重複] → MVP 期の 1 週間には多くて 3〜4 バッチしか
  存在せず、位置が離れていれば混同しにくい。4 色に増やすかは実利用フィードバックで判断
- [事後合流のダイアログが縦に伸びる可能性 (バッチ数が多いとき)] → まずは全候補を単純にリスト表示。折り畳みは
  UI 実装後の UX チューニングで対応
- [Wails bindings シグネチャ変更で、archive 済み add-mealplan-assign-recipe の実装コードとテストを touch する
  必要がある] → 変更範囲は `week-calendar.tsx` の `CopyLeftover` 呼び出しと `mealplan_test.go` の fakeRepo。tasks
  で明示的に列挙し、変更漏れを防ぐ
- [`ListWeek` の DTO に `batchMemberCount` を追加すると、bindings 再生成でフロント側の型も影響] → 既存の Entry
  型を拡張するだけで済み、breaking change は無い (フロントは未使用フィールドを無視する)
- [SplitBatch の「起点日」を `MIN(date)` で決めるが、同日に複数バッチが並ぶユースケース (現状未発生) は
  想定外] → 現状 CopyLeftover は起点日を含む連続 N 日を扱うだけなので、同一 (date, slot) に同じ batch_id が
  複数 position で並ぶことは実装上起きない。同日に別バッチが並ぶことはあり得るが、それは別 batch_id なので
  MIN(date) の起点判定に影響しない。仮に将来「同日に同一 batch_id 複数行」が発生するとしても、
  `MIN(position)` の tiebreaker で決定的に選べる
- [MergeBatch の日付レンジチェックは `LeftoverCopyPresets` の最大値に依存するので、preset を変更すると
  Merge のふるまいも変わる] → 意図的な結合。CopyLeftover と Merge を同じレンジで揃えるという設計原則を
  保つため。preset 変更のインパクトは docs に記載する

## Migration Plan

- migration `00004_meal_plans_batch.sql` (goose):
  1. `ALTER TABLE meal_plans ADD COLUMN batch_id TEXT` (デフォルト NULL でカラム追加)
  2. `UPDATE meal_plans SET batch_id = lower(hex(randomblob(16))) WHERE batch_id IS NULL`
  3. `CREATE INDEX idx_meal_plans_batch_id ON meal_plans(batch_id)`
  4. `NOT NULL` 制約は SQLite の ALTER 制限のため、実質的にはアプリ側 (Assign / CopyLeftover) で必ず batch_id
     を書き込むことで担保する。DB 上は `NOT NULL` を付けず、代わりに `CHECK (length(batch_id) > 0)` を追加する
- ロールバック: goose Down で `DROP INDEX idx_meal_plans_batch_id; ALTER TABLE meal_plans DROP COLUMN batch_id`。
  batch_id を失っても既存の割当・買い物リストの動作 (現状は N 倍計上) に戻るだけ
- Wails bindings は `wails3 generate bindings` で `MealPlanService.CopyLeftover` のシグネチャ拡張、`SplitBatch` /
  `MergeBatch` メソッド追加、`Entry.BatchID` / `Entry.BatchMemberCount` フィールド追加が反映される
