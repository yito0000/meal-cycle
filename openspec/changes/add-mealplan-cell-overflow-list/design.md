## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/mealplan/spec.md](specs/mealplan/spec.md) を参照。

現状は次のとおりである。

- `MealPlanCell` は、品を position 順に並べ、先頭から `preset.visibleItems` 件だけを描く。残りは `<div>` の「他 N 件」になる
- 品行の描画 (チェックボックス、レシピ名のボタン、`CellItemMenu`、ドラッグの開始と終了、ghost の色付け) は、`MealPlanCell` の `map` の中に約 90 行で書かれている
- 品のメニューの開閉は、`MealPlanCell` が `menuPosition` (開いている品の position) で持つ
- `CellItemMenu` は Radix の `DropdownMenu` で、`relative` な品行に重ねた透明な要素を位置の基準にしている
- `frontend/src/components/ui/popover.tsx` (Radix の `Popover`) は、S07 の日付選択で使っている

## Goals / Non-Goals

**Goals:**

- 畳まれた品に、セルの中の品と同じ操作 (作った印、品のメニューの全項目) が届く
- 品行の見た目と挙動を、セルの中と一覧の中で 1 つの部品から出す

**Non-Goals:**

- 一覧の中でのドラッグ

## Decisions

### D1. 畳まれた品をどこに出すか

**選択肢:**

- (a) 「他 N 件」を押すと、セルがその場で縦に伸びて全品を出す
  — Pros: 浮いた面が要らない。ドラッグもそのまま効く
  / Cons: 同じ行のほかのセルも一緒に伸び、4 週表示の一望性が崩れる。どのセルを開いたかを覚える状態が要る
- (b) 「他 N 件」を押すと、セルの近くに全品の一覧をポップオーバーで出す ← 採用
  — Pros: カレンダーの格子は動かない。既存の `Popover` を使える。Esc と外側のクリックで閉じる挙動が最初から付く
  / Cons: 一覧の中の品はドラッグできない
- (c) 「他 N 件」を押すと、期間を 1 週に切り替える
  — Pros: 部品が増えない
  / Cons: 見ていた 4 週の文脈を失う。表示設定は保存されるので、戻す手間も生じる

**採用: (b)**。
4 週表示は俯瞰のための表示であり、格子を動かさないことを優先する。

### D2. 一覧に出す品の範囲

**選択肢:**

- (a) 畳まれた品 (上限を超えた分) だけを出す
  — Pros: 一覧が短い
  / Cons: 「上へ移動」で畳まれた品が上限の内側へ入ると、一覧から消えてセルの中に現れる。押した直後に対象が別の場所へ移る
- (b) そのセルの全品を出す ← 採用
  — Pros: 並び順が 1 か所で見える。移動しても一覧の中にとどまる
  / Cons: セルに見えている品が一覧にも出る

**採用: (b)**。

### D3. 品行の切り出し

品行を `MealItemRow` に切り出し、セルと一覧の両方から使う。

```tsx
type MealItemRowProps = {
  entry: Entry;
  cellItemCount: number;
  fontClass: string;
  draggable: boolean;           // 一覧では false。作り置きの行もセル側で false を渡す
  isDragSource: boolean;
  onAction: (action: EntryAction, entry: Entry) => void;
  onDragStart?: (source: DragSource) => void;
  onDragEnd?: () => void;
  onDropOnRow?: (e: React.DragEvent, position: number) => void;
};
```

品のメニューの開閉は、`MealItemRow` が自分の state として持つ。
`MealPlanCell` の `menuPosition` と `menuButtonRef` は要らなくなる。
他案 (開閉を親に残す) は、セルと一覧の両方に同じ state を書くことになるので見送る。

一覧では `fontClass` に `text-sm` を固定で渡す。
セルの文字サイズは表示密度で縮むが、一覧は読むために開く面なので縮めない。

### D4. ポップオーバーの中の品のメニュー

Radix の `DropdownMenu` は `Popover` の中でも開ける。どちらもポータルへ描かれ、`DropdownMenu` を閉じても `Popover` は開いたまま残る。
品のメニューから開く別のダイアログ (レシピ選択、作り置きを広げる、作り置きをやめるの確認) は `WeekCalendar` が持っており、一覧の外に描かれる。
ダイアログが開くと、`Popover` は外側の操作として閉じる。これは望ましい挙動である (ダイアログの下に一覧が残らない)。

### D5. 一覧を閉じる条件

`CellOverflowList` は、`overflow > 0` のときだけ描く。
品を削除して上限以下になると、部品ごとアンマウントされ、一覧も閉じる。
追加の制御は書かない。

作った印の付け外しや上下の移動では品数が変わらないので、一覧は開いたまま内容だけが更新される。

## Risks / Trade-offs

- [一覧の中の品はドラッグできないので、畳まれた品を別のセルへ動かすには 1 週表示に切り替える必要がある] →
  Non-goal として受け入れる。4 週表示で多い操作は、作った印と削除である
- [品行の切り出しで、ドラッグの ghost の色付け (rAF で一時クラスを剥がす処理) を壊すおそれがある] →
  処理は変えずに移すだけにする。ドラッグの E2E は `test.skip` のままなので、`task dev` で 1 度ドラッグして目視する (tasks.md)
- [`Popover` の中で `DropdownMenu` を閉じたとき、フォーカスが一覧の外へ飛ぶと一覧が閉じる] →
  `CellItemMenu` はすでに `onCloseAutoFocus` でレシピ名のボタンへフォーカスを戻している。同じ処理を `MealItemRow` に移す
