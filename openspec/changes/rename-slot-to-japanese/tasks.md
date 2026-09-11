## 1. i18n 文言差し替え

- [ ] 1.1 `frontend/src/locales/ja.json` の以下の key の値に含まれる「slot」を「時間帯」に置換する:
      `pages.meal_plan.slot_visibility.aria_label` / `cell_actions.slot_menu` / `slot_menu.clear` /
      `slot_menu.copy_days` / `errors.invalid_slot`。完了判定: `npm run type-check` が通り、
      `grep 'slot' frontend/src/locales/ja.json` で残っていないことを確認
- [ ] 1.2 `frontend/src/locales/en.json` を目視レビューし、「Slot」のまま (英語として自然) で
      問題ないことを確認する。完了判定: 変更なしで OK と判断できる

## 2. ドキュメント統一

- [ ] 2.1 `docs/product.md` §S01 の説明文中の「slot」を「時間帯」に置換する。ワイヤー内の label は
      触らない (画像の再生成コストが高いため)。完了判定: `grep 'slot' docs/product.md` で残数が
      技術文脈 (proposal 名など) のみに絞られる
- [ ] 2.2 `docs/scenarios/mealplan.feature` の日本語 Gherkin 本文中の「slot」を「時間帯」に置換する。
      「機能:」「シナリオ:」の見出しも該当箇所は書き換える。完了判定: `grep 'slot' docs/scenarios/mealplan.feature`
      で残数が引数説明などの技術文脈のみに絞られる

## 3. lint / テスト再ゲート

- [ ] 3.1 `task lint` を実行し、Go / Frontend の両方でエラーが無いことを確認する。完了判定: exit code 0
- [ ] 3.2 `go test ./internal/... ./` を実行し、既存テストが全部 pass することを確認する。完了判定:
      exit code 0
- [ ] 3.3 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev` を実行し、
      フロントの型チェックとビルドが通ることを確認する。完了判定: exit code 0
- [ ] 3.4 `task test:e2e` を実行し、既存 E2E テストが引き続き pass することを確認する
      (`ja.slot_visibility.aria_label` を参照するテストが影響を受ける可能性あり)。完了判定: exit code 0
