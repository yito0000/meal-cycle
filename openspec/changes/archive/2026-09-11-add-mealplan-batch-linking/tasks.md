## 1. ドメイン層 (internal/domain/mealplan)

- [x] 1.1 `internal/domain/mealplan/entry.go` を更新し、`Entry` struct に `BatchID string` と
      `BatchMemberCount int` を追加する (完了判定: `go build ./internal/...` が通る)
- [x] 1.2 `internal/domain/mealplan/batch.go` を新設し、`NewBatchID() string` (UUID v7) と `ShareBatch bool` を
      扱う型付きヘルパを実装する (完了判定: `go build ./internal/...` が通る)
- [x] 1.3 `internal/domain/mealplan/batch_test.go` を新設し、以下の table-driven テストを追加する
    - NewBatchID は毎回異なる値を返す
    - NewBatchID の返す値は 32 文字以上の hex 文字列 (UUID v7 相当)
    - (完了判定: `go test ./internal/domain/mealplan/...` が pass する)
- [x] 1.4 `internal/domain/mealplan/copy.go` の `PlanCopy` を更新し、`shareBatch bool` パラメータを追加する。
      shareBatch=true なら返り値の Entry には共通の batch_id を割り振り、false なら 1 行ごとに新 UUID を割り振る
      (完了判定: 下記 1.5 のテストが pass する)
- [x] 1.5 `internal/domain/mealplan/copy_test.go` に PlanCopy の shareBatch 分岐テストを追加する
    - shareBatch=true なら返り値の全 Entry が同一 BatchID
    - shareBatch=false なら返り値の各 Entry が互いに異なる BatchID
    - shareBatch=true と shareBatch=false で BatchID 集合が交わらない
    - (完了判定: `go test ./internal/domain/mealplan/...` が pass する)

## 2. SQLite migration + repository

- [x] 2.1 `internal/storage/sqlite/migrations/00004_meal_plans_batch.sql` を新設し、goose Up で `ALTER TABLE
      meal_plans ADD COLUMN batch_id TEXT` を実行する (完了判定: 下記 2.2 の migration 適用が成功する)
- [x] 2.2 同 migration の Up で既存行に UUID 相当を発行する `UPDATE meal_plans SET batch_id =
      lower(hex(randomblob(16))) WHERE batch_id IS NULL OR batch_id = ''` を追加する (完了判定: 下記 2.3 のテストで
      既存行に batch_id が入る)
- [x] 2.3 同 migration の Up に `CHECK (length(batch_id) > 0)` 制約と
      `CREATE INDEX idx_meal_plans_batch_id ON meal_plans(batch_id)` を追加する。goose Down で index と column を
      drop する (完了判定: `task dev` 起動時に goose migrate が成功する、下記 2.10 のテストが pass する)
- [x] 2.4 `internal/storage/sqlite/mealplan_repo.go` の `ListWeek` を更新し、SELECT に `batch_id` と
      `COUNT(*) OVER (PARTITION BY batch_id) AS batch_member_count` を追加する。返り値 Entry に載せる
      (完了判定: 下記 2.10 のテストが pass する)
- [x] 2.5 `MealPlanRepo.Assign` を更新し、`NewBatchID()` で発行した新 batch_id を挿入する (完了判定: 下記 2.10 の
      テストが pass する)
- [x] 2.6 `MealPlanRepo.CopyLeftover` を更新し、`shareBatch bool` パラメータを受け取る。shareBatch=true なら 1 レシピ
      につき 1 batch_id、shareBatch=false なら 1 行につき新 batch_id を割り振る (完了判定: 下記 2.10 のテストが
      pass する)
- [x] 2.7 `MealPlanRepo.SplitBatch(ctx, date, slot string, position int) (newBatchID string, err error)` を実装する
      (対象行に新 UUID を割り振り、返す。完了判定: 下記 2.10 のテストが pass する)
- [x] 2.8 `MealPlanRepo.MergeBatch(ctx, date, slot string, position int, targetBatchID string) error` を実装する
      (対象行の batch_id を targetBatchID に付け替える。targetBatchID が同 slot に存在しない場合は
      ErrBatchTargetNotFound を返す。完了判定: 下記 2.10 のテストが pass する)
- [x] 2.9 `MealPlanRepo.ListSlotBatches(ctx, date, slot string) ([]BatchSummary, error)` を追加する (合流候補
      ダイアログのため、対象 (date, slot) と同じ slot の他バッチ一覧を返す。完了判定: 下記 2.10 のテストが pass する)
- [x] 2.10 `internal/storage/sqlite/mealplan_repo_test.go` に以下のケースを追加する
    - 既存行 (migration 前) にも batch_id が発行され、それぞれユニーク
    - Assign 3 回で 3 個の異なる batch_id が発行される
    - CopyLeftover(shareBatch=true) すると、コピー元と N 日ぶんが同一 batch_id
    - CopyLeftover(shareBatch=false) すると、N 日ぶんの batch_id が互いに異なる
    - SplitBatch は対象 1 行だけ新 batch_id に、残りは元のまま
    - MergeBatch は対象行を指定 batch_id に付け替える
    - MergeBatch で targetBatchID が同 slot に存在しないと ErrBatchTargetNotFound
    - ListSlotBatches は同じ slot の他バッチだけを返す
    - ListWeek 応答の batch_member_count が正しい (2 行バッチは 2、1 行バッチは 1)
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)

## 3. Wails サービス (internal/service/mealplan.go)

- [x] 3.1 `MealPlanService.CopyLeftover` のシグネチャに `shareBatch bool` を追加し、repo 呼び出しにそのまま渡す
      (完了判定: `go build ./` が通る)
- [x] 3.2 `MealPlanService.SplitBatch(ctx, date, slot string, position int) (string, error)` を追加する
      (完了判定: 下記 3.5 のテストが pass する)
- [x] 3.3 `MealPlanService.MergeBatch(ctx, date, slot string, position int, targetBatchID string) error` を追加する
      (完了判定: 下記 3.5 のテストが pass する)
- [x] 3.4 `MealPlanService.ListSlotBatches(ctx, date, slot string) ([]mealplan.BatchSummary, error)` を追加する
      (完了判定: 下記 3.5 のテストが pass する)
- [x] 3.5 `internal/service/mealplan_test.go` を更新し、以下のケースを fakeRepo 経由で追加する
    - CopyLeftover(shareBatch=true) は全コピー行が同一 BatchID
    - CopyLeftover(shareBatch=false) は各コピー行が異なる BatchID
    - SplitBatch は対象行に新 UUID を返す
    - MergeBatch は成功時 nil、targetBatchID が存在しない場合は ErrBatchTargetNotFound
    - ListSlotBatches は同 slot の他バッチだけを返す
    - CopyLeftover のバリデーション既存ケース (invalid date / slot / days) は shareBatch にかかわらず維持される
    - (完了判定: `go test ./internal/service/...` が pass する)

## 4. Wails バインディング再生成

- [x] 4.1 `wails3 generate bindings` を実行し、`MealPlanService` に `SplitBatch` / `MergeBatch` /
      `ListSlotBatches` メソッドが生成され、`CopyLeftover` のシグネチャが `(date, slot, days, position, mode,
      shareBatch)` になっていることを確認する。`Entry` model に `batchId` / `batchMemberCount` が追加されている
      ことを確認する (完了判定: `frontend/bindings/` の差分に該当メソッドとフィールドが存在する)
- [x] 4.2 `frontend/e2e/bindings-shim/mealplanservice.ts` を更新し、`CopyLeftover` のシグネチャ拡張と 3 メソッド
      追加を反映する (完了判定: `npm run type-check` が通る)

## 5. i18n (frontend/src/locales)

- [x] 5.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に以下のキー群を追加する
    - `mealplan.copy.mode.title` / `mode.leftover` / `mode.distinct` / `mode.leftover_hint` / `mode.distinct_hint`
    - `mealplan.item_menu.split_batch` / `mealplan.item_menu.merge_batch`
    - `mealplan.batch.merge_dialog.title` / `merge_dialog.empty` / `merge_dialog.item_label`
    - `mealplan.batch.legend.leftover` (「同色 = 同じバッチ / 材料は 1 セット」)
    - `mealplan.errors.batch_target_not_found`
    - (完了判定: `npm run type-check` が通り、`npm run build:dev` が成功する)

## 6. フロントエンド: コピーモードダイアログ

- [x] 6.1 `frontend/src/components/mealplan/copy-mode-dialog.tsx` を新設し、「作り置き (1 バッチ共有) / 別調理
      (N バッチ)」を選ぶ 2 択ダイアログを実装する (既定は「作り置き」。props: `open`, `onOpenChange`,
      `onSelect(shareBatch: boolean)`)。完了判定: `npm run type-check` が通る)
- [x] 6.2 `frontend/src/components/mealplan/mealplan-config.ts` に `BATCH_COLORS`
      (Tailwind class 配列、2〜3 色) を追加する (完了判定: `import { BATCH_COLORS } from "./mealplan-config"` が
      解決する)
- [x] 6.3 `frontend/src/components/mealplan/batch-color.ts` を新設し、`colorForBatch(batchID: string,
      memberCount: number): string | null` を実装する (memberCount <= 1 なら null、それ以外は決定的 hash で
      BATCH_COLORS から 1 個返す。完了判定: 下記 6.4 のテストが必要なら追加)
- [x] 6.4 (テストランナー未導入のため保留) `batch-color.ts` の単体テストは Vitest 導入 proposal 後に追加する。本
      proposal では E2E で振る舞いを担保する (完了判定: 判断が本文に明記されている)

## 7. フロントエンド: 分割 / 合流 UI

- [x] 7.1 `frontend/src/components/mealplan/cell-item-menu.tsx` を更新し、「バッチを分ける」「バッチに合流する」
      項目を追加する (合流はサブメニューで候補 batch を列挙する。完了判定: 目視でメニュー項目が出る)
- [x] 7.2 `frontend/src/components/mealplan/batch-merge-dialog.tsx` を新設し、同 slot の他バッチ (レシピ名 + 日付)
      を一覧するダイアログを実装する (props: `open`, `candidates: BatchSummary[]`, `onSelect(batchID)`)。完了判定:
      `npm run type-check` が通る)
- [x] 7.3 `frontend/src/components/mealplan/meal-plan-cell.tsx` を更新し、各行に batch_id / batch_member_count から
      決定した枠色と 🔗 アイコンを描画する (memberCount <= 1 の行は色なし)。完了判定: 目視で連続コピーした複数日に
      共通の枠色が出る)

## 8. フロントエンド: ページ配線 (week-calendar.tsx)

- [x] 8.1 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、連続 N 日コピー実行前に
      `copy-mode-dialog` を挟む (「作り置き / 別調理」→ 衝突チェック → `copy-conflict-dialog` (要る場合のみ)
      → CopyLeftover 呼び出し)。既定選択は「作り置き」。完了判定: `npm run type-check` が通り、目視で
      「連続 3 日」を選ぶとコピーモード選択が出る)
- [x] 8.2 `handleEntryAction` の `split-batch` / `merge-batch` 経路を追加し、SplitBatch / MergeBatch mutation と
      配線する (完了判定: 目視でメニューから分割・合流が実行できる)
- [x] 8.3 `mealplan.errors.batch_target_not_found` を `error-messages.ts` のマッピングに追加する (完了判定:
      `npm run type-check` が通る)

## 9. ドキュメント更新

- [x] 9.1 `docs/scenarios/mealplan.feature` に「作り置き / 別調理の選択」「バッチの分割 / 合流」のシナリオを
      追記する。テスト名は本 proposal の spec の Scenario 名と一字一句揃える (完了判定: E2E test 名と揃った
      1 対 1 対応が取れている)
- [x] 9.2 `docs/product.md` §4/S01 のセル内の描画説明に「同 batch の共通枠色 + 🔗 アイコン」を追記する
      (完了判定: 目視で該当セクションに追記されている)

## 10. E2E シナリオ (frontend/e2e)

- [x] 10.1 `frontend/e2e/mealplan-batch.spec.ts` (新設) に、feature の各シナリオ名で `test(...)` を追加する
    - 作り置きを選ぶと全日程が同じバッチに属する
    - 別調理を選ぶと各日が独立したバッチに属する
    - バッチを分けると独立した新バッチに移る
    - 隣接する同一レシピの行に合流する
    - 2 バッチが同時に存在すると別の色になる
    - (完了判定: `task test:e2e` が pass する。DropdownMenu Portal の操作が難しいシナリオは `test.skip` で
      名前だけ残す)

## 11. lint / format / 最終テスト

- [x] 11.1 `task lint` を実行し、Go / Frontend の両方でエラーがないことを確認する (完了判定: exit code 0)
- [x] 11.2 `task format` を実行し、差分が意図どおりであることを確認する (完了判定: `git diff` が整形差分のみ)
- [x] 11.3 `go test ./internal/... ./` を実行し、全パッケージの単体テストが pass することを確認する
      (完了判定: exit code 0)
- [x] 11.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev` を実行し、フロントの
      型チェックとビルドが通ることを確認する (完了判定: exit code 0)
- [x] 11.5 `task test:e2e` を実行し、E2E シナリオが pass することを確認する (完了判定: exit code 0)
- [x] 11.6 `task dev` で起動し、S01 で「連続 3 日 → 作り置き」と「連続 3 日 → 別調理」の切替、共通枠色の描画、
      🔗 アイコンの表示、`⋯ → バッチを分ける / 合流する` の動作を目視で確認する (完了判定: 各操作が UI に反映
      され、再起動後も残る) — **WSL2 + Windows (PowerShell) 両方で目視確認済み**

## 12. Split の semantics 変更 (D4a: 起点日を残して他を削除)

- [x] 12.1 `internal/storage/sqlite/mealplan_repo.go` の `SplitBatch` 実装を差し替える。対象行が属する
      batch_id 全体に対して、`SELECT MIN(date), MIN(position)` で起点行を特定し、それ以外の同 batch_id 行を
      `DELETE`、起点行に新 UUID を `UPDATE` で付け替える。バッチが 1 行だけの場合は現行 batch_id をそのまま
      返して no-op。完了判定: 下記 12.3 のテストが pass する
- [x] 12.2 `internal/service/mealplan.go` の `SplitBatch` は挙動変更のみ (シグネチャ据え置き)。呼び出しを
      repo にそのまま委譲する。完了判定: `go build ./` が通る
- [x] 12.3 `internal/storage/sqlite/mealplan_repo_test.go` に Split の新挙動テストを追加する
    - 3 行バッチに対して真ん中の行で SplitBatch を呼ぶと、起点日 (最古) の 1 行だけが残り新 UUID を持つ
    - 3 行バッチに対して起点行を指す position で SplitBatch を呼んでも結果は同じ (起点 1 行だけ残る)
    - 単独バッチ (1 行) に対して SplitBatch を呼ぶと、行数もその batch_id も変わらない (no-op)
    - Split 前後で「他のバッチ」の行数は 1 件も変わらない
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)
- [x] 12.4 `internal/service/mealplan_test.go` の fakeRepo `SplitBatch` を新挙動 (起点日以外を削除) に
      合わせて書き直す。既存の Split テストは新挙動基準で assert を書き換える。完了判定:
      `go test ./internal/service/...` が pass する

## 13. Merge の 3 条件制約 (D4b)

- [x] 13.1 `internal/domain/mealplan/validate.go` に新 sentinel `ErrBatchMergeDenied` を追加する
      (メッセージ: `mealplan: batch merge denied`)。完了判定: `go build ./internal/...` が通る
- [x] 13.2 `internal/storage/sqlite/mealplan_repo.go` の `ListSlotBatches` シグネチャに `recipeID string` を
      追加し、SQL の WHERE 節に「同一 recipe_id」「起点行の日付 ±max(`LeftoverCopyPresets`) 日以内」を
      追加する。config 参照経路は repo 生成時に `cfg config.Config` を保持する形にする (call site に
      リテラルを書かない規約)。完了判定: 下記 13.5 のテストが pass する
- [x] 13.3 `internal/storage/sqlite/mealplan_repo.go` の `MergeBatch` に 3 条件の防御チェックを追加する
      (targetBatchID の他行から少なくとも 1 つが 3 条件を満たすか)。違反時は `ErrBatchMergeDenied` を返す。
      完了判定: 下記 13.5 のテストが pass する
- [x] 13.4 `internal/service/mealplan.go` の `ListSlotBatches` シグネチャに `recipeID string` を追加し、
      入口で `mealplan.ValidateDate` / `ValidateSlot` に加えて `recipeID == ""` を弾く。`MergeBatch` は
      挙動変更のみ (シグネチャ据え置き) で repo に委譲。完了判定: `go build ./` が通る
- [x] 13.5 `internal/storage/sqlite/mealplan_repo_test.go` に以下のケースを追加する
    - ListSlotBatches は同一レシピの候補だけを返す (別レシピの batch は除外)
    - ListSlotBatches は 対象行から ±4 日以内 の候補だけを返す (5 日以上離れた batch は除外)
    - ListSlotBatches は同 slot だけを返す (既存)
    - MergeBatch は別レシピの targetBatchID を指定すると ErrBatchMergeDenied を返す
    - MergeBatch は日付レンジ外の targetBatchID を指定すると ErrBatchMergeDenied を返す
    - MergeBatch は 3 条件を満たす targetBatchID には成功する (「1 日飛ばし」は許容: 月火 batch と金の
      単独行がまとめられる)
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)
- [x] 13.6 `internal/service/mealplan_test.go` の fakeRepo `ListSlotBatches` / `MergeBatch` に同じ 3 条件
      フィルタと防御チェックを実装する。既存テストの callsite に recipeID を渡すよう修正、新条件テストを
      追加する。完了判定: `go test ./internal/service/...` が pass する

## 14. bindings 再生成 + フロント配線

- [x] 14.1 `wails3 generate bindings` を実行し、`ListSlotBatches` のシグネチャに `recipeID` が追加された
      ことを確認する。完了判定: `frontend/bindings/.../mealplanservice.ts` の該当関数に第 3 引数がある
- [x] 14.2 `frontend/e2e/bindings-shim/mealplanservice.ts` の `ListSlotBatches` シグネチャに `recipeID` を
      追加する。完了判定: `npm run type-check` が通る
- [x] 14.3 `frontend/src/components/mealplan/week-calendar.tsx` の `openMergeDialog` を、対象 `Entry` の
      `recipeId` を第 3 引数として `MealPlanService.ListSlotBatches` に渡すよう更新する。完了判定:
      `npm run type-check` が通り、目視で「他の作り置きにまとめる」が同レシピ候補だけ表示する

## 15. i18n・error mapping・feature・E2E 追記

- [x] 15.1 `frontend/src/locales/ja.json` / `en.json` に `mealplan.errors.batch_merge_denied` を追加する
      (ja: 「その作り置きにはまとめられません」/ en: `Cannot merge into that batch`)。完了判定:
      `npm run type-check` が通る
- [x] 15.2 `frontend/src/components/mealplan/error-messages.ts` に `mealplan: batch merge denied` →
      `pages.meal_plan.errors.batch_merge_denied` のマッピングを追加する。完了判定: `npm run type-check`
      が通る
- [x] 15.3 `docs/scenarios/mealplan.feature` の「作り置きから外す」「他の作り置きにまとめる」関連シナリオを
      spec 側の新シナリオと 1 対 1 で揃えて書き換える。追加するシナリオ名:
    - 作り置きから外すと起点日だけが残る
    - 単独作り置きの行に対して外す操作は何もしない
    - 同一レシピの近い日付の作り置きにまとめる
    - まとめ候補は同じ slot・同じレシピ・±4 日以内だけを提示する
    - レシピが違う作り置きにはまとめられない
    - 日付レンジ外の作り置きにはまとめられない
    - 1 日飛ばした同一レシピの行はまとめられる
    - (完了判定: 目視で spec と feature の Scenario 名が全部揃う)
- [x] 15.4 `frontend/e2e/mealplan-batch.spec.ts` のテスト名を feature の新シナリオ名に合わせて修正する。
      DropdownMenu Portal 依存のシナリオは引き続き `test.skip` で名前だけ残す。完了判定: `task test:e2e`
      で新テスト名が表示され、既存の passing 数が減らない

## 16. lint / format / テスト再ゲート

- [x] 16.1 `task lint` を実行し、Go / Frontend の両方でエラーが無いことを確認する。完了判定: exit code 0
- [x] 16.2 `task format` を実行し、`git diff` の差分が整形のみであることを確認する。完了判定: 差分が
      整形範囲に限られる
- [x] 16.3 `go test ./internal/... ./` を実行し、全パッケージの単体テストが pass することを確認する。
      完了判定: exit code 0
- [x] 16.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev` を実行し、
      フロントの型チェックとビルドが通ることを確認する。完了判定: exit code 0
- [x] 16.5 `task test:e2e` を実行し、E2E シナリオが引き続き pass することを確認する。完了判定: exit code 0

## 17. メニュー項目の表示条件 (単独作り置き / 複数作り置きに応じた出し分け)

- [x] 17.1 `frontend/src/components/mealplan/cell-item-menu.tsx` の Props に `batchMemberCount: number` を
      追加する。完了判定: `npm run type-check` が通る
- [x] 17.2 `cell-item-menu.tsx` の「作り置きから外す」項目を `batchMemberCount > 1` の行だけ表示するよう
      条件付きレンダリングする。完了判定: 目視で単独作り置きの行には項目が出ない
- [x] 17.3 `cell-item-menu.tsx` の「この品を連続 N 日」サブメニューを `batchMemberCount > 1` の行 (既に
      作り置きになっている) では非表示にする。完了判定: 目視で作り置きに属する行にはサブメニューが出ない
- [x] 17.4 `meal-plan-cell.tsx` から `entry.batchMemberCount` を `CellItemMenu` に渡す配線を追加する。
      完了判定: `npm run type-check` が通る
- [x] 17.5 `docs/scenarios/mealplan.feature` の「事後編集」節に以下 2 シナリオを追記する
    - 単独作り置きの行には「作り置きから外す」項目が出ない
    - 作り置きに属する行には「この品を連続 N 日」項目が出ない
    - (完了判定: feature と spec の Scenario 名が 1 対 1 で揃う)
- [x] 17.6 `task lint && go test ./internal/... ./ && npm --prefix frontend run type-check` を再実行し、
      全パッケージが引き続き pass することを確認する。完了判定: exit code 0
