## MODIFIED Requirements

### Requirement: 1 品の入替 / 削除 / 詳細表示

システムは、割当済みの各品行に対し `⋯` メニューを SHALL 提供する。メニューは「入れ替え」「この品を削除」「この品のレシピ
詳細を見る」「この品を連続 N 日」を含む。入替は 1 品だけを別レシピに置き換え、削除は 1 品だけを消す。詳細表示は S03
レシピ詳細画面に移り、S03 のパンくずまたは Esc キーで呼び出し元の S01 に戻す。

#### Scenario: 1 品だけ入れ替える

- **WHEN** ある (日付, meal) に「肉じゃが」「みそ汁」が並んでおり、ユーザーが「みそ汁」の `⋯ → 入れ替え → 豚汁` を選ぶ
- **THEN** そのセルは「肉じゃが」「豚汁」の 2 品になる (position は変わらない)

#### Scenario: 1 品だけ削除する

- **WHEN** ある (日付, meal) に「肉じゃが」「みそ汁」が並んでおり、ユーザーが「みそ汁」の `⋯ → この品を削除` を選ぶ
- **THEN** そのセルは「肉じゃが」1 品だけになる (position=0)

#### Scenario: 詳細を開くと S02 を経由せずに S01 に戻れる

- **WHEN** ユーザーが品行 `⋯ → この品のレシピ詳細を見る` を選ぶ
- **THEN** S03 レシピ詳細画面が表示され、パンくずの「献立」を押すと S02 を経由せずに呼び出し元の S01 に戻る

### Requirement: S01 表示設定の永続化

システムは、S01 献立カレンダーの表示設定 (時間帯表示 `visibleMeals` と期間長 `periodWeeks`) を
アプリ再起動を跨いで SHALL 保持する。保存場所は WebView の localStorage で、key は
`mealcycle.mealplan.visibleMeals` と `mealcycle.mealplan.periodWeeks` を使う。

読み込み時に値が破損している (JSON パース失敗 / 想定外の enum 値 / storage 読み込み失敗) 場合は、
既定値 (`MEALPLAN_DEFAULTS.defaultVisibleMeals` / `periodWeeks: 1`) にフォールバックする。書き込みが
失敗した (private mode / QuotaExceeded / SecurityError) 場合も静かに握り潰し、アプリの動作は
継続する。

`periodStart` は保存せず、サイドバーから S01 を開くたびに「今週」基準に戻すため、この Requirement の対象外。
ただし S03 レシピ詳細画面から S01 に戻ったときは、S03 を開く前の `periodStart` を保つ
(recipe の「S01 献立カレンダーから S03 詳細画面へ移る」)。

#### Scenario: 時間帯切替が再起動後も保持される

- **WHEN** ユーザーが時間帯切替を「3 食表示」(朝 + 昼 + 夕) に変更してから S01 を閉じ、再度 S01 を
  開く
- **THEN** 時間帯切替は「3 食表示」のままで表示される (既定値の「夕食のみ」に戻らない)

#### Scenario: 期間長切替が再起動後も保持される

- **WHEN** ユーザーが期間切替を「2 週」に変更してから S01 を閉じ、再度 S01 を開く
- **THEN** 期間切替は「2 週」のままで表示される (既定値の「1 週」に戻らない)

#### Scenario: 期間の開始日は毎回「今週」にリセットされる

- **WHEN** ユーザーが `＞` を何度か押して未来の週を表示したまま S01 を閉じ、再度 S01 を開く
- **THEN** 表示は「今日を含む週の月曜」から始まる (前回終了時の週には戻らない)

#### Scenario: localStorage が壊れていても既定値で起動する

- **WHEN** `mealcycle.mealplan.periodWeeks` の値が `3` (許容外) の状態で S01 を開く
- **THEN** 期間長は既定値の 1 週で表示される (アプリは落ちない)

#### Scenario: localStorage が使えなくても S01 は動作する

- **WHEN** localStorage が SecurityError で読み書き不能な環境で S01 を開き、時間帯切替を変更する
- **THEN** 現在のセッションでは変更が反映される (保存はされないが、アプリは落ちない)
