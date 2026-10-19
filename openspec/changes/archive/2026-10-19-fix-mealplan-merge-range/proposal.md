## Why

「他の作り置きにまとめる」が候補を探す日付の範囲が、S09 で作り置きの日数の候補を変えても ±4 日のまま変わらない
(docs/reviews/2026-10-19-design-ux-review.md R46)。
献立のリポジトリが自分で既定値を持ち、S09 で保存した値を受け取っていないためである。
settings の spec は、この探索範囲を保存した値の反映先に挙げており、実装だけが追いついていない。
候補を「2」「7」にすると、広げる操作は 6 日先まで届くのに、まとめる候補は ±4 日しか探さない。

あわせて、まとめる候補の日付と件数が、探す範囲の内側の行だけで決まっている (R47、再現済み)。
10/19 から 10/22 までの 4 行の作り置きを 10/24 の行から見ると、候補は「10/20 夕食 (3 品)」と出る。
mealplan の spec は候補に「その作り置きの中でいちばん早い行」を出すと定めており、これに反する。
R36 で直した献立の取得と同じく、範囲で絞った後の行にしか集計が掛からないことが原因である。

## What Changes

- 「他の作り置きにまとめる」の探索範囲 (±N 日) を、S09 で保存した作り置きの日数の候補の最大値から決める
  - 候補の一覧と、まとめる操作そのものの可否の両方に効かせる
  - 保存したあとは、アプリを再起動しなくても次に候補を開いたときから効く
- まとめる候補の日付と時間帯を、探す範囲に関係なく、その作り置きの同じレシピの行のうちいちばん早い行から取る
- まとめる候補の件数を、探す範囲に関係なく、その作り置きの同じレシピの全行の数にする
- 候補に出すかどうかは従来どおり、「同じレシピの行が 1 行でも範囲内にある作り置き」で決める

### Non-goals

- **まとめるダイアログの日付の表記** (R49)：`fix-mealplan-dialog-polish` で扱う
- **skip している E2E の有効化** (R44)：openspec を通さない `chore:` コミットで扱う
- **探索範囲そのものの決め方の変更**：「日数の候補の最大値」という規則は変えない
- **候補の件数の文言** (「N 品」)：表記は変えない。数える対象だけを spec に書く

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：「まとめ候補の一意性」に、候補の件数が何を数えるかと、範囲の外に続く作り置きの扱いを足す
- `settings`：「保存した値の各画面への反映」に、まとめる候補を探す範囲が変わる Scenario を足す

## Impact

### 新設ファイル

- なし

### 更新ファイル

- `internal/storage/sqlite/mealplan_repo.go`
  - `MealPlanRepo` から `cfg` フィールドと `mergeDayRange` を外し、`config` の import を消す
  - `ListMergeCandidates` と `MergeBatch` に引数 `dayRange int` を足す
  - `ListMergeCandidates` のクエリを、範囲の判定と代表行と件数の集計を分けた形に書き直す
- `internal/storage/sqlite/mealplan_repo_test.go`：`ListMergeCandidates` と `MergeBatch` の呼び出しに日数を渡し、範囲の外に続く作り置きのテストを足す
- `internal/service/mealplan.go`
  - `mealPlanRepo` interface の `ListMergeCandidates` と `MergeBatch` に `dayRange int` を足す
  - `mergeDayRange` を足し、`maxSpreadOffsetDays` をその値から求める
  - `ListMergeCandidates` と `MergeBatch` が `mergeDayRange` の値を渡す
- `internal/service/mealplan_test.go`
  - `fakeMealPlanRepo` の `mergeDayRange` (4 の決め打ち) を消し、引数の日数を使う
  - 作り置きの日数の候補を変えた設定で、探索範囲が変わるテストを足す
- `frontend/e2e/mealplan-batch.spec.ts`：describe「まとめ候補の一意性」に Scenario を 1 つ足す
- `docs/reviews/2026-10-19-design-ux-review.md`：R46、R47 にチェックを入れる

### 生成物 (再生成のみ)

- なし (サービスの公開メソッドの引数と戻り値は変わらない)

### migration

- なし

### 後続 proposal への引き継ぎ

- 時間帯の順を表す `CASE meal` が SQL に増える。時間帯を増やす proposal で、Go 側の `MealRank` との対応をまとめて直す
