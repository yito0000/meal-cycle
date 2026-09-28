## Context

動機と範囲は proposal.md の Why / What Changes を参照。ここでは実装に効く現状だけを書く。

- `recipes` を参照する外部キーは 2 本ある。`recipe_ingredients.recipe_id` と `meal_plans.recipe_id` で、どちらも
  `ON DELETE CASCADE` が付いている。物理削除すると、そのレシピの材料と献立が同じトランザクションで消える
- `recipes` を読むクエリは次の 5 箇所にある
  - `RecipeRepo.List` (S02 一覧と S01 のレシピ選択ダイアログが同じ `["recipes", "list"]` で共有)
  - `RecipeRepo.GetByID` (S03 と S04)
  - `MealPlanRepo` の `ListPeriod` / `ListMergeCandidates` / `loadMealEntries` / `insertMealPlan` (レシピ名の解決)
  - `ShoppingRepo.loadPlannedGroups` (献立の品とレシピ名)
  - `StockRepo.ListSuggestions` (S06 の食材名候補)
- `MealPlanRepo.Assign` と `Replace` は、存在しない `recipe_id` を FK 違反として検出し `mealplan.ErrUnknownRecipe` に変換している
- S03 ドロワー (`recipe-detail-drawer.tsx`) は S02 (`routes/recipes.tsx`) と S01 (`week-calendar.tsx`) の 2 箇所から使われる。
  開閉は呼び出し側の状態 (S02 は search param `detail`、S01 は `useState`) で決まり、ドロワーは `onClose` を呼ぶだけである
- トーストは `sonner` の `Toaster` が `root-layout.tsx` に置いてあり、S05 在庫の削除で使っている
- 確認ダイアログの前例は `unsaved-changes-dialog.tsx` (Radix Dialog、`destructive` ボタン) がある

## Goals / Non-Goals

**Goals:**

- レシピを消しても、そのレシピを使った献立行と、献立行から辿れるレシピ名と材料を失わない
- 「削除済み」の判定を Repository の 1 列に閉じ込め、読み取り側のクエリの変更を最小にする
- 削除済みレシピが新しい献立に入る経路を、UI とサーバの両方で塞ぐ

**Non-Goals:**

- 復元、物理削除、一括削除 (proposal.md の Non-goals を参照)
- 削除済みレシピを献立の各画面で視覚的に区別すること (S03 の表示だけに留める)

## Decisions

### D1. 削除の方式

**選択肢:**

- (a) 物理削除し、CASCADE で献立も消す — Pros: スキーマ変更が無い。「消した」が DB の状態と一致する /
  Cons: 過去の献立が黙って消える。提案機能「久しぶりに」の根拠になる履歴も失われる
- (b) 物理削除し、献立側の FK を `ON DELETE SET NULL` に変えて「削除されたレシピ」行として残す — Pros: レシピ本体は消える /
  Cons: `meal_plans.recipe_id` は NOT NULL で、「未定」= 行が無い状態という不変条件 (design.md §4) を崩す。
  レシピ名も材料も失われるので、S01 と S07 が名前の無い品を扱うことになる
- (c) 論理削除 (`recipes.archived_at`) ← 採用 — Pros: 献立行、レシピ名、材料がすべて残り、S01 と S07 は変更不要。
  復元も列を NULL に戻すだけで済む / Cons: 「削除済みを除外する」条件を読み取り側に足す必要がある。DB からデータは消えない

**採用: (c)**。理由: 献立の履歴を残すことが削除機能の前提条件であり (docs/reviews/2026-09-25-design-ux-review.md R2)、
(a) と (b) はどちらもそれを満たさない。除外条件が要るのは「新しく献立を組むための入口」(一覧、選択肢、割当、入替) だけで、
既存の献立を読むクエリは JOIN のまま変えなくてよい (D4)。
完全ローカルのアプリで、個人の料理データが DB に残ることの不利益は小さい。

### D2. 削除済みを表す列の型

**選択肢:**

- (a) `archived_at TEXT` (NULL = 未削除、値 = RFC3339 UTC の削除時刻) ← 採用 — Pros: 既存の `created_at` / `updated_at` と同じ表現。
  削除時刻が将来の復元 UI や履歴の表示で使える / Cons: 真偽の判定が `IS NULL` になる
- (b) `archived INTEGER NOT NULL DEFAULT 0` — Pros: 判定が素直 / Cons: いつ消したかが残らない

**採用: (a)**。理由: 時刻を持つコストは列 1 本で同じであり、(b) に移る理由が無い。
migration は `ALTER TABLE recipes ADD COLUMN archived_at TEXT` の 1 文で、既存行はすべて NULL (未削除) になる。
Down は `ALTER TABLE recipes DROP COLUMN archived_at` とする (modernc.org/sqlite が同梱する SQLite は 3.35 以降で DROP COLUMN に対応)。
インデックスは張らない。`List` は全件走査で、レシピ数は個人利用で数百件程度にとどまる。

Go 側の `recipe.Recipe` には `ArchivedAt string` (`json:"archivedAt"`) を足す。NULL は空文字で表す。
`*string` にしない理由は、既存の日時フィールドがすべて `string` で、フロントの生成型も `string` に揃うため。

### D3. 公開 API の形

**選択肢:**

- (a) `RecipeService.Delete(ctx, id) error` を足し、Repository 側は `Archive` とする ← 採用 — Pros: フロントとユーザーから見た言葉
  (「削除」) と API 名が一致する。実装が論理削除であることは Repository に閉じる / Cons: Service と Repository でメソッド名が違う
- (b) `RecipeService.Archive(ctx, id)` — Pros: 実装と名前が一致する / Cons: UI は「削除」と呼ぶので、呼び出し側で言葉がずれる
- (c) `Update` の入力に `archivedAt` を含めて更新する — Pros: API が増えない / Cons: 通常の編集保存で誤って削除や復元ができてしまう

**採用: (a)**。

`Delete` の振る舞い:

- 該当 ID が無い → `sqlite.ErrRecipeNotFound`
- 未削除 → `archived_at` に現在時刻 (RFC3339 UTC) を入れる。`updated_at` は変えない (内容の更新ではないため)
- 削除済み → 何もせず nil を返す (冪等)。二重クリックや、S01 と S02 で同じレシピを続けて消した場合にエラーを出さない

`Update` は、対象が削除済みなら新設の `sqlite.ErrRecipeArchived` を返し、行を変えない。
UI からは「編集」ボタンを出さないので通常は届かないが、S04 を開いたまま別経路で削除された場合の最終防衛線になる。
`GetByID` は削除済みも返す (D5)。

### D4. 読み取り側の除外範囲

`archived_at IS NULL` の条件を足す場所と、足さない場所を分ける。

| クエリ | 削除済みの扱い | 理由 |
| --- | --- | --- |
| `RecipeRepo.List` | 除外する | S02 一覧と S01 のレシピ選択ダイアログの入口 |
| `RecipeRepo.GetByID` | 含める | S01 の品から S03 を開けるようにする (D5) |
| `MealPlanRepo.Assign` / `Replace` | 拒否する | 新しい献立に入る経路。`mealplan.ErrUnknownRecipe` を返す |
| `MealPlanRepo` のその他 (一覧、移動、コピー、作り置き) | 含める | 既存の献立行を扱う操作であり、レシピ名の解決に必要 |
| `ShoppingRepo` | 含める | 献立に残っている品の材料は買う必要がある |
| `StockRepo.ListSuggestions` | 含める | D6 を参照 |

`Assign` / `Replace` の拒否は、書き込みの前に `SELECT 1 FROM recipes WHERE id = ? AND archived_at IS NULL` で確かめる。

**選択肢 (拒否の置き場所):**

- (a) `MealPlanRepo.Assign` / `Replace` の中で確かめる ← 採用 — Pros: 同じトランザクション内で確認と書き込みができる。
  既存の FK 違反 → `ErrUnknownRecipe` の変換と同じ層に並ぶ / Cons: mealplan の Repository が recipes の列を知る
- (b) `MealPlanService` から `RecipeRepo` を呼んで確かめる — Pros: 層の依存が素直 / Cons: `MealPlanService` は Repository を
  インタフェースで受けてフェイクでテストしており、依存を増やすとテストの準備が膨らむ。確認と書き込みが別トランザクションになる

**採用: (a)**。理由: mealplan の Repository はすでに `recipes` を JOIN してレシピ名を解決しており、列を 1 本読むことは新しい依存にならない。

`CopyLeftover` / `CopyToOtherMeal` / `MoveEntry` などは、既存の献立行が持つ `recipe_id` を複製または移動する操作なので拒否しない。
作り置きを翌日へ広げる操作は、削除済みレシピであっても「残り物を食べる予定」を表すので、塞ぐと利用者の実態と合わない。

### D5. 削除済みレシピの S03 表示

S02 の一覧からは削除済みレシピに辿り着けない。辿り着くのは S01 の品行メニュー「この品のレシピ詳細を見る」だけである。

**選択肢:**

- (a) `GetByID` が削除済みも返し、ドロワーは「削除済み」の表示を出して「編集」「削除」を隠す ← 採用 — Pros: 献立に残っている品の
  材料や手順を、削除後も確かめられる / Cons: ドロワーに分岐が 1 つ増える
- (b) `GetByID` が削除済みを `ErrRecipeNotFound` にし、既存の「読み込めない」表示を出す — Pros: 実装が少ない /
  Cons: S01 に名前が出ているのに中身を見られず、「なぜ読めないのか」が分からない

**採用: (a)**。「削除済み」は `SheetDescription` の基本情報の行に、タグ / 人数 / 更新日と並べて出す。
フッタはボタンが 0 個になるので、`SheetFooter` ごと描画しない。

### D6. S06 の食材名候補の扱い

**選択肢:**

- (a) 削除済みレシピの材料名も候補に含める (変更しない) ← 採用 — Pros: クエリを変えない。材料名は食材そのものの名前で、
  レシピを消しても「玉ねぎ」が不要な語になるわけではない / Cons: 誤記した材料名を持つレシピを消しても、誤記が候補に残る
- (b) 除外する — Pros: 候補が現役レシピに揃う / Cons: fridge の「食材名の候補補完」要件の振る舞いが変わり、spec delta が増える

**採用: (a)**。誤記の問題は、レシピを直さずに消すという限られた場面でしか起きず、候補に余分な 1 語が出るだけで害は小さい。

### D7. 確認ダイアログの部品と削除後の処理

確認ダイアログは `delete-recipe-dialog.tsx` として新設し、`unsaved-changes-dialog.tsx` と同じ Radix Dialog の構成にする。
props は `open` / `recipeName` / `pending` / `error` / `onConfirm` / `onCancel` とする。

**選択肢 (削除処理の置き場所):**

- (a) ドロワー内で `useMutation` を持ち、成功時に `onClose` を呼ぶ ← 採用 — Pros: S02 と S01 の呼び出し側に変更が要らない
  (どちらも `onClose` で開く前の画面に戻る仕組みをすでに持つ) / Cons: ドロワーが書き込みの責務を持つ
- (b) `onDelete` を props で受け、呼び出し側が削除する — Pros: ドロワーを表示専用に保てる / Cons: S02 と S01 に同じ処理が 2 回書かれる

**採用: (a)**。

成功時の処理は次の順にする。

1. `["recipes", "list"]` を無効化する (S02 一覧と S01 のレシピ選択ダイアログの両方に効く)
2. `["recipes", "detail", id]` を `refetchType: "none"` で無効化する (stale にするだけで、今は再取得しない)。
   ドロワーは閉じ切るまで直前のデータを描く (add-recipe-details design.md D5)。
   即時に再取得すると閉じるアニメーション中に削除済み表示へ切り替わり、`removeQueries` で消すと購読中の observer が
   読み込み中の表示に落ちるため、どちらも避ける。次にドロワーを開いたときは stale なので再取得される
3. `toast` で「「{{name}}」を削除しました」を出す
4. `onClose()` を呼ぶ

`["mealplan"]` と `["shopping"]` は無効化しない。論理削除では献立も材料も変わらないため。

失敗時はダイアログを開いたまま、本文の下に `pages.recipes.delete_confirm.error` を出す。
削除中は両ボタンを無効化し、二重送信を防ぐ。

### D8. 文言

| key | ja |
| --- | --- |
| `pages.recipes.detail.delete` | 削除 |
| `pages.recipes.detail.archived` | 削除済み |
| `pages.recipes.delete_confirm.title` | 「{{name}}」を削除しますか |
| `pages.recipes.delete_confirm.body` | レシピ一覧と、献立のレシピ選択から消えます。すでに献立に入っている分はそのまま残ります。 |
| `pages.recipes.delete_confirm.cancel` | キャンセル |
| `pages.recipes.delete_confirm.confirm` | 削除 |
| `pages.recipes.delete_confirm.error` | 削除できませんでした。もう一度お試しください。 |
| `pages.recipes.delete_confirm.done` | 「{{name}}」を削除しました |

en.json にも同じ key を足す。

## Risks / Trade-offs

- [Risk] 読み取りクエリを今後足すときに `archived_at IS NULL` を付け忘れ、削除済みレシピが新しい入口に出る
  → Mitigation: 除外が要るのは「新しく献立を組む入口」だけであることを D4 の表に残し、`recipe_repo.go` の `List` に 1 行コメントで理由を書く。
  `RecipeRepo_List` のテストに「削除済みを返さない」ケースを入れて退行を検出する
- [Risk] 削除済みレシピを使った献立が S01 で見分けられず、利用者が「消したのに残っている」と戸惑う
  → Mitigation: 確認ダイアログで「献立に入っている分は残る」と先に伝える。品から S03 を開けば「削除済み」が見える。
  戸惑いが実際に出たら、S01 セルでの表示を後続 proposal で足す (proposal.md の引き継ぎ)
- [Risk] 削除したレシピの名前で新しいレシピを作ると、S01 に同名の品が 2 種類並びうる
  → Mitigation: 名前の一意制約はもともと無く、現状でも同名レシピは作れる。本 proposal で新しく生じる問題ではないので対処しない
- [Trade-off] 論理削除なので、利用者が「消した」と思っているデータが DB に残る。完全ローカルで外部送信が無いため許容する
- [Trade-off] 確認ダイアログ方式を採り、S05 在庫の「即削除 + 元に戻す」とは操作感が揃わない。
  レシピは入力量が多く、誤操作の損失が在庫 1 行より大きいため、product.md §4 S03 の既定どおり確認を挟む
