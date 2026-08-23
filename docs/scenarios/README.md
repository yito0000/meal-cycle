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

現状これらの `.feature` は**「読める設計ドキュメント」**として書いており、CI で自動実行はしていない。実装時の単体・統合テストは Go 標準の `testing` パッケージで書く方針。

将来 **E2E テストが欲しくなったタイミングで [cucumber/godog](https://github.com/cucumber/godog) の導入を検討**する（日本語 Gherkin にも対応）。導入時はステップ定義（`前提「X が Y」→ Go 関数呼び出し` の対応表）を書く作業が発生するため、その時点で実行対象にする `.feature` を選別する。
