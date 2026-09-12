## Context

Step 1 (`rename-slot-to-japanese`, archived 2026-09-12) は日本語 UI 側 (i18n value / product.md /
scenarios / E2E test 名) の「slot → 時間帯」だけを対象にし、Go / DB / bindings / i18n key /
en.json value は engineering 語彙として「slot」を残した。

本 change (Step 2) は engineering 語彙側を **meal** に統一する。日本語 UI での「時間帯」は維持し、
コード / スキーマ / 英語 UI では meal に揃える。3 層 (JP UI / EN UI / engineering) の呼び方を決めた:

| 層 | 呼び方 |
|----|--------|
| 日本語 UI (ja.json value / product.md 散文 / scenarios) | 時間帯 |
| 英語 UI (en.json value) | Meal |
| Engineering (Go / DB / bindings / i18n key / コード引用) | meal |

Step 1 の archive では delta の sync を意図的にスキップした (delta が 1 Requirement しか触っておらず
main spec が中途半端になるため)。Step 2 の archive で `openspec/specs/mealplan/spec.md` をこの
change の delta + 実装フェーズの直接編集で最終形にする。

## Goals / Non-Goals

**Goals:**

- Engineering 語彙 (Go / SQLite / Wails bindings / i18n key / en.json value) を **meal** で統一
- 現行 `openspec/specs/mealplan/spec.md` の残 "slot" を全て「時間帯」or 「meal」に整理
- ソースツリー全体で `grep -n slot` (大文字小文字区別なし) が 0 件になる状態を最終ゲートにする
  (ワイヤーフレーム SVG 内 label 除く)

**Non-Goals:**

- 日本語 UI 側の再翻訳 (時間帯 のまま。Step 1 で確定)
- 「作り置き / 別調理」等、他の語彙の見直し
- Requirement のふるまい変更 (テキストの語彙置換のみ)
- 下位互換 (未リリース段階のためカラム rename は破壊的でよい)

## Decisions

### 決定 1: i18n key 名を `meal*` にする (`mealtime*` ではなく)

**選択肢:**

- (a) `pages.meal_plan.meal*` ← 採用 — Pros: engineering 語彙の meal に揃う / 短い /
  key の階層で `meal_plan.meal` は「献立計画→食事」と読めば違和感が薄い
  Cons: 親 key と子 key に meal が 2 回並ぶため、読み手によっては重複感を覚える
- (b) `pages.meal_plan.mealtime*` — Pros: 親子で単語が重ならない / Cons: `mealtime` は
  「食事の時間」ニュアンスで、Slot 型 (breakfast/lunch/dinner) を表すのにやや遠回り
- (c) `pages.mealplan.meal*` — Pros: 親を短くして重複感を消す / Cons: 既存の親 key
  `meal_plan` を変更すると本 proposal のスコープが広がり、影響ファイルが増える

**採用: (a)**。理由: engineering 語彙 (meal) との一致を優先。key の重複は path として明確に
区切られる (`.` で分かれる) ため実用上の視認性は問題にならない。

### 決定 2: SQLite マイグレーションは `ALTER TABLE ... RENAME COLUMN` 1 本

**選択肢:**

- (a) `ALTER TABLE meal_plans RENAME COLUMN slot TO meal` 1 発 ← 採用
  Pros: SQL 1 発で済む / 既存データ (WSL / Windows の開発 DB) が自動で追随する
  Cons: SQLite の `RENAME COLUMN` は 3.25+ 必須 (modernc.org/sqlite は 3.46+ を搭載しており問題なし)。
  `slot` は PRIMARY KEY `(date, slot, position)` の一部だが RENAME COLUMN が PK 定義側も自動追随する
- (b) 「新カラム追加 → データ移送 → 旧カラム drop」の 2 段
  Pros: 3.25 未満対応 / Cons: 未リリース段階で下位互換不要、work が増える
- (c) DB 破棄して migrate up をやり直す (開発者ローカルに手作業を強制)
  Pros: マイグレーション不要 / Cons: 「開発者が DB を手で消す」運用は嫌われる

**採用: (a)**。理由: modernc.org/sqlite が RENAME COLUMN を扱えるため単純 rename で済み、
既存の開発データが失われない。goose マイグレーション 1 本 (up のみ) で完結する。

### 決定 3: `openspec/specs/mealplan/spec.md` の残 "slot" は delta + 実装で最終形にする

**選択肢:**

- (a) 全 15 Requirement を MODIFIED で delta に列挙して sync 時に完全書き換え
  Pros: sync が権威 / Cons: 現行 spec の ~400 行を delta に複製するので、レビューが大変
- (b) Requirement 名が変わる 4 件だけ MODIFIED で delta に置き、他の 11 件の scenario 内
  「slot」プロース置換は実装フェーズで **main spec.md を直接編集** ← 採用
  Pros: delta が読みやすい / 実装差分でも spec.md の履歴が git で残る
  Cons: openspec の想定フロー (delta → archive sync) から少し外れる。archive 時 sync は
  Requirement 名変更分だけの反映になる
- (c) `skip_specs: true` にして delta なし
  Pros: 一貫している (真の refactor だから) / Cons: Step 1 のパターンと不揃い、
  Requirement 名変更を planning artifact に残せない

**採用: (b)**。理由: delta を最小に保つとレビューが楽で、かつ Requirement 名変更 (Step 1 と一部重複) を
planning artifact として残せる。実装フェーズで main spec を直接触るのは Step 1 と同じ運用。archive
時 sync は skip (Step 1 と揃える)。

### 決定 4: Go 型の rename 範囲

**選択肢:**

- (a) エクスポート識別子だけ (`Slot`, `SlotBreakfast`, `ErrInvalidSlot`) → エクスポート版と対応する内部
  変数 (メソッド引数の `slot string` 等) を meal に、内部ヘルパは慣性で残す
- (b) 内部ヘルパ・私設変数も含めて全変更 ← 採用
  Pros: 「slot」を grep で 0 件にする最終ゲートを達成できる / 一貫している
  Cons: 変更ファイルが増える (`week-calendar.tsx` の内部変数 `slots`, `slot` 等)

**採用: (b)**。理由: proposal の Non-goals に「未リリース段階なので下位互換不要」と書いた通り、
一貫性を最優先する。「grep で 0 件」を Definition of Done にすると回帰が防げる。

### 決定 5: e2e HTTP shim (`cmd/e2e-server`) は引数名変更で壊れない

Wails IPC は位置引数で reflect.Call されるため、Go サービスのメソッド引数名を変えても TS からの
呼び出しは影響しない。他案 (引数を struct 化する等) は不要。

## Risks / Trade-offs

- [Risk: 開発者の DB が古いスキーマのまま] → Mitigation: goose migration は起動時に自動 up。
  失敗時は `~/.local/share/meal-cycle/meal-cycle.db` を削除して再作成できることを README で言及
- [Risk: `frontend/bindings/` の再生成漏れ] → Mitigation: 実装フェーズ最終タスクで
  `wails3 generate bindings` を明示。CI (今後) で bindings 生成物と Go シグネチャの整合を
  検査するフックを持てば理想 (別 proposal 案件)
- [Risk: E2E テスト側で古い bindings-shim が残る] → Mitigation: `frontend/e2e/bindings-shim/`
  も同時更新するタスクを tasks.md に明示
- [Risk: docs / spec の残 "slot" 見逃し] → Mitigation: 最終タスクとしてリポジトリ全体で
  `grep -rn slot` (大文字小文字区別なし) を実行して 0 件を確認 (ワイヤーフレーム SVG 除く)
- [Risk: i18n key rename でランタイム 404 (t() 呼び出し先ミスマッチ)] → Mitigation: TypeScript 側は
  すべての `t("pages.meal_plan.slot*")` を機械的に検索して置換、type-check + build で拾う。
  実行時ミスは E2E で確認

## Migration Plan

1. マイグレーション `00005_rename_slot_to_meal.sql` を追加 (up のみ):
   ```sql
   ALTER TABLE meal_plans RENAME COLUMN slot TO meal;
   ```
   `meal_plans` の `slot` は独立インデックスではなく PRIMARY KEY `(date, slot, position)` の一部だが、
   SQLite 3.25+ の `RENAME COLUMN` は PK 定義側も自動追随するため、追加 DDL は不要 (modernc.org/sqlite
   は 3.46+ を搭載)
2. Go 側の `Slot` → `Meal` rename、bindings 再生成、フロント TS 追随
3. i18n key rename + en.json value 更新
4. `openspec/specs/mealplan/spec.md` の残 "slot" (Requirement 名以外) を直接編集
5. `docs/product.md` / `docs/design.md` / `docs/scenarios/mealplan.feature` の残 "slot" を編集
6. `task lint` / `go test ./internal/... ./` / frontend type-check + build:dev / `task test:e2e`
7. リポジトリ全体で `grep -rin slot` (ワイヤーフレーム SVG 除く) の残数が 0 になるまで反復
