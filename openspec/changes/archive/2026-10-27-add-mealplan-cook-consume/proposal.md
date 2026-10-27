## Why

S01 の「作った」印は、調理を終えたことを記録し、S07 の計上から外すところまでで止まっている (add-mealplan-cooked-mark)。
料理で使った材料の分だけ在庫を減らすのは、今も利用者が S05 を開いて手で直す作業である。
覚えていれば直すが、忘れれば S05 の在庫は実物より多いまま残る。
在庫が実物より多いと、S07 は足りない材料を「在庫あり」と示し、買い忘れにつながる。
この先に入れる献立候補の「使い切り」軸 (add-mealplan-suggest-by-stock) も、使い終えた食材を「期限が近い在庫」として勧めかねない。

印を付けた時点で、そのレシピの材料と同名の在庫を減らす候補を示せば、S05 を開く手間が省ける。
ただし材料と在庫の突き合わせは食材名と単位の近似にすぎず、レシピの分量どおりに使ったとも限らない。
そこで自動では減らさず、減らす量を直せる確認のダイアログを挟む。

## What Changes

- S01 で品に「作った」印を付けたとき、減らせる在庫があれば「使った在庫を減らす」ダイアログを開く
  - 印はダイアログを開く前に付く。ダイアログで何を選んでも、印は付いたままになる
  - 減らせる在庫が 1 件も無いときは、ダイアログを開かない
  - 印を外したときは、ダイアログを開かず、在庫も変えない
- ダイアログには、レシピの材料ごとに、同じ食材名と単位を持つ使える在庫の行を並べる
  - 食材名と単位は、S07 と同じく表記をそろえて比べる
  - 期限切れの在庫と、量を持たない材料 (量 0 と量る単位の材料) は対象にしない。常備品と同名の材料も対象にしない
  - 各行に食材名、賞味期限、今の量、減らす量を出す。減らす量は直せる
  - 減らす量の初期値は、材料の量を賞味期限の近い在庫から順に割り当てた値とする
  - 減らせる在庫の無い材料は、ダイアログの下に「在庫が見つからない材料」として名前だけを並べる
- 「減らす」で、全行の減らす量を 1 回でまとめて在庫に反映する
  - 量が 0 になった在庫の行は削除する。減らす量が 0 の行は変えない
  - 1 行でも不正な値があれば、どの在庫も変えない
  - 反映後、減らした在庫の件数を通知する
- 「減らさない」で、在庫を変えずにダイアログを閉じる

### Non-goals

- **印を外したときに在庫を戻すこと**：何をどれだけ減らしたかを保存しないので、戻す量が決まらない。戻したいときは S05 で直す
- **減らした在庫の「元に戻す」**：確認のダイアログで量を確かめてから反映するので、S07 の一括登録と同じく取り消しを持たない
- **人数による換算**：レシピの人数と世帯人数による換算は、世帯人数による換算の proposal で扱う
- **単位の換算**：「g」と「個」のように単位が違う在庫は、S07 と同じく同じ食材とみなさない
- **作り置きを広げた行や D&D で動かした行で開くこと**：印を引き継いだ行では在庫を減らさない。調理は 1 回なので、ダイアログは印を付けた操作のときだけ開く

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：「作った」印を付けたときに「使った在庫を減らす」ダイアログを開く要件を追加する
- `fridge`：料理で使った分の在庫の減らし方 (候補の算出、まとめての反映、検証) の要件を追加する

## Impact

**前提**

- `add-fridge-staples` を先に入れる。常備品と同名の材料を対象から外す規則は、常備品の名前の一覧を使う

**新設ファイル**

- `internal/domain/fridge/consume.go`：`ConsumptionPlan` / `ConsumptionLine` / `Consumption` と、候補を割り当てる `PlanConsumption`
- `internal/domain/fridge/consume_test.go`
- `frontend/src/components/mealplan/consume-stock-dialog.tsx`：「使った在庫を減らす」ダイアログ
- `frontend/src/components/mealplan/consume-form.ts`：ダイアログの行の初期値と検証
- `frontend/src/components/mealplan/consume-form.test.ts`
- `frontend/e2e/mealplan-cook-consume.spec.ts`

**更新ファイル**

- `internal/domain/fridge/errors.go`：消費のエラーコードを追加
- `internal/storage/sqlite/stock_repo.go` と `stock_repo_test.go`：複数行の量を 1 トランザクションで減らす `Consume` を追加
- `internal/service/fridge.go` と `fridge_test.go`：`PlanCookConsumption(entryID)` と `Consume([]Consumption)` を追加
- `frontend/src/components/mealplan/week-calendar.tsx`：`toggle-cooked` で印を付けたあとに候補を取り、ダイアログを開く
- `frontend/src/locales/ja.json` と `frontend/src/locales/en.json`
- `frontend/e2e/bindings-shim/fridgeservice.ts`
- `docs/product.md`：§4 S01 の「作った」チェックボックスの項と、§1 ナラティブの「在庫と献立の自動連動は MVP 後」の記述
- `docs/design.md`：§9 の「料理実施と在庫消費の連動」に proposal 名を添える

**生成物 (再生成のみ)**

- `frontend/bindings/`：`wails3 generate bindings -clean=true -ts -i`

**migration**

- なし。減らした内訳は保存しない

**後続 proposal への引き継ぎ**

- 世帯人数による換算を入れるときは、`PlanConsumption` に渡す材料の量を換算後の量にする
- 献立候補の「使い切り」軸は、この proposal で在庫が減ることを前提に、期限が近い在庫を数える
