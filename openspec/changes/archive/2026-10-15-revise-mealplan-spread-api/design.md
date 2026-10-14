## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/mealplan/spec.md](specs/mealplan/spec.md) を参照。

現状は次のとおりである。

- `WeekCalendar.submitSpread` は、選択を `planSpread` で「同じ時間帯の日付」と「別の時間帯のセル」に分ける
- 同じ時間帯の日付は `computeCopyDays` で「起点からの最大の日数差 + 1」に直し、`CopyLeftover(date, meal, days, position, mode, shareBatch)` を 1 回呼ぶ。
  `CopyLeftover` は起点から連続 `days` 日に入れるので、選んでいない日も埋まる
- 別の時間帯のセルは、1 セルにつき `CopyToOtherMeal(…, mode)` を 1 回呼ぶ。引数に `shareBatch` が無く、常に起点の `batch_id` で挿入する
- 呼び出しは逐次で、失敗した件数を数えて `spread.partial_failure` を出す
- `ListMergeCandidates` は、作り置きごとの代表行を `MIN(date || printf('%04d', position))` で選び、同じ文字列を持つ行と JOIN する。
  時間帯を含まないので、同じ日の同じ position にある 2 行が両方当たる
- サービス層のテストは、`mealPlanRepo` インターフェースのフェイクを使う。リポジトリ層は実 SQLite で確かめる

## Goals / Non-Goals

**Goals:**

- 格子で選んだセルと、反映されるセルを一致させる
- 反映を 1 トランザクションにする
- 画面が使わなくなる経路 (`CopyToOtherMeal`) を残さない

**Non-Goals:**

- `CopyLeftover` の変更 (時間帯全体の「連続 N 日コピー」が使い続ける)
- 対象日の範囲の拡張

## Decisions

### D1. 反映の経路

**選択肢:**

- (a) フロントで、同じ時間帯の日付を 1 日ずつ `CopyLeftover(days=2)` 相当に分けて呼ぶ
  — Pros: Go 側を変えない
  / Cons: `CopyLeftover` は起点日の翌日からしか入れられないので、とびとびの日を表せない。別調理が別の時間帯に効かない問題と、途中失敗の問題も残る
- (b) `CopyToOtherMeal` に `shareBatch` を足し、同じ時間帯のセルもこれで 1 セルずつ入れる
  — Pros: 追加が小さい
  / Cons: 呼び出しがセルの数だけ分かれたままで、途中で失敗すると半端に残る
- (c) セルの一覧を受け取る `SpreadEntry` を 1 本足し、1 トランザクションで反映する ← 採用
  — Pros: 選んだセルがそのまま引数になる。3 つの問題がまとめて消える。フロントの分解 (`planSpread`、`computeCopyDays`) が要らなくなる
  / Cons: サービス、リポジトリ、フェイク、shim にメソッドが 1 つ増える

**採用: (c)**。

```go
// internal/domain/mealplan/spread.go
type SpreadTarget struct {
    Date string `json:"date"`
    Meal Meal   `json:"meal"`
}

// internal/service/mealplan.go
func (s *MealPlanService) SpreadEntry(
    ctx context.Context, date, meal string, position int,
    targets []mealplan.SpreadTarget, mode string, shareBatch bool,
) error
```

### D2. 対象セルの検証をどこに置くか

`mealplan.ValidateSpreadTargets(sourceDate string, sourceMeal Meal, targets []SpreadTarget, maxOffsetDays int) error` を
ドメイン層に置き、table-driven test で確かめる。
次のいずれかで `ErrInvalidSpreadTarget` を返す。

- 一覧が空である
- 日付または時間帯が不正である (`ErrInvalidDate` / `ErrInvalidMeal` はそのまま返す)
- 起点セル自身を含む
- 日付が起点日より前、または起点日 + `maxOffsetDays` より後である
- 同じセルが 2 回出てくる

`maxOffsetDays` はサービスが `max(cfg.LeftoverCopyPresets) - 1` を渡す。
リテラルの 3 は書かない (可変値は `internal/config/defaults.go` に集約する方針)。
他案 (リポジトリの中で検証する) は、SQL と混ざってテストしにくいので見送る。

### D3. 1 セルへの反映の中身

`MealPlanRepo.Spread` は 1 トランザクションで次を行う。

1. 起点の行 (`recipe_id`、`batch_id`) を読む。無ければ `ErrEntryNotFound`
2. 対象セルを (日付, 時間帯の朝 / 昼 / 夕) の順に並べ、1 セルずつ処理する
   - `overwrite`：そのセルの行をすべて消す
   - `append`：そのセルに同じ `recipe_id` の行があれば、何も入れずに次へ進む (既存の `PlanCopy` と同じ二重追加の防止)
   - セルの末尾に、起点と同じ `recipe_id` で 1 行を入れる。`batch_id` は、作り置きなら起点の値、別調理なら新しく発行する
3. `pruneCookedMarks` を呼ぶ (上書きで消えた行の印を掃除する)

既存の `PlanCopy` は使わない。
`PlanCopy` は「1 つの時間帯に複数の品を入れる」ための関数で、今回は 1 品を複数のセルに入れるので、形が合わない。

### D4. `CopyToOtherMeal` をどうするか

**選択肢:**

- (a) 残す
  — Pros: 差分が小さい
  / Cons: 画面から呼ばれないメソッドが、サービス、リポジトリ、フェイク、shim、テスト (100 行超) に残る
- (b) 消す ← 採用
  — Pros: 「別の時間帯へ入れる経路は `SpreadEntry` だけ」と言い切れる
  / Cons: テストの削除を伴う。E2E のヘルパーは使っていないので影響しない

**採用: (b)**。
`CopyToOtherMeal` のテストが確かめていた内容 (同じ `batch_id` で入る、範囲外は拒否、追加での二重防止、上書き) は、`Spread` のテストに移す。

### D5. 選べないセルの判定

統合ダイアログは、表示中の期間の `entries` を受け取っている。
起点の行の `batchId` と `recipeId` をダイアログに渡し、同じ 2 つを持つ行があるセルを選択不可にする。
既定のチェック (起点日の翌日の同じ時間帯) がこの条件に当たるときは、チェックを付けない。

対象日が表示中の期間の外にはみ出す場合 (日曜を起点にすると月〜水曜は翌週) は、`entries` にその日の行が無い。
この場合は選択不可にできないが、反映は `append` の二重防止で何も起きないので、害は無い。

### D6. まとめ候補の代表行

**選択肢:**

- (a) 連結する文字列に時間帯の順位を足す (`date || 朝昼夕の順位 || position`)
  — Pros: 既存のクエリの形を保てる
  / Cons: 文字列の連結で行を特定する作りが残る
- (b) `ROW_NUMBER() OVER (PARTITION BY batch_id ORDER BY date, 朝昼夕の順位, position)` で 1 行目を取る ← 採用
  — Pros: 代表行が必ず 1 行になる。並べる基準が ORDER BY にそのまま書ける
  / Cons: クエリを書き直す

**採用: (b)**。
時間帯の順位は、`shopping_repo.go` と同じ `CASE meal WHEN 'breakfast' THEN 0 …` で作る。

## Risks / Trade-offs

- [`SpreadEntry` の引数が 7 つある] → `add-mealplan-entry-id` で (日付, 時間帯, 並び順) が ID 1 つになる。それまでは既存のメソッドと同じ並びにそろえる
- [E2E サーバは引数を reflect で詰める。構造体のスライスを受けるメソッドは `RecipeService.Import` と `FridgeService.AddMany` で通っている] →
  `cmd/e2e-server/dispatch_test.go` に `SpreadEntry` の形のケースを 1 つ足して確かめる
- [対象日が翌週にはみ出すと、選べないセルの表示が漏れる (D5)] → 反映では二重に入らない。表示の漏れは範囲の拡張の proposal で、ダイアログが自分で取りに行く形に直す
- [`spread.partial_failure` を消すので、古い翻訳キーへの参照が残ると画面にキーが出る] → `week-calendar.tsx` の `errorState.params` も一緒に消し、`npm run build:dev` と E2E で確かめる
