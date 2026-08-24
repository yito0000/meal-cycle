## 1. ドメイン層 (internal/domain/mealplan)

- [x] 1.1 `internal/domain/mealplan/` パッケージを新設し、`Slot` 型 (breakfast / lunch / dinner の const) を定義する
      (完了判定: `go build ./internal/...` が通る)
- [x] 1.2 `Entry` 構造体 (Date string / Slot Slot / Position int / RecipeName string) を追加する
      (完了判定: `go build ./internal/...` が通る)
- [x] 1.3 `WeekStartOf(t time.Time) time.Time` (与えた時刻が属する週の月曜日 00:00 Local を返す) を実装する
      (完了判定: 下記 1.4 のテストが pass する)
- [x] 1.4 `WeekStartOf` の table-driven テストを `internal/domain/mealplan/week_test.go` に追加する
    - 月曜を渡すとその日 00:00 が返る
    - 日曜を渡すと 6 日前の月曜が返る
    - 水曜を渡すと 2 日前の月曜が返る
    - 時分秒が入った時刻を渡しても 00:00 に正規化される
    - (完了判定: `go test ./internal/domain/mealplan/...` が pass する)

## 2. config への既定値追加

- [x] 2.1 `internal/config/defaults.go` の `SlotDisplay` 型 (dinner-only / all-slots) を採用する
      (既存: `SlotDisplayDinnerOnly` / `SlotDisplayAllThree` 定数と `SlotDisplay` 型が既に存在。tasks.md の
      `SlotView` はこの既存名に読み替えて実装。完了判定: `go build ./internal/...` が通る)
- [x] 2.2 `Config` struct と `Defaults()` に `MealPlanCellVisibleItems = 2` を追加する
      (`SlotDisplay = SlotDisplayDinnerOnly` は既存。完了判定: 下記 2.3 のテストが pass する)
- [x] 2.3 `internal/config/defaults_test.go` に既定値の検証テストケースを追加する
    - `Defaults().SlotDisplay` が `SlotDisplayDinnerOnly` と等しい (既存)
    - `Defaults().MealPlanCellVisibleItems` が 2 と等しい (追加)
    - (完了判定: `go test ./internal/config/...` が pass する)

## 3. Wails サービス (MealPlanService)

- [x] 3.1 ルート pkg に `mealplanservice.go` を新設し、`MealPlanService` struct を定義する
      (完了判定: `go build ./` が通る)
- [x] 3.2 `ListWeek(ctx context.Context, weekStart string) ([]mealplan.Entry, error)` を実装し、固定値の
      エントリを返す (月曜〜金曜の夕食に主菜 1〜2 品、うち 1 日は 4 品で「他 N 件」省略確認用)
      (完了判定: 下記 3.3 のテストが pass する)
- [x] 3.3 `mealplanservice_test.go` に BDD 風テストを追加する
    - should return entries for the requested week when weekStart is a monday
    - should return entries whose dates are all within monday..sunday of weekStart
    - should include one slot with more than 2 items to exercise 「他 N 件」表示
    - should keep the same fixed set regardless of weekStart value (本 proposal ではモック応答)
    - (完了判定: `go test ./...` を避け `go test ./internal/... ./` が pass する)
- [x] 3.4 `main.go` の `application.Options.Services` に `MealPlanService` を追加登録する
      (完了判定: `task dev` で起動し、devtools console から `MealPlanService.ListWeek("2026-08-24")` が
      配列を返すこと。目視は最終グループ 7.6 で確認する)

## 4. Wails バインディング再生成

- [x] 4.1 `wails3 generate bindings` を実行し、`frontend/bindings/github.com/yito0000/meal-cycle/` に
      `mealplanservice.js` と `internal/domain/mealplan/models.js` (Entry / Slot) が生成されることを確認する
      (完了判定: `frontend/bindings/` に該当ファイルが存在する)

## 5. フロントエンド (S01 献立カレンダー)

- [x] 5.1 `frontend/src/routes/meal-plan.tsx` の placeholder を実 UI に置き換える
      (route 登録は既存の `router.tsx` で `/` に MealPlanPage が割り当て済み。完了判定: `task dev` 起動後、初期表示に S01 が出る)
- [x] 5.2 サイドバー (`frontend/src/components/layout/sidebar.tsx`) の S01 「献立カレンダー」項目はスキャフォールド済み
      (`/` にリンク + activeProps でハイライト、labelKey `nav.meal_plan`)。追加編集不要
      (完了判定: 目視でサイドバーに 4 項目が出て、S01 がハイライトされる — 最終 7.6 で確認)
- [x] 5.3 `frontend/src/components/mealplan/week-calendar.tsx` を新設し、6rem + 7 列の CSS Grid を組む
      (行 = slot 表示状態: dinner_only なら 1 行、all_three なら 3 行)
      (完了判定: 目視で列 7 個 / 行 1 or 3 が描画される — 最終 7.6 で確認)
- [x] 5.4 列ヘッダに月曜〜日曜の日付ラベルを表示し、「今日」に該当する列を青系背景で視覚的に区別する
      (完了判定: 目視で今日の列がハイライトされる — 最終 7.6 で確認)
- [x] 5.5 slot 表示切替 UI を右上に追加。朝食 / 昼食 / 夕食 の 3 独立チェックボックス (2 択トグルから
      UX 改善、design.md D3 参照)。`useState<Set<string>>` で管理し、初期値は
      `MEALPLAN_DEFAULTS.defaultVisibleSlots` (Go の `DefaultVisibleSlots = ["dinner"]` をフロントでミラー)。
      少なくとも 1 slot は常に ON になるよう最後の 1 個は disabled
      (完了判定: 目視でチェックボックス切替により行数が 1〜3 の範囲で変わる — 最終 7.6 で確認)
- [x] 5.6 セル内で品を position 昇順に縦積み表示し、`MEALPLAN_DEFAULTS.mealPlanCellVisibleItems` を
      超える場合は i18n キー `pages.meal_plan.cell.more_items` で「他 N 件」を表示する
      (完了判定: 固定値の金曜夕食 slot に 4 品置いてあるため「先頭 2 品 + 他 2 件」表示になる — 最終 7.6 で確認)
- [x] 5.7 空セル (0 品) は破線ボーダーの空領域として描画する (＋ ボタン等の操作 UI は本 proposal では追加しない)
      (完了判定: 目視で空 slot に何も描画されない — 最終 7.6 で確認)
- [x] 5.8 TanStack Query で `MealPlanService.ListWeek(weekStart)` をラップし、`weekStart` はマウント時に
      `weekStartOf(new Date())` を "YYYY-MM-DD" に整形して渡す (Go/TS 双方に同名ヘルパを持つ形)
      (完了判定: DevTools Network / React Query devtools で 1 回だけ呼ばれることを確認 — 最終 7.6 で確認)

## 6. i18n

- [x] 6.1 `frontend/src/locales/ja.json` の `pages.meal_plan.*` を拡張し、`today` / `loading` / `load_error` /
      `slot.{breakfast,lunch,dinner}` / `slot_display.{aria_label,dinner_only,all_three}` /
      `cell.more_items` を追加する (既存 `pages.<screen>.*` 命名に揃えるため `mealplan.*` 独立ではなく
      `pages.meal_plan.*` 配下に置く。placeholder キーは削除)
      (完了判定: 目視で日本語が表示される — 最終 7.6 で確認)
- [x] 6.2 `frontend/src/locales/en.json` に同じキーの英訳を追加する
      (完了判定: i18n 言語切替時に英語表示になる。切替 UI が無ければ ja.json との key parity のみ目視確認)
- [x] 6.3 追加コンポーネントに日本語文字列がハードコードされていないことを grep で確認する
      (完了判定: `grep -rn "[ぁ-んァ-ヶ一-龠]" src/components/mealplan src/routes/meal-plan.tsx` が
      no-match で終了)

## 7. 検証 (lint / test / 型 / build)

- [x] 7.1 Go テストが全て pass することを確認する
      (完了判定: `go test ./internal/... ./` が pass する — 4 packages OK)
- [x] 7.2 Go 側 lint が通ることを確認する
      (完了判定: `task lint:go` が pass する — 0 issues)
- [x] 7.3 Frontend の TypeScript 型チェックが通ることを確認する
      (完了判定: `cd frontend && npm run type-check` が pass する)
- [x] 7.4 Frontend の Vite ビルド (dev) が通ることを確認する (import 解決の総合チェック)
      (完了判定: `cd frontend && npm run build:dev` が pass する — 1761 modules built)
- [x] 7.5 Frontend lint (Biome) が通ることを確認する
      (完了判定: `task lint:frontend` が pass する — 21 files, 0 errors)
- [ ] 7.6 `task dev` でアプリを起動し、S01 献立カレンダーが表示されることを目視確認する
      (完了判定: 目視で 7 日 × slot グリッドと固定エントリが描画される、slot トグルで行数が切り替わる。
      Wails GUI は環境依存のため apply 内では実行せず、ユーザー確認に委ねる)
