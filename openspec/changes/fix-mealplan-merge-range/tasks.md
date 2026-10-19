## 1. 探索範囲の日数をサービスから渡す

- [ ] 1.1 `internal/service/mealplan.go` を更新し、`mealPlanRepo` interface の `ListMergeCandidates` と `MergeBatch` の末尾に引数 `dayRange int` を足す。
  `go build ./internal/...` がリポジトリ側の未対応で落ちることを確かめる
- [ ] 1.2 `internal/service/mealplan.go` に `mergeDayRange()` (`configSource` の `LeftoverCopyPresets` の最大値) を足す。
  `maxSpreadOffsetDays()` を `mergeDayRange() - 1` で求める形に書き換える。`go vet ./internal/service/` が通ることで確かめる
- [ ] 1.3 `internal/service/mealplan.go` の `ListMergeCandidates` と `MergeBatch` が `s.mergeDayRange()` を渡すように更新する。
  メソッドのコメントの「±LeftoverCopyPresets max 日以内」を、S09 の値から決めることがわかる書き方に直す。1.6 のテストで確かめる
- [ ] 1.4 `internal/storage/sqlite/mealplan_repo.go` を更新し、`ListMergeCandidates` と `MergeBatch` が引数の `dayRange` を使うようにする。
  `MealPlanRepo` の `cfg` フィールド、`mergeDayRange`、`config` の import を消し、`NewMealPlanRepo` は `db` だけを持たせる。
  `go build ./internal/... ./cmd/...` が通ることで確かめる
- [ ] 1.5 `internal/service/mealplan_test.go` を更新し、`fakeMealPlanRepo` の `ListMergeCandidates` と `MergeBatch` が引数の `dayRange` を使うようにする。
  4 を決め打ちしている `fakeMealPlanRepo.mergeDayRange` を消す。`go test ./internal/service/ -run TestMealPlanService` が通ることで確かめる
- [ ] 1.6 `internal/service/mealplan_test.go` に、作り置きの日数の候補を変えた設定のテストを足す (更新)。
  `staticConfig` に `LeftoverCopyPresets` を差し替えた `config.Config` を渡す。`go test ./internal/service/ -run TestMealPlanService_MergeRange` が通ることで確かめる
  - should list a batch 6 days away as a merge candidate when leftover presets are 2 and 7
  - should not list a batch 3 days away as a merge candidate when leftover presets are only 2
  - should merge into a batch 6 days away when leftover presets are 2 and 7
  - should deny merging into a batch 5 days away with ErrBatchMergeDenied when leftover presets are 2, 3 and 4
- [ ] 1.7 `internal/storage/sqlite/mealplan_repo_test.go` の `ListMergeCandidates` と `MergeBatch` の呼び出しすべてに、既定値と同じ日数 `4` を渡す (更新)。
  `go test ./internal/storage/sqlite/` が通ることで確かめる

## 2. 候補の代表行と件数を作り置きの全体から求める

- [ ] 2.1 `internal/storage/sqlite/mealplan_repo_test.go` の `TestMealPlanRepo_Batch` に、範囲の外に続く作り置きのテストを足す (更新)。
  足した時点で 1 件目と 2 件目が落ちることを `go test ./internal/storage/sqlite/ -run TestMealPlanRepo_Batch` で確かめる
  - 10/19 から 10/22 の夕食の 4 行の作り置きを、10/24 の行から日数 4 で探すと、候補の日付が 10/19、件数が 4 になる
  - 10/19 の昼食と 10/19 から 10/22 の夕食にまたがる作り置きを、10/24 の行から探すと、候補の時間帯が昼食になる
  - 範囲内に同じレシピの行が 1 行も無い作り置きは、別のレシピの行が範囲内にあっても候補に出ない
  - 時間帯全体のコピーで「肉じゃが」と「みそ汁」が共有する作り置きは、「肉じゃが」の行から見た件数に「みそ汁」の行を含めない
- [ ] 2.2 `internal/storage/sqlite/mealplan_repo.go` の `ListMergeCandidates` のクエリを、design.md D2 の形
  (`eligible` で作り置きを選び、同じレシピの全行に `COUNT(*) OVER` と `ROW_NUMBER() OVER` を掛ける) に書き直す。2.1 のテストが通ることで確かめる
- [ ] 2.3 `internal/storage/sqlite/mealplan_repo.go` の `ListMergeCandidates` のコメントを、範囲は判定だけに使い、代表行と件数は作り置きの全体から決めることに書き直す。
  `git diff` で目視して確かめる

## 3. ドキュメント

- [ ] 3.1 `docs/reviews/2026-10-19-design-ux-review.md` の R46 と R47 にチェックを入れ、proposal 名 `fix-mealplan-merge-range` を添える。目視で確かめる

## 4. E2E

- [ ] 4.1 `frontend/e2e/mealplan-batch.spec.ts` の describe「まとめ候補の一意性」に次の Scenario を足す (更新)。
  翌週の月曜から木曜の夕食の作り置きと、土曜 (既存の `SAT`) の夕食の単独の行を `assignRecipe` と `spreadEntry` で用意する。
  `task test:e2e` で通ることを確かめる
  - 探す範囲の外に続く作り置きは最初の行と全体の行数で候補に出る
- [ ] 4.2 settings の Scenario「作り置きの日数の候補を変えるとまとめる候補を探す範囲が変わる」は、1.6 のサービス層のテストで確かめ、E2E には足さない。
  tasks.md のこの行を確認済みにすることで完了とする

## 5. 確認

- [ ] 5.1 `go test ./internal/... ./cmd/...` を実行し、すべて通ることを確かめる
- [ ] 5.2 `cd frontend && npm run test` を実行し、すべて通ることを確かめる
- [ ] 5.3 `cd frontend && npm run type-check` を実行し、エラーが無いことを確かめる
- [ ] 5.4 `cd frontend && npm run build:dev` を実行し、ビルドが通ることを確かめる
- [ ] 5.5 `task test:e2e` を実行し、すべて通ることを確かめる (`check:scenarios` を含む)
- [ ] 5.6 `task lint` を実行し、エラーが無いことを確かめる
