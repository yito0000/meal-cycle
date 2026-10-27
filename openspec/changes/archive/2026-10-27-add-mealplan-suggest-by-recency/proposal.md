## Why

レパートリーが 20〜50 品あっても、献立を考えるときに思い出すのは最近作った数品になりやすい。
同じ料理が続いて飽き、しばらく作っていない料理は候補から抜け落ちていく。
どのレシピをいつ作ったかは、S01 の「作った」印の履歴としてアプリにすでにある。

`add-mealplan-suggest-by-stock` で S01 のレシピ選択ダイアログに候補の欄を作った。
その 2 つめの軸として、しばらく作っていないレシピ (「久しぶりに」) を並べる ([product.md §5](../../../docs/product.md))。

## What Changes

- S01 のレシピ選択ダイアログの「使い切り」の欄の下に、「久しぶりに」の候補の欄を足す
  - レシピの **最終調理日** は、作った印の付いた行の日付のうち最も新しい日とする。印の無い予定の行は数えない
  - 最終調理日から今日までの日数が、S09 で決める日数 (既定 10 日) 以上のレシピを候補にする
  - 一度も作った印を付けていないレシピは候補にしない
  - 今日以降に作った印の無い予定がすでにあるレシピは候補にしない
  - 経過日数の多い順、同じ日数ならレシピ名の昇順に並べ、最大 5 件を出す
  - 各候補には「N 日ぶり」の根拠を添える
  - 検索中に隠すこと、押すと選ぶこと、該当なしの表示は「使い切り」の欄と同じにする
- S09 の献立の節に、「久しぶりに」とみなす日数 (1〜60 日、既定 10 日) の欄を足す

### Non-goals

- **一度も作っていないレシピの扱い**：作ったことの無いレシピは「久しぶり」ではない。新しく試すレシピの候補は別の軸として検討する
- **S01 の外で作った記録**：外食や、献立に入れずに作った料理は記録が無いので数えない
- **最終調理日の保存**：献立の行と作った印から毎回求める。献立の行を消すと最終調理日も変わる

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `suggestion`：「久しぶりに」の候補の要件を追加する
- `settings`：S09 の献立の節に「久しぶりに」とみなす日数を足し、検証と反映先に加える

## Impact

**前提**

- `add-mealplan-suggest-by-stock` を先に入れる (capability `suggestion` と候補の欄の部品を使う)

**新設ファイル**

- `internal/domain/suggestion/by_recency.go`：`ByRecency` と `RecipeHistory`
- `internal/domain/suggestion/by_recency_test.go`
- `internal/storage/sqlite/suggestion_repo.go`：レシピごとの最終調理日と、今日以降の予定の有無を読む `ListRecipeHistory`
- `internal/storage/sqlite/suggestion_repo_test.go`

**更新ファイル**

- `internal/domain/suggestion/suggestion.go`：`Candidate` に `DaysSince`、`Result` に `ByRecency` を追加
- `internal/service/suggestion.go` と `suggestion_test.go`：`List` で `ByRecency` を埋める
- `internal/domain/settings/settings.go` と `validate.go` と `settings_test.go`：`RecencyThresholdDays` と検証 (1〜60)
- `internal/domain/settings/errors.go`：範囲外のエラーコード
- `internal/service/settings.go`：保存する key に `recencyThresholdDays` を足す
- `frontend/src/components/settings/settings-form.tsx` と `frontend/src/components/settings/error-messages.ts`
- `frontend/src/components/mealplan/recipe-picker-dialog.tsx`：「久しぶりに」の `SuggestionSection` を足す
- `frontend/src/locales/ja.json` と `frontend/src/locales/en.json`
- `frontend/e2e/suggestion.spec.ts` と `frontend/e2e/settings.spec.ts`
- `docs/product.md`：§1 の可変値の表 (「久しぶりに」の消費期間閾値を S09 で変更可にする)、§4 S09、§5 の「久しぶりに」の行
- `docs/wireframes/S09_settings.drawio.svg` は更新せず、差分を design.md に残す

**生成物 (再生成のみ)**

- `frontend/bindings/`：`wails3 generate bindings -clean=true -ts -i`

**migration**

- なし。設定は既存の `settings` テーブルの key として保存する

**後続 proposal への引き継ぎ**

- `add-mealplan-suggest-week-preset` は `ByRecency` の結果も週の候補の材料に使う。S09 の献立の節に欄を足すので、この proposal の settings の delta を取り込んでから書く
