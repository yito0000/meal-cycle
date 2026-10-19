## Why

「他の作り置きにまとめる」のダイアログは、候補の日付を `2026-10-20` の形のまま出している
(docs/reviews/2026-10-19-design-ux-review.md R49)。
`revise-mealplan-spread-safety` (R42) で「作り置きを広げる」のダイアログは S01 の格子と同じ `10/20(火)` の形にそろえたが、
同じ直しがまとめるダイアログには入っていない。
S01 から開く 2 つのダイアログで、日付の表記が分かれている。

あわせて、「作り置きを広げる」のダイアログには、一度も描かれない「さらに N 日表示」の経路が残っている (R51)。
S01 は展開の幅に 0 を渡しており、Go は上限を超える対象日を拒否するので、この経路を生かす使い道も無い。

## What Changes

- まとめるダイアログの候補の日付を、S01 の格子と同じ「月/日と曜日」の表記で示す。表記は表示言語に従う
- S01 の 2 つのダイアログが同じ日付の表記を使うよう、表記を作る関数を 1 つにまとめる
- 「作り置きを広げる」のダイアログから、「さらに N 日表示」のボタン、展開の幅の prop、関連する文言を消す

### Non-goals

- **まとめる候補の件数と探す範囲** (R46、R47)：`fix-mealplan-merge-range` で扱う
- **格子の行数を Go から受け取ること** (R48)：`revise-settings-constraints` で扱う
- **S07 や S05 の日付の表記** (`yyyy/mm/dd`)：画面の用途に合わせて別の表記を選んでいるので変えない
- **候補の行の並べ方と件数の文言**：変えない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：「まとめ候補の日付の表記」を足す

## Impact

### 新設ファイル

- `frontend/src/lib/format-meal-date.ts`：`YYYY-MM-DD` を、表示言語の「月/日と曜日」の表記にする関数
- `frontend/src/lib/format-meal-date.test.ts`：上の関数の Vitest

### 更新ファイル

- `frontend/src/components/mealplan/batch-merge-dialog.tsx`：候補の日付を新しい関数で整える
- `frontend/src/components/mealplan/spread-batch-dialog.tsx`
  - `formatRow` と `dateFmt` を新しい関数に置き換える
  - `expandStepDays` の prop、「さらに N 日表示」のボタン、`setFutureDays` の加算を消す
- `frontend/src/components/mealplan/meal-plan-cell.tsx`：同じ表記を組んでいる `dateFmt` を新しい関数に置き換える
- `frontend/src/components/mealplan/week-calendar.tsx`：`SPREAD_EXPAND_STEP` と、`SpreadBatchDialog` への `expandStepDays` の受け渡しを消す
- `frontend/src/locales/ja.json`、`frontend/src/locales/en.json`：`pages.meal_plan.spread.show_more` を消す
- `frontend/e2e/mealplan-batch.spec.ts`
  - describe「まとめ候補の日付の表記」を足す
  - describe「まとめ候補の一意性」の既存のテストが、候補に `YYYY-MM-DD` が含まれることを確かめている箇所を、新しい表記に合わせる
- `docs/reviews/2026-10-19-design-ux-review.md`：R49、R51 にチェックを入れる

### 生成物 (再生成のみ)

- なし (Go のメソッドと型は変わらない)

### migration

- なし

### 後続 proposal への引き継ぎ

- `revise-settings-constraints` は `spread-batch-dialog.tsx` の props をさらに減らす。この change を先に入れる
- S01 の格子の列見出し (`week-calendar.tsx`) は曜日と日付を別の要素で組んでいるので、この change では新しい関数に寄せない
