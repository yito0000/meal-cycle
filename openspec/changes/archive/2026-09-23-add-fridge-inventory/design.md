## Context

S05 は現状プレースホルダ (`frontend/src/routes/fridge.tsx`) だけで、Go 側に在庫のテーブルもサービスも無い。動機は [proposal.md](proposal.md) の Why を参照。

設計を縛っている既存の前提は次のとおり。

- レシピの材料は食材マスタを持たない自由テキスト (`recipe_ingredients.name`)。
  add-recipe-edit で「名寄せが必要になった段階で `add-ingredient-master` を立てる」と決めている
- 期限アラートの閾値は `internal/config/defaults.go` に `ExpiryAlertRedDays = 3` / `ExpiryAlertYellowDays = 7` として既にある。
  call site にリテラルで書かない規約 (CLAUDE.md「可変値は 1 箇所に集約」)
- 日本語 IME を伴う確認は Windows ビルドでしか行えない (CLAUDE.md「動作確認の 2 階建て構成」)。
  WSL の `task dev` では IME 入力を試せないので、変換確定の Enter の扱いは Windows 側で確認する
- フロントにテストランナーが無い。フロントの純関数 (並び替え) の検証は E2E に寄せることになる
- トースト基盤がまだ無い。削除の「元に戻す」(product.md §7 決定事項) で初めて必要になる

## Goals / Non-Goals

**Goals:**

- 在庫の CRUD と「元に戻す」を、既存の 4 レイヤ (Domain / Repository / Service / Frontend) の規約どおりに通す
- 期限状態の判定ロジックを Go 側 1 箇所に置き、後続の `add-mealplan-suggest-by-stock` から再利用できる形にする
- 食材名の入力摩擦を、マスタ無しで下げる (候補補完 + 直近単位の自動入力)
- 追加行と編集行で、入力値の型 / 検証 / キー操作 (Enter / Esc / IME) を共通化する

**Non-Goals:**

- 食材名の正規化 (全角半角、ひらがなカタカナの同一視)。候補の部分一致は表示文字列のまま比較する
- 期限が変わる日付 (深夜 0 時) をまたいだときのリアルタイム再計算。再取得のタイミングで追従すれば足りる (Risks 参照)
- ページング / 仮想スクロール。家庭の冷蔵庫在庫は数十行規模を想定する

## Decisions

### D1. レイヤ配置と命名

**選択肢:**

- (a) capability 名 `fridge` にすべて揃える (`internal/domain/fridge/`, `fridge_repo.go`, `FridgeService`)
  — Pros: grep が 1 語で済む / Cons: repo が扱う実体は在庫行であり、`FridgeRepo.List` だと何の一覧か読み取りにくい
- (b) ドメイン / サービスは `fridge`、エンティティと repo は `StockItem` / `StockRepo` (`stock_repo.go`) ← 採用
  — Pros: design.md §4 のエンティティ名 `StockItem` と一致し、後続の買い物リストから `StockRepo` を読むときも意味が通る
  / Cons: 名前が 2 系統になる
- (c) `internal/domain/stock/` を別パッケージにする — Pros: CLAUDE.md の図に `stock/` がある
  / Cons: 現時点で `fridge` と `stock` を分ける責務の境目が無く、空に近いパッケージが 1 つ増える

**採用: (b)**。capability (画面と機能) は `fridge`、永続化される実体は `StockItem` と呼び分ける。
`internal/domain/fridge/` に `StockItem` 型と期限判定を置き、サービスは `internal/service/fridge.go` の `FridgeService`、
repo は `internal/storage/sqlite/stock_repo.go` の `StockRepo` とする。`FridgeService` は `RecipeService` と同じく
`NewFridgeService(db *sql.DB)` で repo を内部生成する (テストで repo を差し替える需要が今は無いため、MealPlan 型の interface 注入はしない)。

### D2. 食材名の持ち方

**選択肢:**

- (a) 食材マスタ (`ingredients` テーブル) を新設し、在庫は FK 参照 — Pros: design.md §4 の初期案どおり
  / Cons: レシピ材料は自由テキストのままなので名寄せが片側にしか効かない。マスタ管理 UI (改名 / 統合) も要る
- (b) 在庫も `name TEXT` を直持ちし、候補補完で入力を揃える ← 採用
  — Pros: レシピと同じモデルで、将来のマスタ移行を両テーブル同時の in-place migration にできる / Cons: 表記ゆれは防げない

**採用: (b)** (ユーザー判断)。design.md §4 の StockItem を `ingredientId` から `name` に書き換える。

### D3. テーブルとドメイン型

`00006_stock_items.sql` で次のテーブルを作る。

```sql
CREATE TABLE stock_items (
    id         TEXT PRIMARY KEY,          -- UUID v7 (recipes と同じ)
    name       TEXT NOT NULL CHECK (length(name) > 0),
    amount     REAL NOT NULL CHECK (amount > 0),
    unit       TEXT NOT NULL DEFAULT '',
    expires_at TEXT,                      -- 'YYYY-MM-DD' / NULL = 期限なし
    added_at   TEXT NOT NULL,             -- RFC3339 UTC
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_stock_items_name ON stock_items (name);
```

`name` の index は候補集計 (D5) の `GROUP BY name` 用。期限順の並び替えはフロントで行うので `expires_at` には張らない。

ドメイン型 `fridge.StockItem` は次のフィールドを持つ。

- 入力 / 永続化：`ID` / `Name` / `Amount float64` / `Unit` / `ExpiresAt string` / `AddedAt` / `UpdatedAt`
- 読み出し時の算出値：`ExpiryStatus` (`none` / `fresh` / `soon` / `urgent` / `expired`) / `DaysLeft *int`

「期限なし」の表現は 2 案あった。

- (a) `ExpiresAt *string` — Pros: NULL と 1:1 / Cons: Wails バインディングで `string | null` になり、フォームの空文字との変換が両側に散る
- (b) `ExpiresAt string`、空文字 = 期限なし ← 採用 — Pros: `<input type="date">` の値と同じ表現で、`Recipe` の文字列日時と揃う
  / Cons: DB 境界で `NULL ⇔ ""` 変換が要る

**採用: (b)**。変換は repo の scan / insert の 1 箇所に閉じる。`DaysLeft` は「期限なし」を 0 と区別する必要があるため `*int` のままにする。
`ExpiryStatus` / `DaysLeft` は `Recipe.IngredientCount` と同じく読み出し専用で、入力側の値は無視する。

### D4. 期限状態の判定位置

**選択肢:**

- (a) フロントで算出する (閾値は `mealplan-config.ts` と同様にフロント側 mirror)
  — Pros: 端末の日付変化に即追従 / Cons: 閾値が Go とフロントで二重管理になり、後続の Go 側提案ロジックとも重複する
- (b) Go のドメイン関数で算出し、`List` の戻り値に載せる ← 採用
  — Pros: `config.Defaults()` を直接読めて mirror 不要。table-driven test で境界値を検証できる。提案機能から再利用できる
  / Cons: 日付をまたいでも再取得までは表示が古い

**採用: (b)**。`fridge.Evaluate(expiresAt string, today time.Time, red, yellow int) (ExpiryStatus, *int)` を純関数で置く。
`today` は引数で受け、サービスが `time.Now()` をローカル日付の 0 時に丸めて渡す。サービスは `now func() time.Time` を
フィールドに持ち、テストで固定日付に差し替える。判定規則は spec の「賞味期限の状態表示」に従い、`daysLeft < 0` を
`expired`、`daysLeft <= red` を `urgent`、`daysLeft <= yellow` を `soon`、それ以外を `fresh`、期限なしを `none` とする。
当日 (`daysLeft == 0`) は今日中なら使えるので `urgent` に残し、`expired` には含めない。残り日数はローカル日付同士の差で、
時刻は使わない (UTC で計算すると日本時間の 0〜9 時に 1 日ずれるため)。

### D5. 食材名候補の集計

**選択肢:**

- (a) フロントで `FridgeService.List` と `RecipeService.GetByID` 全件から組み立てる
  — Pros: Go の追加が無い / Cons: レシピ一覧 projection は材料を持たないので N 回の詳細取得が要る
- (b) `StockRepo.ListSuggestions` が `stock_items` と `recipe_ingredients` を 1 クエリで集計する ← 採用
  — Pros: 1 往復で済み、直近単位の決定規則を SQL 1 箇所に置ける / Cons: `stock_repo.go` が `recipe_ingredients` を読む

**採用: (b)**。戻り値は `fridge.Suggestions{ Names []NameSuggestion{Name, Unit}; Units []string }`。
食材名ごとの直近単位は、在庫の `added_at` 最新行を優先し、在庫に無い名前はレシピの `updated_at` 最新の材料行から取る
(ウィンドウ関数 `ROW_NUMBER() OVER (PARTITION BY name ORDER BY ...)` で 1 名前 1 行に絞る)。
`Units` は両テーブルの空でない `unit` の distinct。候補の部分一致と 8 件への絞り込みはフロントで行う。
recipe 側テーブルを読むのは参照のみで、`recipe` capability の要件は変えない。`add-ingredient-master` でマスタ検索に置き換える。

### D6. 候補補完つき入力の実装

**選択肢:**

- (a) ネイティブ `<input list>` + `<datalist>` — Pros: 追加コード最小
  / Cons: WebKitGTK と WebView2 で表示と挙動が揃わない。候補選択時に単位を埋めるイベントが取れない
- (b) shadcn Combobox (cmdk + Radix Popover) — Pros: shadcn 運用に乗る
  / Cons: 依存が 2 つ増える。cmdk は「リストから選ぶ」前提で、自由入力を主にする用途には合わせ込みが要る
- (c) 自前の `stock-name-input.tsx` (WAI-ARIA combobox パターン: `role="combobox"` + `role="listbox"`) ← 採用
  — Pros: 自由入力を主にでき、IME 合成中のキー処理を細かく制御できる / Cons: キーボード操作とフォーカス管理を自前で書く

**採用: (c)**。IME 対策として、`keydown` で `e.nativeEvent.isComposing || e.keyCode === 229` のときは Enter / 上下キーを無視する
(WebKit 系は `compositionend` の直後に `isComposing=false` の Enter が来ることがあり、`keyCode 229` 判定を併用する)。
この判定は追加行 / 編集行の Enter 送信でも使うので `stock-form.ts` に `isImeComposing(e)` として置く。

### D7. 並び替えと検索の実行場所

他案として Go 側で `ORDER BY` / `WHERE name LIKE` する方法も検討したが、S02 レシピ一覧 (`recipe-sort.ts`) と同じく、
全件取得してフロントで並べ替え / 絞り込みを行う。在庫は数十行規模で、切り替えのたびに往復する理由が無い。
`fridge-sort.ts` に `FridgeSortKey = "expiry-asc" | "name-asc" | "added-desc"` と比較関数を置く。
食材名の比較は `localeCompare(b, "ja")` とし、期限なしは期限順で常に末尾、同順位は食材名で安定させる。
並び替えと検索の状態はページコンポーネントの `useState` に持ち、永続化しない。

### D8. 削除と「元に戻す」

**選択肢:**

- (a) 論理削除 (`deleted_at` 列) + 復元で NULL に戻す — Pros: 復元が UPDATE 1 本
  / Cons: 全クエリに `deleted_at IS NULL` が要り、後続の買い物リスト集計でも除外漏れのリスクを抱える。掃除処理も要る
- (b) 物理削除 + `Restore(item)` で同じ ID / `AddedAt` のまま再 insert ← 採用
  — Pros: スキーマが単純で、フロントは削除前の行をそのまま渡すだけ / Cons: 復元用に「ID と追加日時を指定する insert」を公開する
- (c) トースト表示中は削除を遅延し、閉じたら実行 — Pros: DB 操作が 1 回
  / Cons: 表示中にアプリを閉じたときの扱いと、キャッシュ上の「削除予定」状態の管理が複雑になる

**採用: (b)**。`Restore` は入力検証に加えて ID (空でない) と `AddedAt` (RFC3339) を検証し、同じ ID が既にあれば
`ErrStockAlreadyExists` を返す (二重押し対策)。`UpdatedAt` は復元時刻にする。
フロントは削除 mutation の `onSuccess` で、削除前の行オブジェクトを閉じ込めたトーストを出す。

### D9. トースト基盤

**選択肢:**

- (a) `sonner` + shadcn の `sonner.tsx` を手動コピー ← 採用 — Pros: shadcn 公式のトースト。action ボタン、自動消去、
  重なり表示が揃っている / Cons: 依存が 1 つ増える
- (b) 自前の単一スロットトースト — Pros: 依存なし / Cons: 買い物リストの一括登録など後続でも使う見込みがあり、作り直しになる

**採用: (a)**。shadcn の `sonner.tsx` は `next-themes` を import するので、その部分を削ってコピーする (本アプリはライトテーマ固定)。
`<Toaster />` は `root-layout.tsx` に 1 つ置く。表示時間は 5 秒 (sonner の既定 4 秒より少し長くし、押し逃しを減らす)。
これはユーザー可変値ではない (product.md §1 の一覧に無い) ので `config` には置かず、`use-fridge.ts` の定数にする。
sonner は外部通信をせず、バンドルに同梱されるので「完全ローカル」の前提は崩れない。

### D10. 行内編集の同時編集

**選択肢:**

- (a) 複数行を同時に編集可能 — Pros: 制約が無い / Cons: 保存 / キャンセルのボタンが複数並び、Enter がどの行に効くか曖昧
- (b) 別の行の「編集」で現在の編集を破棄して切り替える — Pros: 操作が軽い / Cons: 未保存の変更が黙って消える
- (c) 編集中は他行の「編集」「削除」を無効化する ← 採用 — Pros: 変更が消えない / Cons: 切り替えにキャンセルが 1 手要る

**採用: (c)**。`fridge-table.tsx` が `editingId: string | null` を持ち、各行に `disabled` を渡す。追加行は編集中も使える
(買い物直後に既存行を直しながら追加を続ける操作を妨げないため)。

### D11. 入力検証とエラーの受け渡し

検証上限 (食材名 100 字 / 量 99999 / 単位 20 字) は実装上限でありユーザー可変値ではないので、`RecipeService` と同様に
`internal/service/fridge.go` のパッケージ定数に置く。Go 側の sentinel は次のとおりで、メッセージは `fridge: ` で始める。
フロントの `error-messages.ts` は mealplan と同じ部分文字列マッチで i18n key に変換する。

| sentinel                  | message                        | 発生箇所                  |
| ------------------------- | ------------------------------ | ------------------------- |
| `ErrStockNameEmpty`       | `fridge: name empty`           | service                   |
| `ErrStockNameTooLong`     | `fridge: name too long`        | service                   |
| `ErrStockAmountOutOfRange`| `fridge: amount out of range`  | service                   |
| `ErrStockUnitTooLong`     | `fridge: unit too long`        | service                   |
| `ErrStockInvalidExpiry`   | `fridge: invalid expiry date`  | service (`time.Parse("2006-01-02")`) |
| `ErrStockInvalidRestore`  | `fridge: invalid restore`      | service (ID 空 / AddedAt 不正) |
| `ErrStockNotFound`        | `fridge: stock item not found` | repo (Update / Delete)    |
| `ErrStockAlreadyExists`   | `fridge: stock item exists`    | repo (Restore)            |

sentinel は `internal/domain/fridge/errors.go` にまとめる (repo と service の両方が返すため、`recipe` のように
repo 側と service 側に分散させない)。フロント側の検証は同じ規則を `stock-form.ts` の `validateStockForm` に持ち、
保存ボタン押下時に走らせる。量は `<Input type="number" step="any">` の文字列を `Number()` で変換し、`NaN` は範囲外として扱う。

### D12. データ取得と再取得

TanStack Query のキーは `["fridge", "list"]` と `["fridge", "suggestions"]` の 2 つ。Add / Update / Delete / Restore の
mutation は成功時に両方を invalidate する (候補は在庫名の増減で変わるため)。レシピ保存でも候補は変わるが、
`["fridge", "suggestions"]` は S05 を開くたびに再取得されるので (`staleTime: 0`、既定)、レシピ側の mutation から invalidate はしない。

### D13. 賞味期限の入力欄

他案 (年月日のセレクト 3 つ、「+3 日」などのプリセットボタン) も検討したが、ワイヤーは単一の日付欄なので `<Input type="date">` を使う。
WebKitGTK (WSL の `task dev`) ではピッカーが出ず文字入力欄として描画される場合があるが、値の形式 (`YYYY-MM-DD`) は同じなので
検証は変わらない。プリセットボタンは入力頻度を見てから別 proposal で検討する。

### D14. 期限切れの表示

当初は期限切れを 🔴 (`urgent`) に含めていたが、「今日中に使う」と「使えるか捨てるか判断する」が同じ見た目になり区別できなかった。
期限切れを独立した状態 `expired` に分けたうえで、表示方法を次の 3 案から選ぶ。

**選択肢:**

- (a) 色ドットの色だけを変える (例: 濃い赤 / 黒) — Pros: 変更が最小
  / Cons: 色覚によっては 🔴 と区別がつかず、分けた意味が薄れる
- (b) 期限切れだけを一覧上部の別枠にまとめる — Pros: 並び順に関係なく常に目に入る
  / Cons: 既定の「賞味期限が近い順」ではすでに先頭に並ぶので効果が重複する。「食材名」「追加日」順のときに
  期限切れだけ枠に固まるのかという並び替えとの新しい規則が要る
- (c) 色ドットの代わりに「期限切れ」の文字バッジを出し、行の背景を薄い赤にする ← 採用
  — Pros: ドットと形が違うので色に頼らず見分けられる。並び替えの規則を変えずに済む / Cons: 行の見た目が他と揃わない

**採用: (c)** (ユーザー判断)。バッジは `expiry-dot.tsx` が `expired` のときに描画し、行の背景は `fridge-item-row.tsx` が
`expiryStatus` を見て付ける。後続の提案機能 (`add-mealplan-suggest-by-stock`) で期限切れ在庫を候補に含めるかは、その proposal で決める。

## Risks / Trade-offs

- [Risk] アプリを開いたまま日付が変わると、残り日数と色ドットが再取得まで 1 日古い
  → Mitigation: TanStack Query の既定 `refetchOnWindowFocus` で、ウィンドウに戻ったときに再取得される。S05 を開き直しても再取得される
- [Risk] 表記ゆれ (「玉ねぎ」「たまねぎ」「タマネギ」) が別の食材として並び、後続の買い物リストで在庫と一致しない
  → Mitigation: 候補補完で既存表記に寄せる。根本対応は `add-ingredient-master` に引き継ぐ (proposal の Impact に記載)
- [Risk] 自前 combobox のキーボード操作とフォーカス管理に抜けが出る
  → Mitigation: E2E で上下キー / Enter / Esc の操作を 1 シナリオ通す。IME 合成中の挙動は Windows ビルドで手動確認する (tasks に明記)
- [Risk] 「元に戻す」を押す前にトーストが消えると復元できない
  → Mitigation: 表示時間を 5 秒にする。product.md の決定事項 (確認ダイアログなし) の範囲内の割り切りとして受け入れる
- [Risk] `stock_repo.go` が `recipe_ingredients` を読むので、recipe 側のスキーマ変更で候補集計が壊れる
  → Mitigation: `stock_repo_test.go` にレシピ材料由来の候補ケースを入れ、recipe 側の変更時に落ちるようにしておく
- [Risk] shadcn `sonner.tsx` から `next-themes` を削る手修正が、将来 shadcn 側の更新を取り込むときに差分になる
  → Mitigation: 削った箇所にコメントを 1 行残す

## Migration Plan

- `00006_stock_items.sql` は新規テーブルの追加のみで既存データに触れない。起動時の goose 自動適用で入る
- Down は `DROP TABLE stock_items` (在庫データは失われる)
- WSL の DB と Windows の DB は別ファイル (CLAUDE.md) なので、Windows 側の確認は空の在庫から始める
