## Context

- 動機は [proposal.md](proposal.md) 参照
- 現状の repo は `Assign` / `Replace` / `Remove` / `Reorder` / `CopyLeftover` / `CopyToOtherMeal`
  等の操作を提供している。移動 (move) は明示的には無い
- 「削除 + 割当」の 2 段呼び出しでは、batchId が失われる (Assign が新規 batch を発行するため)。
  作り置きに属する 1 行を別セルに動かすときはこれが困る (作り置きが解体される)

## Goals / Non-Goals

**Goals:**

- 1 行を別セルに 1 操作で移動できるようにする
- 作り置きに属する行を移動しても batchId を維持し、作り置きが解体されないようにする
- 同一セル内の並び替えは既存 Reorder に流し、既存 UI (上下ボタン) と挙動を揃える

**Non-Goals:**

- touch / モバイル対応
- コピー D&D (`Ctrl` キー等)
- 複数選択 D&D

## Decisions

### D1. 作り置きに属する行の D&D 挙動

**選択肢:**

- (a) 個別移動を許容し batchId 維持 (当初案) — Cons: Windows 手動確認で「作り置き 3 行のうち 1 行だけ
  D&D で離れた日に飛ぶ」挙動が非直感的だった。作り置きは 3 行まとまって初めて意味を持つ運用で、個別
  にずらせると視覚 (連続 N 日) と DB が食い違う
- (b) 個別移動時に新しい batchId を発行 (SplitBatch 相当) — Cons: 破壊的操作が意図せず発動する
- (c) **batch member 行 (`batchMemberCount > 1`) は draggable=false にして個別 D&D を禁止** ← 採用 —
  Pros: 事故を確実に防ぐ。移動したい場合は既存 `⋯ → 作り置きから外す` で単独行にしてから D&D する
  2 段で明示的に扱える。repo/service の MoveEntry は既存の batchId 維持仕様のまま (backward-compatible)
- (d) batch 全体を差分日数ぶんまとめてシフトする — Cons: target 側の衝突 (既存割当) 解決や meal 変更
  時の挙動が複雑になり、MVP 対象外

**採用: (c)**。フロント側の draggable ゲートで運用上の禁止を実現し、repo/service API は据え置く。
`MoveEntry` の DB 挙動は「batchId 維持」で backward-compatible のまま (単独行の移動時に元 batchId を
そのまま持ち越す仕様は残す)。

### D2. 同一セル内 D&D の扱い

**選択肢:**

- (a) **UI 側で同一セル判定 → ReorderMeal に流す** ← 採用 — Pros: サーバ API はシンプル
  (MoveEntry は「異セル間」のみ)、既存 Reorder の position 管理を再利用できる
- (b) MoveEntry の source == target 時は内部で Reorder 相当を実行 — Pros: フロント側の分岐が減る
  Cons: 1 メソッドの責務が広がり、テストケースが増える
- (c) 同一セル内 D&D は no-op — Cons: ユーザーの期待から外れる (「肉じゃがを副菜の下に持って行きたい」
  ケースを塞ぐ)

**採用: (a)**。フロント側で `source.date === target.date && source.meal === target.meal` を検知して
既存 `ReorderMeal` を呼ぶ。position は D&D の drop 相対位置 (drop 対象 entry の前 or 後) から計算する。

### D3. Repo 実装のトランザクション境界

**選択肢:**

- (a) **1 トランザクション内で「対象行削除 + source position 詰め直し + target 末尾に insert」を
  すべて実行** ← 採用 — Pros: 原子性、失敗時のロールバックが自動
- (b) 3 つの独立クエリで実行 — Cons: 途中で失敗するとカレンダーが壊れる (対象行が消えたのに target
  に無い等)

**採用: (a)**。実装は `Remove` + `insertMealPlan` の内容を新関数の中でシーケンシャルに走らせる。
既存の `Remove` が position 詰め直しをやっているのを流用したいが、`Remove` は独自 tx を張るので
関数内部の共通ヘルパ (`removeRowTx(ctx, tx, date, meal, position) error`) を切り出してから使う。

### D4. drop 位置の解釈 (行の間 or 行の下)

**選択肢:**

- (a) **セルにドロップ = 末尾に挿入** ← 採用 — Pros: 実装がシンプル。他セルに移動するときの通常
  ユースケースは「そこに置く」で、細かい position 指定は要らない
- (b) 行間にドロップ = その位置に挿入、セル余白にドロップ = 末尾 — Cons: ドロップターゲット判定が
  複雑、UI 実装コスト増

**採用: (a)**。移動先の position は常に max+1。同一セル内 D&D は D2 のとおり Reorder 経由なので、
そちらで既存 UI (上下ボタン) 相当の position 指定を扱う。

### D5. D&D 時の視覚フィードバック

**選択肢:**

- (a) **ドラッグ中はセル本体に `outline-2 outline-blue-500` を追加、drop hover 時は
  `bg-blue-50` に強調** ← 採用 — Pros: 動的な視覚フィードバックが自然、キーボード操作の focus と
  デザインコンセプト共有
- (b) ゴースト要素を表示 (HTML5 D&D のデフォルト) — 現状のブラウザ挙動をそのまま使う
- (c) カレンダー全体を薄暗くして drop 可能セルだけ明るく表示 — Cons: 過剰

**採用: (a) + (b)**。ドラッグ中のブラウザデフォルトゴーストは活用しつつ、drop 可能セルの hover 時に
枠色を強調する。同一セル (source) 自身にも hover 枠を出すが、drop しても no-op (source と target が
同じ場合は D2 の判定で Reorder に流れるが、drop 対象 entry が無いと index=同 position で no-op)。

**追記 (Windows 手動確認フィードバック 3-4 回目)**: 掴んだ行 (ghost, カーソル追従) と元の位置に残る
source 行を **役割で塗り分ける** (3 役: ghost / source / drop target):

- **ghost (掴んで動く方)**: 薄い緑 (`bg-emerald-100 ring-2 ring-emerald-400`)。カーソルに追従し、
  「何を掴んでいるか」を主役として提示する。
- **source (元の位置に残る方)**: `bg-neutral-100 opacity-40` の薄いグレー placeholder。「ここに元々
  あった」ことは伝えるが視点を奪わない。
- **drop 可能セル**: 青の outline (blue-400)、既存のまま。

初期案では source 側に emerald を載せていたが、browser の ghost キャプチャが同じ style を拾って
「同じ濃さの緑が 2 つ」になり、視点が source にも引っ張られた (実機フィードバック 3 回目)。次の案で
source を灰色フェードに落として時間差の塗り分けを試したところ、今度は ghost が真っ白になってしまい
「掴んでる感」が薄れた (4 回目)。結果、以下の 3 段構えに落ち着いた:

**実装ポイント (時間差の塗り分け)**: browser の drag ghost キャプチャは `onDragStart` 完了直後に
同期で走る。この性質を利用して:

1. `onDragStart` の同期パスで DOM に一時的に emerald 系の class を `classList.add` で載せる (`e.currentTarget` 直操作)
2. browser がその DOM 状態を ghost 画像として capture する
3. `requestAnimationFrame` で一時 class を `classList.remove` し、その直後に `setDragSource` を呼ぶ
4. React re-render で source 行は `bg-neutral-100 opacity-40` の灰色フェードに切り替わる

これで ghost = 薄い緑 (capture 時点の DOM) / source = 灰色フェード (state 反映後の DOM) と、
同じ DOM ノードを時間軸で 2 種類の姿として使い分けられる。source の追跡は親コンポーネント
(WeekCalendar) の `dragSource: DragSource | null` state で持ち、各セルに props で配って該当 entry
(date/meal/position 一致) だけを装飾する。

### D6. カーソル affordance

**選択肢:**

- (a) 品行に `cursor-pointer`、drag 中も既定カーソル — Cons: 「掴める」ことが視覚的に判らない
  (Windows 手動確認で判明)
- (b) **単独割当行は `cursor-grab` (drag 中は `cursor-grabbing`)、batch member 行は
  `cursor-not-allowed`** ← 採用 — Pros: hover 時点で「掴めるか / 掴めないか」が判る。CSS だけで完結、
  D1 の draggable ゲートと視覚的に一貫

**採用: (b)**。tooltip (title 属性 or aria-label) は既存の「⋯ → 作り置きから外す」の存在を暗黙に
伝える表現に留め、明示的な「作り置きから外してから移動して下さい」等の説明は増やさない (⋯ メニューに
既に該当項目があるため)。

## Risks / Trade-offs

- [Wails WebView の HTML5 D&D 対応] → Chromium 系なので標準的にサポートされる。macOS 版でも同様
- [Playwright での D&D 自動化] → Playwright の `dragTo` は近似操作で、`onDragStart` /
  `onDragOver` / `onDrop` を確実には triggering しない。E2E は `test.skip` でシナリオ名だけ
  残し、実装は手動確認で担保する
- [作り置き 3 行のうち 1 行を離れた日に移すと、レンジ外に飛ばせる] → 「他の作り置きにまとめる」の
  ±4 日制約と衝突しない。移動なので既存 batch の他行が動くわけではなく、単に 1 行が離れた日に
  ぶら下がる形になる。UX 上はやや気持ち悪いが、SplitBatch を明示的に呼び出さない限り batch は
  維持する
- [D&D 中に mouseup を逃す事故] → ブラウザの `onDragEnd` で state をクリアする防御

## Migration Plan

- 新規メソッド追加のみで既存 API 変更なし。既存 UX (Reorder / Assign / Remove) はそのまま動作
- ロールバック: `MoveEntry` を repo/service から削除し、フロント側 D&D handlers を削除すれば元に戻る
- Wails bindings は再生成で更新される (`.gitignore` 対象)
