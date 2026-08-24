## Why

MVP の中核画面である S01 献立カレンダーを、まず「1 週間分の日 × slot グリッドを表示する」ところまで通し、以降の割当・複数週・買い物リスト連携 proposal の
土台にする。実データ層 (repository + migration) はまだ整っていないため、本 proposal では **Wails サービスが固定値を返す**ところで打ち止めにし、
UI + サービス境界 + ドメイン型 を先に確定させる。これにより、次の `add-mealplan-assign-recipe` は「サービスの中身を repo 実装に差し替えるだけ」で
進められるようにする。

## What Changes

- 新規サイドバー項目 S01「献立カレンダー」を追加し、TanStack Router の初期ルートを S01 に設定する
- 週カレンダー UI (7 日 × slot グリッド、CSS Grid) を実装。1 セル = 1 slot、slot 内は 0..N 品を縦積みで表示する
- 現在の週 (今日を含む ISO 週、月曜始まり) を対象に、日付ヘッダと slot ラベルを描画する
- slot 表示切替 (右上トグル「夕食のみ」/「3 食表示」、既定は「夕食のみ」) を実装。切替はクライアント state のみで、サービスは常に全 slot を返す
- Wails サービス `MealPlanService` を新設し、`ListWeek(weekStartDate)` 相当のメソッドで固定値の `[]MealPlanEntry` を返す
- ドメインパッケージ `internal/domain/mealplan/` を新設し、`MealPlanEntry` / `Slot` / `Weekday` の型と、週の起点日を算出するヘルパを置く
- i18n キーを `frontend/src/locales/{ja,en}.json` に追加。日本語 UI 文字列はすべて `t()` 経由
- `internal/config/defaults.go` に「既定 slot 表示 = 夕食のみ」を追加し、UI の初期値はここから引く

### Non-goals

- **永続化しない**: SQLite テーブル / migration / repository は本 proposal では作らない (次以降の proposal でサービス実装を差し替える前提)
- **セル操作は未実装**: 空セル `＋` / 品行 `⋯` / slot 全体 `⋯` / 作り置きコピー は本 proposal のスコープ外 (別 proposal `add-mealplan-assign-recipe` 等で追加)
- **前後ナビ / 期間切替**: 「＜」「＞」「1 週 / 2 週 / 4 週」ボタンは UI 上に**表示しない** (期間切替は `add-mealplan-multi-week` で足す)
- **買い物リスト CTA**: 画面下部の「この期間の買い物リストを見る」は表示しない (`add-shopping-list` で追加)
- **候補提示 (MVP 後)**: 「✨ 翌週の候補を出す」CTA・使い切り/久しぶりに/時短の 3 軸は対象外
- **他画面 (S02 レシピ一覧 等)**: サイドバーの他項目はスタブ (プレースホルダ) のまま。実体は各 capability の proposal で足す

## Capabilities

### New Capabilities

- `mealplan`: 献立カレンダーの週表示と、日 × slot への品割当 (今回は表示のみ、割当以降は後続 proposal で追加)。
  本 proposal で追加する要件は「週表示」「slot 表示切替」「固定値でのエントリ描画」の 3 要件。

### Modified Capabilities

なし (mealplan は本 proposal で新設)。

## Impact

- **コード**:
  - Go: `internal/domain/mealplan/` 新設、`mealplanservice.go` (package main) 新設、`main.go` の `application.Options.Services` に登録
  - Frontend: `frontend/src/routes/mealplan.tsx` 新設、`frontend/src/components/mealplan/` 配下に週カレンダー用コンポーネント一式
  - i18n: `frontend/src/locales/{ja,en}.json` に `mealplan.*` キー群を追加
  - config: `internal/config/defaults.go` に `DefaultSlotView` を追加
- **バインディング再生成**: `wails3 generate bindings` を走らせて `frontend/bindings/` を更新 (git 管理外)
- **依存追加なし**: 新規 npm パッケージ / Go モジュールは追加しない
- **テスト**: Go 側は domain (週起点計算) + service (固定値応答) の単体テスト。Frontend 側は runner 未導入のため type-check + build のみ

### 後続 proposal への引き継ぎ

- `MealPlanService.ListWeek` の**中身**を repository 呼び出しに差し替える (`add-mealplan-assign-recipe` の一部として実施)。
  シグネチャと戻り値型は本 proposal で確定させておく
- SQLite の `meal_plans` テーブル + migration + `internal/storage/sqlite/mealplan_repo.go` は `add-mealplan-assign-recipe` で新設
- 前後ナビ / 期間切替 UI (`＜` `＞` / 「1 週」「2 週」「4 週」トグル) は `add-mealplan-multi-week` で追加
- セル操作 (品追加・入替・削除・連続 N 日) は `add-mealplan-assign-recipe` で追加
- 空セル → レシピ選択ダイアログは `add-recipe-listing` (S02 のミニ版として再利用) に依存
