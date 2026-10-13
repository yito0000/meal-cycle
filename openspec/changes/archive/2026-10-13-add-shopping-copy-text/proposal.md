## Why

meal-cycle は完全ローカルのデスクトップアプリで、スーパーの店内では S07 買い物リストを開けない。
S07 で何を買うかが分かっても、その内容を PC の外に持ち出す手段が無いので、店では記憶か手書きのメモに頼ることになる
(docs/reviews/2026-09-25-design-ux-review.md R1)。

買い物リストは、スマホで見られて初めて店内で役に立つ。
外部と通信しない方針のままでも、テキストをクリップボードにコピーできれば、LINE やメモアプリに貼ってスマホに送れる。
R1 で挙がった手段 (コピー、印刷、LAN 配信) の中で、これがいちばん安く作れて効果が大きい。

## What Changes

- S07 に「テキストでコピー」ボタンを置く。押すと、買い物リストをプレーンテキストにしてクリップボードにコピーし、トーストで知らせる
- コピーするテキストは次の形にする
  - 1 行目：対象期間 (例：`買い物リスト 2026/10/05〜2026/10/11`)
  - 材料テーブルのうち「在庫なし」と「一部在庫あり」の行を、材料テーブルと同じ順で 1 行ずつ (例：`- 玉ねぎ 2 個`)。
    「一部在庫あり」の行には `(一部在庫あり)` を添える
  - 「家にあるか確かめる材料」のうち「在庫なし」の項目があれば、空行と見出しを挟んで食材名だけを並べる
- 「在庫あり」の行と項目はコピーしない。在庫の状態での絞り込みと購入済チェックは、コピーする内容に影響しない
- コピーする行も項目も無いとき (すべて在庫あり) は、ボタンを押せない状態にする
- クリップボードに書き込めなかったときは、その旨をトーストで知らせる

### Non-goals

- **印刷**：Wails の webview (Windows の WebView2、Linux の WebKitGTK) で `window.print()` がどう振る舞うかを確かめていない。
  スマホへの持ち出しはコピーで足りるので、印刷は必要になったら別の proposal で扱う
- **LAN 上への配信 (QR コード)**：ローカル完結の原則とぶつかるため扱わない
- **テキストの形式の選択**：チェックリスト記法 (`- [ ]`) や Markdown 表など、形式を選ぶ UI は持たない
- **用途 (レシピ名) と献立の欄**：店内では何を買うかだけが要るので、テキストに含めない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `shopping`：買い物リストをテキストでクリップボードにコピーする要件「買い物リストのテキストコピー」を ADDED で足す

## Impact

### 新設ファイル

- `frontend/src/components/shopping/shopping-text.ts`：買い物リストからコピーするテキストを組み立てる純関数
- `frontend/src/components/shopping/copy-shopping-text-button.tsx`：「テキストでコピー」ボタン。クリップボードへの書き込みとトーストを受け持つ
- `frontend/src/lib/clipboard.ts`：クリップボードへの書き込み。`navigator.clipboard` が使えないときは Wails の runtime に切り替える
- `frontend/e2e/shopping-copy.spec.ts`：コピーの E2E

### 更新ファイル

- `frontend/src/routes/shopping.tsx`：対象期間の表示の横にボタンを置く
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.shopping.copy.*` を追加する
- `docs/product.md`：§4 S07 にテキストコピーの項を足す
- `docs/wireframes/S07_shopping_list.v2.drawio.svg`：対象期間の行にボタンを描き足す (可視 SVG と `content=` 内の編集 XML の両方)
- `openspec/specs/shopping/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- なし。Go 側とバインディングは変えない

### migration

- なし

### 前提と順序

- 他の進行中の change とはファイルが重ならない
- `add-ingredient-name-matching` と `add-mealplan-cooked-mark` は S07 の行の中身を変えるが、この change はフロントが受け取った行を整形するだけなので、どちらが先でもよい

### 後続 proposal への引き継ぎ

- 印刷用 CSS (`window.print()`)。Windows ビルドで WebView2 の印刷ダイアログが出るかを確かめてから切る
