## Why

家族で使う場合、操作しない家族の消費はアプリに届かず、S05 の在庫は少しずつ実物とずれていく
([product.md §1](../../../docs/product.md)「家族で使う場合」)。
ずれを直すには冷蔵庫を開けて 1 品ずつ確かめるしかないが、今の S05 では 1 行ごとに「編集」を押し、量を直して「保存」を押す必要がある。
在庫が 20 行あると 60 回近い操作になり、見直しが続かない。

[design.md §9](../../../docs/design.md) は「在庫の見直しを速くする手段：S05 の各行を『ある / 減った / ない』で次々に片付ける」を家族向けの拡張に挙げている。
本 proposal はその手段を入れる。

## What Changes

- S05 の在庫のテーブルの上に「見直す」ボタンを置き、押すと見直しの状態に入る
- 見直しの状態では、各行の「編集」「削除」の代わりに「ある」「減った」「ない」の 3 つのボタンを出す
  - 「ある」：在庫を変えずに、その行を見直し済みにする
  - 「減った」：その行の量の欄だけを入力欄にし、今の量を選択した状態でフォーカスする。Enter で量を保存して見直し済みにする
  - 「ない」：その行を削除し、通知の「元に戻す」で戻せる (今の削除と同じ作法)
- 見直し済みの行は薄く表示し、次の未見直しの行へフォーカスを移す
- テーブルの上に「見直し済み N / M 件」を出し、「見直しを終える」で通常の表示に戻る。見直し済みの印は保存しない
- 見直しの状態では、インライン追加行、検索欄、並び替えはそのまま使える。行内編集の「編集」は出さない

### Non-goals

- **常備品の欄**：常備品はすでに 3 つの状態ボタンで 1 回の操作で切り替わるので、見直しの状態の対象にしない
- **見直しの記録の保存**：いつ見直したかを残すと、見直しの催促や在庫の信頼度の表示が欲しくなる。必要になったら別の proposal にする
- **「減った」の量の候補 (半分、1 つ減らす等)**：単位が自由テキストなので候補を一般化できない。量を打ち直す形にとどめる
- **量 0 での確定を「ない」とみなすこと**：今の量の検証 (0 より大きい) をそのまま使い、0 はエラーにする

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `fridge`：S05 に在庫の見直しの状態を足す。既存の在庫の更新と削除の振る舞いは変えない

## Impact

**新設ファイル**

- `frontend/src/components/fridge/stock-review.ts`：見直し済みの ID の集合と、次にフォーカスする行を求める純関数
- `frontend/src/components/fridge/stock-review.test.ts`
- `frontend/src/components/fridge/fridge-review-row.tsx`：見直しの状態の 1 行 (「ある」「減った」「ない」と量の入力欄)
- `frontend/src/components/fridge/fridge-review-bar.tsx`：「見直し済み N / M 件」と「見直しを終える」
- `frontend/e2e/fridge-stock-review.spec.ts`
- `docs/wireframes/S05_fridge.v3.drawio.svg`：見直しの状態の S05

**更新ファイル**

- `frontend/src/routes/fridge.tsx`：見直しの状態の切替と「見直す」ボタン
- `frontend/src/components/fridge/fridge-table.tsx`：見直しの状態のとき `fridge-review-row` を描く
- `frontend/src/components/fridge/use-fridge.ts`：量だけを更新する mutation を足す (既存の `FridgeService.Update` を呼ぶ)
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：見直しの文言
- `docs/product.md`：§4 S05 に見直しの状態の節を足す。§1「家族で使う場合」の段落を実装済みに直す
- `docs/design.md`：§9 の「在庫の見直しを速くする手段」を実装済みに直す
- `docs/wireframes/README.md`：v3 を足す

**生成物 (再生成のみ)**

- なし (Go のサービスは変えない)

**migration**

- なし

**後続 proposal への引き継ぎ**

- 見直しの記録 (最終見直し日) を持つなら、`stock_items` に列を足すか、見直しの履歴テーブルを作るかを決める
- 「減った」に量の候補を出すなら、単位ごとの刻み (個は 1、g は 50 など) を S09 の可変値にするかを決める
