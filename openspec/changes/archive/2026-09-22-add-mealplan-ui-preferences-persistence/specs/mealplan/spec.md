## ADDED Requirements

### Requirement: S01 表示設定の永続化

システムは、S01 献立カレンダーの表示設定 (時間帯表示 `visibleMeals` と期間長 `periodWeeks`) を
アプリ再起動を跨いで SHALL 保持する。保存場所は WebView の localStorage で、key は
`mealcycle.mealplan.visibleMeals` と `mealcycle.mealplan.periodWeeks` を使う。

読み込み時に値が破損している (JSON パース失敗 / 想定外の enum 値 / storage 読み込み失敗) 場合は、
既定値 (`MEALPLAN_DEFAULTS.defaultVisibleMeals` / `periodWeeks: 1`) にフォールバックする。書き込みが
失敗した (private mode / QuotaExceeded / SecurityError) 場合も静かに握り潰し、アプリの動作は
継続する。

`periodStart` は「今週基準」で毎回リセットするため、この Requirement の対象外。

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
