## 1. i18n 文言差し替え

- [x] 1.1 `frontend/src/locales/ja.json` の value に含まれる「slot」を「時間帯」に置換する
      (対象: `slot_visibility.aria_label` / `cell_actions.add_more` / `cell_actions.slot_menu` /
      `slot_menu.clear` / `slot_menu.copy_days` / `copy.conflict.body` / `batch.merge_dialog.empty` /
      `errors.invalid_slot` / `errors.reorder_mismatch`)。JSON key (例: `slot_menu`) は code-side
      識別子なので残す。完了判定: `grep -n '"[^"]*slot[^"]*"' frontend/src/locales/ja.json` の
      ヒットが JSON key 位置のみになること
- [x] 1.2 `frontend/src/locales/en.json` を目視レビューし「Slot」のまま (英語として自然) で
      問題ないことを確認する。完了判定: 変更なしで OK と判断できる

## 2. ドキュメント統一

- [x] 2.1 `docs/product.md` 散文中の「slot」を「時間帯」に置換する (§1 可変値表 / §S01 説明文 /
      §5 拡張案 / §7 決定事項を含む全体)。バッククォート囲みのコード引用 (例: `` `(date, slot)` ``)
      は残す。ワイヤー内 label は触らない (画像再生成コストのため)。完了判定:
      `grep -n 'slot' docs/product.md` の残数がバッククォート囲みの行 (コード引用) のみになること
- [x] 2.2 `docs/scenarios/mealplan.feature` の日本語 Gherkin 本文および `機能:` / `シナリオ:`
      見出しの「slot」を「時間帯」に置換する。ステップの引数説明などの技術文脈 (存在すれば) は残す。
      完了判定: `grep -n 'slot' docs/scenarios/mealplan.feature` の残数がコード引数説明 or
      バッククォート囲みのみになること
- [x] 2.3 `frontend/e2e/mealplan-assign.spec.ts` の `describe` / `test.skip` の名前を 2.2 で更新した
      feature 見出しと一字一句一致させる。完了判定: feature の `機能:` / `シナリオ:` 行と E2E テスト
      名を目視で突き合わせて一致

## 3. lint / テスト再ゲート

- [x] 3.1 `task lint` を実行し、Go / Frontend の両方でエラーが無いことを確認する。完了判定: exit code 0
- [x] 3.2 `go test ./internal/... ./` を実行し、既存テストが全部 pass することを確認する。完了判定:
      exit code 0
- [x] 3.3 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev` を実行し、
      フロントの型チェックとビルドが通ることを確認する。完了判定: exit code 0
- [x] 3.4 `task test:e2e` を実行し、E2E テストが引き続き pass することを確認する
      (test 名を 2.3 で書き換えたため、runner が新名で拾えているかを合わせて確認)。完了判定: exit code 0
