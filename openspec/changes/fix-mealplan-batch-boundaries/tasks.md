## 1. 作り置きの行数を全行で数える

- [x] 1.1 `internal/storage/sqlite/mealplan_repo_test.go` の `TestMealPlanRepo_Batch` に、`ListPeriod` の作り置きの行数のテストを足す (更新)。
  足した時点で 1 件目と 2 件目が落ちることを `go test ./internal/storage/sqlite/ -run TestMealPlanRepo_Batch` で確かめる
  - 日曜の夕食から翌週月曜の夕食へ広げた作り置きを、日曜を含む 7 日で取得すると、日曜の行の `BatchMemberCount` が 2 になる
  - 土曜、日曜、翌週月曜に広げた作り置きを、土曜と日曜を含む 7 日で取得すると、2 行とも `BatchMemberCount` が 3 になる
  - 期間の中だけに 2 行ある作り置きは、従来どおり `BatchMemberCount` が 2 になる
  - 単独の行は、同じ日に別の作り置きの行があっても `BatchMemberCount` が 1 になる
- [x] 1.2 `internal/storage/sqlite/mealplan_repo.go` の `ListPeriod` を更新し、`COUNT(*) OVER (PARTITION BY m.batch_id)` を
  `(SELECT COUNT(*) FROM meal_plans a WHERE a.batch_id = m.batch_id)` に置き換える。1.1 のテストが通ることで確かめる
- [x] 1.3 `internal/storage/sqlite/mealplan_repo.go` の `ListPeriod` のコメントを、「期間外の行も数える」理由 (S07 と同じ数え方) に書き直す。
  `TestMealPlanRepo_Batch` の既存のテスト名 `ListPeriod populates batch_member_count via window function` を
  `ListPeriod populates batch_member_count from every row of the batch` に改める (`mealplan_repo_test.go` 更新)。
  `go test ./internal/storage/sqlite/` が通ることで確かめる

## 2. 「作り置きをやめる」で残す行を時間帯の順で選ぶ

- [x] 2.1 `internal/storage/sqlite/mealplan_repo_test.go` の `TestMealPlanRepo_Batch` に、`SplitBatch` の残す行のテストを足す (更新)。
  足した時点で 1 件目が落ちることを `go test ./internal/storage/sqlite/ -run TestMealPlanRepo_Batch` で確かめる
  - 月曜の昼食に B、A の順で入れ、A を月曜の夕食へ広げてから昼食の A でやめると、昼食の A が残り夕食の A が消える
  - 同じ作り置きで夕食の A を選んでやめても、残るのは昼食の A である
  - 月曜の夕食と火曜の朝食にまたがる作り置きは、月曜の夕食が残る (日付が時間帯より優先される)
  - 昼食の B は削除されず、position 0 のまま残る
- [x] 2.2 `internal/storage/sqlite/mealplan_repo.go` の `SplitBatch` を更新し、残す行を選ぶクエリの並びを
  `ORDER BY date, CASE meal WHEN 'breakfast' THEN 0 WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END, position` にする。
  関数のコメントの「起点日 (… MIN(position) を tiebreaker にした行)」も同じ規則に直す。2.1 のテストが通ることで確かめる

## 3. E2E

- [x] 3.1 `frontend/e2e/mealplan-batch.spec.ts` に describe「表示期間をまたぐ作り置き」を足す (更新)。
  翌週の土曜、日曜と、その次の月曜を `nextWeekDay(5)`、`nextWeekDay(6)`、`nextWeekDay(7)` で作り、`assignRecipe` と `spreadEntry` で用意する。
  `task test:e2e` で通ることを確かめる
  - 期間の外に続く作り置きの行にも枠色が付く (品行の `class` に枠色が入り、`border-transparent` でない)
  - 期間の外に続く作り置きの行はドラッグできない (品行の `draggable` が `false`)
  - 期間の外に続く作り置きの行にも作り置きをやめる項目が出る
  - 作り置きをやめる確認には期間の外で削除される行も数える
- [x] 3.2 `frontend/e2e/mealplan-batch.spec.ts` に describe「作り置きをやめたときに残る行」を足す (更新)。`task test:e2e` で通ることを確かめる
  - 同じ日の昼食と夕食にまたがる作り置きをやめると昼食の行が残る

## 4. ドキュメント

- [x] 4.1 `docs/product.md` の §4 S01「作り置きをやめる」の箇条に、「最初の日に複数の時間帯の行があれば、朝食、昼食、夕食の順で早い行を残す」を書き足す。
  `git diff docs/product.md` で 1 か所だけの変更であることを確かめる
- [x] 4.2 `docs/reviews/2026-10-15-design-ux-review.md` の R36 と R38 にチェックを入れ、proposal 名 `fix-mealplan-batch-boundaries` を添える。目視で確かめる

## 5. 確認

- [x] 5.1 `go test ./internal/... ./cmd/...` を実行し、すべて通ることを確かめる
- [x] 5.2 `cd frontend && npm run type-check` を実行し、エラーが無いことを確かめる
- [x] 5.3 `cd frontend && npm run build:dev` を実行し、ビルドが通ることを確かめる
- [x] 5.4 `task test:e2e` を実行し、すべて通ることを確かめる (`check:scenarios` を含む)
- [x] 5.5 `task lint` を実行し、エラーが無いことを確かめる
