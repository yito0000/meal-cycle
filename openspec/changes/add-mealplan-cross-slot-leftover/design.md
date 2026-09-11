## Context

- 動機は [proposal.md](proposal.md) 参照。前提として `add-mealplan-batch-linking` の Merge 3 条件のうち
  「同 slot」制約が実生活と乖離することが目視確認で判明した
- 現状の repo:
  - `ListSlotBatches(ctx, date, slot, recipeID)` は同 slot でフィルタ
  - `MergeBatch` は「対象行 (date, slot, position) を targetBatchID に付け替え」だが同 slot 内の他行が
    条件を満たすかだけをチェック

## Goals / Non-Goals

**Goals:**

- 「別の時間帯 (slot) にも同じ作り置きを持ち越せる」ユースケースを提供
- 買い物リスト計上 (distinct batch_id 単位で 1 セット) の意味を保つ
- 既存 UI (作り置きから外す / 他の作り置きにまとめる) との一貫性を保つ

**Non-Goals:**

- CopyLeftover の slot 跨ぎ拡張 (「連続 3 日を昼食と夕食両方に」のような複雑 UX)
- ドラッグ&ドロップ

## Decisions

### D1. 「別 slot にも足す」の API 形状

**選択肢:**

- (a) `MergeBatch` を「対象行を targetBatchID に付け替え」 の意味のまま使い、フロント側で「対象 slot に
  空行を Assign → その batch_id を狙って Merge」の 2 段呼び出し — Cons: 中間状態が「別バッチの新行」で
  枠色チカチカ、失敗時ロールバック複雑
- (b) 専用サービスメソッド `CopyToOtherSlot(ctx, sourceDate, sourceSlot, sourcePosition, targetDate,
  targetSlot) error` を追加 ← 採用 — Pros: サーバ側 1 トランザクションで完結、意図が API に載る、
  衝突解決 (append / overwrite) は CopyLeftover の repo ヘルパを共有できる

**採用: (b)**。理由: `CopyLeftover` は「同 slot × N 日」、`CopyToOtherSlot` は「別 slot × 対象日」で
軸が対称的。両方をサーバで 1 メソッドで完結させれば UI 側は迷わない。

### D2. `ListSlotBatches` → `ListMergeCandidates` に改名するか

**選択肢:**

- (a) 名前を維持し、内部フィルタから同 slot を外す — Cons: メソッド名が実態と乖離する
  (「slot」名を含むメソッドが slot をフィルタしない)
- (b) `ListMergeCandidates(ctx, date, slot, recipeID)` に改名 ← 採用 — Pros: 意図が名前に載る、bindings
  再生成で漏れなく更新できる

**採用: (b)**。理由: 名前は API 契約の一部で、実装意図を表すべき。フロント側の呼び出しは 1 箇所
(`week-calendar.tsx` の `openMergeDialog`) だけなので、更新コストは小さい。`slot` パラメータは残す
(表示情報として使う候補が「別 slot にある」ことを判別するため)。

### D3. 「別の時間帯にも作り置きを追加」の入力 UI

**選択肢:**

- (a) 対象 (対象日, 対象 slot) をユーザーに選ばせるダイアログ — Pros: 明示的、誤操作リスク低。 Cons:
  ステップが 1 つ増える
- (b) セル間のドラッグ&ドロップ — Cons: MVP 対象外、実装複雑
- (c) セル右上の `＋` メニュー内に「作り置きを持ち越し」項目を追加 — Cons: `＋` は「新規追加」の意味で
  混乱する

**採用: (a)**。理由: UX 明示的、実装がシンプル。ダイアログは「対象日を日付ピッカーで選ぶ + 対象 slot を
セグメント切替」の 2 入力。対象日はデフォルトで「今日 + 1 日」、対象 slot はデフォルトで「昼食」
(現実の使い方に一致)。

### D4. 対象日 / 対象 slot のバリデーション

**選択肢:**

- (a) サーバ側で「対象日は起点日から ±4 日以内」「対象 slot は元と異なる」を検証、違反時
  `ErrBatchMergeDenied`
- (b) フロント側で先にバリデーションし、日付ピッカーで選択不可にする
- (c) 両方 (defense in depth) ← 採用

**採用: (c)**。フロントで選択候補を絞る (誤操作防止) + サーバ側で最終防衛 (直接 API を叩かれた場合の
安全)。既存の `ErrBatchMergeDenied` を流用。

## Risks / Trade-offs

- [「別 slot にも作り置きを追加」で `CopyLeftover` と ロジックが重複しがち] → 内部の insert 関数
  (`insertMealPlan`) と衝突解決 (`prepareCopySource` / `PlanCopy`) を共有することで DRY を保つ
- [同一 (date, slot) に既に別バッチの同一レシピがある場合、UI は「重複」と表示すべきか] → 現状の
  `CopyLeftover` も PlanCopy 内で重複を除外している。同じロジックを流用
- [別 slot 跨ぎでの視覚識別 (枠色) が「一週間で 2 slot 表示」中に見えづらい] → 3 slot 表示の場合は
  カレンダー内で縦方向に離れてしまう。MVP はこの見え方でも許容、色付けの重複問題として観察する

## Migration Plan

- サービス API のリネーム (`ListSlotBatches` → `ListMergeCandidates`) と新メソッド追加のみ。
  DB スキーマ変更なし
- 実装後の見え方: `⋯` メニューに「別の時間帯にも作り置きを追加」項目が増える。「他の作り置きにまとめる」
  の候補ダイアログに別 slot 候補も含まれる (「月曜・夕食」「火曜・昼食」等 slot 名付き表示)
- ロールバック: 新メソッド `CopyToOtherSlot` を消し、`ListMergeCandidates` を `ListSlotBatches` に
  戻して同 slot フィルタを再有効化すれば元の挙動になる
