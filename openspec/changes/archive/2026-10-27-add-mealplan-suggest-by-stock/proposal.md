## Why

献立を組むとき、利用者は S05 で期限の近い在庫を確かめ、それを使えるレシピを頭の中で探してから S01 のセルに割り当てている。
ペルソナの「残りの材料を考えて献立を考えるのが面倒」は、この往復のことである ([product.md §1](../../../docs/product.md))。
在庫の期限とレシピの材料はどちらもアプリが持っているので、突き合わせは機械に任せられる。

[product.md §5](../../../docs/product.md) は、全自動で献立を組むのではなく、根拠つきの候補を並べて利用者が選ぶ方針を決めている。
その最初の軸として、期限の近い在庫を使えるレシピ (「使い切り」) を、S01 のレシピ選択ダイアログの上部に出す。
在庫が実物に近いほど候補は役に立つので、作った印で在庫を減らす `add-mealplan-cook-consume` のあとに入れる。

## What Changes

- 献立候補を扱う capability `suggestion` を新設する
- S01 のレシピ選択ダイアログの上部に、「使い切り」の候補の欄を足す
  - 期限が 🔴 か 🟡 の在庫 (期限切れを除く) と食材名が同じ材料を持つレシピを候補にする。単位は問わない
  - 候補は、使える在庫の数の多い順、同じ数なら最も近い期限の早い順、さらにレシピ名の昇順に並べ、最大 5 件を出す
  - 各候補には、使える在庫の食材名と最も近い期限までの日数を根拠として 1 行で添える (例：「玉ねぎ、にんじん が期限 2 日以内」)
  - 候補を押すと、下の一覧の行を押したときと同じく、そのレシピを選んで閉じる
  - 該当するレシピが無いときは、欄を消さずに「該当なし」と出す
  - 検索欄に文字を入れている間は、候補の欄を隠して一覧だけを出す
- 削除済みのレシピは候補に出さない

### Non-goals

- **「久しぶりに」と「時短」の軸**：「久しぶりに」は `add-mealplan-suggest-by-recency` で足す。「時短」は所要時間の項目が無いので見送る
- **週単位の候補 (週プリセット)**：`add-mealplan-suggest-week-preset` で扱う
- **候補の件数や対象とする期限の段階を S09 で変えること**：期限の段階は既存の 🔴 / 🟡 の日数 (S09 で変えられる) に従う。件数は見た目の調整値として `defaults.go` に置き、S09 には出さない
- **在庫の量とレシピの量の突き合わせ**：候補にするかどうかは食材名だけで決める。使い切れるかどうかまでは見ない
- **セルの日付に合わせた期限の判定**：期限は今日を起点に数える。来週のセルを選んでいても、今日から見た期限で候補を出す

## Capabilities

### New Capabilities

- `suggestion`：S01 で献立を組むときに、根拠つきのレシピ候補を示す。この proposal では「使い切り」の軸を入れる

### Modified Capabilities

(なし)

## Impact

**前提**

- `add-mealplan-cook-consume` を先に入れる

**新設ファイル**

- `internal/domain/suggestion/suggestion.go`：`Candidate` / `StockReason` / `Result`
- `internal/domain/suggestion/by_stock.go`：「使い切り」の候補を算出する `ByStock`
- `internal/domain/suggestion/by_stock_test.go`
- `internal/service/suggestion.go`：`SuggestionService` (`List`)
- `internal/service/suggestion_test.go`
- `frontend/src/components/mealplan/suggestion-section.tsx`：候補の欄 1 つ分 (見出し、件数、候補の行、該当なし)
- `frontend/src/components/mealplan/use-suggestions.ts`
- `frontend/e2e/bindings-shim/suggestionservice.ts`
- `frontend/e2e/suggestion.spec.ts`

**更新ファイル**

- `internal/config/defaults.go` と `defaults_test.go`：`SuggestionLimit` (既定 5) を追加
- `main.go`：`application.Options.Services` に `service.NewSuggestionService(db, settingsSvc)` を追加
- `cmd/e2e-server/main.go` と `cmd/e2e-server/dispatch_test.go`：registry に `SuggestionService` を足し、件数の検査を 7 にする
- `frontend/e2e/bindings-shim/index.ts`：`SuggestionService` を re-export
- `frontend/src/components/mealplan/recipe-picker-dialog.tsx`：検索欄の下に候補の欄を置く
- `frontend/src/locales/ja.json` と `frontend/src/locales/en.json`
- `docs/product.md`：§4 S01 の「MVP 後拡張」と §5 の「使い切り」の行に proposal 名を添える
- `docs/design.md`：§8 の `add-mealplan-suggest-by-stock` の行

**生成物 (再生成のみ)**

- `frontend/bindings/`：`wails3 generate bindings -clean=true -ts -i`

**migration**

- なし

**後続 proposal への引き継ぎ**

- `add-mealplan-suggest-by-recency` は `suggestion.Result` に軸を 1 つ足し、ダイアログには `SuggestionSection` をもう 1 つ並べる
- `add-mealplan-suggest-week-preset` は `ByStock` の結果を週の候補の材料に使う
- 「時短」の軸は、レシピに所要時間の項目を足す proposal のあとで検討する
