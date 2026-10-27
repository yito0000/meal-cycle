## Why

家族で使う場合も、アプリを操作するのは献立を担当する 1 人とする前提を docs に置いた ([product.md §1「家族で使う場合」](../../../docs/product.md))。
この前提では、ほかの家族が牛乳を飲んでも、食パンを食べても、その消費はアプリに届かない。
S05 は量と賞味期限で在庫を持つので、こうした品を登録すると、量はすぐ実物とずれる。
ずれた量で S07 の在庫判定をすれば、牛乳は「在庫あり」のまま切れてしまうだろう。

量を正しく保つ手段は無い。
操作する人が知り得るのは、冷蔵庫を開けたときに見た「まだある」「残り少ない」「もう無い」という状態だけである。
そこで、牛乳や食パンのように家族が日常的に食べる品を **常備品** として、量を持たず状態だけを持つ在庫に分ける。
「少ない」「ない」にした常備品は、献立に関係なく S07 に載り、買い忘れを防ぐ。

## What Changes

- S05 に常備品の欄を足す
  - 常備品は食材名と状態 (「ある / 少ない / ない」) だけを持つ。量、単位、賞味期限は持たない
  - 欄の中で常備品を追加する。追加した常備品の状態は「ある」から始まる
  - 各常備品の状態は、3 つの選択肢を押すだけで切り替わり、その場で保存される
  - 削除は確認を挟まず即実行し、トーストの「元に戻す」で状態ごと復元する (在庫の削除と同じ作法)
  - 表記をそろえた食材名が同じ常備品は、2 つ登録できない
  - 並び順は食材名の昇順で固定する。状態を切り替えても行は動かない
  - S05 の検索と並び替えは、これまでどおり在庫のテーブルだけに効く
- S07 に常備品の欄を足す
  - 状態が「少ない」「ない」の常備品を、対象期間の献立に関係なく並べる。「ない」を先に、それぞれ食材名の昇順とする
  - 各項目に食材名と状態を出し、期間の献立で使うときはその用途 (レシピ名) も添える
  - 各項目の「買った」で状態を「ある」に戻し、欄から外す。トーストの「元に戻す」で元の状態に戻せる
  - 期間に献立が無いときも、すべて作った印付きのときも、欄は表示する
- 献立の材料のうち、表記をそろえた食材名が常備品と同じものは、材料テーブルにも「家にあるか確かめる材料」にも出さない
  - 常備品が「少ない」「ない」なら、その材料のレシピ名を常備品の欄の用途に添える
  - 常備品が「ある」なら、どこにも出さない。量で突き合わせても実物とずれるので、状態の判断を優先する
- S07 の「テキストでコピー」に、常備品の欄の項目を含める
  - 材料テーブルの行のあとに見出し「常備品」を置き、「- 牛乳」「- 食パン (少ない)」の形で並べる
  - 献立や材料が無い期間でも、コピーする常備品があればボタンを表示する

### Non-goals

- **常備品の名前の変更**：誤記は削除して登録し直す。状態しか持たないので、登録し直して失うものが無い
- **在庫と常備品の相互変換**：S05 の在庫の行を常備品に移す操作や、その逆は持たない
- **同じ名前の在庫の登録を止めること**：常備品と同名の在庫も、これまでどおり S05 に追加できる。S07 の材料の扱いだけが常備品に寄る
- **S07 のチェックと一括登録に常備品を乗せること**：常備品は量を持たないので、量を入れるダイアログに乗せない。「買った」だけで戻す
- **状態の変化の履歴や、「少ない」にしてからの経過日数**：状態は最新の 1 つだけを持つ
- **S09 の設定項目**：状態の段階や並び順に、利用者ごとに変えたい値が無い
- **在庫の見直しを速くする手段、世帯人数による換算、献立のテキストコピー**：[design.md §9](../../../docs/design.md) のとおり別の proposal にする

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `fridge`：S05 に常備品の欄を足す。常備品の永続化、一覧、追加、状態の切替、削除と元に戻す、入力検証の要件を追加する
- `shopping`：S07 に常備品の欄と「買った」を足す。常備品と同名の材料を材料テーブルと「家にあるか確かめる材料」から外す。
  空状態で常備品の欄を出す条件と、テキストコピーに常備品を含める規則を変える

## Impact

**新設ファイル**

- `internal/storage/sqlite/migrations/00011_staples.sql`：`staples` テーブル
- `internal/domain/fridge/staple.go`：`Staple` と `StapleState`
- `internal/domain/ingredient/name.go`：食材名の表記をそろえる `NormalizeName` (`shopping.normalizeKey` から移す)
- `internal/domain/ingredient/name_test.go`：`internal/domain/shopping/normalize_test.go` のケースを移す
- `internal/storage/sqlite/staple_repo.go` と `internal/storage/sqlite/staple_repo_test.go`
- `internal/domain/shopping/staple.go` と `internal/domain/shopping/staple_test.go`：常備品の欄の項目の算出
- `frontend/src/components/fridge/use-staples.ts`：常備品の query と mutation
- `frontend/src/components/fridge/staple-panel.tsx`：S05 の常備品の欄 (追加行と一覧)
- `frontend/src/components/fridge/staple-row.tsx`：常備品 1 行 (状態の切替と削除)
- `frontend/src/components/fridge/staple-state-toggle.tsx`：「ある / 少ない / ない」の切替
- `frontend/src/components/shopping/shopping-staple-list.tsx`：S07 の常備品の欄
- `frontend/src/components/shopping/shopping-text.test.ts`：テキストコピーの組み立ての単体テスト
- `frontend/e2e/fridge-staples.spec.ts` と `frontend/e2e/shopping-staples.spec.ts`
- `docs/wireframes/S05_fridge.v2.drawio.svg`：常備品の欄を加えた S05 の構成

**更新ファイル**

- `internal/domain/fridge/errors.go`：常備品のエラーコードを追加
- `internal/domain/shopping/normalize.go`：`normalizeKey` を `ingredient.NormalizeName` に委ねる。`normalize_test.go` は削除する
- `internal/domain/shopping/list.go`：`StapleItem` と `List.Staples` を追加
- `internal/domain/shopping/aggregate.go` と `aggregate_test.go`：常備品と同名の材料を除く
- `internal/service/fridge.go` と `fridge_test.go`：`ListStaples` / `AddStaple` / `SetStapleState` / `DeleteStaple` / `RestoreStaple`
- `internal/service/shopping.go` と `shopping_test.go`：常備品を読んで `List` に載せる
- `frontend/src/routes/fridge.tsx`：常備品の欄を在庫の右 (狭い幅では下) に置く
- `frontend/src/routes/shopping.tsx`：常備品の欄、空状態、コピーのボタンの表示条件
- `frontend/src/components/fridge/error-messages.ts`：常備品のエラーコードを i18n key に対応づける
- `frontend/src/components/shopping/use-shopping-list.ts`：`staples` の null を空配列にそろえる
- `frontend/src/components/shopping/shopping-text.ts`：常備品の行を組み立てに加える
- `frontend/src/components/shopping/copy-shopping-text-button.tsx`：常備品を受け取って組み立てに渡す
- `frontend/src/locales/ja.json` と `frontend/src/locales/en.json`
- `frontend/e2e/bindings-shim/fridgeservice.ts`：常備品のメソッドを追加
- `frontend/e2e/helpers.ts`：`addStaple` / `setStapleState` / `stapleRow` / `shoppingStapleItem` を追加
- `docs/product.md`：§1「家族で使う場合」、§4 S05 と S07、§7 決定事項
- `docs/design.md`：§4 の常備品のデータモデル、§9 の候補に proposal 名を添える
- `docs/wireframes/README.md`：`S05_fridge.v2.drawio.svg` の行を追加

**生成物 (再生成のみ)**

- `frontend/bindings/`：`wails3 generate bindings -clean=true -ts -i`

**migration**

- `00011_staples.sql`：常備品 1 品 = 1 行の `staples` テーブルを作る

**後続 proposal への引き継ぎ**

- 在庫の見直しを速くする手段は、S05 の在庫の行を対象にする。常備品の状態の切替はこの proposal で 1 クリックになっているので、対象に含めない
- 献立のテキストコピーは、常備品を含めない。常備品は買い物の話で、献立を家族に渡す目的とは別である
- 常備品の名前の変更が要ると感じたら、行内編集を足す proposal にする。重複の検査はこの proposal の規則を流用できる
- 食材マスタ (`add-ingredient-master`) を入れるときは、`staples.name` も `stock_items.name` と同時に移す
