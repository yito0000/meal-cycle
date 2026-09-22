## 1. repo: MoveEntry の追加

- [x] 1.1 `internal/storage/sqlite/mealplan_repo.go` に `MoveEntry(ctx, sourceDate, sourceMeal
      string, sourcePosition int, targetDate, targetMeal string) error` を新規追加する。1 tx 内で:
    - 対象行 (source) の `recipe_id` と `batch_id` を SELECT
    - source 行を DELETE + source セル内の後続 position を詰め直す
    - target セルの `MAX(position)` + 1 位置に (recipe_id, batch_id) を INSERT する
- [x] 1.2 内部ヘルパ `removeRowTx(ctx, tx, date, meal, position) error` を切り出す。既存 `Remove` の
      内部処理を共通化して MoveEntry から再利用する
- [x] 1.3 同一セル (source == target) を repo に渡した場合は `ErrInvalidPosition` を返し、UI 側で
      ReorderMeal に流させる (責務分離)
- [x] 1.4 `internal/storage/sqlite/mealplan_repo_test.go` に以下ケースを追加する
    - MoveEntry: 別セルへの単純移動で source が空、target 末尾に 1 行追加される
    - MoveEntry: source 行 削除後に後続 position が詰め直される
    - MoveEntry: 作り置きに属する行を移動しても batch_id が維持される
    - MoveEntry: 別時間帯 (source lunch → target dinner) でも batch_id が維持される
    - MoveEntry: 存在しない source は ErrEntryNotFound
    - MoveEntry: source == target は ErrInvalidPosition
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)

## 2. service 層

- [x] 2.1 `internal/service/mealplan.go` の `mealPlanRepo` インターフェイスに `MoveEntry` を追加する
- [x] 2.2 `MealPlanService.MoveEntry(ctx, sourceDate, sourceMeal string, sourcePosition int,
      targetDate, targetMeal string) error` を追加。入力バリデーション (date / meal / position) を
      通し、repo に委譲する
- [x] 2.3 `internal/service/mealplan_test.go` の `fakeMealPlanRepo` に `MoveEntry` を実装し、service
      層のバリデーションを検証する
    - 不正 date / 不正 meal / 負 position のケースをそれぞれ型付きエラーで弾く
    - 別セルへの移動が反映される
    - (完了判定: `go test ./internal/service/...` が pass する)

## 3. Wails bindings 再生成 + shim 更新

- [x] 3.1 `wails3 generate bindings` を実行する。完了判定: `MoveEntry` が生成される
- [x] 3.2 `frontend/e2e/bindings-shim/mealplanservice.ts` に `MoveEntry` を追加する。完了判定:
      `npm run type-check` が通る

## 4. i18n 追加

- [x] 4.1 `frontend/src/locales/{ja,en}.json` に以下 key を追加する
    - `mealplan.dnd.item_grabbable_label` — ドラッグ可能行の aria-label (「ドラッグして別セルへ移動」)
    - `mealplan.dnd.cell_drop_label` — ドロップ可能セルの aria-label
    - `mealplan.dnd.moved_announcement` — 移動後のスクリーンリーダーアナウンス
      (「{{recipeName}} を {{fromDate}} {{fromMeal}} から {{toDate}} {{toMeal}} に移動しました」)

## 5. フロント: D&D handler

- [x] 5.1 `frontend/src/components/mealplan/meal-plan-cell.tsx` の品行 `<div>` に `draggable`,
      `onDragStart` を追加。dataTransfer に `{ date, meal, position }` を JSON で載せる
- [x] 5.2 同ファイルのセル外側 `<div>` に `onDragOver` (preventDefault で drop 許可) と `onDrop`
      handler を追加。onDrop で source 情報をパースし、コールバック
      `onDropEntry(source, target)` を呼ぶ
- [x] 5.3 ドラッグ中の視覚状態を管理する。`useState<boolean>` を親コンポーネント
      (WeekCalendar) に置き、`onDragStart` で true、`onDragEnd` で false。true の間は各セルに
      枠強調を付ける (Tailwind: `outline-2 outline-blue-400`)
- [x] 5.4 `frontend/src/components/mealplan/week-calendar.tsx` に `MoveEntry` mutation を追加。
      `onDropEntry` を受け取り、以下で分岐する
    - source == target (同一セル): 既存の `ReorderMeal` を呼ぶ (drop 対象 entry と source の
      position を入れ替える)
    - source != target (別セル): `MoveEntry` を呼ぶ

## 6. ドキュメント + E2E

- [x] 6.1 `docs/product.md` §S01 に「D&D による 1 行 移動」を追記する。既存の「セル内の操作」の
      直後に、「移動」を独立した節として書く
- [x] 6.2 `docs/scenarios/mealplan.feature` に以下シナリオを追加する
    - 別セルにドロップすると 1 行だけ移動する
    - 作り置きに属する行を移動しても作り置きが維持される
    - 別時間帯にドロップしても batch_id は維持される
    - 同一セル内でドロップすると並び替えとして扱う
    - 既存割当のあるセルにドロップすると末尾に追加される
- [x] 6.3 `frontend/e2e/mealplan-*.spec.ts` に該当シナリオの test 名を追加する (Playwright の D&D は
      近似実行のため `test.skip` で名前だけ残し、手動確認で担保する)

## 7. lint / format / テスト最終ゲート

- [x] 7.1 `task lint`。完了判定: exit code 0
- [x] 7.2 `task format`。完了判定: 整形差分のみ
- [x] 7.3 `go test ./internal/... ./`。完了判定: exit code 0
- [x] 7.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev`。完了判定:
      exit code 0
- [x] 7.5 `task test:e2e`。完了判定: exit code 0
- [x] 7.6 Windows ネイティブビルドで手動確認する。日本語 IME の絡む場面ではないが、D&D は
      OS 依存の挙動があるため実機確認は必須。手順は CLAUDE.md「動作確認の 2 階建て構成」に従う

## 8. Windows 実機フィードバック反映 (design D1/D6 差分)

- [x] 8.1 `frontend/src/components/mealplan/meal-plan-cell.tsx` の品行 `<div>` を、`entry.batchMemberCount > 1`
      のとき `draggable={false}` にする。onDragStart / onDrop / onDragEnd の handler は付けない
- [x] 8.2 同ファイル品行に cursor クラスを追加する
    - 単独割当行 (`batchMemberCount <= 1`): `cursor-grab active:cursor-grabbing`
    - batch member 行 (`batchMemberCount > 1`): `cursor-not-allowed`
- [x] 8.3 `docs/scenarios/mealplan.feature` の D&D シナリオを更新する
    - 「作り置きに属する行を移動しても作り置きが維持される」を削除
    - 「作り置みに属する行はドラッグできない」を追加 (spec.md の新規シナリオと同じ文面)
- [x] 8.4 `frontend/e2e/mealplan-batch.spec.ts` の D&D セクションを spec 更新に合わせて差替え
    - 「作り置きに属する行を移動しても作り置きが維持される」を削除
    - 「作り置みに属する行はドラッグできない」を `test.skip` で追加
- [x] 8.5 lint / format / go test / type-check / test:e2e を再実行し、全 pass を確認
- [x] 8.6 Windows ネイティブビルドで再確認 (cursor が grab/grabbing/not-allowed に切り替わること /
      batch member 行が drag 開始しないこと)

## 9. source 行の視覚強調 (Windows 実機フィードバック 2 回目)

- [x] 9.1 `frontend/src/components/mealplan/week-calendar.tsx` の `isDragging: boolean` state を
      `dragSource: DragSource | null` に置き換え、onEntryDragStart で source をセット、onEntryDragEnd と
      handleDropEntry で null に戻す
- [x] 9.2 `MealPlanCell` に props `dragSource: DragSource | null` を追加。既存 `isAnyDragging` は
      `dragSource !== null` に置き換え (呼び出し側で導出)
- [x] 9.3 `meal-plan-cell.tsx` で ghost / source / drop target を 3 色で塗り分ける
      (Windows 実機フィードバック 3-5 回目)
    - ghost (カーソル追従): 薄い緑 (`bg-emerald-100 ring-2 ring-emerald-400`)
    - source (元の位置に残る): `bg-neutral-100 opacity-40` の灰色フェード
    - drop target セル: 青 outline (blue-400、既存)
    - 実装: `onDragStart` で emerald class を DOM に `classList.add` (同期 → browser がキャプチャ) →
      `requestAnimationFrame` で `classList.remove` してから `setDragSource` を呼ぶ。同じ DOM ノードを
      時間軸で 2 種類の姿として使い分けることで、ghost と source の色分けが両立する
- [x] 9.4 lint / format / go test / type-check / test:e2e を再実行し、全 pass を確認
- [x] 9.5 Windows ネイティブビルドで再確認 (source 行がハイライトされ、どの品を移動中か視認できる)
