## Why

S01 献立カレンダーは現在 1 週固定表示で、**前後の週を見る手段が UI 上に無い**。目視確認で「翌週が
表示できない」ことが明らかになった。product.md §S01 には期間切替 (1 週 / 2 週 / 4 週) と前後ナビが
計画として書かれているが実装されておらず、`add-mealplan-batch-linking` で導入した「連続 N 日コピー」で
翌週跨ぎのデータを作れる (例: 日曜起点で連続 3 日) にも関わらず、その翌週側の行が UI から確認できない。
S01 の基本操作としても最低限「次週 / 前週」に移動できないと使い物にならない。

## What Changes

- S01 画面の**期間切替 UI** (1 週 / 2 週 / 4 週) と **前後ナビゲーション** (`＜` / `＞` / 「今週」) を
  画面右上に追加する
- **既定表示は 1 週間**を維持 (現状と同じ)
- 期間切替を「2 週」にすると横方向に 14 列表示 (必要に応じ横スクロール)、「4 週」は行方向に 4 週分
  折り返し表示 (product.md §S01 記載通り)
- `MealPlanService.ListWeek(weekStart)` を **`ListPeriod(periodStart, periodWeeks)`** に拡張、
  または新規メソッド `ListPeriod` を追加して既存 `ListWeek` は互換 wrapper として残す (design.md で確定)
- フロント側 `useQuery` の queryKey に `periodStart` と `periodWeeks` を含める。TanStack Query の cache 単位が
  週から期間に変わる
- 前後ナビは:
  - `＜`: `periodStart` を `periodWeeks` 分過去にシフト
  - `＞`: `periodStart` を `periodWeeks` 分未来にシフト
  - 「今週」: `periodStart` を「今日を含む週」の月曜日に戻す
- 期間切替と `resize-mealplan-cells` の `CellSizePreset` (期間 × slot 数) が正しく連動する
  (2 週 / 4 週表示のときはセルが小さくなる、既に設計済み)

## Non-goals

- **ドラッグ&ドロップでの日付移動**: 別 proposal
- **月間表示 (30 日カレンダー)**: 4 週で十分。今スコープ外
- **前後ナビでの keyboard shortcut** (`←` / `→` キー): 後続改善
- **URL クエリで期間を保持**: memory history なので不要 (別途 S09 で「起動時の既定期間」設定は検討)
- **買い物リスト画面 (S07) の期間追従**: 別 proposal (`add-shopping-list`) で対応

## Capabilities

### Modified Capabilities

- `mealplan`:
  - **MODIFIED**: 「週表示の初期状態」Requirement — 「1 週」表示だけでなく「1 / 2 / 4 週」の期間切替が
    可能になる旨に拡張。初期表示は「今日を含む週 (1 週表示)」で不変
  - **MODIFIED**: 「週データ取得の境界」Requirement — 週 (7 日) 固定だった取得境界を、任意の連続
    期間 (7 / 14 / 28 日) に一般化
  - **ADDED**: 「期間切替」Requirement (1 週 / 2 週 / 4 週 の切替と、各期間の表示レイアウト)
  - **ADDED**: 「前後ナビゲーション」Requirement (`＜` `＞` `今週` ボタンで期間を移動)

## Impact

- **新設ファイル**:
  - `frontend/src/components/mealplan/period-nav.tsx` — 期間切替トグル + 前後ナビの UI コンポーネント
- **更新ファイル**:
  - `internal/service/mealplan.go` — `ListPeriod(periodStart string, periodWeeks int) ([]Entry, error)`
    追加 (または `ListWeek` を拡張。design で確定)
  - `internal/storage/sqlite/mealplan_repo.go` — 同上のクエリ実装
  - `internal/service/mealplan_test.go` / `internal/storage/sqlite/mealplan_repo_test.go` — 期間 7/14/28 日の
    ケース追加
  - `frontend/src/routes/index.tsx` (or `meal-plan.tsx` 相当) — `periodStart` / `periodWeeks` state を持ち、
    `WeekCalendar` に渡す。`useQuery` の queryKey に含める
  - `frontend/src/components/mealplan/week-calendar.tsx` — props に `periodWeeks` を追加、grid 列数と
    行折り返しロジックを period に応じて分岐
  - `frontend/src/components/mealplan/mealplan-config.ts` — `CellSizePreset` の pickCellPreset に
    `periodWeeks` を渡す (resize-mealplan-cells で既に定義済み前提)
  - `frontend/src/locales/{ja,en}.json` — 期間切替 / ナビの i18n key を追加
  - `frontend/e2e/bindings-shim/mealplanservice.ts` — `ListPeriod` 追加 (または `ListWeek` シグネチャ変更)
  - `docs/product.md` §S01 — 実装済みとして期間切替 / 前後ナビの説明を更新
  - `docs/scenarios/mealplan.feature` — 前後ナビ / 期間切替のシナリオを追加 (既存の @planned 相当を
    活性化)
- **生成物 (再生成のみ)**: `wails3 generate bindings` — サービス変更を反映
- **migration**: なし

### 後続 proposal への引き継ぎ

- **`add-mealplan-batch-linking`**: 週跨ぎ作り置きの可視化が可能になる (Split の起点判定 / Merge レンジ判定
  は既に週跨ぎ対応済み。ナビが動けば画面上でも追跡できる)
- **`add-shopping-list` (S07)**: 期間切替と同期する CTA を張るために、`periodStart` / `periodWeeks` を
  引数化する下地ができる
- **`add-mealplan-cross-slot-leftover`**: 別 proposal だが、期間ナビの実装後は跨ぎシナリオの表示検証が
  楽になる
