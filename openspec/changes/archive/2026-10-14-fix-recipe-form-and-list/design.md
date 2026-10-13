## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/recipe/spec.md](specs/recipe/spec.md) を参照。

現状は次のとおりである。

- `RecipeEditForm` は、キャンセルとパンくずの `onSelect` を `leave(go)` で包む。dirty なら `pendingLeave` に移動先の関数を入れ、`UnsavedChangesDialog` を開く
- サイドバーは TanStack Router の `<Link>` で、`leave` を通らない
- 保存は、`routes/recipes/new.tsx` と `$recipeId.edit.tsx` の `onSave` の中で、サービスの呼び出しに成功したあと `router.navigate` する
- `RecipeFormValues` は `servings: number` と、バインディングの `RecipeIngredient` (`amount: number`) をそのまま持つ。
  入力欄の `onChange` は `Number.parseInt` / `Number.parseFloat` の結果を入れ、`NaN` なら 1 または 0 に倒す
- S06 のフォーム (`stock-form.ts`) は、量を文字列で持ち、保存時に検証して数値へ直す
- `RecipeTable` は `r.updatedAt` (RFC3339、UTC) をそのまま描く。S03 は `formatDisplayDate(formatDate(new Date(recipe.updatedAt)))` で整形している
- ルータは memory history で、`@tanstack/react-router` は 1.170 系である

## Goals / Non-Goals

**Goals:**

- S04 から出る経路を、種類を問わず 1 か所で止める
- 数値欄の持ち方を S06 とそろえる

**Non-Goals:**

- フォームのライブラリの導入

## Decisions

### D1. 画面の移動をどこで止めるか

**選択肢:**

- (a) サイドバーの `<Link>` に、S04 が dirty かどうかを伝える仕組み (context やグローバルな state) を足す
  — Pros: 既存の `leave` の作りを保てる
  / Cons: 出口が増えるたびに同じ配線が要る。レイアウトがレシピのフォームの事情を知ることになる
- (b) TanStack Router の `useBlocker` で、S04 にいるあいだの画面の移動そのものを止める ← 採用
  — Pros: サイドバー、パンくず、キャンセル、将来増える出口を、フォームの中の 1 か所で扱える。`pendingLeave` が要らなくなる
  / Cons: 保存に成功したあとの移動まで止めてしまうので、通す仕組みが要る (D2)
- (c) 入力を下書きとして保存し、戻ってきたら復元する
  — Pros: 確認ダイアログが要らない
  / Cons: 下書きの保存先、破棄の操作、編集と新規の区別が要る。今回の範囲を超える

**採用: (b)**。

```tsx
const blocker = useBlocker({
  shouldBlockFn: () => form.isDirty && !leavingAfterSave.current,
  withResolver: true,
  enableBeforeUnload: false,
});
```

- `blocker.status === "blocked"` のあいだ `UnsavedChangesDialog` を開く
- 「破棄」は `blocker.proceed()`、「編集を続ける」は `blocker.reset()` を呼ぶ
- キャンセルとパンくずは、`leave` で包まずに移動先の関数をそのまま呼ぶ。dirty なら `useBlocker` が止める

`enableBeforeUnload` は false にする。
Wails の webview にはページの再読み込みや URL 欄が無く、`beforeunload` の確認は出番が無い。

### D2. 保存に成功したあとの移動を通す

**選択肢:**

- (a) 保存に成功したら、フォームの初期値を現在の値で置き換えて dirty を解いてから移動する
  — Pros: 「保存済みなら dirty でない」という意味に沿う
  / Cons: 移動は `onSave` (ルート側) の中で起きる。初期値の置き換えは state の更新で、次の描画まで `isDirty` に反映されない
- (b) 保存を始めるときに ref を立て、`shouldBlockFn` でその ref を見る ← 採用
  — Pros: ref は同期で読めるので、描画を待たずに通せる
  / Cons: 失敗したときに ref を戻し忘れると、以後の移動が素通りになる

**採用: (b)**。
`saveMutation` の `onMutate` で `leavingAfterSave.current = true`、`onError` で `false` に戻す。
戻し忘れは E2E (保存に失敗したあとでサイドバーを押すと確認される) では確かめにくいので、`onError` の 1 か所に書き、コメントで理由を残す。

### D3. 数値欄の持ち方

**選択肢:**

- (a) 数値のまま持ち、空のときだけ別のフラグで覚える
  — Pros: 型の変更が小さい
  / Cons: 「1.」のような入力途中の表記も失う。フラグと値の 2 つを同期させる必要がある
- (b) フォームの中では文字列で持ち、保存時に数値へ直す ← 採用
  — Pros: S06 と同じ持ち方になる。入力途中の内容がそのまま残る
  / Cons: `RecipeFormValues` がバインディングの型から離れ、変換が 2 か所 (読み込み、保存) に要る

**採用: (b)**。

```ts
export type IngredientFormRow = { name: string; amount: string; unit: string };
export type RecipeFormValues = {
  name: string; servings: string; tags: string[];
  ingredients: IngredientFormRow[]; steps: string; notes: string;
};
```

- 読み込み (`recipeToFormValues`)：`servings` は `String(r.servings)`。`amount` は、0 なら空文字、それ以外は `String(amount)` にする。
  量 0 は「量を持たない材料」を表すので、欄に 0 を出さないほうが保存した意味に合う
- 保存 (`handleSave`)：`servings` は `Number(values.servings)` に直し、空や整数でない値は人数のエラーにする。
  `amount` は、空なら 0、それ以外は `Number(amount)` にする
- `onSave` に渡す `SaveInput` の型 (数値) は変えない。ルート側の 2 ファイルは触らない

dirty の判定は文字列どうしの deep 比較になる。
「200」を「200.0」に書き換えると dirty になるが、利用者が欄を触った結果なので、確認が出て困る場面ではない。

### D4. 更新日の整形

`frontend/src/lib/format-date.ts` に `formatTimestampDate(rfc3339: string): string` を足し、S02 と S03 の両方から呼ぶ。
中身は S03 が今やっている変換 (`new Date` でローカルの年月日を取り、`yyyy/mm/dd` にする) と同じである。
他案 (Go 側で整形した文字列を返す) は、Go がサーバの時刻帯で整形することになり、「日時は RFC3339 の UTC で渡す」という既存の契約を崩すので見送る。

並べ替え (`compareRecipes`) は、これまでどおり RFC3339 の文字列で比べる。表示だけを変える。

### D5. 行をキーボードで開く

**選択肢:**

- (a) `<tr>` に `tabIndex={0}`、`role="button"`、`onKeyDown` を付ける
  — Pros: 見た目が変わらない
  / Cons: 表の行にボタンの役割を載せると、読み上げで列の構造が失われる
- (b) レシピ名のセルの中身を `<button>` にする ← 採用
  — Pros: フォーカス、Enter、Space が標準の挙動で付く。表の構造はそのまま残る
  / Cons: 行のクリックとボタンのクリックが二重に発火しないよう、片方に寄せる必要がある

**採用: (b)**。
ボタンの `onClick` で `stopPropagation` し、行の `onClick` (行のどこを押しても開く) は残す。
ボタンの見た目は文字のままにし、フォーカスのときだけ輪郭を出す。

## Risks / Trade-offs

- [`useBlocker` は、同じルートの中の search の書き換えも止めうる] → S04 は search を書き換えない。`shouldBlockFn` に条件は足さない
- [保存に失敗したあと ref が立ったままだと、未保存の確認が出なくなる] → D2 のとおり `onError` で戻す
- [量 0 の材料を編集で開くと、欄が空で表示される] → D3 のとおり意図した表示である。S03 も量 0 の数字を出さない
- [既存の E2E が、人数欄や量の欄に `fill` で値を入れている] → `fill` は欄を空にしてから入力するので、文字列で持つ形でも同じ結果になる。`task test:e2e` で確かめる
