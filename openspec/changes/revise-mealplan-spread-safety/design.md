## Context

動機は proposal.md の Why を参照。

統合ダイアログ (`SpreadBatchDialog`) は、S01 が持つ表示期間の行 (`entries`) を `existingEntries` として受け取り、
既存割当の背景色と「起点と同じ作り置きの同じレシピ」の判定に使っている。
格子は起点日から 3 日後までの 4 行なので、起点が週の後半にあると、格子の一部が表示期間の外に出る。

反映は `MealPlanService.SpreadEntry` → `MealPlanRepo.Spread` の 1 トランザクションで行う。
上書きのときは対象セルを `DELETE` してから入れ、消した行の内容は返さない。
品の削除と時間帯全体のクリアは、`revise-mealplan-destructive-actions` で消した行 (`RemovedEntry`) を返し、
`RestoreEntries` で元のセルの元の並び順に戻せるようになっている。

削除済みレシピの確認 (`ensureActiveRecipe`) は `Assign` と `Replace` だけが呼んでいる。
時間帯の順 (`mealRank`) はリポジトリ層にだけあり、ドメイン層の `ValidateSpreadTargets` は日付しか比べていない。

## Goals / Non-Goals

**Goals:**

- 格子の既存割当と選べないセルの判定を、表示期間に関係なく正しくする
- 「作り置きを広げる」を、追加でも上書きでも 1 回の操作で取り消せるようにする
- 起点と同じ日の起点より前の時間帯と、削除済みレシピの品を、広げる対象から外す
- 統合ダイアログの日付の表記と、「作った」チェックボックスの読み上げ名を直す

**Non-Goals:**

- 時間帯全体の「連続 N 日コピー」の取り消しと衝突判定 (proposal の Non-goals を参照)
- 取り消し用の情報をサーバ側に保存すること。品の削除と同じく、通知が持つ情報だけで戻す

## Decisions

### D1. 格子の範囲の献立の取り方

**選択肢:**

- (a) 統合ダイアログが開いたときに、起点日から 7 日ぶんを既存の `ListPeriod(起点日, 1)` で取得する ← 採用
  — Pros: サービスに手を入れずに済む。`ListPeriod` は開始日を月曜に寄せることを呼び出し側の責任にしており、任意の日付を受け付ける。
  query key を `["mealplan", "spread-grid", 起点日]` にすれば、既存の `invalidateQueries({ queryKey: ["mealplan"] })` で一緒に古くなる
  / Cons: 格子は 4 日なのに 7 日ぶん読む。ダイアログを開くたびに 1 回読み取りが増える
- (b) 格子の範囲だけを返す新しいサービスのメソッドを足す — Pros: 読む量が最小 / Cons: 用途が 1 つのメソッドが増え、bindings と E2E の shim も増える
- (c) S01 の取得範囲を、表示期間の前後 3 日ぶん広げる — Pros: ダイアログは今のまま `entries` を使える /
  Cons: S01 の取得の意味 (表示期間の行だけを返す) が変わり、spec の「週データ取得の境界」のシナリオと食い違う

**採用: (a)**。読み取りが 1 回増えても、1 台 1 ユーザーの規模では体感できない。
取得が終わるまでは、格子のチェックボックスを押せない状態で表示する (背景色が付く前に上書きを選べないようにするため)。
`existingEntries` の受け渡しはやめ、ダイアログは取得した行だけを使う。

### D2. 取り消しの単位と API

**選択肢:**

- (a) `SpreadEntry` が `SpreadResult { addedEntryIds, removed []RemovedEntry }` を返し、`UndoSpread(result)` が 1 トランザクションで戻す ← 採用
  — Pros: 追加した行の削除と消した行の復元が一緒に戻り、一部だけ戻った状態にならない。復元は既存の `restoreEntryTx` を使い回せる
  / Cons: サービスのメソッドが 1 つ増え、`SpreadEntry` の戻り値が変わる (BREAKING、内部 API)
- (b) 画面が `RemoveEntry` を追加した行の数だけ呼び、続けて `RestoreEntries` を呼ぶ
  — Pros: サービスに手を入れない / Cons: 途中で失敗すると一部だけ戻る。spec の「一部だけ戻った状態にはしない」を守れない
- (c) 上書きのときだけ取り消しを出す — Pros: 通知が減る / Cons: 追加で広げたときに、どのセルに入ったかを通知で確かめる手段が無い。
  追加でも同じ品があるセルは黙って飛ばしているので、セルの数を示す通知には意味がある

**採用: (a)**。通知は追加でも上書きでも出す。
`UndoSpread` は、追加した行のうちまだ残っているものだけを `DELETE` し、詰め直し (`renumberMeal`) をセルごとに 1 回行う。
続けて `removed` を `restoreEntryTx` で戻し、最後に `pruneCookedMarks` を呼ぶ。
`removed` の中に同じ ID の行がすでにあれば、`restoreEntryTx` が `ErrInvalidRestore` を返し、全体を巻き戻す。
`UndoSpread` の入力は画面を経由して戻ってくるので、`RestoreEntries` と同じ検証 (日付、時間帯、並び順、ID、日時の形式) を
サービス層で通す。検証は既存の `RestoreEntries` の中身を関数に切り出して共有する。

### D3. 上書きで消した行の読み方

上書きのセルを `DELETE` する前に、既存の `loadRemovedEntries` で `m.date = ? AND m.meal = ?` の行を読む。
読んだ行には作った印の日時が付いているので、戻したときに印も戻る。
同じセルを 2 回読むことは無い (`ValidateSpreadTargets` が重複したセルを拒否する)。

### D4. 起点より前の時間帯の拒否

**選択肢:**

- (a) ドメイン層に `MealRank` を移し、`ValidateSpreadTargets` で「起点日と同じ日で、時間帯の順位が起点以下」を拒否する ← 採用
  — Pros: 検証が既存のドメイン関数に収まる。リポジトリ層の `mealRank` も `mealplan.MealRank` に寄せて 1 か所にできる / Cons: なし
- (b) リポジトリ層の `Spread` で拒否する — Pros: 既存の `mealRank` を使える / Cons: 検証がドメインとリポジトリに分かれる

**採用: (a)**。エラーは既存の `ErrInvalidSpreadTarget` を使う (画面からは選べないので、専用の文言は要らない)。
フロントは `MEAL_RENDER_ORDER` の添字で同じ判定をし、該当セルを選択不可の印で描く。
読み上げ名は `pages.meal_plan.spread.earlier_meal_cell_label` (「起点より前の時間帯」) にする。

### D5. 削除済みレシピの拒否とエラー

**選択肢:**

- (a) 新しい `ErrArchivedRecipe` を返す ← 採用 — Pros: 通知で「削除済みのレシピは広げられません」と理由を示せる /
  Cons: エラーの対応表に 1 行増える
- (b) `Assign` と同じ `ErrUnknownRecipe` を返す — Pros: 対応表が変わらない / Cons: 「指定したレシピが見つかりません」と出るが、
  利用者の目の前のセルにはそのレシピの品が見えているので、理由が伝わらない

**採用: (a)**。`Assign` と `Replace` は spec どおり `ErrUnknownRecipe` のままにする (存在しないレシピと区別しない)。
`Spread` は起点の行のレシピを、`CopyLeftover` は起点の時間帯の全品のレシピを、トランザクションの中で確かめる。

時間帯全体のコピーの扱いは、次の 2 案から選んだ。

- (a) 削除済みレシピの品が 1 つでもあれば全体を拒否する ← 採用 — Pros: 「どのセルにも反映しない」の既存の約束と同じ形になる /
  Cons: 残りの品だけコピーしたい場合は、先に削除済みの品を消す必要がある
- (b) 削除済みレシピの品だけを飛ばしてコピーする — Pros: 手間が少ない / Cons: 黙って飛ばすと、コピーしたつもりの品が欠ける

### D6. 日付の表記と読み上げ名

統合ダイアログの行見出しは `Intl.DateTimeFormat(i18n.language, { month: "numeric", day: "numeric", weekday: "short" })` で整えている。
起点の説明 (`origin_hint`) とチェックボックスの読み上げ名 (`cell_checkbox_label`) も、同じ `formatRow` を通した文字列を渡す。
「作った」チェックボックスは `pages.meal_plan.cell.cooked` を「{{name}}を作った」に改め、レシピ名を渡す。
既存の E2E は `getByRole("checkbox", { name: t.cell.cooked })` で探しているが、
文言が変わるので、`t.cell.cooked.replace("{{name}}", レシピ名)` で組み立てる形に直す。

## Risks / Trade-offs

- [Risk] 統合ダイアログを開いた直後、取得が終わるまで格子を操作できない → 取得は 1 回の SQLite の読み取りで、待ちはほぼ見えない。
  取得に失敗したときは、ダイアログを閉じてエラーを通知で示す
- [Risk] 取り消しの通知を出しているあいだに別の操作で同じセルを変えると、戻したあとの並び順が上書き前と違うことがある
  → `RestoreEntries` と同じく、戻す位置はセルの品数で頭打ちにする。spec は「元のセルの元の並び順」を、間に別の操作が無い場合について約束している
- [Risk] `SpreadEntry` の戻り値が変わり、E2E のヘルパー (`spreadEntry`) と `cmd/e2e-server` のテストが壊れる → tasks.md で同時に直す
- [Risk] 削除済みレシピを含む時間帯をコピーできなくなり、これまでできた操作が失敗する
  → 通知で理由を示す。削除済みの品を消せばコピーできる

## Migration Plan

スキーマとデータは変わらない。既存の DB はそのまま使える。
