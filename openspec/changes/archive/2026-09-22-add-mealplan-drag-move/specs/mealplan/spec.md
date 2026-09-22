## ADDED Requirements

### Requirement: 1 行 移動 (ドラッグ&ドロップ)

システムは、S01 の 1 品行を別セルに移動する操作を SHALL 提供する。ユーザーはカレンダー上の
品行をドラッグして別セルにドロップすることで、その 1 行を「元セルから削除 → ドロップ先セルの末尾
に挿入」する。移動前の作り置き情報 (`batch_id`) と `recipe_id` は据え置きで、単独割当行
(`batch_member_count == 1`) を移動しても batch_id は維持される。

**作り置みに属する行 (`batch_member_count > 1`) は D&D 対象外**。UI 側で `draggable=false` にして
個別の drag を禁止し、cursor は `not-allowed` を表示する。移動したい場合は既存の
`⋯ → 作り置きから外す` で単独行にしてから D&D する 2 段操作で扱う。repo/service の `MoveEntry` は
backward-compatible のまま (batch_id 維持仕様は残す) だが、フロントは batch member 行からは呼び出さない。

同じセル内 (source と target の date, meal が同じ) にドロップした場合は、既存の並び替え操作
(`ReorderMeal`) で position を差し替える。ドロップターゲットが drop 前後で変化しない場合は
`meal_plans` は変わらない。

D&D 中は drop 可能セルに視覚的なハイライトが SHALL 表示される。ドラッグ元セル自身に戻した場合は
no-op となる。品行 hover 時、draggable な行は `cursor: grab` (drag 中は `grabbing`)、batch member 行は
`cursor: not-allowed` を表示する。ドラッグ開始後、次の 3 役を色で塗り分ける:

- **カーソルに追従する ghost**: 薄い緑 (emerald 系) で描画され、「何を掴んでいるか」を主役として示す。
- **元の位置に残る source 行**: 灰色フェード (`bg-neutral-100 opacity-40`) の placeholder。ここに
  元々あった、の意味だけ伝え、視点は奪わない。
- **drop 可能セル**: 青の outline (blue-400)、drop 対象として提示する。

ghost と source は同じ DOM ノードを時間差で塗り分ける実装 (onDragStart 内で emerald class を DOM に
直接付けて browser にキャプチャさせ、`requestAnimationFrame` で剥がした後に state を反映して灰色
フェードへ切り替える)。

サーバ側は `MoveEntry(sourceDate, sourceMeal, sourcePosition, targetDate, targetMeal)` を
提供し、1 トランザクションで実行する:

1. 対象行を削除する
2. 削除後の source セル内 position を詰め直す
3. target セル (date, meal) の position = max + 1 に、同じ `recipe_id` + 同じ `batch_id` で
   挿入する

失敗時 (存在しない source / 不正な date / meal 等) は既存の型付きエラー (`ErrEntryNotFound`,
`ErrInvalidDate`, `ErrInvalidMeal`, `ErrInvalidPosition`) を返す。

#### Scenario: 別セルにドロップすると 1 行だけ移動する

- **WHEN** 月曜・夕食に「肉じゃが」だけがある状態で、その行を火曜・夕食セルにドラッグ&ドロップする
- **THEN** 月曜・夕食は空セルになる
- **AND** 火曜・夕食に「肉じゃが」が 1 品として追加される (position = 0)

#### Scenario: 作り置みに属する行はドラッグできない

- **WHEN** 月・火・水曜・夕に同じ作り置みの「肉じゃが」が並んでいる状態で、水曜の行の drag を試みる
- **THEN** その行は `draggable=false` で drag が開始せず、cursor は `not-allowed` を表示する
- **AND** カレンダー上の他セルにも drop 可能ハイライトは表示されない
- **AND** `meal_plans` は変わらない

#### Scenario: 別時間帯にドロップしても batch_id は維持される

- **WHEN** 月曜・夕食に単独の「肉じゃが」がある状態で、その行を月曜・昼食セルに D&D する
- **THEN** 月曜・夕食は空セルになる
- **AND** 月曜・昼食に「肉じゃが」が追加され、元の `batch_id` を維持する

#### Scenario: 同一セル内でドロップすると並び替えとして扱う

- **WHEN** 月曜・夕食に「肉じゃが」「みそ汁」がある状態で、「みそ汁」を「肉じゃが」の上にドロップする
- **THEN** 月曜・夕食は「みそ汁」「肉じゃが」の順で position が入れ替わる (ReorderMeal 経由)

#### Scenario: 移動元行 (自セル) 自身に戻すと何も起きない

- **WHEN** 月曜・夕食の「肉じゃが」をドラッグ開始し、同じセル (drop 対象の行間で変化なし) にドロップ
  する
- **THEN** `meal_plans` は変わらず、UI 上の表示も変わらない

#### Scenario: 既存割当のあるセルにドロップすると末尾に追加される

- **WHEN** 月曜・夕食に「肉じゃが」、火曜・夕食に「みそ汁」がある状態で、月曜の「肉じゃが」を
  火曜・夕食セルにドロップする
- **THEN** 月曜・夕食は空セルになる
- **AND** 火曜・夕食は「みそ汁」「肉じゃが」の 2 品構成になる (position 0, 1)

#### Scenario: ドラッグ中は drop 可能セルがハイライトされる

- **WHEN** ユーザーが品行のドラッグを開始する
- **THEN** カレンダー上の全セルが drop 可能状態を示す視覚 (枠色) を持つ
- **AND** ドラッグ中のセルにマウスをかざすと、そのセルは強調表示に切り替わる
