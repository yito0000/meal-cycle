## Context

- 動機は [proposal.md](proposal.md) 参照。前提として `add-mealplan-batch-linking` の Merge 3 条件のうち
  「同時間帯」制約が実生活と乖離することが目視確認で判明した
- 現状の repo:
  - `ListMealBatches(ctx, date, meal, recipeID)` は同時間帯でフィルタ
  - `MergeBatch` は「対象行 (date, meal, position) を targetBatchID に付け替え」だが同 時間帯内の他行が
    条件を満たすかだけをチェック

## Goals / Non-Goals

**Goals:**

- 「別の時間帯 (時間帯) にも同じ作り置きを持ち越せる」ユースケースを提供
- 買い物リスト計上 (distinct batch_id 単位で 1 セット) の意味を保つ
- 既存 UI (作り置きから外す / 他の作り置きにまとめる) との一貫性を保つ

**Non-Goals:**

- CopyLeftover の 時間帯跨ぎ拡張 (「連続 3 日を昼食と夕食両方に」のような複雑 UX)
- ドラッグ&ドロップ

## Decisions

### D1. 「別時間帯にも足す」の API 形状

**選択肢:**

- (a) `MergeBatch` を「対象行を targetBatchID に付け替え」 の意味のまま使い、フロント側で「対象時間帯 に
  空行を Assign → その batch_id を狙って Merge」の 2 段呼び出し — Cons: 中間状態が「別バッチの新行」で
  枠色チカチカ、失敗時ロールバック複雑
- (b) 専用サービスメソッド `CopyToOtherMeal(ctx, sourceDate, sourceMeal, sourcePosition, targetDate,
  targetMeal) error` を追加 ← 採用 — Pros: サーバ側 1 トランザクションで完結、意図が API に載る、
  衝突解決 (append / overwrite) は CopyLeftover の repo ヘルパを共有できる

**採用: (b)**。理由: `CopyLeftover` は「同時間帯 × N 日」、`CopyToOtherMeal` は「別時間帯 × 対象日」で
軸が対称的。両方をサーバで 1 メソッドで完結させれば UI 側は迷わない。

### D2. `ListMealBatches` → `ListMergeCandidates` に改名するか

**選択肢:**

- (a) 名前を維持し、内部フィルタから同時間帯 を外す — Cons: メソッド名が実態と乖離する
  (「Meal」名を含むメソッドが時間帯をフィルタしない)
- (b) `ListMergeCandidates(ctx, date, meal, recipeID)` に改名 ← 採用 — Pros: 意図が名前に載る、bindings
  再生成で漏れなく更新できる

**採用: (b)**。理由: 名前は API 契約の一部で、実装意図を表すべき。フロント側の呼び出しは 1 箇所
(`week-calendar.tsx` の `openMergeDialog`) だけなので、更新コストは小さい。`meal` パラメータは残す
(表示情報として使う候補が「別 時間帯にある」ことを判別するため)。

### D3. 「別の時間帯にも作り置きを追加」の入力 UI

**選択肢:**

- (a) 対象 (対象日, 対象時間帯) をユーザーに選ばせるダイアログ — Pros: 明示的、誤操作リスク低。 Cons:
  ステップが 1 つ増える
- (b) セル間のドラッグ&ドロップ — Cons: MVP 対象外、実装複雑
- (c) セル右上の `＋` メニュー内に「作り置きを持ち越し」項目を追加 — Cons: `＋` は「新規追加」の意味で
  混乱する

**採用: (a)**。理由: UX 明示的、実装がシンプル。ダイアログは「対象日を日付ピッカーで選ぶ + 対象時間帯 を
セグメント切替」の 2 入力。対象日はデフォルトで「今日 + 1 日」、対象 時間帯はデフォルトで「昼食」
(現実の使い方に一致)。

### D4. 対象日 / 対象時間帯 のバリデーション

**選択肢:**

- (a) サーバ側で「対象日は起点日から ±4 日以内」「対象 時間帯は元と異なる」を検証、違反時
  `ErrBatchMergeDenied`
- (b) フロント側で先にバリデーションし、日付ピッカーで選択不可にする
- (c) 両方 (defense in depth) ← 採用

**採用: (c)**。フロントで選択候補を絞る (誤操作防止) + サーバ側で最終防衛 (直接 API を叩かれた場合の
安全)。既存の `ErrBatchMergeDenied` を流用。

## Risks / Trade-offs

- [「別時間帯にも作り置きを追加」で `CopyLeftover` と ロジックが重複しがち] → 内部の insert 関数
  (`insertMealPlan`) と衝突解決 (`prepareCopySource` / `PlanCopy`) を共有することで DRY を保つ
- [同一 (date, meal) に既に別バッチの同一レシピがある場合、UI は「重複」と表示すべきか] → 現状の
  `CopyLeftover` も PlanCopy 内で重複を除外している。同じロジックを流用
- [別時間帯跨ぎでの視覚識別 (枠色) が「一週間で 2 時間帯表示」中に見えづらい] → 3 時間帯表示の場合は
  カレンダー内で縦方向に離れてしまう。MVP はこの見え方でも許容、色付けの重複問題として観察する

## Migration Plan

- サービス API のリネーム (`ListMealBatches` → `ListMergeCandidates`) と新メソッド追加のみ。
  DB スキーマ変更なし
- 実装後の見え方: `⋯` メニューに「別の時間帯にも作り置きを追加」項目が増える。「他の作り置きにまとめる」
  の候補ダイアログに別時間帯候補も含まれる (「月曜・夕食」「火曜・昼食」等 時間帯名付き表示)
- ロールバック: 新メソッド `CopyToOtherMeal` を消し、`ListMergeCandidates` を `ListMealBatches` に
  戻して同時間帯 フィルタを再有効化すれば元の挙動になる
