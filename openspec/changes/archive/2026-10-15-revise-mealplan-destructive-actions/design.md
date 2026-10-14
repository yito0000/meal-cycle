## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/mealplan/spec.md](specs/mealplan/spec.md) を参照。

現状は次のとおりである。

- `WeekCalendar` は全 mutation を `withHandlers` で包み、失敗すると `errorState` に i18n のキーを入れる。
  `errorState` はカレンダー上部の赤い帯として描かれ、次の成功で消える
- `MealPlanService.RemoveEntry` と `ClearMeal` は `error` だけを返す
- 品の削除やクリアで、ある (作り置き, レシピ) の行が 1 つも無くなると、`pruneCookedMarks` がその「作った」印を消す
- `SplitBatch` は、起点以外の行を消し、起点の行に新しい `batch_id` を振る。消えた行は表示中の期間の外にあることもある
- S05 は、削除前の行をフロントが持っておき、通知の「元に戻す」で `FridgeService.Restore(item)` に渡して作り直す

## Goals / Non-Goals

**Goals:**

- 削除とクリアを、S05 と同じ「即実行 + 元に戻す」にそろえる
- 元に戻したあとの状態 (並び順、作り置き、作った印) を、削除の前と同じにする
- 「作り置きをやめる」で行が消えることを、実行の前に伝える

**Non-Goals:**

- 汎用の undo スタック (操作の履歴を何段も戻す)

## Decisions

### D1. 元に戻すための情報を誰が持つか

**選択肢:**

- (a) フロントが、削除の前に一覧から行の内容を控える (S05 と同じ)
  — Pros: Go 側の戻り値を変えない
  / Cons: 一覧の `Entry` は「作った」印を真偽値でしか持たず、印を付けた日時を戻せない。
  一覧を取り直す前の連続操作では、控えた内容が古い
- (b) 削除とクリアが、消した行の内容を返す ← 採用
  — Pros: 消したその時点の内容が手に入る。印の日時も含められる
  / Cons: 2 つのメソッドの戻り値が変わる
- (c) 論理削除にして、元に戻すときにフラグを倒す
  — Pros: 並び順をそのまま保てる
  / Cons: 献立を読む全クエリ (S01、S07、まとめ候補) に条件が増える。並び順の一意性と両立させにくい

**採用: (b)**。

```go
// internal/domain/mealplan/removed.go
type RemovedEntry struct {
    Date      string `json:"date"`
    Meal      Meal   `json:"meal"`
    Position  int    `json:"position"`
    RecipeID  string `json:"recipeId"`
    BatchID   string `json:"batchId"`
    CreatedAt string `json:"createdAt"`
    CookedAt  string `json:"cookedAt"` // 空文字 = 印なし
}
```

- `RemoveEntry(date, meal, position) (RemovedEntry, error)`
- `ClearMeal(date, meal) ([]RemovedEntry, error)` (並び順の昇順)
- `RestoreEntries(entries []RemovedEntry) error`

### D2. 元に戻す処理の中身

`MealPlanRepo.RestoreEntries` は 1 トランザクションで、渡された行を (日付, 時間帯, 並び順) の昇順に 1 行ずつ戻す。

1. そのセルの現在の品数を数える。戻す並び順が品数より大きければ、末尾 (品数) に置く
2. 戻す並び順以降の行を、1 つずつ後ろへずらす。ずらし方は `fix-mealplan-renumber-after-reorder` の退避の方式を使う
   (全行を `positionShiftOffset` だけ退避し、`+1` した位置へ戻す)
3. 元の `recipe_id`、`batch_id`、`created_at` で行を入れる
4. `CookedAt` が空でなく、その (作り置き, レシピ) の印が無ければ、同じ日時で印を入れる

削除のあとに別の品が同じセルへ足されていても、1 と 2 により失敗しない。
この場合、戻した品は元の並び順に入り、あとから足された品はその後ろへずれる。

レシピが論理削除されていても戻せる (既存の献立は削除済みレシピを参照し続ける方針)。
レシピの行そのものが無い場合は外部キー違反になり、`ErrUnknownRecipe` を返す。

サービスは、各行の日付、時間帯、並び順 (0 以上)、`RecipeID` と `BatchID` が空でないこと、`CreatedAt` と `CookedAt` が RFC3339 であることを確かめる。
違反は `ErrInvalidRestore` で拒否する。一覧が空なら何もしない。

### D3. 「作り置きをやめる」の取り消し

**選択肢:**

- (a) 削除と同じく、消した行を返して「元に戻す」を付ける
  — Pros: 3 つの操作の作法がそろう
  / Cons: やめる操作は、残った行の `batch_id` も新しくする。元に戻すには、行を戻すだけでなく、残った行の `batch_id` と印を元の値へ戻す必要がある。
  `RestoreEntries` とは別の処理になり、あいだに別の操作が挟まったときの扱い (残った行がすでに別の作り置きにまとめられていた、など) も決めなければならない
- (b) 実行の前に確認のダイアログを出す ← 採用
  — Pros: Go 側を変えない。「選んだ行が残るとは限らない」という、名前だけでは伝わらない点を文章で示せる
  / Cons: 「S05 の削除は確認なし」の方針とは別の作法が 1 つ増える
- (c) 何も足さず、改名だけにする
  — Pros: 最小
  / Cons: 表示中の期間の外にある行も消える操作が、1 クリックで走るまま残る

**採用: (b)**。
削除とクリアは「見えている品が消える」操作で、消えたことが画面でわかる。
やめる操作は、見えていない日の行まで消えるので、事後の通知より事前の確認が合う。
product.md §7 の決定事項に、この使い分けを書く。

削除される行の数は、選んだ行の `batchMemberCount - 1` である。フロントが持っている値で足りる。

### D4. 新しい名前

**選択肢:**

- (a) 「作り置きをやめる」 ← 採用
  — Pros: 作り置き全体に効く操作だとわかる。「広げる」の逆として読める
  / Cons: どの行が残るかは名前に入らない (確認のダイアログで補う)
- (b) 「作り置きを最初の日だけにする」
  — Pros: 結果が名前に入る
  / Cons: メニューの項目として長い。ほかの項目 (「作り置きを広げる」「他の作り置きにまとめる」) と長さがそろわない
- (c) 「作り置きを解除」
  — Pros: 短い
  / Cons: 「この行を解除する」とも読め、今回直したい誤解が残る

**採用: (a)**。
i18n のキー (`item_menu.split_batch`) と、Go 側の `SplitBatch` の名前は変えない。
画面の文言だけの変更であり、内部の名前を変えると差分が広がるためである。

### D5. 通知の出し方

失敗は `toast.error(t(key))`、削除とクリアの成功は「元に戻す」付きの `toast(…, { action })` にする。
表示時間は S05 と同じ `UNDO_TOAST_MS` (5 秒) を使い、定数を `frontend/src/lib/undo-toast.ts` に移して共有する。
他案 (S01 用に別の定数を持つ) は、同じ作法の値が 2 か所に分かれるので見送る。

連続して 2 品を削除すると、通知は 2 つ積まれ、それぞれが自分の消した行を戻す。
`sonner` は通知ごとに `action` を持てるので、追加の状態管理は要らない。

`withHandlers` は残し、`setErrorKey` の代わりに `toast.error` を呼ぶ。
献立の読み込みの失敗 (`routes/meal-plan.tsx` の `isError`) は、これまでどおりカレンダーの位置に出す。

## Risks / Trade-offs

- [通知が消えたあとは戻せない] → S05 と同じ割り切りである。5 秒は S05 で使っている値で、短いという声が出たら 1 か所で変えられる
- [元に戻すまでのあいだに同じセルが変わると、並び順が削除の前と完全には一致しない] → D2 のとおり失敗はしない。戻した品は元の位置に入る
- [クリアした時間帯の品数が多いと、通知の文言が長くなる] → 文言は「N 品を削除しました」と件数だけにする
- [確認のダイアログが増えて、やめる操作が 1 手増える] → 行が消える操作であり、頻度も低い。D3 のとおり受け入れる
- [`RemoveEntry` の戻り値の変更で、E2E の shim と生成物の型がずれる] → shim とバインディングの再生成を同じタスクグループに置く
