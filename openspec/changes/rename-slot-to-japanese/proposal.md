## Why

「slot」は engineering 語彙で、ユーザーが目にする UI 文言としては直感的でない。i18n 文言 (aria-label や
ダイアログ題) に混じっているため、「slot 表示切替」「slot メニュー」等の表現が S01 の説明に登場する。
日本語アプリなので日本語で通したい。「時間帯」なら短くて自然。

## What Changes

- UI 文言 (i18n key) 内の「slot」を **「時間帯」** に置き換える
- 対象 i18n key (例):
  - `pages.meal_plan.slot_visibility.aria_label`: 「slot 表示切替」→「時間帯の表示切替」
  - `pages.meal_plan.cell_actions.slot_menu`: 「slot メニュー」→「時間帯メニュー」
  - `pages.meal_plan.slot_menu.clear`: 「slot 全体をクリア」→「時間帯全体をクリア」
  - `pages.meal_plan.slot_menu.copy_days`: 「slot 全体を連続 {{days}} 日コピー」→「時間帯全体を連続
    {{days}} 日コピー」
  - `pages.meal_plan.errors.invalid_slot`: 「slot の指定が不正です」→「時間帯の指定が不正です」
  - (英語側 `en.json` は既に "Slot" / "Slot menu" 等自然なので変更不要)
- docs/product.md 全体の散文中の「slot」を「時間帯」に統一 (§1 可変値表 / §S01 説明文 / §5 拡張案 /
  §7 決定事項を含む)。**バッククォート囲みのコード引用 (例: `` `(date, slot)` ``) は残す** —
  Go 型名 / DB カラム名を指す構文引用は engineering 語彙として保持する (design.md 参照)
- docs/scenarios/mealplan.feature の日本語 Gherkin 内の「slot」を「時間帯」に置換 (シナリオ本文 +
  `機能:` / `シナリオ:` 見出しを含む)
- 上記 feature の見出しを書き換えることに合わせ、`frontend/e2e/mealplan-assign.spec.ts` の
  `describe` / `test.skip` の名前も同時に追随させる (scenarios と E2E の対応を保つ規約のため)

## Non-goals

- **Go の型名 (`Slot`, `SlotDinner` etc)、DB カラム名 (`slot`)、bindings の型名の変更**: engineering 語彙
  として残す (テーブル / API / ソースコード全体に及ぶ変更で影響が大きい割にメリットが薄い)
- **英語文言の見直し**: 英語側 (`en.json`) の "Slot" は英語話者にとって自然なので変えない
- **朝食 / 昼食 / 夕食のラベル自体**: 既に日本語なので影響なし

## Capabilities

### Modified Capabilities

- `mealplan`:
  - **MODIFIED**: 「slot 表示切替」Requirement — 説明文中の「slot」を「時間帯」に統一。ふるまいは
    不変。Scenario 名の「slot」を「時間帯」に書き換える (spec の Scenario 名変更を伴う)

## Impact

- **新設ファイル**: なし
- **更新ファイル**:
  - `frontend/src/locales/ja.json` — 「slot」を含む文字列を「時間帯」に置換 (上記対象 key)
  - `frontend/src/locales/en.json` — 変更なし (念のためレビュー)
  - `docs/product.md` — 散文中の「slot」を「時間帯」に置換 (バッククォート囲みのコード引用は残す)
  - `docs/scenarios/mealplan.feature` — 日本語 Gherkin 本文 + 見出しの「slot」を「時間帯」に置換
  - `frontend/e2e/mealplan-assign.spec.ts` — feature の見出し変更に合わせ、`describe` / `test.skip`
    の名前を「時間帯」に追随
  - `openspec/specs/mealplan/spec.md` (archive 時に sync) — 現行スナップショットは archive で反映
- **生成物 (再生成のみ)**: なし
- **migration**: なし

### 後続 proposal への引き継ぎ

- なし。以降の proposal でもユーザー可視の文言では「時間帯」を使う。engineering 語彙 (Go / TS / DB) は
  「slot」のまま残す
