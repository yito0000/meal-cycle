## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/shopping/spec.md](specs/shopping/spec.md) を参照。

S07 の現状は次のとおりである。

- `ShoppingService.List` の `items` は、Go 側で並び順が決まっている (`covered` 以外が先、`covered` が後、それぞれ最初に必要になる日、食材名、単位の順)
- `ShoppingPage` は `isPantryItem` (必要量 0) で `items` を材料テーブルの行と「家にあるか確かめる材料」の項目に分けている
- 必要量の表記は `formatQuantity(amount, unit)` で作る。量る単位は「大さじ 2」のように単位を前に置く
- 期間の表示は `formatDisplayDate` で `yyyy/mm/dd` にしている
- アプリはまだクリップボードを使っていない。フロントから `@wailsio/runtime` を直接 import している箇所も無い
  (生成されたバインディングが内部で使うだけで、E2E ではバインディングごと shim に差し替わる)
- E2E のサーバは `http://127.0.0.1:<port>` で配信しており、Chromium では secure context として扱われる

## Goals / Non-Goals

**Goals:**

- S07 がすでに持っている `items` だけでテキストを組み立て、Go 側とバインディングに手を入れない
- Windows の実機 (WebView2) と E2E (headless Chromium) の両方でコピーが動くようにする

**Non-Goals:**

- テキストの形式を利用者が選べるようにすること
- コピーした日時や内容を記録すること

## Decisions

### D1. テキストを組み立てる場所

**選択肢:**

- (a) フロントの純関数で組み立てる ← 採用
  — Pros: 画面に出ている表記 (`formatQuantity`、`formatDisplayDate`、i18n の文言) をそのまま使える。Go 側が変わらない
  / Cons: 純関数のユニットテストを書く場所が無い (フロントに test runner が無い)
- (b) `ShoppingService` にテキストを返すメソッドを足す
  — Pros: Go の table-driven test で組み立てを検証できる
  / Cons: Go 側は日本語の文言を持たない方針 (design.md §7) なので、見出しや「一部在庫あり」を返せない。
  文言をフロントから渡すと引数が膨らむ

**採用: (a)**。
`frontend/src/components/shopping/shopping-text.ts` に `buildShoppingText(items, period, t)` を置く。
`items` は `ShoppingService.List` の `items` をそのまま受け取り、関数の中で `isPantryItem` と在庫の状態で振り分ける。
画面の絞り込み (`stockFilter`) とチェック (`checked`) は引数に取らない。spec のとおり、これらはテキストに影響しないからである。
返り値は `{ text: string; count: number }` にして、トーストの件数とボタンの押せる / 押せないの判定にも同じ `count` を使う。

組み立ての検証は E2E で行う。
Vitest を入れる proposal が来たら、この関数が最初のユニットテストの対象になる。

### D2. クリップボードへの書き込み方

**選択肢:**

- (a) `navigator.clipboard.writeText` だけを使う
  — Pros: 実装が 1 行。E2E の Chromium でそのまま動く
  / Cons: secure context でしか使えない。Linux の開発環境 (WebKitGTK) は独自スキームで配信しており、secure context と見なされるかを確かめていない
- (b) Wails の runtime の `Clipboard.SetText` だけを使う
  — Pros: webview の種類と secure context に左右されない
  / Cons: E2E には Wails の runtime が無いので動かない。E2E で差し替えるには vite の e2e プラグインを広げる必要がある
- (c) `navigator.clipboard.writeText` を先に試し、使えないか失敗したら Wails の runtime に切り替える ← 採用
  — Pros: E2E は (a) の経路、実機は (a) か (b) のどちらかで動く
  / Cons: 経路が 2 つになる

**採用: (c)**。
`frontend/src/lib/clipboard.ts` に `writeClipboardText(text): Promise<void>` を置く。

1. `navigator.clipboard?.writeText` があれば呼ぶ。成功したら終わる
2. 無いか、例外になったら `await import("@wailsio/runtime")` で runtime を読み込み、`Clipboard.SetText(text)` を呼ぶ
3. それも失敗したら例外をそのまま投げる。ボタンはこれを受けて失敗のトーストを出す

runtime を動的 import にするのは、E2E のビルドで runtime のモジュールを読み込ませないためである。
E2E はテストの中でクリップボードの権限を与えるので、1 の経路で成功し、2 には来ない。

失敗の経路 (spec の「クリップボードに書き込めないとその旨を知らせる」) は E2E に入れない。
権限を与えずに Chromium の `writeText` を失敗させると、2 で E2E に無い runtime を読みに行くことになり、失敗の理由が実機と違ってしまう。
この Scenario は、ボタンが例外を受けて失敗のトーストを出すことをコードレビューで確かめる。

### D3. ボタンの置き場所と状態

ボタンは対象期間の表示 (`shopping-period`) と同じ行の右端に置く。
材料テーブルの近く (絞り込みの横や登録ボタンの横) に置く案も考えた。
だが絞り込みの横に置くと「表示中の行だけをコピーする」と読まれやすく、登録ボタンの横に置くとチェックした行が対象に見える。
コピーは期間の買い物リスト全体に効くので、期間の表示の横に置く。

- `data.dishes.length === 0` (献立が無い) と `data.items.length === 0` (材料が無い) のときは出さない
- `count === 0` のときは `disabled` にする。材料テーブルが「在庫あり」だけで、別の欄もすべて在庫ありのときにこうなる
- 書き込み中はボタンを `disabled` にし、連打で二重にコピーしないようにする

### D4. テキストの形

行頭は `- ` にする。
LINE とメモアプリ (iOS の「メモ」、Google Keep) に貼ると、どちらでも箇条書きとして読める記号だからである。
チェックリスト記法 (`- [ ]`) にしなかったのは、Markdown を解釈しないアプリに貼ると `[ ]` がそのまま残って読みにくいためである。

見出し、行、「一部在庫あり」の注記、別の欄の見出しは、すべて i18n のキーにする。
`- ` と `(一部在庫あり)` のような記号の付け方も言語ごとに変えられるよう、行全体をテンプレートのキーにする (例：`"- {{name}} {{quantity}}"`)。
改行は `\n` にする。貼る先はスマホのアプリと、LF を解釈する今の Windows のメモ帳を想定している。

## Risks / Trade-offs

- [Risk] WebView2 で `navigator.clipboard.writeText` が権限の確認を出す、または拒否される → D2 の runtime の経路に切り替わる。Windows ビルドで両方の経路のどちらが使われても、貼り付けられることを確かめる
- [Risk] Linux の開発環境で runtime の `Clipboard.SetText` も失敗する → 開発時の動作確認では失敗のトーストが出るだけで、データは壊れない。確認は Windows ビルドで行う
- [Risk] 画面の表記とテキストの表記がずれる → 必要量と日付の表記は画面と同じ関数 (`formatQuantity`、`formatDisplayDate`) を通す

## Migration Plan

- データの移行は無い。フロントの変更だけで、戻すときはコミットを revert する
