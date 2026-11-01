## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/fridge/spec.md](specs/fridge/spec.md) と [specs/shopping/spec.md](specs/shopping/spec.md) を参照。

この proposal に関わる現状は次のとおりである。

- 在庫は `stock_items` の 1 行 = 1 品で、`amount REAL NOT NULL CHECK (amount > 0)` を持つ。量を持たない行を入れる余地は無い
- `FridgeService` が S05 の在庫を、`ShoppingService` が S07 の算出を公開している。どちらも `main.go` で `settingsSvc` を受け取って登録する
- 食材名の表記をそろえる関数は `internal/domain/shopping/normalize.go` の非公開の `normalizeKey` だけにある。
  ひらがなとカタカナ、全角と半角、英字の大文字と小文字をそろえ、漢字とかなはそろえない
- S07 の材料の算出 (`shopping.Aggregate`) は、作った印の付いた組を飛ばし、量を持たない材料を食材名だけで合算してから、在庫の状態を付けて並べる
- S07 の画面は、献立が無い期間には `empty.no_plans` だけを出し、材料の列も献立の欄も出さない
- `useShoppingList` は他画面の mutation から invalidate されない。S07 は開くたびに再取得する (add-shopping-list design.md D8)
- 開発は長期の機能ブランチ `family/main` で行う。この期間、main ではスキーマを変えない ([docs/release.md](../../../docs/release.md))

## Goals / Non-Goals

**Goals:**

- 常備品を、在庫の量と賞味期限の仕組みに手を入れずに足す。S05 の在庫のテーブルと S07 の在庫判定の規則は、常備品と同名の材料を除けば変わらない
- 常備品と材料の突き合わせに、在庫との突き合わせと同じ表記のそろえ方を使う

**Non-Goals:**

- 常備品の行の並べ替えを利用者に選ばせること。食材名の昇順だけにする
- 状態の切替を元に戻す操作。切り替えた状態は、同じ欄でもう一度押せば戻せる

## Decisions

### D1. 常備品を持つテーブル

**選択肢:**

- (a) `stock_items` に種別の列を足し、常備品の行は量と単位と賞味期限を NULL にする
  — Pros: テーブルが 1 つで済む
  / Cons: `amount` の CHECK と NOT NULL を外すことになり、SQLite では表の作り直しが要る。
  在庫を読むすべての箇所 (S05 の一覧、S07 の在庫判定、候補補完) に種別の分岐が入る
- (b) `staples` テーブルを別に作る ← 採用
  — Pros: `stock_items` と既存の読み出しに手が入らない。状態の列に CHECK を付けられる
  / Cons: 常備品と同名の在庫が両方のテーブルにあり得る

**採用: (b)**。
常備品と在庫は持つ列が重ならず、重なるのは食材名だけである。
同名の両立は Non-goals のとおり許し、S07 での扱いを D6 で決める。

```sql
CREATE TABLE staples (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL CHECK (length(name) > 0),
    state      TEXT NOT NULL CHECK (state IN ('enough', 'low', 'out')),
    added_at   TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

`id` は在庫と同じく UUID v7 を Repository で発行する。
`added_at` は「元に戻す」で復元したときに保つ値で、並び順には使わない。
`updated_at` は状態を切り替えるたびに進める。

### D2. 状態の表し方

**選択肢:**

- (a) 整数 (0 / 1 / 2) で持つ
  — Pros: 大小比較で並べられる / Cons: DB とバインディングを読んでも意味がわからない
- (b) 文字列の列挙 `enough` / `low` / `out` で持つ ← 採用
  — Pros: `fridge.ExpiryStatus` と同じ形で、フロントの型にもそのまま出る / Cons: 並び順は別に定義する

**採用: (b)**。`internal/domain/fridge/staple.go` に `StapleState` 型と 3 つの定数を置く。
S07 の並び順 (「ない」が先) は D6 の算出で決め、値の大小には頼らない。

### D3. 同じ常備品を 2 つ登録させない判定

**選択肢:**

- (a) `name` に UNIQUE 制約を付ける
  — Pros: DB が保証する / Cons: 「ギュウニュウ」と「ぎゅうにゅう」のような表記の違いを同じとみなせない
- (b) サービスで表記をそろえた名前を既存の常備品と比べる ← 採用
  — Pros: S07 の突き合わせと同じ基準で重複を判定できる
  / Cons: 2 つの追加がほぼ同時に来ると、両方とも通る余地がある

**採用: (b)**。S07 で同じ常備品とみなす 2 つの名前を、S05 では別々に登録できてしまうと、
材料がどちらの常備品に寄るかが決まらなくなる。
同時の追加は操作する人が 1 人のデスクトップアプリでは起きにくく、起きても S07 では最初の 1 件に寄せる (D6) ので表示は壊れない。

表記をそろえる関数は、`fridge` と `shopping` の両方から使うので、新しいパッケージに移す。

- (i) `shopping.normalizeKey` を公開して `fridge` 側から import する
  — Cons: 在庫のドメインが買い物リストのドメインに依存する向きになる
- (ii) `internal/domain/ingredient/name.go` に `NormalizeName` として移す ← 採用
  — Pros: どちらのドメインからも依存の向きが自然になる。後の `add-ingredient-master` の置き場所にもなる
- (iii) 同じ関数を `fridge` にも書く
  — Cons: そろえ方を変えたときに片方だけ直る

`shopping.normalizeKey` は `ingredient.NormalizeName` を呼ぶだけの関数に縮め、呼び出し側は変えない。
`normalize_test.go` のケースは `internal/domain/ingredient/name_test.go` に移す。

### D4. 常備品の操作を公開するサービス

**選択肢:**

- (a) `FridgeService` にメソッドを足す ← 採用
  — Pros: `main.go` と `cmd/e2e-server` のサービス登録、`dispatch_test.go` の登録数が変わらない。S05 の画面と 1 対 1 で対応する
  / Cons: `FridgeService` のメソッドが増える
- (b) `StapleService` を新設する
  — Pros: ファイルが分かれる / Cons: サービス登録とバインディングの import 先が増える。capability は `fridge` のままなので 1 capability = 1 ファイルの規約からも外れる

**採用: (a)**。足すメソッドは次の 5 つで、実体は `internal/storage/sqlite/staple_repo.go` の `StapleRepo` に置く。

- `ListStaples(ctx) ([]fridge.Staple, error)`：全件を返す。並び順はフロントで決める (D7)
- `AddStaple(ctx, name string) (fridge.Staple, error)`：名前を検証し、状態 `enough` で 1 件作る
- `SetStapleState(ctx, id string, state fridge.StapleState) (fridge.Staple, error)`：状態を変え、`updated_at` を進める
- `DeleteStaple(ctx, id string) error`
- `RestoreStaple(ctx, item fridge.Staple) (fridge.Staple, error)`：削除前の ID、名前、状態、`added_at` で作り直す。重複の検査も通す

エラーは `internal/domain/fridge/errors.go` に `fridge: staple ...` の形で足す。
フロントの `stockErrorOf` は `fridge: name empty` のような部分文字列で判定しているので、
常備品のコードに `staple` を挟み、既存の規則に誤って当たらないようにする。

### D5. 常備品を S07 に載せる経路

**選択肢:**

- (a) `ShoppingService.List` の返り値に `Staples` を足す ← 採用
  — Pros: 同名の材料を除く処理と、常備品の欄の算出を同じ 1 回の読み出しで行える
  / Cons: `shopping.List` の型とバインディングが変わる
- (b) S07 で `FridgeService.ListStaples` も呼び、フロントで突き合わせる
  — Pros: Go 側の買い物リストの算出は変わらない
  / Cons: 材料テーブルから同名の材料を外すには、表記のそろえ方をフロントにも書くことになる

**採用: (a)**。`shopping.List` に `Staples []StapleItem` を足す。
`StapleItem` は `ID`、`Name`、`State`、`Recipes` を持ち、「少ない」「ない」の常備品だけが入る。
空状態の判定のために `HasStapleIngredients bool` も足す (D10)。

### D6. 常備品と同名の材料の扱い

`shopping.Aggregate` に常備品の名前の一覧を渡し、表記をそろえた食材名が一致する材料を合算の前に飛ばす。
量のある材料も量を持たない材料も、単位を問わず飛ばす。
飛ばした材料のレシピ名は、新設の `shopping.StapleItems(batches, staples)` が献立の順に集める。
`StapleItems` は次の順で常備品の欄の項目を作る。

1. 状態が `enough` の常備品を除く
2. 計上する組 (作った印の無い組) の材料を献立の順に走査し、名前が一致する常備品の `Recipes` にレシピ名を重複なく足す
3. 状態が `out` の項目を先、`low` の項目を後にし、それぞれ食材名の昇順に並べる

表記をそろえた名前が同じ常備品が 2 件あれば (D3 の同時追加)、`ListStaples` の順で最初の 1 件にだけレシピ名を足す。

他の案として、常備品と同名の材料を材料テーブルに残し、在庫の列に常備品の状態を出すことも考えた。
だが材料テーブルの在庫の列は量の突き合わせの結果を示す列で、量を持たない常備品の状態を混ぜると、「一部在庫あり」と「少ない」の区別がつかなくなる。
材料を常備品の欄に寄せれば、1 つの品が S07 の 2 か所に出ることも無い。

### D7. S05 の常備品の欄の置き場所と形

在庫のテーブルの右に `lg:grid-cols-[1fr_20rem]` で欄を置き、狭い幅では在庫のテーブルの下に回す。
S07 の「この期間の献立」と同じ組み方で、利用者が 2 画面で同じ形を覚えれば済む。

他の案として、欄を在庫のテーブルの上に置くことも考えた。
だが S05 は「一番上にインライン追加行が常駐する」ことを前提に組んであり、その上に別の欄を挟むと追加行の位置が常備品の件数で動く。

欄の中身は次のとおりとする。

- 見出し「常備品」と、量を持たず状態だけを持つ品であることを添える説明
- 追加行：食材名の入力 (`IngredientNameInput` を在庫の追加行と同じ候補で使う) と「追加」ボタン。
  食材名の欄の Enter で追加し、日本語入力の変換確定の Enter では追加しない。追加に成功したら入力を空にしてフォーカスを残す
- 一覧：1 行に食材名、状態の切替、削除。並び順は `name.localeCompare(other, "ja")` (在庫の「食材名」の並びと同じ比較) で固定する
- 常備品が 1 件も無いときは、一覧の位置にその旨を出す

S05 の検索欄と並び替えは在庫のテーブルにだけ効かせる。
常備品は数が少なく、状態を見渡すために全件を並べておくほうが役に立つためである。

### D8. 状態の切替の操作

**選択肢:**

- (a) 「ある / 少ない / ない」の 3 つのボタンを横に並べ、選択中のボタンを押下状態で示す ← 採用
  — Pros: 1 回押せば目的の状態になる。押下状態は `aria-pressed` で読み上げにも伝わる
  / Cons: 1 行の横幅を取る
- (b) 押すたびに「ある → 少ない → ない」と巡回する 1 つのボタン
  — Pros: 幅を取らない / Cons: 「ある」から「ない」へは 2 回押すことになり、押しすぎると巡回して戻る
- (c) セレクトボックス
  — Cons: 開いて選ぶ 2 操作になる

**採用: (a)**。部品は `staple-state-toggle.tsx` に置き、S05 だけで使う。
押すと `SetStapleState` を呼び、成功したら常備品の一覧を invalidate する。
保存中はその行の切替を無効にし、連打で順序が入れ替わるのを防ぐ。
状態の色は、「ない」を赤系、「少ない」を黄系、「ある」を無彩色にする。色だけに頼らず、ボタンの文字で状態を示す。

### D9. S07 の「買った」

常備品の欄の各項目に「買った」ボタンを置く。
押すと `SetStapleState(id, "enough")` を呼び、成功したら S07 の買い物リストと S05 の常備品の一覧を invalidate する。
S07 の一覧は他画面の mutation では invalidate しない方針だが、S07 自身の操作は一括登録と同じく invalidate する (add-shopping-purchase design.md D6)。

トーストに「元に戻す」を置き、押すと押す前の状態 (`low` か `out`) で `SetStapleState` を呼ぶ。
削除ではなく状態の変更なので、`RestoreStaple` は使わない。
「買った」で項目は欄から消えるので、ミューテーションのコールバックは、在庫の削除と同じくページ側 (`ShoppingPage`) で扱う。

### D10. S07 の常備品の欄の置き場所と空状態

常備品の欄は左の列 (材料の列) の中で、登録ボタンの下、「家にあるか確かめる材料」の上に置く。
買う品が決まっている欄を、確かめるだけの欄より先に読ませるためである。
材料テーブルより上に置く案は、期間の献立から算出する本体を常備品が押し下げるので採らない。

空状態ごとの表示は次のとおりとする。どの場合も、常備品の欄は `staples` が 1 件以上のときだけ出す。

| 期間の状態 | 材料の列に出すもの | 献立の欄 |
| --- | --- | --- |
| 献立が無い | `empty.no_plans` と常備品の欄 | 出さない |
| すべて作った印付き | `empty.all_cooked` と常備品の欄 | 出す |
| 材料が無い | `empty.no_ingredients` と常備品の欄 | 出す |
| 材料がすべて常備品と同名 | `empty.only_staples` と常備品の欄 | 出す |
| 量のある材料が無い | `empty.no_quantified`、常備品の欄、家にあるか確かめる材料 | 出す |
| それ以外 | 絞り込み、材料テーブル、登録ボタン、常備品の欄、家にあるか確かめる材料 | 出す |

献立が無い期間は、これまで 2 列の組みを出さずに 1 行のメッセージだけを出していた。
この proposal では、献立が無い期間も 1 列のまま、メッセージの下に常備品の欄を置く。

常備品と同名の材料だけからなる期間は、`items` が空になり、このままでは「材料が無い」に当たる。
だが献立のレシピに材料はあるので、`empty.no_ingredients` の文言は合わない。
常備品がすべて「ある」なら `staples` も空になり、`items` と `staples` だけでは「材料が無い」と区別できない。
そこで `shopping.List` に `HasStapleIngredients bool` を足し、常備品に寄せた材料が 1 件でもあれば true にする。
`items` が空で `HasStapleIngredients` が true のときは、`empty.no_ingredients` の代わりに
「献立の材料はすべて常備品です」(`empty.only_staples`) を出し、その下に常備品の欄を出す。

### D11. テキストコピーへの常備品の組み込み

`buildShoppingText` に常備品の項目を渡し、次の順で組み立てる。

1. 見出し
2. 材料テーブルの「在庫なし」「一部在庫あり」の行
3. 常備品の項目があれば、空行、見出し「常備品」、`- 牛乳` または `- 食パン (少ない)` の行を欄と同じ順に
4. 「家にあるか確かめる材料」の在庫の無い項目があれば、空行、見出し、項目の行

件数は 2〜4 の行の合計とする。
ボタンを出す条件は「献立があって材料が 1 件以上ある」から、「その条件を満たすか、常備品の項目が 1 件以上ある」に広げる。
「ない」には印を添えない。材料テーブルの「在庫なし」に印を添えないのと合わせ、買う必要がある状態を既定として読ませる。

### D12. ワイヤーとの差分

S05 は常備品の欄で画面の組みが 1 列から 2 列に変わるので、`docs/wireframes/S05_fridge.v2.drawio.svg` を新しく作る。
旧ワイヤーは初期の構想の記録として残す ([docs/wireframes/README.md](../../../docs/wireframes/README.md) の例外の規則)。

S07 は左の列に欄が 1 つ増えるだけで、組みは `S07_shopping_list.v2.drawio.svg` のままである。
ワイヤーは更新せず、差分 (登録ボタンの下に常備品の欄が入ること) をこの節に残す。

## Risks / Trade-offs

- [Risk] 常備品と同名の材料が S07 の材料テーブルから消え、常備品が「ある」のときはどこにも出ない。
  レシピで 1 L 使う牛乳が、残りわずかでも「ある」のままなら買い物リストに載らない
  → 状態を見直すのは操作する人の判断に任せる前提 (proposal の Why) に従う。
  常備品の欄の用途には、その常備品を使うレシピ名を出し、「少ない」「ない」にしたときに献立とのつながりが見えるようにする
- [Risk] `shopping.List` の型が変わり、e2e の shim と `useShoppingList` の null の扱いに漏れがあると S07 が落ちる
  → shim の戻り値型は `shopping.List` を使うので、`staples` は型に自動で入る。`useShoppingList` で `staples ?? []` にそろえる
- [Risk] `normalizeKey` の移動で、S07 の合算と在庫判定の結果が変わる
  → 関数の中身は変えずに移し、既存の `aggregate_test.go` と移したテーブルテストをそのまま通す
- [Risk] `family/main` の期間に main でスキーマ変更が入ると、`00011` が重なって goose が失敗する
  → [docs/release.md](../../../docs/release.md) のとおり、この期間は main でスキーマを変えない
- [Trade-off] 状態の切替に「元に戻す」が無い。押し間違えたら、もう一度押して戻す。
  3 つのボタンのどれを押しても結果が画面で見えるので、トーストで知らせる必要は薄い

## Migration Plan

- `00011_staples.sql` は表を 1 つ作るだけで、既存の表には触れない。既存の DB は空の常備品の欄から始まる
- マイグレーションの前に、起動時の控えの仕組み (shell の「データベース更新前の控え」) が DB を複製する
- goose は前進のみなので、このバージョンの DB を古いバージョンで開くことはできない。リリースでは MINOR を上げる
- Down は `DROP TABLE staples` とする
