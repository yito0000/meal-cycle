## Context

動機は proposal.md の Why を参照。

S09 の保存と読み出しは `SettingsService.Get` / `Save` / `Reset` が担い、戻り値は保存できる値だけを持つ `settings.Values` である
(add-settings-screen design.md D4)。
「消せない日数」は Go の `settings.ShoppingRequiredDays` と、フロントの `SHOPPING_REQUIRED_DAYS` の 2 か所にある。
既定の日数を消したときに選び直す 7 日は、フロントの `SHOPPING_FALLBACK_DAYS` にだけある。

「作り置きを広げる」の対象日の上限は、Go の `MealPlanService.maxSpreadOffsetDays` が検証に使う。
フロントの `WeekCalendar` は同じ値を `Math.max(...leftoverCopyPresets) - 1` で計算し、`SpreadBatchDialog` に渡す。
`SpreadBatchDialog` は既存の品の判定に `ListPeriod(source.date, 1)` で 7 日分の献立を取る。
`ListPeriod` は 1 / 2 / 4 週しか受け付けないので、日数をそのまま渡せない。

## Goals / Non-Goals

**Goals:**

- 設定に結びついた規則を Go だけに置き、フロントは戻り値を表示に使うだけにする
- 統合ダイアログの格子の行数と、既存の品を判定する献立の範囲を、同じ値から決める

**Non-Goals:**

- `settings.Values` の形は変えない。保存の往復に、保存しない値を混ぜない
- S01 の期間の選択肢 (1 / 2 / 4 週) は扱わない (proposal.md の Non-goals)

## Decisions

### D1. S09 の制約を返す口

**選択肢:**

- (a) `SettingsService.Constraints()` を足し、`settings.Constraints` を返す ← 採用
  — Pros: `Values` の往復に保存しない値が混ざらない。制約は版の中で変わらないので、フロントは 1 回読めば足りる /
  Cons: S09 を開くときの呼び出しが 1 つ増える
- (b) `Values` に `shoppingRequiredDays` などを足す — Pros: 呼び出しが増えない /
  Cons: `Save` に送り返される値になり、Go は受け取った値を捨てる必要がある。`diffRows` が既定値との差を見るときにも除外がいる
- (c) フロントに写しを残し、Vitest で値を固定する — Pros: Go を触らない /
  Cons: Vitest は Go の値を読めないので、両方を直し忘れる事故は防げない

**採用: (a)**。`Constraints` は次の 2 つを持つ。

- `shoppingRequiredDays`：`ShoppingRequiredDays` の複製
- `shoppingFallbackDays`：既定の日数を消したときに選び直す日数。`ShoppingRequiredDays` の先頭 (7) とし、必ず消せない日数に含まれる値にする

フロントは `useSettingsConstraints()` (`staleTime: Infinity`) で読み、S09 の画面だけで使う。
`SettingsGate` には入れない。S09 以外の画面が使わない値で、起動を待たせる理由が無いためである。

### D2. 統合ダイアログの格子に必要な値の返し方

**選択肢:**

- (a) `MealPlanService.SpreadGrid(entryID)` を足し、起点から選べる日数と、その範囲の献立を 1 回で返す ← 採用
  — Pros: 検証に使う上限と、判定に使う献立の範囲が、同じ呼び出しの中で同じ値から決まる。
  リポジトリの `ListPeriod` は日数で範囲を受け取れるので、週の丸めが要らない / Cons: `ListPeriod` と似た読み出しが 1 つ増える
- (b) 上限だけを返す `SpreadMaxOffsetDays()` を足し、献立はフロントが週数に丸めて `ListPeriod` で取る
  — Pros: メソッドが小さい / Cons: 1 / 2 / 4 週への丸めをフロントに書くことになり、写しが別の形で残る
- (c) 上限を `Constraints` に入れる — Pros: 口が 1 つで済む /
  Cons: 上限は作り置きの日数の候補から決まり、保存のたびに変わる。`Constraints` は設定の query key の下で取り直さない値なので、古いまま残る

**採用: (a)**。`SpreadGrid` の戻り値は次のとおり。

- `maxOffsetDays`：`maxSpreadOffsetDays()` と同じ値
- `entries`：起点の行の日付から `maxOffsetDays + 1` 日分の献立。中身は `ListPeriod` と同じ `Entry`

フロントは query key `["mealplan", "spread-grid", source.id]` (`source.id` は起点の行の ID) で取る。
`["mealplan"]` の取り直しと、S09 の保存のあとの取り直し (設定以外の全 query) の両方で古い値が消える。

### D3. 格子の行数を持つ場所

`SpreadBatchDialog` は `initialFutureDays` を props で受け取らず、`SpreadGrid` の `maxOffsetDays` で行を作る。
読み終えるまでは、今の「取得前は反映できない」(`canSubmit` が `isLoaded` を見る) と同じく、チェックボックスを無効にする。
行の数が読み終えるまで決まらないので、読み込み中は起点日の 1 行だけを描く。
`WeekCalendar` に残る `spreadPresetDays` (既定でチェックする日数) は、利用者に見せる初期値の決め方で、Go の検証とは関係しないので残す。

## Risks / Trade-offs

- [Risk] 統合ダイアログを開いてから格子の行がそろうまで、行数が 1 から増えて見える
  → 1 台のローカル SQLite で、読み出しは数 ms で終わる。今も既存の品の色分けは同じ待ち方をしている
- [Risk] e2e の bindings shim に足し忘れると、E2E ビルドだけが型チェックで落ちる
  → shim の `index.ts` が生成物との型の一致を検査するので、足し忘れは `npm run type-check` で見つかる
- [Risk] `shoppingFallbackDays` を `ShoppingRequiredDays` の先頭に結びつけると、先頭の値を変えたときに選び直す日数も変わる
  → 選び直す日数は「必ず候補に残る日数」であることだけが必要で、7 である必要は無い。`settings_test.go` で、消せない日数に含まれることを確かめる

## Migration Plan

スキーマとデータは変わらない。既存の DB はそのまま使える。
