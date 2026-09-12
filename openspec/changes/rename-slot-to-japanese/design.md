## Context

meal-cycle は日本語 UI が既定。「slot」は engineering 語彙 (Go の `Slot` 型 / DB カラム名 / Wails
bindings) と、UI 文字列 (i18n 経由) / ドキュメント / Gherkin シナリオに散在している。
UI 側の日本語文言に混入している状態は、ユーザーには engineering 語彙が漏れて見える。

現状:

- `frontend/src/locales/ja.json` の一部 value に「slot」が残っている
- `docs/product.md` §1 表 / §S01 説明文 / §5 拡張案 / §7 決定事項 に「slot」が混在
- `docs/scenarios/mealplan.feature` に日本語 Gherkin 混じりで「slot」が残る
- `openspec/specs/mealplan/spec.md` (現行スナップショット) にも「slot」が残る
- Go / TS のコード / DB スキーマ / bindings は「slot」のまま (Non-goal)

## Goals / Non-Goals

**Goals:**

- ユーザー可視の日本語文言から「slot」を排除する
- 用語を 1 つに揃える (混在させない)
- ドキュメント (product.md / scenarios / spec) と i18n の間で語彙を一致させる

**Non-Goals:**

- Go の型名 (`Slot`, `SlotDinner` 等) / DB カラム名 / bindings 型名の変更 — engineering 語彙として残す
- 英語 UI (`en.json`) の "Slot" の見直し — 英語話者にとって自然
- ワイヤーフレーム (`docs/wireframes/*.drawio.svg`) 内 label — SVG 再生成コストが高い

## Decisions

### 訳語の選定

**選択肢:**

- (a) 「食事枠」 — Pros: slot の直訳寄りで DB 用語との対応が付く / Cons: やや堅い、日常語彙ではない
- (b) 「食事区分」 — Pros: 表・仕様書と親和 / Cons: 説明的で長い、UI に置くと重い
- (c) 「時間帯」 ← 採用 — Pros: 短い / 日本語として自然 / 朝食・昼食・夕食を包含する語感が既にある /
  Cons: 「時間の帯」というやや広い意味を持つため、他文脈で衝突しないか要注意
- (d) 「ミールスロット」等のカタカナ化 — Pros: 直訳 / Cons: 「slot」を残すのと本質的に同じで解決しない

**採用: (c) 時間帯**。理由: 短くて自然、既に S01 の「夕食」等と語感が揃う。i18n 文言・見出し・
Gherkin いずれの文脈でも読み下せる。他文脈で「時間帯」が別意味に使われている箇所は現状 product.md /
scenarios にはないことを grep で確認済み。

### 置換範囲: `docs/product.md` は全体 (データモデル文脈含む)

**選択肢:**

- (a) §S01 のみ (proposal 当初案) — Pros: 最小変更 / Cons: §1 表・§5 拡張案・§7 決定事項の
  「slot 表示」「1 slot あたりの品数」等が取り残され、読み手に混乱を与える
- (b) ユーザー可視の UI 語彙のみ (§1 表・§S01・§5) — Pros: 中庸 / Cons: §1 末尾や §7 の
  「同一 slot に 0..N 行」「(date, slot) の一意制約」等が残り、文脈判定が読み手に依存する
- (c) 全体置換 (データモデル文脈含む) ← 採用 — Pros: product.md 内で語彙を完全に統一できる /
  Cons: 「slot」が指すものが DB 用語なのか UI 語彙なのか、置換後は文脈から判断する必要がある

**採用: (c) 全体置換**。理由: product.md は「利用者が読む」ドキュメント (UX / 画面 / Gherkin) で
あって、DB カラム名を参照する目的の技術文書ではない。データモデル解説は
[docs/design.md #4](../../../docs/design.md) 側にある。ただし **コード / スキーマの構文引用
(バッククォート囲みの `` `(date, slot)` `` 等) は残す** — grep で「コード上の識別子」と
「散文の語彙」を区別できるようにするため。

### `docs/scenarios/mealplan.feature` の見出しと E2E テストの追随

Gherkin の `機能:` / `シナリオ:` 行にも「slot」が混じっている。E2E テスト (`frontend/e2e/`) の
`describe` / `test.skip` 名は scenarios feature の見出しと一字一句一致させる規約 (CLAUDE.md) の
ため、feature を書き換えたら E2E テストコードも同時に追随する必要がある。

**採用**: feature 本文・見出し・E2E テストの `describe` / `test.skip` を同時に書き換える。
tasks.md にサブタスクとして E2E コード側の書き換えを明示する。他案 (E2E は後追い) は
scenarios と E2E の対応が壊れた期間ができるため採らない。

### `openspec/specs/mealplan/spec.md` (現行スナップショット) の反映タイミング

**選択肢:**

- (a) apply 中に手で書き換える — Pros: レビュー時点で現行と一致 / Cons: openspec のフローと二重管理
- (b) archive 時に `openspec-sync-specs` 経由で自動反映 ← 採用 — Pros: 通常フローに乗る / Cons:
  archive 前の中間状態で現行 spec と change delta が乖離する期間ができる

**採用: (b) archive 時 sync**。理由: openspec の通常フロー (change delta → archive 時 sync) に
乗せるのが素直。中間期間の乖離は proposal 進行中はどの change でも発生する既定の挙動。

## Risks / Trade-offs

- [Risk: 「時間帯」が別意味と衝突] → mitigation: product.md / scenarios / i18n を grep して現状で
  他意味の「時間帯」使用がないことを確認済み。今後別文脈で使う場合は「勤務時間帯」等の修飾語で
  区別する
- [Risk: DB カラム名との認知乖離] → mitigation: 開発者は Go の `Slot` / DB カラム `slot` に触れる
  たびに「UI では時間帯」と読み替える必要がある。docs/design.md 側で対応関係を明記する運用でカバー
- [Risk: 過剰置換でコード引用も書き換える] → mitigation: バッククォート囲みやコードブロック内は
  残す運用。tasks.md の完了判定 grep で「バッククォート囲み以外」を確認する
