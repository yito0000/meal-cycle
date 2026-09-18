## Why

現在の `add-mealplan-batch-linking` は「作り置き = 同一時間帯内」で閉じている。しかし現実には
「前日の夕食に作った肉じゃがを、翌日の昼食に食べる」というユースケースが頻繁にある。作り置き 1 バッチ
を **複数時間帯に跨って共有** できないと、買い物リスト計算 (1 バッチ = 1 セット計上) と現実の食生活が
乖離する。

具体例:
- 月曜 夕食: 肉じゃが (作る) → 1 セット材料計上
- 火曜 昼食: 肉じゃが (前日の残り) → 現状は別バッチ扱い → もう 1 セット計上される
- 火曜 夕食: 肉じゃが (前日の残り) → 現状も同時間帯内なら同一バッチ → 1 セットのまま

昼食に持ち越したいユーザーが「材料 2 セット」と表示されると、実際より多く買ってしまう原因になる。

## What Changes

- **`MergeBatch` の 3 条件から「同時間帯」を外す**。新条件: 同一レシピ / 起点日から ±max(`LeftoverCopyPresets`)
  日以内 の 2 条件で許容 (別時間帯間でもまとめられる)
- **`ListMealBatches` の候補フィルタから同時間帯制約を外し**、名前を `ListMergeCandidates` に改名
  (「Meal」名を含むメソッド名が実態と合わなくなるため)
- **新 UI「別の時間帯にも作り置きを追加」**: 品行 `⋯` に新項目を追加し、選んだ (別時間帯, 対象日) に
  同一 batch_id + 同一 recipeID の行を追加する。CopyLeftover が「同時間帯 × N 日」を扱うのに対し、
  こちらは「他時間帯 × 1 日」を扱う 対称的な操作
- **`CopyToOtherMeal(ctx, sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal) error`** を
  新規追加 (repo + service)
- **既存 CopyLeftover は変更なし**: 「同時間帯内で連続 N 日」の意味を維持 (作り置きコピーの主軸)
- **UI 上の候補ダイアログ (`batch-merge-dialog.tsx`)** は「時間帯名」列を追加して、別時間帯候補も
  「月曜・昼食」のように識別できるようにする
- **視覚識別 (枠色 + 🔗)** は別時間帯跨ぎでもそのまま機能する (batch_id 単位で決まるため既存ロジックで OK)

## Non-goals

- **CopyLeftover の時間帯跨ぎ対応**: 「連続 N 日を昼食と夕食の両方に」等の複雑な UX は今スコープ外。
  「同一時間帯内で連続 N 日」+「単発で別時間帯に足す」の組合わせで賄う
- **ドラッグ&ドロップでの時間帯間移動**: 別 proposal
- **朝食 / 昼食 / 夕食以外の時間帯追加** (おやつ、夜食): 別 proposal

## Capabilities

### Modified Capabilities

- `mealplan`:
  - **MODIFIED**: 「事後編集による作り置きの外し / まとめ」Requirement — 「他の作り置きにまとめる」の
    3 条件から「同時間帯」を外し、2 条件 (同一レシピ / ±4 日以内) に緩和。別時間帯候補も提示・許容する
  - **ADDED**: 「作り置きを別時間帯にも追加」Requirement — 品行 `⋯` から別時間帯の同日 or 隣接日に
    同一 batch_id + 同一 recipe の行を追加する操作を提供する

## Impact

- **新設ファイル**:
  - `frontend/src/components/mealplan/copy-to-meal-dialog.tsx` — 「別の時間帯にも作り置きを追加」の
    ダイアログ (対象日 + 対象時間帯を選ぶ)
- **更新ファイル**:
  - `internal/storage/sqlite/mealplan_repo.go` — `MergeBatch` の防御チェックから「同時間帯」を外す。
    `ListMealBatches` → `ListMergeCandidates(ctx, date, meal, recipeID)` に改名 (呼び出し元の meal は
    表示情報用に受け取るが、フィルタからは削る)。`CopyToOtherMeal(ctx, sourceDate, sourceMeal,
    sourcePosition, targetDate, targetMeal) error` を新規追加 (対象行の batch_id と recipe_id を
    引き継いで新行を作る)
  - `internal/service/mealplan.go` — 同上の変更を反映。`ListMergeCandidates` は同時間帯制約なく候補を
    返す。`CopyToOtherMeal` を新規サービスメソッドとして追加
  - `internal/service/mealplan_test.go` / `internal/storage/sqlite/mealplan_repo_test.go` — 別時間帯跨ぎの
    merge / CopyToOtherMeal のケースを追加
  - `internal/domain/mealplan/validate.go` — 変更なし (既存 `ErrBatchMergeDenied` を使い回す)
  - `frontend/src/components/mealplan/batch-merge-dialog.tsx` — 候補行の表示に「時間帯」列 (夕食 / 朝食
    等) を追加
  - `frontend/src/components/mealplan/cell-item-menu.tsx` — 「別の時間帯にも作り置きを追加」メニュー項目を
    追加 (`onCopyToMeal`)
  - `frontend/src/components/mealplan/week-calendar.tsx` — `openMergeDialog` は `recipeID` だけ渡す (meal
    フィルタが外れたため引数簡略化)。`handleEntryAction` に `copy-to-meal` 経路を追加
  - `frontend/e2e/bindings-shim/mealplanservice.ts` — `ListMealBatches` → `ListMergeCandidates`、
    `CopyToOtherMeal` を追加
  - `frontend/src/locales/{ja,en}.json` — `item_menu.copy_to_meal` / `copy_to_meal_dialog.*` を追加
  - `docs/scenarios/mealplan.feature` — 「別時間帯への作り置き追加」「別時間帯間のまとめ」シナリオを追加
  - `docs/product.md` §S01 — 説明を更新
- **生成物 (再生成のみ)**: `wails3 generate bindings`
- **migration**: なし (batch_id / recipe_id で全部済むため既存スキーマで対応)

### 後続 proposal への引き継ぎ

- **`add-shopping-list`**: 「distinct batch_id 単位で 1 セット計上」ロジックは 時間帯に無関係になるので、
  実装がむしろシンプルになる (meal をキーに含めなくてよい)
- **`add-mealplan-cook-consume`**: 「作った」マークは batch_id 単位。時間帯跨ぎでも 1 バッチ = 1 回の
  調理として扱う (在庫連動時に「別時間帯に持ち越した分」を二重計上しない)
