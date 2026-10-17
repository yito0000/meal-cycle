## Context

動機は proposal.md の Why を参照。

S01 は献立の各行が持つ「作り置きの行数」(`batchMemberCount`) だけを見て、枠色、ドラッグの可否、
「作り置きをやめる」の表示、確認のダイアログの削除行数を決めている。
この値は献立の取得で `COUNT(*) OVER (PARTITION BY m.batch_id)` として数えているが、
ウィンドウ関数は `WHERE m.date >= ? AND m.date < ?` で期間に絞ったあとの行にしか掛からない。
S07 の献立の取得 (`ShoppingRepo.loadPlannedGroups`) は相関サブクエリで全行を数えており、正しい値を返している。

「作り置きをやめる」の残す行は `ORDER BY date, position LIMIT 1` で選んでいる。
`ListMergeCandidates` は R23 で `date, 朝昼夕の順, position` に直したが、この並べ方が `SplitBatch` には入っていない。

## Goals / Non-Goals

**Goals:**

- `batchMemberCount` を、期間に関係なくその作り置きの全行の数にする
- `SplitBatch` の残す行を、日付、時間帯、並び順の順で選ぶ

**Non-Goals:**

- フロントの判定式 (`batchMemberCount > 1`) は変えない。値が正しくなれば 4 つの表示はそのまま直る
- 期間の外の行そのものを S01 に返すことはしない

## Decisions

### D1. 作り置きの行数の数え方

**選択肢:**

- (a) 相関サブクエリ `(SELECT COUNT(*) FROM meal_plans a WHERE a.batch_id = m.batch_id)` ← 採用
  — Pros: S07 と同じ書き方になり、2 画面の数え方がそろう。`idx_meal_plans_batch_id` が効く / Cons: 行ごとにサブクエリが走る
- (b) 期間に掛かる作り置きの件数を CTE で先に集計して JOIN する
  — Pros: 集計が 1 回で済む / Cons: S07 と書き方が分かれる。28 日表示でも行数は高々数百で、(a) との差は体感できない
- (c) サービス層で、期間内の batch_id の一覧を別クエリで数え直して埋める
  — Pros: SQL が単純なまま / Cons: 2 回の読み取りのあいだに書き込みが挟まると食い違う。トランザクションが要る

**採用: (a)**。理由: S07 と同じ数え方にそろえることが、この不具合の再発を防ぐいちばん確かな手段である。
性能差は、1 台 1 ユーザーの献立の規模では問題にならない。

### D2. 「作り置きをやめる」で残す行の並べ方

**選択肢:**

- (a) `ORDER BY date, 朝昼夕の順, position` ← 採用 — Pros: `ListMergeCandidates` の代表行と同じ規則になる / Cons: なし
- (b) 選んだ行を残す — Pros: 利用者の意図に近い場合がある / Cons: spec の「最初の日の行だけを残す」と、
  `revise-mealplan-destructive-actions` で決めた確認のダイアログの文言を変えることになる。R31 で挙動は変えないと決めている

**採用: (a)**。時間帯の順は `ListMergeCandidates` と同じ `CASE meal WHEN 'breakfast' THEN 0 ...` で書く。
Go 側の `mealRank` と SQL の `CASE` の 2 か所の対応は、既存のコメントが示すとおり手で合わせる。

### D3. テストの置き場所

数え方と並べ方はどちらもリポジトリ層の SQL にあるので、リポジトリ層のテストで確かめる。
サービス層のテストはフェイクのリポジトリを使っているため、この 2 点を確かめられない。
画面の 4 つの表示は E2E で確かめる (tasks.md の 3 章)。

## Risks / Trade-offs

- [Risk] 期間の外に続く作り置きの端の行が D&D できなくなり、これまでの「端の行なら動かせた」使い方が通らなくなる
  → spec はもともと作り置きの行の D&D を禁じており、端の行だけ動かせたのは不具合である。動かしたい場合の手順は product.md に書いてある
- [Risk] SQL の `CASE meal` と Go の `mealRank` が 3 か所に増え、片方だけ直す事故が起きうる
  → 時間帯は 3 つで固定しており、追加の予定は無い。今回は増やすだけにとどめ、まとめるのは時間帯を増やすときに行う

## Migration Plan

スキーマとデータは変わらない。既存の DB はそのまま使える。
