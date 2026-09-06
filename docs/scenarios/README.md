# シナリオ

meal-cycle の受け入れ観点を Gherkin 形式で残す場所。capability 単位でファイルを分ける。

- 参照元: [../product.md](../product.md) #6
- 記法: 日本語 Gherkin（`# language: ja` を各ファイル冒頭に置く）
- 意図: **受け入れ観点の骨子**を capability 横断で見渡せる形にする。実装時は proposal 側 `openspec/changes/<name>/specs/<capability>/spec.md` に、事前条件・データ例つきで転記する

## ファイル一覧

| ファイル                                            | 対応 capability | MVP スコープ           |
| --------------------------------------------------- | --------------- | ---------------------- |
| [recipe.feature](recipe.feature)                    | recipe          | ✓                      |
| [fridge.feature](fridge.feature)                    | fridge          | ✓                      |
| [mealplan.feature](mealplan.feature)                | mealplan        | ✓（候補提示は MVP 後） |
| [shopping.feature](shopping.feature)                | shopping        | ✓                      |

## ファイル命名

`<capability>.feature`（`.feature` は Cucumber 系ツールの標準拡張子）。GitHub でシンタックスハイライトが効く。将来ツール（cucumber-go 等）で実行する余地を残すため `.md` にはしない。

複数の `機能` ブロックを 1 ファイルにまとめて OK。ファイル数を増やすより 1 ファイル内で `機能:` を並べる方が見通しが良い。

## 記法メモ

- `# language: ja` を先頭に置くと Cucumber 系ツールが日本語キーワードを認識する
- キーワード: `機能` / `シナリオ` / `シナリオアウトライン` / `例` / `背景` / `前提` / `もし` / `ならば` / `かつ` / `しかし`
- ワイヤーで表現済みの見た目（列順、色ドット等）はシナリオでは再記述しない。[../wireframes/](../wireframes/) を正とする
- 日付は絶対表記で書く（例: `2026-08-24`）。相対表記だと時間が経つと解釈が変わる

## 今後の運用

- 新規 proposal を切るとき、対応する capability の `.feature` に新しい `シナリオ` を追加する
- proposal を archive するとき、確定した内容を `openspec/specs/<capability>/spec.md` にも合成する
- 「まだ実装していないが受け入れ観点は決まっている」シナリオは、機能ブロックの前に `@planned` タグを付けて明示（ツール実行時にスキップできる）

## 自動テストとしての実行について

`.feature` は実行可能なテストではなく、**受け入れ観点を capability 横断で見渡すためのドキュメント**として書いている。
実行するのは Playwright ベースの E2E (`frontend/e2e/`) で、対応は**テスト名の一致**で取る。

- `describe` 名を `機能:` 行に、`test` 名を `シナリオ:` 行に一字一句揃える
- 未実装のシナリオは `test.skip` で名前だけ置く
- 対応の確認手順は [../../frontend/e2e/README.md](../../frontend/e2e/README.md) にある

現在 `recipe.feature` の 17 シナリオはすべてテスト名として存在する (うち 7 件は未実装で skip)。
`mealplan.feature` / `fridge.feature` / `shopping.feature` は画面が未実装なので、対応する capability の
proposal で E2E を足していく。

### godog を入れない理由

当初は [cucumber/godog](https://github.com/cucumber/godog) で `.feature` を直接実行する案を検討したが見送った。

サービス層は `internal/service/*_test.go` が `t.Run("should … when …")` の BDD 風で書かれており、
godog を足しても増えるのは Gherkin の見た目だけでカバー範囲は変わらない。
さらに「検索欄に入力する」型のシナリオを Go のステップに写すと、実際にはフロントにある検索とソートを
Go 側で再実装した嘘の検証になる。

判断の詳細は [../../openspec/changes/archive/2026-09-07-add-e2e-harness/proposal.md](../../openspec/changes/archive/2026-09-07-add-e2e-harness/proposal.md) の Non-goals にある。
