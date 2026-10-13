## 1. 品のメニューの開き方の試作

- [x] 1.1 `frontend/src/components/mealplan/cell-item-menu.tsx` (更新) と `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) で、design.md D1 (b) と D2 の構成を最小限に組む
    - 開閉を自前の状態で持ち、レシピ名のボタンの `click` で開く
    - `DropdownMenuTrigger` は品行に重ねた透明な要素にし、`pointer-events: none` にする
    - メニューを `side="right"` で開く
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `task dev` で、試作の挙動を確かめる
    - レシピ名をクリックすると、品行の右にメニューが開く
    - 日曜の列の品では、メニューが左に回り込む
    - 単独の品をドラッグしても、メニューは開かず D&D になる
    - レシピ名のボタンからドラッグを始められる
    - チェックボックスを押しても、ドラッグが始まらない
    - Enter と Space でメニューが開き、Esc で閉じてフォーカスがレシピ名に戻る
    - 完了判定: 目視で確認する
- [x] 1.3 1.2 の結果を `openspec/changes/revise-mealplan-item-row/design.md` の D1 と D2 に追記する。期待どおりでない点があれば、代わりの案に切り替えて D1 / D2 を更新する。
  完了判定: design.md の差分を目視で確認する

## 2. 品行 (S01)

- [x] 2.1 `frontend/src/components/mealplan/cell-item-menu.tsx` (更新) を、1 の試作をもとに仕上げる
    - `⋯` のトリガーを外し、`open` と `onOpenChange` を props で受け取る
    - 「作った」「作った印を外す」の項目と、`cooked` / `onToggleCooked` の props を外す (design.md D3)
    - 冒頭のコメントのメニュー項目の列挙を直す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) で、品行を「作った」チェックボックスとレシピ名のボタンの構成にする (design.md D2)
    - チェックボックスは `aria-label={t("pages.meal_plan.cell.cooked")}` を持ち、`entry.cooked` を反映する
    - チェックボックスの変更で `onEntryAction("toggle-cooked", entry)` を呼ぶ
    - レシピ名のボタンは `aria-haspopup="menu"` を持ち、クリックで品のメニューを開く
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.3 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) から 🔗 の描画を外す (design.md D4)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.4 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) で、品行のカーソルとホバー時の表示を変える (design.md D8)
    - すべての品行を `cursor-pointer` にし、`cursor-not-allowed` と `cursor-grab` をやめる
    - ホバー時に背景色を変える
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.5 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) で、レシピ名を `line-clamp-2` と `overflow-wrap: anywhere` で 2 行まで折り返す (design.md D5)。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.6 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) で、セルの内側の余白を `p-1`、品行の左右の余白を `px-0.5` にする (design.md D6)。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.7 `frontend/src/components/mealplan/week-calendar.tsx` (更新) で、時間帯ラベルの列を `6rem` から `2.5rem` にする (design.md D6)。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.8 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` (更新) から、使わなくなるキーを外す
    - `pages.meal_plan.item_menu.mark_cooked`
    - `pages.meal_plan.item_menu.unmark_cooked`
    - `pages.meal_plan.cell_actions.item_menu`
    - 完了判定: `grep -rn` で各キーの参照が `frontend/src` と `frontend/e2e` に無いことを確かめ、`cd frontend && npm run build:dev` が通る

## 3. E2E

- [x] 3.1 `frontend/e2e/helpers.ts` (更新) の `openItemMenu` を、レシピ名のボタンをクリックしてメニューを開く形にする。
  同名の品が複数あるときに何番目の行かを指定できる引数を足す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/e2e/mealplan-cooked.spec.ts` (更新) を、チェックボックスでの付け外しに合わせる
    - 作ったを選ぶと品に印が付く (チェックボックスを押し、品のメニューが開かないことも確かめる)
    - 作り置きの 1 行に印を付けると同じ作り置きの全行に印が付く
    - 別調理の行には印が広がらない
    - 作った印を外す
    - 作った印は再起動後も保持される
    - 作り置きとして広げた行は印を引き継ぐ (品のメニューは `openItemMenu` で開く)
    - 入れ替えた品は印の無い状態になる (同上)
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.3 `frontend/e2e/mealplan-item-row.spec.ts` (新設) に describe「セル内の品表示」を置く
    - 複数品は position 順に縦積みされる (各行にチェックボックスがあり、`⋯` が無いことも確かめる)
    - 品行をクリックすると品のメニューが開く
    - キーボードで品のメニューを開く
    - 1 行に収まらないレシピ名は 2 行まで折り返される
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.4 `frontend/e2e/mealplan-batch.spec.ts` (更新) の describe「作り置きの視覚識別」を「作り置きの枠色による識別」に改める
    - 既存の skip の「2 つの作り置きが同時に存在すると別の色になる」は名前をそのまま残す
    - 作り置きの品行には 🔗 アイコンを表示しない (新規に有効なテストとして書く)
    - 完了判定: `task test:e2e` で上記が pass し、`check:scenarios` が通る
- [x] 3.5 `frontend/e2e/mealplan-batch.spec.ts` (更新) の describe「1 行 移動 (ドラッグ&ドロップ)」に、skip で名前だけを足す
    - ドラッグした品行では品のメニューが開かない (Playwright の D&D では発火が安定しないため、Windows 実機で確かめる旨をコメントに書く)
    - 完了判定: `task test:e2e` の `check:scenarios` が通る

## 4. ドキュメント

- [x] 4.1 `docs/product.md` (更新) の §4 S01 を直す
    - 「セル内の操作」の品行の `⋯` の記述を、レシピ名のクリックで開く品のメニューに改め、「作った / 作った印を外す」を項目から外す
    - 「作った」の説明を、品行の左端のチェックボックスで付け外しする形に改める
    - 修飾キー付き D&D の説明の `⋯ → 作り置きを広げる` を、品のメニューの「作り置きを広げる」に改める
    - 「作り置きの視覚識別」の `🔗` アイコンの記述を外し、枠の色だけで見分ける形に改める
    - レシピ名を 2 行まで折り返すことを足す
    - 完了判定: 差分を目視で確認する

## 5. 検証

- [x] 5.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 5.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 5.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 5.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 5.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 5.6 `openspec validate revise-mealplan-item-row --strict` を実行する。完了判定: エラー 0 件
- [x] 5.7 試作と同じ方法で、1088px と 870px の画面幅でレシピ名に残る幅を測る。完了判定: 1088px で単独の品と作り置きの品のどちらも 60px 以上ある
- [x] 5.8 `task build:win` で Windows ビルドを作り、実機で確かめる
    - レシピ名のクリックでメニューが開き、画面の右端の列では左に開く
    - 単独の品をドラッグするとメニューは開かず、別のセルに移動する
    - チェックボックスで作った印を付け外しできる
    - 完了判定: 目視で確認する
