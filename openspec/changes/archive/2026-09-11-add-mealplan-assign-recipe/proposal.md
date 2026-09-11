## Why

S01 献立カレンダーは `add-mealplan-week-view` で「表示」までを済ませたが、`MealPlanService.ListWeek` はまだ固定 dataset
を返している。ユーザーが実際にレシピを日 × slot に割り当て、翌週の献立を組み立てて保存できるようにする — この capability が
無いと後続の「作り置きコピー」「買い物リスト生成」「候補提示」もすべて空回りする。

## What Changes

- `meal_plans` テーブルを追加し、`(date, slot, position, recipe_id)` を主単位として保存する。1 slot = 0..N 品、position 順
- `MealPlanService.ListWeek` の実装を repo 呼び出しに差し替える。`RecipeName` は `recipes` と JOIN してサーバ側で解決する
- `MealPlanEntry` に `RecipeID` フィールドを追加する。`RecipeName` は表示用フィールドとして残し、JOIN 結果を載せる
- 入力バリデーションを domain 層のヘルパにまとめる。全割当系メソッドは date が "YYYY-MM-DD" であること / slot が
  breakfast|lunch|dinner のいずれかであること / position が非負かつ対象 slot に存在すること / Reorder の recipe ID
  集合が既存 slot と要素として一致すること / CopyLeftover の days が `config.LeftoverCopyPresets` に含まれることを
  検証する。違反時は `ErrInvalidDate` / `ErrInvalidSlot` / `ErrInvalidPosition` / `ErrEntryNotFound` /
  `ErrReorderMismatch` / `ErrInvalidCopyDays` のいずれかを返し、`meal_plans` は変化しない
- `MealPlanService` に割当系メソッドを追加する:
  - `AssignRecipe(date, slot, recipeId)` — 指定 slot に末尾追加 (position は max+1)
  - `ReplaceEntry(date, slot, position, recipeId)` — 1 品だけ入れ替え
  - `RemoveEntry(date, slot, position)` — 1 品削除 (残り品の position を詰め直す)
  - `ClearSlot(date, slot)` — slot 全体を削除
  - `ReorderSlot(date, slot, orderedRecipeIds)` — slot 内の並びを再指定
  - `CopyLeftover(date, slot, days, position?, mode)` — 1 品または slot 全体を連続 N 日コピー (position 未指定なら slot 全体)
- 割当 UI: 空セル `＋` / 割当済セル右上 `＋` / 品行 `⋯` (入替 / 削除 / 詳細 / 連続 N 日) / slot 右下 `⋯` (全体クリア / 全体
  連続 N 日コピー / 並び替え) を実装する
- レシピ選択ダイアログ (S02 のミニ版) を `frontend/src/components/mealplan/recipe-picker-dialog.tsx` に新設する。名前部分
  一致検索、`RecipeService.List` の結果を再利用する
- 作り置きコピーの衝突ダイアログ (追加 / 上書き / キャンセル) を実装する。コピー先に既存 slot がある場合の分岐は
  `CopyLeftover` の `mode` (`append` / `overwrite`) で切り替える。プレビュー計算はフロントで先に行い、衝突があるときだけ
  ダイアログを開く
- slot 並び替えは MVP では上下ボタン式 (ドラッグは非対応)。1 slot 内 position を 1 個ずつスワップする
- i18n キーを `frontend/src/locales/{ja,en}.json` に追加 (`mealplan.picker.*` / `mealplan.cell.actions.*` / `mealplan.copy.*`)
- 表示用の「他 N 件」閾値は `config.MealPlanCellVisibleItems` を継続利用 (既存)。作り置きコピーの N 候補は
  `config.LeftoverCopyPresets` を利用 (既存)

### Non-goals

- **期間切替 / 前後ナビ**: 「1 週 / 2 週 / 4 週」トグルと「＜」「＞」は `add-mealplan-multi-week` で扱う
- **買い物リスト生成**: 「この期間の買い物リストを見る」CTA は `add-shopping-list` で扱う
- **候補提示 (MVP 後)**: 使い切り / 久しぶりに / 時短 / 週プリセットは対象外
- **ドラッグ&ドロップ並び替え**: 上下ボタンのみ。DnD ライブラリの追加は行わない
- **食材マスタとの連動**: 割当した瞬間に材料が引かれる等の副作用は無い (在庫連動は
  `add-mealplan-cook-consume` で扱う)
- **バリデーション以外の Undo/Redo**: 単一操作の取り消しは MVP スコープ外
- **タイムゾーン跨ぎ**: 日付は "YYYY-MM-DD" のローカル日付のまま。TZ 変換ロジックは足さない

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `mealplan`: 現行 spec は「週表示 (固定値)」までを規定している。本 proposal で以下を追加・改訂する:
  - **ADDED**: 割当 (追加 / 入替 / 削除 / slot 全クリア / 並び替え) の Requirement
  - **ADDED**: 作り置きコピー (1 品 / slot 全体、既存割当への衝突解決) の Requirement
  - **ADDED**: レシピ選択ダイアログの Requirement (名前検索、選択で dialog close、S02 本画面には遷移しない)
  - **MODIFIED**: 「週データ取得の境界」を「サービス実装は SQLite repo 呼び出しに差し替え、`RecipeName` は JOIN で
    解決する」に更新
  - **MODIFIED**: 「セル内の品表示」に「品行 `⋯` / セル右下 `⋯` のアクション要件」を追加

## Impact

- **新設ファイル**:
  - `internal/storage/sqlite/migrations/00003_meal_plans.sql` — `meal_plans` テーブル + index
  - `internal/storage/sqlite/mealplan_repo.go` — repo 実装 (List / Assign / Replace / Remove / ClearSlot / Reorder /
    CopyLeftover)
  - `internal/storage/sqlite/mealplan_repo_test.go` — `t.TempDir()` で実 SQLite に対する tests
  - `internal/domain/mealplan/copy.go` — 作り置きコピー先日付列挙 (N 日分の連続日を返す) と、衝突検出ヘルパ
  - `internal/domain/mealplan/copy_test.go` — table-driven test
  - `internal/domain/mealplan/validate.go` — date / slot / position の validate ヘルパと `ErrInvalidDate` /
    `ErrInvalidSlot` / `ErrInvalidPosition` / `ErrEntryNotFound` / `ErrReorderMismatch` / `ErrInvalidCopyDays` /
    `ErrInvalidCopyMode` の型付きエラー宣言
  - `internal/domain/mealplan/validate_test.go` — table-driven test
  - `frontend/src/components/mealplan/recipe-picker-dialog.tsx` — S02 のミニ版レシピ選択ダイアログ
  - `frontend/src/components/mealplan/cell-item-menu.tsx` — 品行 `⋯` メニュー
  - `frontend/src/components/mealplan/slot-menu.tsx` — slot 右下 `⋯` メニュー
  - `frontend/src/components/mealplan/copy-conflict-dialog.tsx` — 作り置きコピーの衝突解決ダイアログ
- **更新ファイル**:
  - `internal/service/mealplan.go` — `ListWeek` を repo 呼び出しに差し替え。割当系メソッド追加
  - `internal/service/mealplan_test.go` — 既存の固定値 tests を repo ベースに書き換え、割当系の BDD 風テストを追加
  - `internal/domain/mealplan/entry.go` — `Entry` に `RecipeID string` を追加。既存 `RecipeName` はそのまま JOIN 表示用
  - `main.go` — `NewMealPlanService()` を `NewMealPlanService(db)` にし、`recipes` と同じく `*sql.DB` を注入
  - `frontend/src/routes/meal-plan.tsx` — `useMutation` で割当系サービス呼び出しをラップし、成功時に `mealplan.week`
    query を invalidate する
  - `frontend/src/components/mealplan/week-calendar.tsx` — セル `＋` ボタン、品行 `⋯`、slot 右下 `⋯` の hookup
  - `frontend/src/components/mealplan/meal-plan-cell.tsx` — 空セル / 割当済セル / メニュー配置
  - `frontend/src/locales/ja.json` / `frontend/src/locales/en.json` — `mealplan.picker.*` / `mealplan.cell.actions.*` /
    `mealplan.copy.*` キー群
- **生成物 (再生成のみ)**: `frontend/bindings/` を `wails3 generate bindings` で更新
- **migration**: `00003_meal_plans.sql` を追加。`meal_plans(date TEXT, slot TEXT, position INTEGER, recipe_id TEXT
  REFERENCES recipes(id))` + `PRIMARY KEY(date, slot, position)` + `INDEX (date)`
- **依存追加なし**: 新規 npm / Go モジュールは無し。shadcn の Dialog / DropdownMenu / Command は必要に応じて手動コピー

### 後続 proposal への引き継ぎ

- **期間切替**: `add-mealplan-multi-week` が `ListWeek` を `ListRange(startDate, weeks)` に拡張する想定。本 proposal では
  1 週固定のまま
- **買い物リスト**: `add-shopping-list` が `meal_plans` を材料展開して集計する。本 proposal は割当までで止める
- **RecipeName 表示の JOIN**: 表示用に `recipes.name` を JOIN するが、レシピ名変更後の反映タイミングは
  「次に S01 を開いたとき」までで十分。リアルタイム反映は不要
- **並び替え UX**: 上下ボタン方式で MVP を通す。DnD が必要なら別 proposal で `dnd-kit` 追加を検討
