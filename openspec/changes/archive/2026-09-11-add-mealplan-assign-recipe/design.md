## Context

- 動機は [proposal.md](proposal.md) 参照。既存 `add-mealplan-week-view` で S01 の表示層と `MealPlanService.ListWeek`
  シグネチャは確定済み。本 proposal は「サービス内部を repo に差し替え + 割当系メソッドを追加」する
- 参照ドキュメント: [docs/product.md #4/S01](../../../docs/product.md)、
  [docs/wireframes/S01_meal_calendar.drawio.svg](../../../docs/wireframes/S01_meal_calendar.drawio.svg)、
  [docs/scenarios/mealplan.feature](../../../docs/scenarios/mealplan.feature) の「献立へのレシピ割当」「作り置きコピー」節、
  [docs/design.md §4](../../../docs/design.md) の MealPlan
- 既存の関連ファイル:
  - `internal/domain/mealplan/entry.go` — `Entry { Date, Slot, Position, RecipeName }` (本 proposal で `RecipeID` を追加)
  - `internal/service/mealplan.go` — 固定 dataset を返す `ListWeek`
  - `internal/storage/sqlite/recipe_repo.go` — 参考 (トランザクション、UUID 生成、時刻精度の扱い)
  - `frontend/src/components/mealplan/` — セル / カレンダー / slot 表示切替
  - `internal/config/defaults.go` — `LeftoverCopyPresets = [2, 3, 4]`、`MealPlanCellVisibleItems = 2` (既存)

## Goals / Non-Goals

**Goals:**

- ユーザーが 1 週間の献立を組み終えて再起動しても状態が残る、最小の割当機能を通す
- サービス API を「1 操作 = 1 メソッド」で切り、フロントは楽観更新ではなく成功時 invalidate で扱う (競合が起きない
  シングルユーザー前提)
- 作り置きコピーは 1 品 / slot 全体 / 衝突解決の 3 経路を Go 側に閉じ込め、フロントはプレビュー計算と mode 選択のみ
- 並び替えは UX 最小の「上下ボタン」で通す。DnD 導入は後回し

**Non-Goals:**

- 期間切替 (1 週 / 2 週 / 4 週) / 前後ナビ / 買い物リスト CTA / 候補提示 (proposal.md の Non-goals 参照)
- Undo / Redo、ドラッグ&ドロップ並び替え、複数選択操作
- 食材マスタ・在庫連動 (`add-mealplan-cook-consume` 以降で扱う)

## Decisions

### D1. `meal_plans` テーブルのキー設計

**選択肢:**

- (a) `id TEXT PRIMARY KEY` (UUID) + `UNIQUE(date, slot, position)` — Pros: 単一 ID で 1 行を指せる、position の
  ズレを気にせず update できる。 Cons: `(date, slot, position)` の主単位に 2 系統の識別子が並ぶ。position 詰め直しの
  ときに UNIQUE 制約と衝突しやすく、一時的な position 値を経由するトランザクションが必要
- (b) `PRIMARY KEY (date, slot, position)` (複合主キー、id なし) ← 採用 — Pros: 主単位そのものが主キーで concepts と
  一致。読み取り時の ORDER BY が主キーの部分一致で効き、`date` 単独の INDEX を別に張るだけで済む。 Cons: position
  の詰め直し (削除・並び替え) で複数行の UPDATE が要る。SQLite は主キー UPDATE を許容するが 1 トランザクション内で
  順序 (下から詰める / 一時オフセット) に注意が必要

**採用: (b)**。理由: MealPlan は 1 品行に対して安定した外部識別子を持たせる必要が無い (料理履歴などの参照先にはならず、
UI 側も `(date, slot, position)` で扱えば十分)。トランザクション内での position 詰め直しは domain 層のヘルパで
一括処理できる。既存 `recipe_ingredients` テーブルも同じ `(recipe_id, position)` 複合主キー方式を採っており、
プロジェクト全体の一貫性が取れる

### D2. `RecipeID` と `RecipeName` の返し方

**選択肢:**

- (a) `Entry` は `RecipeID` のみ持ち、フロントで別クエリ `RecipeService.Get(id)` を叩いて名前を解決 — Pros: DTO が薄い、
  レシピ名変更の追随が「詳細を開いたときに最新」で自然。 Cons: 週表示の N セルぶんクエリが増える、TanStack Query の
  cache mismatch でチラつく
- (b) `Entry` に `RecipeID` と `RecipeName` の両方を載せ、Go 側で `meal_plans` × `recipes` を JOIN して返す ← 採用
  — Pros: 1 クエリで完結、UI は取得済みデータをそのまま描ける。名前変更は「次に週を開いたとき」に自動反映される。
  Cons: DTO が広くなる、削除済みレシピを掴んだままの meal_plan 行が現れると `RecipeName` が null になる
  — が、外部キー `ON DELETE CASCADE` で `meal_plans` 側も消すので発生しない

**採用: (b)**。理由: 単一クエリの読み出し性と UI 実装の単純さが優先。既存 `recipe_repo.List` も JOIN で
`IngredientCount` を返しており、同じパターンを踏襲できる

### D3. サービス API の粒度

**選択肢:**

- (a) `MealPlanService.Batch(operations []Op) error` の 1 メソッドに集約 — Pros: 将来 Undo/Redo や複数選択操作を足すとき
  に自然。 Cons: 型付けが弱くなり、フロント側で discriminated union を組む必要がある。Wails の Go→TS bindings は
  interface / any の相互運用が弱い
- (b) 操作ごとに 1 メソッド `AssignRecipe` / `ReplaceEntry` / `RemoveEntry` / `ClearSlot` / `ReorderSlot` /
  `CopyLeftover` ← 採用 — Pros: bindings が素直な TS 関数になり、フロントは `useMutation` を 1 対 1 で貼れる。
  Cons: メソッド数がやや多い

**採用: (b)**。理由: 現時点で複数操作を 1 リクエストにまとめる要件はない (作り置きコピーは Go 側で 1 トランザクション
にまとまる)。将来 batch が必要になったら追加すればよく、いま複雑度を先取りしない

### D4. 作り置きコピーの衝突解決の責務分担

**選択肢:**

- (a) フロントが (存在チェック → 衝突ダイアログ → mode 決定 → サーバに `mode=append|overwrite` で送信) の全ステップを
  担う。サーバは指示された mode を実行するだけ — Pros: UI ロジックが明快、サーバは冪等な mode に応じた処理をするだけで
  済む。 Cons: 存在チェックのためにフロントが週データを見て衝突予測をする。並行編集 (別ウィンドウ) には弱いが、
  meal-cycle はシングルウィンドウのローカルアプリなので問題にならない
- (b) サーバが常に append を試み、衝突があれば `ErrConflict` を返す。フロントはエラーを見てダイアログを開き、
  `mode=overwrite` で再送 — Pros: サーバ側で状態を見て判定するので確実。 Cons: 「上書き宣言」を含むエラーレスポンスを
  定義する必要があり、round trip が 2 回になる

**採用: (a)**。理由: シングルユーザー / シングルウィンドウ前提で衝突予測はフロント側で正確にできる。1 リクエストで
完結する方が「キャンセルしたのに 1 回目のリクエストが飛んだ」ようなノイズも避けられる。サーバは `CopyLeftover(date,
slot, days, position?, mode)` を受け取り、`mode=append` は末尾追加、`mode=overwrite` は各コピー先 slot を空にして
から追加する

### D5. 起点日を含めるか (連続 N 日の解釈)

**選択肢:**

- (a) N 日 = 起点日 + 追加で N 日 (合計 N+1 日) — Pros: 「もう N 日ぶん増やしたい」という直感に合う。 Cons: 「連続 3 日
  作り置き」というコピーが月火水木の 4 日を埋めるという混乱が起きる
- (b) N 日 = 起点日を含む合計 N 日 ← 採用 — Pros: 「3 日作り置き = 3 日ぶん食べる」というペルソナの発話と一致する。
  scenarios ([mealplan.feature の該当 scenario](../../../docs/scenarios/mealplan.feature)) の期待値ともこちらが一致する。
  Cons: 起点日には既に品があるので「起点日への重複 append」を防ぐガードが要る

**採用: (b)**。理由: ペルソナと scenarios が (b) 前提で書かれており、そちらに合わせる。起点日への重複回避は
domain 層 (`CopyDates(date, days) []string` が起点日を含む配列を返し、サービスが起点日の position 重複を弾く) で扱う

### D6. Recipe 選択ダイアログの再利用度

**選択肢:**

- (a) S02 レシピ一覧のコンポーネントを流用し、「表示 mode = pick」を prop で切り替える — Pros: コード重複が減る、フィルタ
  仕様が一致する。 Cons: S02 は行クリックでドロワー、ダイアログは行クリックで選択、と挙動が根本的に違う。1 コンポーネントに
  分岐を仕込むと後で読みにくい
- (b) ダイアログ専用の `recipe-picker-dialog.tsx` を新設し、`RecipeService.List` の結果を共有する ← 採用 — Pros: S02
  との責務分離が明快。ダイアログは検索 + 選択の最小 UI に閉じる。 Cons: 一覧の描き方が微妙に重複する (テーブル vs リスト)

**採用: (b)**。理由: MVP 段階でレシピ数は多くて 50 前後の想定で、テーブル形式にする必然性が低い。ダイアログは名前と
タグの 1 列リストで十分。将来 S10 レシピ検索が入るタイミングで共通化を再検討する

### D7. バリデーションの置き場所

**選択肢:**

- (a) repo 層で SQL 制約 (CHECK / トリガ) と Go 側検査を両方持つ — Pros: DB レベルで整合が保たれる。 Cons: SQLite の
  CHECK 制約でスロット名やフォーマットを弾くと migration が肥大化する。エラーが `sqlite3: constraint failed:` の
  形で返り、フロントに型付きコードで伝えづらい
- (b) domain 層に `Validate*` ヘルパを置き、service 層で全メソッドの入口 (repo に触れる前) から呼ぶ ← 採用 — Pros:
  失敗時に `meal_plans` は一切触られない。エラーは `ErrInvalidDate` 等の型付きセンチネルとして bindings 経由でフロントに
  届き、i18n key へマップできる。テストが repo に依存せず、domain 単体で回せる。 Cons: SQL 制約による「最終防衛線」
  が無くなる — が、書き込み経路はサービス層 1 箇所に集約されているので実運用上の穴は無い

**採用: (b)**。理由: 「i18n は Go 側が型付きコード、日本語文字列はフロントの locales で持つ」というプロジェクト
方針 ([docs/design.md §7](../../../docs/design.md)) と整合する。バリデーションのテストがドメイン層で完結し、
`t.TempDir()` の DB を毎回作らずに済む。境界チェックの視覚的な source of truth を 1 ファイル (`validate.go`)
に集中させることで、後続の proposal (fridge / shopping) でも同じ形を使い回せる

### D8. `CopyLeftover` の日数上限をどこで縛るか

**選択肢:**

- (a) service 層で `config.LeftoverCopyPresets` と照合し、含まれなければ `ErrInvalidCopyDays` — Pros: プロジェクト
  規約 (リテラル閾値を call site に書かない) と一致し、S09 設定 UI 導入時に config を差し替えるだけで許容値が広がる。
  Cons: フロント側のプリセット (`config` 由来) と Go 側の許容値が二重に定義され、乖離しないよう `config.Defaults()`
  を単一 source として揃える運用が必要
- (b) domain 層で `days >= 2` のような固定条件を書く — Pros: シンプル。 Cons: `2 / 3 / 4` の可変性が call site に
  埋め込まれ、S09 で外に出すときの二度手間になる

**採用: (a)**。理由: 「1 日は無意味 (起点日のみ)、5 日以上はプリセットとして提示していないので受け付けない」という
制約はユーザー可変値そのもの。config 集約ルールに乗せておく方が後で楽

### D9. 並び替え UI

**選択肢:**

- (a) 上下ボタン (`↑` `↓`) — Pros: 追加ライブラリ不要、Wails webview の互換性を気にしなくて済む。 Cons: 4 品以上を大きく
  動かすときの操作数が増える
- (b) ドラッグ&ドロップ (dnd-kit 等) — Pros: 直感的。 Cons: 依存ライブラリが増える。webview 内のポインタイベントを詰める
  必要がある

**採用: (a)**。理由: MVP スコープを最小に保ち、まず「割当ができる」を通す。DnD が実利用で欲しくなってから別 proposal
で `dnd-kit` を追加する

## Risks / Trade-offs

- [position の詰め直しが 1 品削除ごとに走る] → SQLite の書き込みは高速で、slot 内の品は 4 程度に収まる想定なので実害
  なし。詰め直しは domain 層のヘルパに集約し、副作用を repo メソッドの中で閉じる
- [`RecipeName` を DTO に混ぜたことでレシピ名変更後の反映が「次に週を開いたとき」まで遅れる] → 献立カレンダーは低頻度
  更新の画面なので体感問題は無い。i18n や検索は `RecipeID` を主キーに扱うので不整合は生じない
- [作り置きコピー先の衝突予測をフロントに置いた] → シングルユーザー前提で問題ないが、将来クラウド同期を入れたら (b)
  にリファクタする必要がある。仕様上 `mode` はサーバが解釈するので、判定ロジックだけ移せば済む
- [複合主キー `(date, slot, position)` は Wails bindings 側で複数キーの型が扱いにくい可能性] → フロントは「行を指す
  キー」を必要とせず、`AssignRecipe(date, slot, recipeId)` のような操作単位で API を叩くので影響しない
- [SQLite の外部キー `recipe_id REFERENCES recipes(id) ON DELETE CASCADE`] → recipe 削除で meal_plan 行が消えるのは
  仕様どおり。ただし現状の `recipe_repo` に delete 実装があるかは別で確認する (無ければ本 proposal では触らず、
  後続の `add-recipe-delete` で扱う)

## Migration Plan

- migration `00003_meal_plans.sql` を goose Up で流す。既存 DB (`add-mealplan-week-view` までしか触っていない DB)
  では `meal_plans` テーブルが未作成なので、初回起動時の migrate で追加される
- ロールバック: goose Down で `DROP TABLE IF EXISTS meal_plans`。既存レシピデータには影響しない
- Wails bindings は `wails3 generate bindings` を再走してフロントの `MealPlanService` 型を更新する。`.gitignore` 対象
  のため、開発者は `task dev` / `task build` を回せば自動で反映される
