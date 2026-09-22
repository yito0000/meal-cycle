## Why

現在 1 品の「別セルへの移動」は `⋯ → 削除` + 「別セルの ＋ から選び直し」の 2 段操作でしか実現
できない。作り置き前提で毎週回すペルソナにとって「木曜の肉じゃがを金曜に移したい」「昼食予定を
夕食に振り替えたい」のような小さな入れ替えは頻繁に発生し、その度にレシピ選択ダイアログを開き直すのは
負担が大きい。

ドラッグ&ドロップで「セル間の 1 行 移動」を提供すれば、直感的な 1 操作で完結する。既存の作り置き
コピー UX (`add-mealplan-cross-meal-leftover` および後続の `revise-mealplan-leftover-ux`) は
**コピー**、今回追加するのは **移動** で、意味が対照的だがユーザーモデルではどちらも「並べ替え」の
一部として扱える。

具体例:

- 木曜・夕食に置いた「肉じゃが」を、実生活の都合で金曜に振り替えたい
- 現状: `⋯ → 削除` → 金曜・夕食セルの `＋` → レシピ選択ダイアログ → 「肉じゃが」を検索して選ぶ (4 操作)
- 目標: 木曜・夕食セルの「肉じゃが」行を金曜・夕食セルにドラッグ (1 操作)

## What Changes

- **`MoveEntry(sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal) error`** を repo と
  service に新規追加する。1 行を別セルに移動する:
  - 対象行を削除 (source の position は詰め直す)
  - 対象セル (targetDate, targetMeal) の末尾 (position = max+1) に同じ recipeId + 同じ batchId で
    挿入する
  - **作り置きに属する行 (batchMemberCount > 1) は UI で draggable=false**。個別に動かすと作り置きの
    まとまりが意図せず崩れるため (Windows 手動確認で判明)、移動したい場合は既存の
    `⋯ → 作り置きから外す` で単独行にしてから D&D する 2 段操作を推奨する
  - repo/service の `MoveEntry` 自体は batchId を据え置きで動かす (backward-compatible)。フロントが
    batch member 行を送らないことで運用上の禁止を実現する
- **フロント D&D**: HTML5 Drag and Drop API を使う。`<div draggable>` を品行に、`onDragStart` で
  source 情報を dataTransfer に載せ、cell 側の `onDragOver` / `onDrop` で target を受け取り、
  MoveEntry を呼ぶ
- **同一セル内 D&D は既存 Reorder に流す**: source と target が同じ (date, meal) の場合は
  ReorderMeal で position を差し替える。既存の Reorder UI (上下ボタン) は残す
- **ドロップターゲット可視化**: ドラッグ中は各セルにハイライト枠を付ける。同一セル (自身) にドロップ
  したときは no-op
- **モバイル / touch はサポート外**: MVP は desktop (Wails webview) 用途に絞る。将来は
  `@dnd-kit/core` 等の導入で対応可能だが、今スコープ外
- **既存の連続 N 日コピー / 作り置きを広げる / 他の作り置きにまとめる / 別セルに追加は影響を受けない**。
  それらは「コピー」であり、D&D は「移動」

## Non-goals

- **D&D コピー** (Ctrl キー押下時に移動でなくコピーになる等の修飾キー UI): 移動と混同されやすく、
  「作り置きを広げる」UX でカバー済み
- **touch (モバイル / タブレット) 対応**: WSL2 開発時の対象外、Windows デスクトップ想定
- **セル間で複数行を選択して一括移動**: MVP は 1 行ずつ
- **作り置きに属する行 (batchMemberCount > 1) の D&D 移動**: 個別移動は作り置きのまとまりを崩す。
  移動したい場合は既存の `⋯ → 作り置きから外す` で単独行にしてから D&D する 2 段で対応する。
  「batch 全体を N 日ぶんまとめてシフト」も MVP 対象外 (target 側の衝突解決や meal 変更時の挙動が
  複雑化するため)
- **時間帯ラベルや日付ヘッダに対する D&D**: 「日付列を丸ごと別の日にドラッグ」等は Non-goal
- **Playwright での D&D 自動化**: Playwright の drag は近似しかできないので、E2E は
  test.skip でシナリオ名だけ残す

## Capabilities

### Modified Capabilities

- `mealplan`:
  - **ADDED**: 「1 行 移動」Requirement — Repo/Service で MoveEntry を公開し、UI からは D&D で
    駆動する。同一セル内は既存 Reorder に流す

## Impact

- **新設ファイル**:
  - なし (D&D は既存コンポーネントに handler を追加する形で実装)
- **更新ファイル**:
  - `internal/storage/sqlite/mealplan_repo.go` — `MoveEntry` メソッドを追加。1 トランザクションで
    「対象行削除 + source position 詰め直し + target 末尾に insert」
  - `internal/service/mealplan.go` — `MoveEntry(ctx, sourceDate, sourceMeal, sourcePosition,
    targetDate, targetMeal) error` を追加。同一セルの場合は ReorderMeal に流すか (要検討 →
    design D2)、または UI 側で分岐する
  - `internal/storage/sqlite/mealplan_repo_test.go` / `internal/service/mealplan_test.go` — 新規
    ケース追加
  - `frontend/src/components/mealplan/meal-plan-cell.tsx` — 品行に `draggable`, セル本体に
    `onDragOver`/`onDrop`。ドラッグ中は視覚ハイライト
  - `frontend/src/components/mealplan/week-calendar.tsx` — `MoveEntry` mutation を配線
  - `frontend/e2e/bindings-shim/mealplanservice.ts` — `MoveEntry` を追加
  - `frontend/src/locales/{ja,en}.json` — D&D 中のスクリーンリーダー用アナウンス key を追加
  - `docs/product.md` §S01 — D&D による 1 行 移動 UX を追記
  - `docs/scenarios/mealplan.feature` — シナリオ追加
  - `frontend/e2e/mealplan-*.spec.ts` — テスト名を追加 (実行は test.skip)
- **生成物**: `wails3 generate bindings` (`.gitignore` 対象なので変更コミットなし)
- **migration**: なし (既存スキーマで完結)

### 後続 proposal への引き継ぎ

- **touch 対応**: 別 proposal でスマホ / タブレット向けに `@dnd-kit/core` 等を導入する
- **セル間の別バッチ移動 UX**: MoveEntry は batchId をそのまま持ち越すため、`⋯` から「移動先で
  作り置きから外す」等の追加操作は別 proposal で扱う
- **Playwright での D&D**: Playwright の限界を確認しつつ、実運用に耐える E2E は次イテレーション
