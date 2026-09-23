## Context

S07 は現状プレースホルダ (`frontend/src/routes/shopping.tsx`) だけで、Go 側に買い物リスト用のサービスは無い。
動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/shopping/spec.md](specs/shopping/spec.md) を参照。

集計に使うデータはすでに 3 つのテーブルにそろっている。
設計を縛るのは、それぞれの持ち方の癖である。

- `meal_plans` は 1 行 = 1 品で、作り置きは `batch_id` の共有で表す。
  mealplan spec「作り置きの意味付け」は、買い物リストが作り置き単位で 1 セットの材料を計上すると約束している
- 品の入れ替え (`MealPlanRepo.Replace`) は `batch_id` を変えずに `recipe_id` だけを差し替える。
  そのため、1 つの `batch_id` に異なるレシピの行が混ざる状態が起こりうる
- `recipe_ingredients` と `stock_items` の食材名はどちらも自由テキストで、食材マスタも単位換算も無い。
  `recipe_ingredients.amount` には 0 を入れられる (レシピ入力の検証は量の範囲を見ていない)
- 在庫の期限状態は `fridge.Evaluate` が Go 側で判定する。閾値は `config.Defaults()` から読む
- S01 の `periodStart` は React の state で、永続化していない (週数だけが localStorage に残る)。
  ルータは memory history なので、URL を介した状態共有は router の search param に限られる
- ユーザー可変な値は `internal/config/defaults.go` に集め、フロントは `mealplan-config.ts` のような手動 mirror で読む運用である

この条件で厳密な不足量を出そうとすると、表記ゆれと単位違いのせいで数字の見かけの精度だけが上がる。
本 change は、リストを「期間の献立で使う材料の目安」と位置づけ、在庫との関係を 3 段階の状態に丸める (proposal.md Why)。

## Goals / Non-Goals

**Goals:**

- 作り置きの数え方と在庫の状態判定を、Go のドメイン層 1 箇所の純関数に閉じ込め、table-driven test で固める
- 期間を「開始日 + 日数」で扱い、S01 経由でもサイドバー経由でも S07 上の切替でも、サービスの呼び方を 1 通りにする
- 後続の `add-shopping-purchase` と `add-ingredient-master` が、集計の中身に手を入れずに差し替え点へ触れる形にする
- 材料が何から計上されたかを、献立の一覧として同じ 1 回の取得で返し、計上の単位と一覧の行を 1 対 1 に揃える

**Non-Goals:**

- 集計結果のキャッシュや差分更新。1 期間の献立は多くて数十行、材料は数百行規模であり、開くたびの全件算出で足りる
- 献立や在庫を変更したときの S07 への即時反映。S07 を開いたときに再取得すれば足りる (D8)

## Decisions

### D1. 集計を置く場所

**選択肢:**

- (a) フロントで組み立てる。`MealPlanService.ListPeriod` と `RecipeService.GetByID` と `FridgeService.List` を呼び、TS で集計する
  — Pros: Go 側の追加が無い
  / Cons: `ListPeriod` は週数 (1 / 2 / 4) しか受けず、3 日や 4 日の期間を取れない。
  集計ロジックがテストランナーの無いフロントに入り、E2E でしか検証できない。レシピの数だけ IPC が往復する
- (b) SQL で読み出しまでを行い、合算と状態判定は Go のドメイン層の純関数で行う ← 採用
  — Pros: 作り置きのまとめ上げは SQL の `GROUP BY` で済み、合算規則は DB 抜きで table-driven test にできる
  / Cons: 新しいサービスと repo が 1 つずつ増える
- (c) 合算と状態判定まで SQL 1 本で行う
  — Pros: Go 側のコードが最小
  / Cons: 期限切れ判定 (ローカル日付と閾値) を SQL に書くことになり、条件分岐の検証が実 DB 経由のテストに寄る

**採用: (b)**。repo (`ShoppingRepo.ListPlannedBatches`) は「期間内の作り置きと、そのレシピの材料」だけを返し、
ドメイン (`shopping.Aggregate`) が合算、状態判定、並び替えを行う。サービスは両者と在庫の読み出しをつなぐ。

### D2. サービスと repo の配置

**選択肢:**

- (a) `MealPlanService` に `ListShopping` を足す — Pros: ファイルが増えない
  / Cons: capability が `mealplan` と `shopping` にまたがり、CLAUDE.md の「1 capability = 1 サービスファイル」から外れる
- (b) `internal/service/shopping.go` に `ShoppingService` を新設する ← 採用
  — Pros: 規約どおり。後続の `add-shopping-purchase` もここにメソッドを足せる / Cons: e2e-server の登録と shim が 1 つずつ増える

**採用: (b)**。`NewShoppingService(db *sql.DB)` は `FridgeService` と同じく内部で `sqlite.NewShoppingRepo(db)` と
`sqlite.NewStockRepo(db)` を生成する。今のところ repo を差し替えてテストする需要は無く、サービス層のテストも実 SQLite で書く。
`now func() time.Time` をフィールドに持ち、テストでは固定日付を差し込む (期限切れ判定のため)。

### D3. 作り置きの計上単位

**選択肢:**

- (a) 献立の行ごとに計上する — Pros: 実装が最も単純
  / Cons: 作り置き 3 日分で材料が 3 倍になり、mealplan spec の約束に反する (ユーザー判断でも不採用)
- (b) `(batch_id, recipe_id)` の組ごとに、組の全行 (期間外を含む) の最も早い日付を調理日とし、調理日が期間内の組だけを数える
  — Pros: 期間の頭にある「もう作った作り置きの残り」を除外できる
  / Cons: 期間外の行まで読む必要があり SQL が複雑になる。まだ作っていない作り置きも除外しうる。
  目安のリストに求める精度を超えている (ユーザー判断で不採用)
- (c) 期間内の行だけを `(batch_id, recipe_id)` でまとめ、組ごとに 1 回計上する ← 採用
  — Pros: SQL が `WHERE` + `GROUP BY` だけで済む。入れ替えでレシピが混ざった作り置きでも、レシピごとに材料が出る
  / Cons: 期間の頭に前の期間から続く作り置きがあると、すでに作った料理の材料も載る (D14 の献立一覧で気づけるようにする)

**採用: (c)**。SQL は `meal_plans` を期間で絞ってから `(batch_id, recipe_id)` で `GROUP BY` し、組ごとに期間内の `MIN(date)` を
`FirstDate` として返す。`FirstDate` は用途欄と行の並び順、献立の一覧の並び順に使う。
組の単位を `batch_id` だけにしない理由は、入れ替え後の作り置きで片方のレシピの材料が落ちるのを防ぐためである。

### D4. 食材の突き合わせキー

**選択肢:**

- (a) trim 後の「食材名 + 単位」の完全一致 ← 採用
  — Pros: 予測しやすい。add-fridge-inventory の引き継ぎ事項 (「名前一致は trim 後の完全一致」) と一致する
  / Cons: 「玉ねぎ」と「たまねぎ」、「g」と「グラム」は別の行になる
- (b) NFKC 正規化やかな統一をかけてから比較する — Pros: 全角半角のゆれを吸収できる
  / Cons: 在庫側の候補補完 (S06) は正規化していないため、S05 と S07 で同じ文字列の扱いが食い違う。
  ひらがなとカタカナの同一視は語によって誤判定がある
- (c) 食材名だけで突き合わせ、単位は無視する — Pros: 同名在庫を拾いやすい
  / Cons: 「豚肉 200 g」と「豚肉 1 パック」を同じ量として比べることになり、状態の判定が意味を失う

**採用: (a)**。単位が空文字の材料は「空文字という単位」として扱い、単位が空の在庫とだけ一致する。
同名で単位だけが違う在庫は突き合わせず、補足表示もしない (ユーザー判断)。

### D5. 在庫の状態の表し方

**選択肢:**

- (a) 不足量 (必要量 − 在庫量) を出し、0 以下の行は隠す — Pros: そのまま買い物メモになる
  / Cons: 表記ゆれや単位違いで突き合わせが外れると、数字だけが正確そうに見える。在庫で足りる材料が画面から消え、
  「期間に何を使うか」が見えなくなる (ユーザー判断で不採用)
- (b) 必要量と在庫量の両方を数値で並べる — Pros: 情報量が最大
  / Cons: 列が増え、目安として眺めるには細かすぎる
- (c) 必要量の合計だけを出し、在庫との関係を「在庫なし / 一部在庫あり / 在庫あり」の 3 段階で添える ← 採用
  — Pros: 全材料が見え、買うべきものは状態と並び順で分かる。突き合わせの近似が数字の精度として表に出ない
  / Cons: 「一部在庫あり」のときにいくつ買えばよいかは利用者が判断する

**採用: (c)**。`shopping.StockStatus` を `none` / `partial` / `covered` の 3 値で定義する。
在庫量 `s` と必要量 `r` から、`s == 0` なら `none`、`s > 0 && s < r` なら `partial`、`s > 0 && s >= r` なら `covered` とする。
量 0 の材料 (`r == 0`) は同じ式で `none` か `covered` のどちらかになり、特別扱いのコードを持たない。
並び順は `covered` 以外を上、`covered` を下にまとめ、それぞれの中を `FirstDate`、食材名、単位の昇順にする。

### D6. 期限切れ在庫の除外

**選択肢:**

- (a) 期限切れ在庫も在庫に数える — Pros: S05 の一覧と量が一致する / Cons: 使えない在庫のせいで「在庫あり」と出る
- (b) `fridge.Evaluate` が `expired` を返す在庫を除外する ← 採用 (ユーザー判断)
  — Pros: 期限判定を S05 と同じ関数に揃えられる。今日が期限の在庫は `urgent` なので数えられる
  / Cons: 期限切れを捨てずに使うつもりの利用者には「在庫なし」と出る

**採用: (b)**。サービスが `s.now()` と `s.cfg.ExpiryAlertRedDays` / `s.cfg.ExpiryAlertYellowDays` で `Evaluate` を呼び、
`expired` 以外の在庫だけを `shopping.StockAmount` に変換して `Aggregate` に渡す。閾値は色分け用で結果に影響しないが、
`Evaluate` のシグネチャに合わせて config から読む (リテラルを書かない規約のため)。

### D7. 量の計算と比較

**選択肢:**

- (a) `float64` のまま合算し、必要量と在庫量を小数第 2 位で丸めてから比較する ← 採用
  — Pros: `recipe_ingredients.amount` / `stock_items.amount` が REAL で、既存の型と揃う
  / Cons: 小数第 3 位以下の差は判定に効かない
- (b) 小数ライブラリ (`shopspring/decimal` 等) を入れる — Pros: 誤差が出ない / Cons: 依存が 1 つ増え、目安の用途には過剰
- (c) 1/1000 単位の整数に変換して計算する — Pros: 依存なしで誤差が出ない / Cons: 入力が小数第 3 位を超えると結局丸めが要る

**採用: (a)**。`0.1 + 0.2` と `0.3` の比較で「一部在庫あり」に倒れるのを防ぐため、比較の前に丸める。
丸めの桁数は表示の桁数 (fridge の `formatAmount` と同じ小数第 2 位) に合わせる。ユーザーが変える値ではないので
`config` ではなく `internal/domain/shopping/aggregate.go` の定数に置く。返す必要量も丸めた値にする。

### D8. 期間の受け渡しと切替

**選択肢:**

- (a) 期間を localStorage に保存し、S01 と S07 で共有する — Pros: サイドバーから開いても前回の期間が出る
  / Cons: product.md は S01 の開始日を毎回リセットすると決めており、保存すると意味が変わる。サイドバーからの既定期間とも矛盾する
- (b) React context などのアプリ内共有 state に置く — Pros: 保存しない
  / Cons: 画面を離れても期間が残り、「サイドバーから開いたら既定期間」を実現するには別途リセットが要る
- (c) 期間を router の search param (`from` / `days`) に置く ← 採用
  — Pros: S01 の CTA も S07 上の切替も「search を書き換えて遷移する」1 つの操作になる。
  サイドバーのリンク (search 無し) は自然に既定期間になり、S07 で選んだ期間が残らない (spec どおり)。S01 の state 設計に触れない
  / Cons: search の検証 (zod) を 1 つ足す必要がある

**採用: (c)**。`/shopping` ルートに `validateSearch` を足し、`from` (YYYY-MM-DD) と `days` (数値) を任意項目で受ける。
両方が妥当 (`days` がプリセットに含まれる) なときだけ採用し、どちらかが欠けるか不正なら既定期間 (今日、既定日数) にする。
この判定は `frontend/src/components/shopping/shopping-period.ts` の純関数に置く。
S01 の CTA は `days = periodWeeks * 7` を渡す。7 / 14 / 28 はいずれもプリセットに含まれる。
S07 上の切替は `navigate({ search, replace: true })` で search を書き換える。memory history なので履歴は溜めない。

既定期間の「今日」はフロントで決めてサービスに渡す。サービスの `List` は開始日と日数を受けるだけにし、期間の算出を持ち込まない。

再取得は TanStack Query の既定 (`staleTime: 0`、マウント時に再取得) に任せる。
queryKey は `["shopping", from, days]` とし、献立や在庫の mutation 側から明示的に invalidate しない。
S07 を開くたびにマウントされるので、他画面での変更は開き直した時点で反映される。

### D9. 日数プリセットの置き場所と検証

**選択肢:**

- (a) 日数を 1〜28 の範囲で受け、プリセットは UI だけが持つ — Pros: サービスの検証が単純
  / Cons: プリセットの定義が UI 側だけになり、Go の config に集約する規約から外れる
- (b) `config.Defaults()` に `ShoppingPeriodDayPresets` と `ShoppingDefaultPeriodDays` を置き、サービスはプリセットに含まれる日数だけを受ける ← 採用
  — Pros: CLAUDE.md の「可変値は 1 箇所に集約」に従う。`LeftoverCopyPresets` と同じ運用になり、S09 で差し替える箇所も揃う
  / Cons: フロントに手動 mirror (`shopping-config.ts`) が要り、Go と同期を保つ必要がある

**採用: (b)**。既定値は `ShoppingPeriodDayPresets = [3, 4, 7, 14, 28]`、`ShoppingDefaultPeriodDays = 7`。
source of truth は `defaults_test.go` で、product.md §1 の可変値一覧にも同じ値を載せる。
プリセットに無い日数は `shopping.ErrInvalidPeriodDays` を返す。
S01 の 28 日 (4 週表示) を受けられるよう、28 をプリセットに含める。

### D10. 返り値の形

`ShoppingService.List` は次の `shopping.List` を返す (JSON タグは camelCase、スライスは nil でなく空配列)。

| フィールド | 内容 |
|---|---|
| `periodStart` / `periodEnd` / `periodDays` | 対象期間。`periodEnd` は期間に含まれる最後の日 (表示用) |
| `dishes` | 計上した献立 (`Dish`) の配列。1 要素 = `(batch_id, recipe_id)` の組 1 つ。空なら「献立が無い」空状態 (D14) |
| `items` | 材料の行 (`Item`) の配列。並び順は spec どおりにサーバ側で確定済み |

`Item` は `name` / `unit` / `amount` (丸めた必要量の合計) / `stockStatus` (`none` / `partial` / `covered`) /
`recipes` (レシピ名、期間内で最初に出てくる日の順) を持つ。在庫量の数値は返さない (D5)。

空状態は `dishes` と `items` の組み合わせで判定する。`dishes` が空なら「献立が無い」、`dishes` があって `items` が空なら「レシピに材料が無い」。

### D11. 量の表示整形の共有

**選択肢:**

- (a) S07 用に整形関数を新しく書く — Pros: fridge に触れない / Cons: S05 と S07 で同じ量の見え方がずれうる
- (b) fridge の `formatAmount` (`frontend/src/components/fridge/stock-form.ts`) を `frontend/src/lib/format-amount.ts` に移し、両画面から使う ← 採用
  — Pros: 表示規則 (整数は小数点なし、小数は第 2 位まで) が 1 箇所になる / Cons: fridge のファイルを 2 つ触る

**採用: (b)**。移動のみで挙動は変えない。量 0 のときの「—」は S07 のテーブル側で出し分け、`formatAmount` には持ち込まない。

### D12. ワイヤーとの差分

ワイヤー `S07_shopping_list.drawio.svg` は初期ビジョンとして凍結し、更新しない (docs/wireframes/README.md の運用)。
本 proposal の実装とワイヤーの差分は次のとおり。

- チェックボックス列と下部の「☑ を冷蔵庫に登録して完了」ボタンは描画しない。`add-shopping-purchase` で足す
- ワイヤーの説明文「必要な食材のうち、冷蔵庫に無いもの」は、在庫で足りる材料も並べるため「この期間の献立で使う材料の目安」に変える
- ワイヤーの見出し補足「(S01 の表示期間に連動)」の代わりに、開始日の入力と日数プリセットの切替を置く
- 「在庫」列を追加し、「在庫あり」の行を薄く表示して末尾にまとめる
- 空状態のメッセージ 2 種を追加する (ワイヤーには無い)
- 開始日の入力はアプリ内のカレンダー (D15) で、日付は yyyy/mm/dd で表示する
- 材料テーブルの右に「この期間の献立」欄を追加する。狭い幅では材料テーブルの下に回す (D14)

### D13. エラーの扱い

開始日の検証は `mealplan.ValidateDate` を再利用して `mealplan.ErrInvalidDate` を返す。
日数の検証は shopping 固有の規則 (プリセットへの所属) なので、`internal/domain/shopping/errors.go` に `ErrInvalidPeriodDays` を新設する。
日付の検証を shopping 側に複製する案も考えたが、同じ「日付文字列の妥当性」を 2 箇所に持つことになるため採らない。
S07 は search param を検証してから呼ぶため、通常の操作でこれらのエラーには到達しない。
到達した場合 (と DB エラー) は、エラー種別で分けずに「買い物リストの読み込みに失敗しました」を表示する。

### D14. 期間の献立一覧

材料の計上の元になった献立を、S07 の右側の欄に一覧する。狙いは、D3 で受け入れた近似 (期間の頭に前から続く作り置きも数える) を、
利用者が画面で確かめられるようにすることである。

**行の単位の選択肢:**

- (a) 日付ごとに、その日の時間帯と品を並べる (S01 の縮小版) — Pros: 見慣れた形
  / Cons: 作り置きが何日にも分かれて出るため、材料を 1 回分しか数えていないことが一覧から読み取れない
- (b) 計上する調理 1 回分 (`(batch_id, recipe_id)` の組) ごとに 1 行にし、期間内の日付と時間帯を添える ← 採用 (ユーザー判断)
  — Pros: 材料の計上と一覧の行が 1 対 1 に対応し、何を何回分として数えたかがそのまま読める
  / Cons: 日付順の献立表としては読みにくい

**操作の選択肢:**

- (a) 参照専用 ← 採用 (ユーザー判断) — Pros: 除外状態の保持先という論点を持ち込まない
  / Cons: もう作った作り置きを外すには開始日をずらすしかない
- (b) 行ごとのチェックで計上から外す — Pros: 期間を変えずに外せる
  / Cons: サービスに除外対象を渡す引数が増え、除外状態の保持も決める必要がある。調理済みの扱いは `add-mealplan-cook-consume` の領分と重なる

**採用: 行の単位は (b)、操作は (a)**。

データは `ShoppingRepo.ListPlannedBatches` の結果をそのまま使う。`PlannedBatch` に次の項目を足す。

- `Dates []string`：期間内でその組が割り当てられた日付 (昇順、重複なし)
- `Meals []string`：期間内でその組が割り当てられた時間帯 (朝 / 昼 / 夕の順、重複なし)
- `BatchMemberCount int`：その `batch_id` の全行数 (期間外を含む)。2 以上なら 🔗 を付ける。S01 の `batchMemberCount` と同じ意味
- `StartedBeforePeriod bool`：同じ `(batch_id, recipe_id)` の行が開始日より前にあるか。SQL の `EXISTS` 副問い合わせで求める

サービスは `PlannedBatch` から材料を除いた `Dish` (`batchId` / `recipeId` / `recipeName` / `dates` / `meals` / `batchMemberCount` /
`startedBeforePeriod`) の配列を
`List.Dishes` として返す。`batchId` と `recipeId` の組は、フロントで行の key に使う一意な識別子である。
並び順は、期間内で最初に出てくる行の (日付, 時間帯, セル内の position) の昇順とし、ドメイン側で確定させる。
そのため `PlannedBatch` には最初の行の時間帯と position (`FirstMeal` / `FirstPosition`) も持たせる。
同じ時間帯の中をレシピ名順にする案も考えたが、S01 で利用者が並べた順 (主菜 → 副菜 → 汁物) と食い違うため採らない。
用途欄のレシピ名もこの順に並べる。

日付の表示 (連続なら「9/24〜26」、飛び飛びなら「9/24, 9/26」) はフロントの純関数 `format-dish-dates.ts` で組み立てる。
表記は言語で変わる可能性があり、Go 側に日本語の書式を持たない方針 (CLAUDE.md の i18n 規約) に従うためである。
月をまたぐ範囲は「9/30〜10/2」のように終端にも月を付ける。

レイアウトは、`lg` 以上の幅で `grid-cols-[1fr_20rem]` の 2 列とし、それより狭いときは 1 列にして献立の欄を材料テーブルの下に置く。
他案として折り畳みで上部に置く案も考えたが、材料と献立を見比べる用途には常に見えている方が合うため採らない (ユーザー判断)。

### D15. 開始日の入力と日付の表記

最初の実装では開始日に `<input type="date">` を使った。WSL の `task dev` (WebKitGTK) で見ると、ポップアップのカレンダーが小さく、
表示形式も mm/dd/yyyy だった。どちらも WebView と OS の言語設定が決める部分で、アプリの CSS や属性では変えられない。
Windows の WebView2 では日本語環境なら yyyy/mm/dd で出る見込みだが、環境ごとに見た目が変わること自体を避けたい。

**選択肢:**

- (a) `<input type="date">` のまま — Pros: 依存が増えない / Cons: 大きさも表示形式も環境任せで、今回の指摘を解消できない
- (b) 年 / 月 / 日の 3 つのセレクトに分ける — Pros: 依存が増えず、形式は固定できる
  / Cons: 曜日が見えず、「来週の月曜から」のような選び方がしにくい
- (c) shadcn/ui の Calendar (`react-day-picker`) を Popover (`@radix-ui/react-popover`) で開く DatePicker にする ← 採用
  — Pros: 大きさ、曜日の見出し、月の表示を Tailwind で決められ、どの WebView でも同じ見た目になる。shadcn の手動コピー運用に沿う
  / Cons: 依存が 2 つ増える。E2E はネイティブの日付入力への `fill` ではなく、カレンダーの日付をクリックする操作になる

**採用: (c)** (ユーザー判断)。共通部品 `src/components/ui/date-picker.tsx` として作り、今回は S07 の開始日にだけ使う。
S05 / S06 の賞味期限の入力も同じ問題を抱えるが、fridge capability の変更になるため後続の小さな change で置き換える。

日付の表記は、開始日のボタンと「対象期間: 2026/09/21 〜 2026/09/27」の両方を「yyyy/mm/dd」に揃える。
内部の値 (search param、サービスへの引数) は従来どおり `YYYY-MM-DD` のままにし、表示の直前だけ `frontend/src/lib/format-date.ts` で変換する。
カレンダーの曜日と月の見出しは i18next の言語 (ja / en) に合わせる (`react-day-picker/locale` の `ja` / `enUS`)。

## Risks / Trade-offs

- [Risk] 期間の頭に前の期間から続く作り置きがあると、すでに作った料理の材料も載る
  → 献立の一覧でその行に「期間前から」の印が付くので、利用者は気づいて読み飛ばすか、開始日をずらして外せる (D14)。
  `add-mealplan-cook-consume` で調理済みを記録できるようになったら、除外条件を足す余地がある
- [Risk] S01 の期間は月曜始まりなので、週の途中に S01 から開くと、過ぎた日の献立の材料も載る
  → S07 上で開始日を今日に変えれば外せる。サイドバーから開けば最初から今日始まりになる
- [Risk] 表記ゆれ (「玉ねぎ」と「たまねぎ」) や単位違いで在庫が突き合わず、「在庫なし」と出る
  → S06 の候補補完がレシピ材料名と直近の単位を候補に出すので、在庫を登録する時点で表記は揃いやすい。
  状態はあくまで目安で、根本対応は `add-ingredient-master`
- [Risk] 「一部在庫あり」では買う量が分からない
  → 必要量の合計は表示しているので、利用者が手元を見て判断する。数値の内訳は、正確さの見かけを避けるため出さない (D5)
- [Risk] フロントの `shopping-config.ts` と Go の `config.Defaults()` がずれる
  → `mealplan-config.ts` と同じ手動 mirror の運用で、ずれるとサービスが `ErrInvalidPeriodDays` を返すので E2E で気づける。
  S09 で ConfigService に移せば解消する
- [Risk] S07 を開いたまま別の手段で在庫が変わっても表示は更新されない
  → サイドバーの切り替えで S07 はアンマウントされるため、単一ウィンドウのアプリでは開き直しで追従する (D8)

## Migration Plan

スキーマ変更は無い。既存 DB のまま動く。ロールバックはコードの revert だけで済む。
