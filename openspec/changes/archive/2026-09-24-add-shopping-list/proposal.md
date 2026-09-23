## Why

献立を組み終えたあと、その期間にどんな材料がどれだけ要るかは、今のところ利用者が頭の中で拾い集めるしかない。
S01 に並んだレシピの材料を 1 つずつ見て、冷蔵庫 (S05) にあるかを思い出す作業である。
product.md §1 のナラティブは「組み終えたら必要な食材が自動で集計される」ことを前提にしており、
MVP 4 機能のうち買い物リスト (S07) だけがまだプレースホルダのまま残っている。

前提はすでにそろっている。
献立は作り置き単位 (`batch_id`) で保存され、mealplan spec は「買い物リストは作り置き単位で 1 セットの材料を計上する」と
約束済みである。在庫も add-fridge-inventory で SQLite に入った。
本 proposal では、この 2 つを突き合わせて、期間の献立で使う材料の一覧を S07 に出す。

このリストに求めるのは厳密な不足量の計算ではなく、買い物の前に眺める材料の目安である。
食材名は自由テキストで単位換算も無いため、在庫との突き合わせはどうしても近似になる。
そこで不足量を引き算して見せるのをやめ、期間に要る材料をすべて並べたうえで、在庫の有無を 3 段階の状態で添える。
対象期間も、S01 の週単位に縛らず、S07 上で開始日と日数 (3 / 4 / 7 / 14 / 28 日) を選べるようにする。
作り置きで 2〜3 日ずつ回すペルソナは、次の買い物までの数日分だけを見たい場面が多いためである。
材料の横には、計上の元になった献立の一覧を並べる。どの献立を何回分として数えたかが見えれば、
期間の頭に前から続く作り置きが混ざっていても、利用者が自分で気づいて開始日をずらせる。

design.md §8 の順序案どおり、購入済チェックと冷蔵庫への一括登録は後続の `add-shopping-purchase` に分ける。

## What Changes

- `shopping` capability を新設する。買い物リストは保存せず、開くたびに献立と在庫から算出する。
- Wails サービス `ShoppingService` を追加し、`List(periodStart, periodDays)` を公開する。
  返り値は対象期間 (開始日と終了日と日数)、計上した献立 (調理 1 回分ごと) の配列、材料の行の配列である。
- 集計規則は次のとおり。いずれも Go 側のドメインで行う。
  - 作り置きは 1 回分として数える。同じ作り置きに属する行が期間内に何日分あっても、材料は 1 セットだけ計上する
  - 食材の突き合わせは「trim 後の食材名 + 単位」の完全一致で行う。同名でも単位が違えば別の行になる
  - 各行に在庫の状態を付ける。同じ食材名と単位の在庫 (期限切れを除く) の合計が必要量以上なら「在庫あり」、
    0 より多く必要量未満なら「一部在庫あり」、無ければ「在庫なし」とする
  - 量が 0 の材料 (塩少々など) も他の材料と同じように並べる。量の欄は「—」とし、在庫が 1 件でもあれば「在庫あり」になる
- S07 画面 (`/shopping`) を実装する。上から「見出し」「開始日と日数の切替」「対象期間の表示」
  「食材 / 必要量 / 在庫 / 用途の 4 列テーブル」を置く。
  - 必要量の列には期間の必要量の合計を出す (在庫を引いた量ではない)
  - 在庫の列には「在庫なし / 一部在庫あり / 在庫あり」を出す。「在庫あり」の行は薄く表示し、リストの末尾にまとめる
  - 用途の列には、その食材を使うレシピ名を献立の一覧と同じ順 (日付、時間帯、S01 のセル内の順) に重複なく並べる
  - 材料テーブルの右に「この期間の献立」欄を置く。1 行 = 計上する調理 1 回分で、レシピ名、期間内の日付、時間帯を出す。
    作り置きには 🔗、期間より前から続く作り置きには「期間前から」の印を付ける。参照専用で、計上から外す操作は無い。
    画面の幅が足りないときは材料テーブルの下に回す
  - 献立が 1 件も無い期間と、献立はあるがレシピに材料が登録されていない期間とで、異なる空状態メッセージを出す
- 対象期間の決まり方を定める。
  - S01 から開いた場合は S01 の表示期間 (開始日と 7 / 14 / 28 日) を初期値にする
  - サイドバーから開いた場合は「今日から 7 日間」を初期値にする
  - S07 上で開始日と日数を切り替えると、その期間で算出し直す
- 日数プリセットと既定日数を `internal/config/defaults.go` に追加し、product.md §1 の可変値一覧にも載せる。
- S01 献立カレンダーの下部に「この期間の買い物リストを見る」ボタンを追加し、表示中の期間を渡して S07 へ遷移する。
- `docs/scenarios/shopping.feature` を今回の仕様に合わせて書き換える。在庫にある食材を除外する前提の 2 シナリオは、
  在庫の状態を表示するシナリオに置き換える。

### Non-goals

- **購入済チェックと冷蔵庫への一括登録**：`add-shopping-purchase` で扱う。ワイヤーにあるチェックボックス列と
  下部の「☑ を冷蔵庫に登録して完了」ボタンは本 proposal では描画しない。
- **不足量の算出と表示**：必要量から在庫量を引いた値や、その内訳は出さない。在庫の状態は 3 段階の目安にとどめる。
- **単位違いの在庫の扱い**：同名で単位だけが違う在庫は突き合わせず、補足表示もしない。
- **人数による量の換算**：献立側に人数の情報が無いため、レシピに書かれた量をそのまま使う。
- **単位換算と表記ゆれの吸収** (g と kg、「玉ねぎ」と「たまねぎ」)：将来の `add-ingredient-master` で扱う。
- **任意の日数や終了日の指定**：日数はプリセットから選ぶ。プリセットの変更は S09 設定画面 (MVP 後) で扱う。
- **S07 で選んだ期間の保持**：サイドバーから開き直すと既定期間に戻る。
- **買い物リストの保存と編集** (行の手動追加や削除、量の手修正)：リストは常に算出結果であり、手で編集しない。
- **献立の一覧からの操作**：計上から外す、レシピ詳細を開く、S01 の該当セルへ飛ぶ、といった操作は付けない。
- **印刷やテキスト出力**：外部に持ち出す手段は MVP では用意しない。
- **Frontend テスト runner の導入**：Vitest は未導入のまま。フロントは type-check / build:dev / E2E で検証する。

## Capabilities

### New Capabilities

- `shopping`：献立と冷蔵庫在庫から、期間の献立で使う材料を算出して S07 に一覧表示する
  (対象期間の決定と切替、作り置き単位の計上、同一食材の合算、在庫の状態、期間の献立一覧、空状態)。

### Modified Capabilities

- `mealplan`：S01 から表示中の期間を引き継いで S07 を開く導線 (「この期間の買い物リストを見る」ボタン) を追加する。

## Impact

### 新設ファイル

- `internal/domain/shopping/list.go`：`PlannedBatch` / `Dish` / `StockAmount` / `StockStatus` / `Item` / `List` 型
- `internal/domain/shopping/errors.go`：`ErrInvalidPeriodDays`
- `internal/domain/shopping/aggregate.go`：集計の純関数 `Aggregate` (作り置き単位の材料展開、合算、在庫の状態判定、並び替え)
- `internal/domain/shopping/aggregate_test.go`：`Aggregate` の table-driven test
- `internal/storage/sqlite/shopping_repo.go`：`ShoppingRepo.ListPlannedBatches` (期間内の作り置き、日付と時間帯、材料の読み出し)
- `internal/storage/sqlite/shopping_repo_test.go`：実 SQLite での読み出しテスト
- `internal/service/shopping.go`：`ShoppingService` と入力検証、期限切れ在庫の除外
- `internal/service/shopping_test.go`：サービス層の BDD 風テスト
- `frontend/src/components/shopping/shopping-config.ts`：日数プリセットと既定日数のフロント側 mirror
- `frontend/src/components/shopping/shopping-period.ts`：search param の検証と、既定期間の算出
- `frontend/src/components/shopping/shopping-period-picker.tsx`：開始日の入力と日数プリセットの切替
- `frontend/src/components/shopping/shopping-table.tsx`：材料テーブル
- `frontend/src/components/shopping/stock-status-badge.tsx`：在庫の状態の表示
- `frontend/src/components/shopping/shopping-dish-list.tsx`：「この期間の献立」欄
- `frontend/src/components/shopping/format-dish-dates.ts`：献立の日付の範囲表記 / 列挙表記の整形
- `frontend/src/components/shopping/use-shopping-list.ts`：TanStack Query の query フック
- `frontend/src/lib/format-amount.ts`：量の表示整形。fridge の `formatAmount` を移設して S05 と S07 で共有する
- `frontend/e2e/bindings-shim/shoppingservice.ts`：E2E 用 shim
- `frontend/e2e/shopping.spec.ts`：`shopping.feature` のシナリオ名に揃えた E2E

### 更新ファイル

- `internal/config/defaults.go`：`ShoppingPeriodDayPresets` ([3, 4, 7, 14, 28]) と `ShoppingDefaultPeriodDays` (7) を追加
- `internal/config/defaults_test.go`：上記 2 値の既定値テストを追加
- `main.go`：`application.Options.Services` に `service.NewShoppingService(db)` を追加
- `cmd/e2e-server/main.go`：`newRegistry` に `ShoppingService` を追加し、コメントのサービス数を 5 に更新
- `cmd/e2e-server/dispatch_test.go`：`TestNewRegistry` の期待サービス一覧に `ShoppingService` を追加
- `frontend/e2e/bindings-shim/index.ts`：`ShoppingService` を re-export と型突き合わせに追加
- `frontend/e2e/helpers.ts`：`openShoppingList` と、レシピ作成 / 割当 / 作り置きの API ヘルパを追加
- `frontend/src/router.tsx`：`/shopping` ルートに `validateSearch` (`from` / `days`) を追加
- `frontend/src/routes/shopping.tsx`：プレースホルダを S07 本体に差し替え
- `frontend/src/routes/meal-plan.tsx`：下部に「この期間の買い物リストを見る」ボタンを追加
- `frontend/src/components/fridge/stock-form.ts`：`formatAmount` を削除 (`frontend/src/lib/format-amount.ts` へ移設)
- `frontend/src/components/fridge/fridge-item-row.tsx`：`formatAmount` の import 元を `@/lib/format-amount` に変更
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.shopping.*` を追加し、`placeholder` を削除。
  `pages.meal_plan.shopping_cta` を追加
- `docs/scenarios/shopping.feature`：シナリオの書き換えと追加
- `docs/product.md`：§1 の可変値一覧に日数プリセットと既定日数を追加し、§4 S07 を今回の仕様に書き換える

### 生成物 (再生成のみ)

- `frontend/bindings/`：`wails3 generate bindings` で `ShoppingService` と `shopping` モデルを生成

### migration

- なし。既存の `meal_plans` / `recipes` / `recipe_ingredients` / `stock_items` を読むだけで、スキーマは変えない

### 後続 proposal への引き継ぎ

- `add-shopping-purchase`：チェック状態の保持先 (DB / localStorage / メモリ) はそちらで決める。
  行の識別子は本 proposal の「食材名 + 単位」の組を使える。一括登録は `FridgeService.Add` を再利用する。
  チェック対象を「在庫あり」以外の行に絞るかどうかもそちらで決める
- `add-ingredient-master`：突き合わせキーを「食材名 + 単位」から食材 ID に差し替える。単位換算を入れるなら
  `Aggregate` の合算と在庫の状態判定の箇所に閉じる
- `add-mealplan-cook-consume`：調理済みを記録できるようになったら、献立の一覧の「期間前から」の印に加えて、
  調理済みの作り置きを計上から外す条件を足せる
- S09 設定画面：`ShoppingPeriodDayPresets` / `ShoppingDefaultPeriodDays` を設定ストアに移す。
  フロントの mirror (`shopping-config.ts`) も ConfigService 経由に置き換える
