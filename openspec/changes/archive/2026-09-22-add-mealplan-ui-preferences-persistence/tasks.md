## 1. localStorage ヘルパの新設

- [x] 1.1 `frontend/src/lib/mealplan-preferences.ts` を新設する。以下 3 つを export する
    - `readPreference<T>(key: string, defaultValue: T, validate: (raw: unknown) => T | null): T`
      — localStorage read + JSON.parse + validate。失敗時 defaultValue。try/catch で握り潰す
    - `writePreference(key: string, value: unknown): void` — JSON.stringify + localStorage.setItem。
      try/catch で握り潰す
    - `useMealPlanPreference<T>(key, defaultValue, validate)` React hook — 初期値は read で解決、
      setter が呼ばれたら state 更新 + write
- [x] 1.2 key prefix は `mealcycle.mealplan.` に統一する。ハードコード禁止で
      `mealPlanKey(name: string)` 内部ヘルパを作る
- [x] 1.3 hook のテスト方針: 現状フロントに test runner が無いため、Vitest 導入 (別 proposal) を
      待つ。それまでは E2E で観察する

## 2. `meal-plan.tsx` の状態を差し替え

- [x] 2.1 `frontend/src/routes/meal-plan.tsx` の `visibleMeals` を `useMealPlanPreference` に
      差し替える。validate は「Meal enum の集合であること」
- [x] 2.2 同 `periodWeeks` を `useMealPlanPreference` に差し替える。validate は
      `AllowedPeriodWeeks (1|2|4)` にマッチすること
- [x] 2.3 `periodStart` は変更しない (既定は `weekStartOf(new Date())` のまま)

## 3. ドキュメント + E2E

- [x] 3.1 `docs/product.md` §S01 に「表示設定 (時間帯 / 期間長) は再起動後も保持される」を追記する
- [x] 3.2 `docs/scenarios/mealplan.feature` に以下シナリオを追加する
    - 時間帯切替が再起動後も保持される
    - 期間長切替が再起動後も保持される
    - 期間の開始日は毎回「今週」にリセットされる
- [x] 3.3 `frontend/e2e/mealplan-assign.spec.ts` (or 新 spec) に該当シナリオの test 名を追加する
      (DropdownMenu Portal に依存しないので実行可能な想定 — reload で state を確認する)

## 4. lint / format / テスト最終ゲート

- [x] 4.1 `task lint`。完了判定: exit code 0
- [x] 4.2 `task format`。完了判定: 整形差分のみ
- [x] 4.3 `go test ./internal/... ./`。完了判定: exit code 0 (Go 側変更は無いが regression 確認)
- [x] 4.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev`。完了判定:
      exit code 0
- [x] 4.5 `task test:e2e`。完了判定: exit code 0
