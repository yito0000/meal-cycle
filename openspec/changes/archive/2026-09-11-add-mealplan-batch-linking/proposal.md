## Why

`add-mealplan-assign-recipe` で「連続 N 日コピー」は入ったが、コピーで作られた行と単独割当された行が DB 上でまったく
同じ形になっており、「作り置き (1 バッチを N 日で食べ回す)」なのか「別調理 (N バッチぶん材料が必要)」なのかを区別
できない。ペルソナは作り置き主体だが、実際にはどちらもあり得る。この意味付けを決めないまま
`add-shopping-list` に進むと、材料が常に「出現回数分」に計上される (= 現実と乖離した買い物リスト) 実装で
固まってしまう。買い物リストを実装する前にデータモデルを確定させる必要がある。

## What Changes

- `meal_plans` に `batch_id TEXT NOT NULL` を追加する (migration `00004_meal_plans_batch.sql`)。同じ batch_id を
  持つ行は「1 バッチを共有」= 材料は 1 セットだけ計上する
- 単独割当 (`AssignRecipe`) は行ごとに新しい UUID を発行する (=「別調理 1 品」の意味になる)
- `CopyLeftover` に `share_batch bool` パラメータを追加する。true なら N 行に同一 batch_id を割り振る (= 作り置き)、
  false なら 1 行ごとに新 UUID (= 別調理)。既存の起点行は変更しない (append) か、あるいはコピー先ぶんだけ新 batch_id
  を作る (overwrite) — 挙動は design で決める
- フロントの「連続 N 日」ダイアログ (品行 `⋯` / slot 右下 `⋯` 両方) に「作り置き (1 バッチ共有) / 別調理 (N バッチ)」
  の 2 択を追加する。デフォルトは「作り置き」(ペルソナと一致)
- 既存衝突ダイアログ (「追加 / 上書き / キャンセル」) と組み合わせて、`share_batch` は共通に選ぶ (「上書き」でも
  「追加」でも、コピー元と同じ batch_id を共有するかは同じ意味)
- 事後編集 UI: 品行 `⋯` に「作り置きから外す / 他の作り置きにまとめる」を追加する。すでに割当された行の
  batch_id を後から変更できる
  - 「作り置きから外す」= 対象行が属する作り置きに対して、**起点日 (バッチ内で最も古い日付の行) を残し、
    それ以外の行を全削除**、残った起点行に新 batch_id を発行して単独作り置きにする。連続 N 日コピーの
    逆演算として動作する (バッチが 1 行しか含まない場合は no-op)
  - 「他の作り置きにまとめる」= 同じ slot 内で **(1) 同一レシピ / (2) 起点日から
    ±max(`LeftoverCopyPresets`) 日以内 / (3) 別バッチ** の 3 条件を満たす候補にのみまとめられる。
    別レシピや遠い過去の作り置きへのまとめは不可 (「1 回の調理」の意味と整合しないため)
  - メニュー項目の表示条件: 「作り置きから外す」は複数行の作り置きに属する行だけに出す。
    「この品を連続 N 日」は単独作り置きの行だけに出す (既に作り置きの行では更に連続コピーする意味が無い)
- UI 表示: 同じ batch_id の行に共通の枠色 (2〜3 色ローテーション) と 🔗 アイコンを付ける。単独 (batch_id が
  1 行だけ) の行は色を付けない
- 色ローテーション定数は `frontend/src/components/mealplan/mealplan-config.ts` に置く (call site にリテラルを書かない
  プロジェクト規約と一致)

### Non-goals

- **買い物リストの実装**: 材料計上のロジック本体は `add-shopping-list` で扱う。本 proposal は「batch_id という区別が
  データとして存在する」ところまでで止める
- **バッチのメタ情報**: 「バッチの作成日 (実際に調理した日)」「賞味期限」等の追加カラムは扱わない (先に色分け UX と
  データ意味付けを固定する)
- **在庫連動**: 「作った」マークで在庫を減らす連動は `add-mealplan-cook-consume` で扱う
- **ドラッグ&ドロップでのバッチ変更**: 事後編集は `⋯` メニュー経由のみ。DnD は後回し
- **ダイアログを毎回スキップする「常にこの選択」設定**: S09 設定画面で対応する

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `mealplan`: 現行 spec は「連続 N 日コピー」を「行を N 個生成する」までしか規定していない。本 proposal で以下を
  追加・改訂する:
  - **ADDED**: バッチ共有の Requirement (batch_id の同値 = 1 バッチ、材料は 1 セット)
  - **ADDED**: 連続 N 日コピー時のバッチ選択 UI の Requirement (「作り置き」/「別調理」の 2 択、デフォルト
    「作り置き」)
  - **ADDED**: 事後編集の Requirement (バッチを分ける / 合流する)
  - **ADDED**: バッチの視覚識別 Requirement (同 batch の共通枠色 + 🔗 アイコン)
  - **MODIFIED**: 「作り置きコピー (1 品単位)」「作り置きコピー (slot 全体)」「作り置きコピー先の衝突解決」の 3
    Requirement は、batch_id パラメータが加わる形で振る舞いが拡張される

## Impact

- **新設ファイル**:
  - `internal/storage/sqlite/migrations/00004_meal_plans_batch.sql` — `meal_plans` に `batch_id TEXT NOT NULL` を追加
    し、既存行にはそれぞれ独自の UUID を発行する (下位互換: 既存の連続コピー結果は「別調理」扱いに寄せる。
    データ量は極小なので影響なし)
  - `internal/domain/mealplan/batch.go` — `NewBatchID()` (UUID v7)、`ShareBatch bool` を扱う型付きヘルパ
  - `internal/domain/mealplan/batch_test.go` — table-driven test
  - `frontend/src/components/mealplan/copy-mode-dialog.tsx` — 「作り置き / 別調理」の 2 択ダイアログ (連続コピー
    実行前)
  - `frontend/src/components/mealplan/batch-color.ts` — batch_id → 表示色 (Tailwind class) の決定的 hash
  - `frontend/src/components/mealplan/batch-edit-menu.tsx` — 品行 `⋯` の「バッチを分ける / 合流する」項目 (既存
    メニューを拡張)
- **更新ファイル**:
  - `internal/storage/sqlite/mealplan_repo.go` — `Assign` は new UUID、`CopyLeftover` は `shareBatch` に応じて
    batch_id を発行、`SplitBatch(ctx, date, slot, position)` (起点日以外を削除して新 UUID を返す実装) /
    `MergeBatch(ctx, date, slot, position, targetBatchID)` (3 条件チェックあり) /
    `ListSlotBatches(ctx, date, slot, recipeID)` を追加 (シグネチャに `recipeID` を含めて候補フィルタ)
  - `internal/service/mealplan.go` — `CopyLeftover(..., shareBatch bool)` にシグネチャ拡張、`SplitBatch` /
    `MergeBatch` / `ListSlotBatches(..., recipeID)` を追加
  - `internal/domain/mealplan/validate.go` — 新 sentinel `ErrBatchMergeDenied` を追加 (Merge の 3 条件
    違反時に返す)
  - `internal/service/mealplan_test.go` — 追加ケース (バッチ共有 / 分割 / 合流)
  - `internal/storage/sqlite/mealplan_repo_test.go` — 追加ケース
  - `internal/domain/mealplan/entry.go` — `Entry` に `BatchID string` を追加
  - `frontend/src/components/mealplan/meal-plan-cell.tsx` — batch_id をもとに枠色 + 🔗 アイコンを描画
  - `frontend/src/components/mealplan/week-calendar.tsx` — CopyLeftover 呼び出しに `shareBatch` を渡す、
    copy-mode-dialog を組み込む
  - `frontend/src/components/mealplan/cell-item-menu.tsx` — 「バッチを分ける / 合流する」項目を追加
  - `frontend/src/components/mealplan/mealplan-config.ts` — `BATCH_COLORS` 定数を追加
  - `frontend/src/locales/{ja,en}.json` — `mealplan.copy.mode.*` / `mealplan.item_menu.split_batch` /
    `mealplan.item_menu.merge_batch` / `mealplan.batch.legend.*` 追加
  - `docs/scenarios/mealplan.feature` — バッチ選択・分割・合流の Gherkin シナリオを追記
  - `docs/product.md` — S01 セル内の「同 batch 枠色 + 🔗 アイコン」を挙動として追記 (ワイヤーは v2 として更新するか
    別 issue にする)
- **生成物 (再生成のみ)**: `wails3 generate bindings` で `MealPlanService` の `CopyLeftover` シグネチャ変更、
  `SplitBatch` / `MergeBatch` 追加を反映
- **migration**: `00004_meal_plans_batch.sql`。既存行への UUID 割り振りは goose の Up ステートメントで
  `UPDATE meal_plans SET batch_id = lower(hex(randomblob(16)))` 相当を実行 (実装は design.md で確定)

### 後続 proposal への引き継ぎ

- **add-shopping-list**: 材料計上は「distinct batch_id 単位で 1 セット」で計算する。本 proposal で batch_id を
  Entry に載せることで、S07 実装時に MealPlan だけ見れば計上ロジックが完結する
- **add-mealplan-cook-consume**: 「作った」マークは batch_id 単位で扱う (1 バッチ = 1 回の調理)。本 proposal の
  batch_id をそのまま参照する
- **設定画面 S09**: 「連続 N 日のデフォルトを『作り置き』か『別調理』か」を設定できるようにする。本 proposal は
  MVP 期のデフォルトを「作り置き」で固定
