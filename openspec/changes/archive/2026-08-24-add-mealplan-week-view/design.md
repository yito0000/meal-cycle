## Context

- 参考: [proposal.md](proposal.md) — Why / Non-goals / 後続 proposal への引き継ぎ
- MVP の中核画面 S01 のうち、表示層 + サービス境界 + ドメイン型 を先行して固める。
- 現状 `openspec/specs/` は空で、mealplan capability も初出。後続 `add-mealplan-assign-recipe` が
  「サービス実装を repo 呼び出しに差し替える」形で載せられるよう、**サービス I/F と DTO の型は本 proposal で確定**させる。
- 参照ドキュメント: [docs/product.md #1, #4/S01](../../../docs/product.md) と
  [docs/wireframes/S01_meal_calendar.drawio.svg](../../../docs/wireframes/S01_meal_calendar.drawio.svg)、
  [docs/design.md #4/MealPlan](../../../docs/design.md)。

## Goals / Non-Goals

**Goals:**

- 週カレンダーの UI + サービス境界 + ドメイン型 を確定させ、次 proposal で repo に差し替えるだけで動くようにする
- 固定値 (シード) は Go 側 (サービス) に置き、フロントは常に Wails 経由で取得する
  (次 proposal でサービス内部だけを repo に差し替えられる)
- 1 セル = 0..N 品のレイアウトを CSS Grid で成立させる (品数超過は「他 N 件」で省略)

**Non-Goals:**

- SQLite テーブル / migration / repository の追加。永続化なし
- セル操作 (＋ / ⋯ / 連続 N 日 / slot 全クリア)、期間切替 (1 週 / 2 週 / 4 週)、前後ナビ (＜ / ＞)、
  下部 CTA (買い物リスト / 週プリセット)、レシピ選択ダイアログ (S02 のミニ版) の実装
- フロントエンド testing runner (Vitest 等) の導入 — 別 proposal で扱う

## Decisions

### D1. 固定値の置き場所

**選択肢:**

- (a) フロントエンド (`src/data/mockMealPlan.ts` 等) にモックを持つ — Pros: サービス層を作らなくて済み、
  次 proposal でサービスとフロントを同時に足せる。 Cons: 次 proposal でフロントとサービスの
  両方を触ることになる。TS 側の型と Go 側の型で二重管理になり、Wails バインディング再生成時の差異検出が効かない
- (b) Wails サービス (`MealPlanService.ListWeek`) が Go 側で固定 `[]MealPlanEntry` を返す ← 採用 — Pros:
  型は Go が source of truth、TS 側は自動生成 bindings を通す。次 proposal では**サービスの中身だけ**を
  repo 呼び出しに差し替えるので、フロントは無変更で済む。 Cons: 表示だけのために Wails サービスを 1 個作る overhead

**採用: (b)**。理由: 次 proposal (`add-mealplan-assign-recipe`) がフロントとサービス両方を書き換える
コストの方が高い。境界を先に固定して差し替えを 1 レイヤに閉じ込める方が、本 proposal の目的
(「後続の土台にする」) に合う。ユーザー選択でも (b) が指定されている。

### D2. 週の起点日 (月曜始まり)

**選択肢:**

- (a) 日曜始まり — Pros: JavaScript `Date.getDay()` が 0=日曜。 Cons: 日本の作り置きペルソナは
  「日曜夜に来週分を組む」→ 月曜始まりの方が「来週」を素直に扱える
- (b) 月曜始まり ← 採用 — Pros: [product.md #1](../../../docs/product.md) のナラティブ
  「日曜の夜に来週 1 週間分の献立を組む」に一致。ISO 8601 週の慣習にも合う。 Cons: JS 側で
  月曜正規化コードが必要

**採用: (b)**。理由: プロダクトの想定利用シナリオ (日曜夜に月〜金の夕食を組む) に一致。
週起点計算は Go 側の `internal/domain/mealplan` に置き、フロントはサービスから受け取った日付リストを
そのまま描画する (フロントに日付ロジックを持ち込まない)。

### D3. slot 表示切替 UI と状態管理

**UX 選択肢** (2 択トグル vs 個別チェックボックス):

- (a) 「夕食のみ」/「3 食表示」の 2 択トグル (ワイヤーフレーム初期案) — Pros: UI が最小。ワイヤーと一致。
  Cons: 「朝食 + 夕食だけ ON」のような中間状態が表現できない。3 食すべて要る人はまれ (作り置きペルソナは
  夕食メイン、たまに朝も見たい程度)、トグルの選択肢が実利用に噛み合わない
- (b) 朝食 / 昼食 / 夕食 の 3 つの独立チェックボックス ← 採用 — Pros: デフォルトは夕食のみ ON、
  必要に応じて朝食や昼食を追加できる。UX が実利用パターンに一致。 Cons: ワイヤー ([S01](../../../docs/wireframes/S01_meal_calendar.drawio.svg))
  との差分が発生する (ワイヤーは 2 択トグルで描かれている)

**採用: (b)**。理由: 作り置きペルソナは基本夕食のみ、たまに朝食も可視化したいというユースケースが自然
で、2 択より 3 チェックの方がその中間状態を素直に扱える。ワイヤーとの差分は「初期案からの UX 改善」
として扱い、ワイヤー本体は更新せず、以後は spec 側 (この spec.md) を実装の真実とする。

**制約**: 少なくとも 1 slot が常に ON (すべて OFF はカレンダーが 0 行になるため禁止)。UI 側で最後の
チェックボックスは disabled にする。

**状態管理選択肢:**

- (a) URL / ルート params に載せる — Pros: 深リンク・再現性が高い。 Cons: Wails は memory history
  固定で URL 共有ができないため利点が消える
- (b) React コンポーネント state (`useState<Set<Slot>>`) のみ ← 採用 — Pros: シンプル。切替は 1 セッション
  内でしか使われない (再起動でデフォルトに戻ってよい)。 Cons: 再起動で状態が消える

**採用: (b)**。理由: ローカル単一ユーザーで URL 共有の概念が無い。永続化は S09 で再検討。
初期値は `internal/config/defaults.go` の `DefaultVisibleSlots []string` (`["dinner"]`) から読む。

### D4. サービスから返す DTO の形

**選択肢:**

- (a) 週構造 (`WeekView { days: [{date, slots: [{slot, entries: [...]}]}] }`) を返す — Pros: フロントは
  そのまま描画できる。 Cons: 空 slot を明示エントリで返すことになり (「slot は必ずある、entries が 0」)、
  仕様書の「空 slot はエントリ無し」と揃わない。次 proposal で `entries` の add/remove を扱う際に
  ネスト構造の更新パスが増える
- (b) 平坦なエントリ配列 `[]MealPlanEntry { date, slot, position, recipeName }` ← 採用 — Pros:
  MealPlan は「1 行 = 1 品」というデータモデル ([design.md #4](../../../docs/design.md)) と一致。
  空 slot は「エントリが無い」で表現でき、追加・削除もエントリ単位で表現しやすい。 Cons: フロントで
  (date, slot) キーで group by する処理が必要

**採用: (b)**。理由: 永続化層のスキーマ (`meal_plans` は 1 行 1 品) と DTO の形を揃えることで、
次 proposal で repo からの結果をそのまま返せる。フロント側の group by は 7 日 × 3 slot = 21 セル
分の一回だけで軽い。

**DTO 仕様 (Go)**:

```go
package mealplan

type Slot string

const (
    SlotBreakfast Slot = "breakfast"
    SlotLunch     Slot = "lunch"
    SlotDinner    Slot = "dinner"
)

type Entry struct {
    Date       string // "YYYY-MM-DD"
    Slot       Slot
    Position   int    // 0-based, 同一 (date, slot) 内で一意
    RecipeName string // 本 proposal では固定値
}
```

`RecipeID` は本 proposal では持たない (レシピ capability が未着手のため)。次 proposal で
`RecipeID` を追加し、フロントは名前 + ID 両方を受け取る形に拡張する。

### D5. サービスメソッドのシグネチャ

`MealPlanService.ListWeek(ctx context.Context, weekStart string) ([]Entry, error)`

- `weekStart` は "YYYY-MM-DD" 形式の月曜日。フロントから明示的に指定する
- 「今日を含む週の月曜日」の算出は Go の `internal/domain/mealplan.WeekStartOf(today time.Time) time.Time` で行い、
  フロントは起動時に一度呼び出して結果を保持する
- 本 proposal では `weekStart` が何であっても同じ固定エントリセットを返す (前後ナビが無いため実質固定)

他案 (フロントで週起点を計算し `weekStart` を渡さない、あるいはサービス側で `time.Now()` を使う) も
検討したが、次 proposal で複数週対応するときに `weekStart` パラメタが不可欠なので今から入れる。

### D6. 品数超過の閾値

`internal/config/defaults.go` に `MealPlanCellVisibleItems = 2` として置く。
セル内に表示するのは先頭 2 品、超過分は「他 N 件」で省略。
リテラル `2` を UI に直書きしない (S09 で設定可能にする前提)。

## Risks / Trade-offs

- **[Risk] 固定値のまま archive されると次 proposal との差分が「サービス実装ごっそり書き換え」になる**
  → **Mitigation**: サービス I/F・DTO 型・週起点計算ヘルパを本 proposal で確定させ、次 proposal では
  サービスメソッドの本体のみを書き換える形に閉じ込める。tasks.md の最後で「次 proposal の diff は
  service 本体と repository の追加に限定される」ことを確認する
- **[Risk] Wails バインディング再生成を忘れて TS 側の型が古いまま動く** → **Mitigation**: tasks.md に
  `wails3 generate bindings` タスクを明示。開発中は `task dev` が自動再生成するため通常運用では発生しにくい
- **[Risk] 「今日」の判定がタイムゾーンで揺れる** → **Mitigation**: Go 側は `time.Now().Local()` で
  ローカル時刻を使う (完全ローカルアプリ前提)。テストは `WeekStartOf(t time.Time)` に時刻を注入して検証
- **[Risk] slot 切替を state で持つと再起動時に「3 食表示」に戻せない** → **Mitigation**: 本 proposal では
  受容する (毎起動「夕食のみ」開始)。S09 (設定 UI) 追加時に永続化を再検討

## Migration Plan

新規追加のみで既存機能への破壊的変更なし。ロールバックは変更 revert で完結する。
