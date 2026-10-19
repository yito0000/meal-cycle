## Context

動機は proposal.md の Why を参照。

S01 で「月/日と曜日」の表記を作っている箇所は 2 つある。
`SpreadBatchDialog` の `formatRow` と、`MealPlanCell` の `dateFmt` で、どちらも
`new Intl.DateTimeFormat(i18n.language, { month: "numeric", day: "numeric", weekday: "short" })` に、
`YYYY-MM-DD` をローカル日付の `Date` にして渡している。
`BatchMergeDialog` は候補の `date` を i18n の補間にそのまま差し込んでいる。

`SpreadBatchDialog` の `expandStepDays` は、`WeekCalendar` の `SPREAD_EXPAND_STEP = 0` からしか渡されない。
`expandStepDays > 0` のときだけ「さらに N 日表示」のボタンを描き、押すと `futureDays` を増やす。
増やした先の日付は `SpreadEntry` の検証 (`maxSpreadOffsetDays`) で拒否される。

## Goals / Non-Goals

**Goals:**

- S01 のダイアログとセルの「月/日と曜日」の表記を、1 つの関数から作る
- 一度も描かれない展開の経路を消す

**Non-Goals:**

- 列見出しの表記 (曜日と日付を別の要素で並べる) は変えない
- 表記の形そのもの (`10/20(火)`、`Tue, 10/20` など) は `Intl.DateTimeFormat` に任せ、自前で組み立てない

## Decisions

### D1. 日付の表記を作る場所

**選択肢:**

- (a) `frontend/src/lib/format-meal-date.ts` に `formatMealDate(isoDate, language)` を置き、ダイアログとセルから呼ぶ ← 採用
  — Pros: 表記の形が 1 か所で決まり、R42 と R49 のような直し漏れが起きにくい。純関数なので Vitest で表記を固定できる /
  Cons: ファイルが 1 つ増える
- (b) `BatchMergeDialog` に `SpreadBatchDialog` と同じ `Intl.DateTimeFormat` を書く — Pros: 変更が最小 /
  Cons: 同じ書き方が 3 か所になり、次に表記を変えるときにまた揃え直しが要る
- (c) Go の `BatchSummary` に表示用の日付を持たせる — Pros: フロントで整えなくて済む /
  Cons: Go は表示文言を持たないという方針 (design.md §7) に反する。表示言語の切替にも追従できない

**採用: (a)**。`language` は呼び出し側が `useTranslation()` の `i18n.language` を渡す。
関数は `Intl.DateTimeFormat` を言語ごとに 1 つだけ作って使い回す (`Map` に保持する)。
セルは期間の日数ぶん描かれるので、描画のたびに作り直さないためである。

### D2. 展開の経路の消し方

他案として「展開の幅を Go の上限まで伸ばして生かす」を検討したが、
格子の初期の行数がすでに Go の上限と同じなので、伸ばす余地が無い。
prop、ボタン、`futureDays` の加算、`show_more` の文言を消す。
`futureDays` の state は `initialFutureDays` を保持するだけになるので、この change では `initialFutureDays` を直接使う形にする。
`initialFutureDays` そのものを消すのは `revise-settings-constraints` に任せる。

### D3. E2E の確かめ方

新しい Scenario の E2E は、候補の行に `formatMealDate(MON, "ja")` と同じ文字列が含まれ、`MON` (`YYYY-MM-DD`) が含まれないことを確かめる。
E2E は Vitest と同じ関数を import して期待値を作る。
`Intl` の出力は実行環境の ICU に左右されるため、期待値を文字列で直書きしないためである。
`まとめ候補の一意性` の既存のテストが `toContainText(MON)` で日付を確かめている箇所も、同じ期待値に置き換える。

## Risks / Trade-offs

- [Risk] E2E が `src/lib` の関数を import すると、テストと実装が同じ誤りを共有する
  → 表記の形そのものは Vitest で `10/20(火)` (`ja`) と `Tue, 10/20` (`en`) を直書きして固定する。E2E は「同じ関数で整えた日付が出ていること」だけを確かめる
- [Risk] `fix-mealplan-merge-range` が足す E2E も候補の日付を確かめる
  → その E2E は、この change の 3 章で新しい表記に合わせて直す。適用の順は `fix-mealplan-merge-range` を先にする

## Migration Plan

スキーマとデータは変わらない。
