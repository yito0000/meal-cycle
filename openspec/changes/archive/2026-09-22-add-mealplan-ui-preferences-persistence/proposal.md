## Why

S01 献立カレンダーの表示設定 (時間帯表示切替 / 期間長切替) は現状 `useState` の初期値で毎マウント
既定値 (夕食のみ / 1 週) にリセットされる。ユーザーが「3 食表示」に切り替えても、アプリを閉じて
再度開くと夕食のみに戻る。作り置き前提のペルソナでも「今週は朝も入力したい」等の切替はよくあり、
毎回設定し直すのは負担が大きい。

具体例:

- 月曜朝: 「3 食表示 + 2 週」に切り替えて 1 週間の朝昼夕を並べて計画
- 火曜朝: アプリを起動 → 夕食のみ + 1 週に戻っている → 毎回切り替え直し

期間 (`periodStart`) は「今週基準」で毎回リセットするのが自然なため、永続化対象から外す。
「時間帯表示」「期間長」だけを保存する。

## What Changes

- **`visibleMeals` の永続化**: `Set<string>` を localStorage に保存 (JSON 化した文字列配列)
- **`periodWeeks` の永続化**: `1 | 2 | 4` を localStorage に保存
- **読み込み時のフォールバック**: パース失敗 / 値異常 / storage 読み込み失敗はすべて既定値
  (`MEALPLAN_DEFAULTS.defaultVisibleMeals` / `periodWeeks: 1`) にフォールバック。エラーで
  アプリが落ちない防御
- **書き込み失敗の握り潰し**: try/catch で握り潰し、ログも出さない (private mode 等での
  QuotaExceeded / SecurityError 対策)
- **1 箇所のフックに集約**: `frontend/src/lib/mealplan-preferences.ts` (新設) に read/write ヘルパを置く。
  `meal-plan.tsx` はこのヘルパから `useLocalStoragePreference` 相当を呼び出す

## Non-goals

- **S09 設定画面**: MVP 後実装。ConfigService 経由の永続化に移管する予定 (design.md D1 参照)
- **periodStart の永続化**: 「今週」を基準にするのが期待挙動なので保存しない
- **他ページの UI 設定永続化**: レシピ一覧の sort / search 状態など、別 proposal で扱う
- **クロスデバイス同期**: 完全ローカルアプリなので不要
- **設定画面 UX**: 設定に直接触るユーザー UI は追加しない (既存の Meal 表示切替 / 期間切替のまま)

## Capabilities

### Modified Capabilities

- `mealplan`:
  - **ADDED**: 「S01 表示設定の永続化」Requirement — visibleMeals と periodWeeks の状態を
    アプリ再起動を跨いで保持する

## Impact

- **新設ファイル**:
  - `frontend/src/lib/mealplan-preferences.ts` — localStorage read/write ヘルパ + React hook
    (`useMealPlanPreference<T>(key, defaultValue, validate)`)
- **更新ファイル**:
  - `frontend/src/routes/meal-plan.tsx` — `useState` を `useMealPlanPreference` に差し替え
  - `docs/product.md` §S01 — 「表示設定は再起動後も保持される」を追記
  - `docs/scenarios/mealplan.feature` — シナリオ 2 件追加 (時間帯 / 期間長)
- **生成物**: なし
- **migration**: なし (localStorage は初回起動時に空、既定値にフォールバック)

### 後続 proposal への引き継ぎ

- **S09 設定画面**: localStorage に置いた key を SQLite `ui_preferences` テーブルに移す。
  移行時は localStorage を読んで DB に write、以降 DB を source of truth にする
- **`revise-mealplan-leftover-ux`**: 表示設定の UX と作り置き UX 統合の間に依存はない
