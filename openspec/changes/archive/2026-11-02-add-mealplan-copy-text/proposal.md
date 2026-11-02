## Why

家族で使う場合も、アプリを操作するのは献立を担当する 1 人で、ほかの家族には献立や買い物リストをテキストで渡す
([product.md §1](../../../docs/product.md)「家族で使う場合」)。
買い物リストは S07 の「テキストでコピー」で渡せるようになったが、献立そのものを渡す手段が無い。
今は S01 を見ながら日付とレシピ名をメッセージアプリに打ち直している。

[design.md §9](../../../docs/design.md) は「献立のテキストでのコピー：S07 の『テキストでコピー』と同じ形で、期間の献立を家族に渡す」を家族向けの拡張に挙げている。

## What Changes

- S01 の下部、「この期間の買い物リストを見る」の横に「献立をテキストでコピー」ボタンを置く
- 押すと、表示中の期間の献立をクリップボードにコピーし、コピーした日数を通知で知らせる
  - 1 行目に「献立 11/2(月)〜11/8(日)」の形で期間の見出しを置く
  - 以降は日付と時間帯ごとに 1 行、「11/2(月) 夕: 肉じゃが、味噌汁」の形で並べる。品の順は S01 のセル内の順
  - 献立の無い日と時間帯の行は出さない
  - 時間帯表示 (夕食のみ / 3 食) にかかわらず、期間内の全時間帯を含める
  - 作った印と作り置きの印は含めない
- 表示中の期間に献立が 1 件も無いときはボタンを押せない

### Non-goals

- **曜日や時間帯の絞り込み**：コピーの前に範囲を選ぶダイアログは置かない。必要な部分は受け取った側で読み飛ばせる
- **共有先との連携 (メールやメッセージアプリを開く)**：完全ローカルの方針 ([design.md §2](../../../docs/design.md)) に合わせ、クリップボードまでにとどめる
- **材料の同梱**：買い物リストは S07 のコピーで渡す
- **書式の設定 (Markdown、絵文字の有無など)**：まず 1 つの書式で使ってみる

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：S01 に献立のテキストコピーを足す

## Impact

**新設ファイル**

- `frontend/src/components/mealplan/mealplan-text.ts`：期間の献立の行からテキストと日数を組む純関数
- `frontend/src/components/mealplan/mealplan-text.test.ts`
- `frontend/src/components/mealplan/copy-mealplan-text-button.tsx`
- `frontend/e2e/mealplan-copy-text.spec.ts`

**更新ファイル**

- `frontend/src/routes/meal-plan.tsx`：下部の CTA の並びにボタンを足す
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：見出し、行、時間帯の略称、通知の文言
- `docs/product.md`：§4 S01 にテキストでコピーの節を足す。§1「家族で使う場合」を実装済みに直す
- `docs/design.md`：§9 の献立のテキストでのコピーを実装済みに直す
- `docs/wireframes/S01_meal_calendar.v2.drawio.svg`：下部にボタンを足す

**生成物 (再生成のみ)**

- なし (Go のサービスは変えない。S01 が取得済みの `ListPeriod` の結果から組む)

**migration**

- なし

**後続 proposal への引き継ぎ**

- 曜日の絞り込みや書式の選択が欲しくなったら、コピーの前に開くダイアログを足す
- S07 のコピーと献立のコピーを 1 回で渡したい声が出たら、両方をまとめる操作をどちらの画面に置くかを決める
