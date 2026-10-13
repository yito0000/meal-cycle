## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は `specs/recipe/spec.md` の delta を参照。

現状の実装で、この change に効く点は次のとおり。

- レシピの入力制約は `internal/service/recipe.go` の `validateRecipeInput` にあり、`Create` と `Update` が保存の前に走らせる。
  違反は型付きのエラー (`ErrRecipeNameTooLong` など) で返り、フロントが i18n key に対応付ける
- `RecipeRepo.Create` は 1 件ごとにトランザクションを張る。複数件を 1 トランザクションで登録する口は無い
- 手順とメモは Markdown の単一テキストで保存し、S03 で `RecipeMarkdown` が描画する
- S03 の本文 (`RecipeDetailBody`) は、表示と「編集」「削除」の操作が 1 つのコンポーネントにまとまっている
- E2E は Wails の IPC を HTTP に差し替えて React と Go サービスを本物のまま通すので、OS ネイティブのファイルダイアログは通らない
- `RecipeService` は `config.Config` を持っていない。`ShoppingService` は `cfg: config.Defaults()` を持っている
- `gopkg.in/yaml.v3` と `golang.org/x/text` は、golangci-lint などの tool 依存経由で既に `go.mod` に indirect として載っている

## Goals / Non-Goals

**Goals:**

- ファイルの読み取りから重複の判定までを Go 側で行い、table-driven test で形式と量の表記ゆれを網羅できるようにする
- 取り込む前の確認 (読むだけ) と取り込み (書く) を別の呼び出しにし、確認の段階では DB に何も書かない
- ファイル選択を E2E で通せるようにする

**Non-Goals:**

- 取り込んだレシピを S02 でハイライトすること。既存のハイライトは 1 件の ID しか持たないため、件数の通知で代える
- 取り込んだレシピの並び順を、ファイルの順に揃えること (D7)

## Decisions

### D1. インポート形式

**選択肢:**

- (a) CSV
  — Pros: 表計算ソフトで一覧を眺められる
  / Cons: 複数行の手順と、レシピごとに数の違う材料を 1 行に収められない。ファイルを 2 本に分けるか、セルに区切り文字を埋め込むことになり、
  AI の出力も手での修正も崩れやすい
- (b) JSON (1 ファイルに配列)
  — Pros: 構造をそのまま表せ、読み取りが簡単
  / Cons: 手順の Markdown を 1 本の文字列に `\n` で詰めることになり、人が読んで直しにくい。1 件の誤りで全件が読めなくなる
- (c) 1 レシピ = 1 Markdown ファイル、先頭に YAML frontmatter ← 採用
  — Pros: 手順とメモは Markdown のまま本文に書け、保存形式と一致する。1 件ずつ開いて直せ、誤りがあってもそのファイルだけが落ちる。
  AI に出させる形として定着している
  / Cons: frontmatter と本文の節の 2 か所を読む必要がある。YAML の書き間違い (インデント、コロンを含む値) で読めないことがある

**採用: (c)**。書き間違いには、エラーの理由をファイルごとに出すことと、`docs/recipe-import.md` のプロンプトで値を引用符で囲ませることで備える。

本文の節は、行頭が `## ` のレベル 2 の見出しだけを区切りに使う。見出しの文字列は trim してから「手順」「Steps」「メモ」「Notes」と比べる
(英字は大文字と小文字を区別しない)。レベル 3 以下の見出しは節の中身としてそのまま残す。
どちらの見出しも無いときは、本文全体を手順とする。最初の見出しより前の本文は、どちらかの見出しがあるときは捨てる。
手順とメモの中身は、前後の空行を除いて保存する。

元レシピの URL の置き場所は、ユーザーとの相談でメモに決めた。
frontmatter に `source` 項目を作ってアプリがメモに書き足す案もあるが、書き足すときの見出しの文言 (「出典:」) を Go 側に持つことになり、
Go 側に表示文言を持たない方針 (docs/design.md §7) とぶつかる。プロンプトでメモの節に URL を書かせれば足りる。

### D2. ファイルの受け取り方

**選択肢:**

- (a) Wails のネイティブのファイルダイアログを Go 側で開き、Go 側でファイルを読む
  — Pros: パスが取れ、フォルダの走査も Go で書ける。サイズの大きいファイルもフロントを通らない
  / Cons: E2E の HTTP 経路ではダイアログを開けない。テスト用の差し替え口を別に作ることになる
- (b) フロントの `<input type="file">` (ファイル用は `multiple`、フォルダ用は `webkitdirectory`) で選び、
  名前と中身の組を Go に渡す ← 採用
  — Pros: Playwright の `setInputFiles` でファイルとフォルダのどちらも通せる。Go 側はファイルシステムに触れず、純粋な関数でテストできる
  / Cons: `webkitdirectory` は標準外の属性で、WebView の実装に依存する。中身を IPC で 1 回に送るので、件数と大きさに上限が要る

**採用: (b)**。配布先の WebView2 (Windows) は Chromium 系で `webkitdirectory` に対応する (Windows 実機で確認済み)。
開発時の WSL で動く WebKitGTK では効かないが、開発環境に限った制限として受け入れる (Risks を参照)。
上限は 100 件、1 ファイル 64 KB とし (spec の Requirement「S02 からのレシピのインポート」)、合計で 6.4 MB 程度に収める。
手順 4000 字とメモ 2000 字は UTF-8 で 18 KB 程度なので、64 KB は正しいレシピのファイルを落とさない。
上限はユーザーが変える値ではなく実装上の上限なので、`recipeNameMaxRunes` と同じく `internal/service/recipe.go` の定数に置く。

`.md` 以外を除く処理と件数の判定はフロントで行い、Go 側でも件数を再確認する (`ErrRecipeImportTooManyFiles`)。
ファイル名はフォルダ選択時の相対パス (`webkitRelativePath`) を使い、無いときは `name` を使う。

### D3. サービスの API

**選択肢:**

- (a) `Import(files)` 1 本で、読み取りから登録までを行う
  — Pros: 呼び出しが 1 回で済む
  / Cons: 確認画面を出せない
- (b) `PreviewImport(files) []ImportCandidate` と `Import(recipes []Recipe) []Recipe` の 2 本 ← 採用
  — Pros: 確認の段階で DB に何も書かない。取り込む側は既存の `Recipe` をそのまま受け取れる
  / Cons: 確認から取り込みまでの間に DB が変わると、重複の判定が古くなる
- (c) `PreviewImport` の結果を Go 側に一時保存し、`Import(ids)` で確定する
  — Pros: フロントからレシピの中身を送り返さずに済む
  / Cons: サービスが状態を持つことになり、ダイアログを閉じたときの破棄も要る

**採用: (b)**。重複は警告であって登録を止める条件ではないので、判定が古くなっても害は無い。
`Import` は受け取ったレシピの全件に `validateRecipeInput` を当ててから登録する。フロントから来た値を検証なしに書かない。

```go
// internal/domain/recipe/importfile.go
type ImportFile struct {
    Name    string `json:"name"`    // 表示用のファイル名 (フォルダ選択なら相対パス)
    Content string `json:"content"`
}

type ImportCandidate struct {
    FileName  string       `json:"fileName"`
    Recipe    Recipe       `json:"recipe"`
    Issue     ImportIssue  `json:"issue"`     // 空なら取り込める
    IssueArg  string       `json:"issueArg"`  // 問題の材料名など、メッセージに差し込む値
    Duplicate bool         `json:"duplicate"`
}
```

`PreviewImport` の手順は次のとおり。

1. 件数が上限を超えていれば `ErrRecipeImportTooManyFiles` を返す
2. ファイルごとに、大きさを確かめ、`recipe.ParseImportFile` で読み、`validateRecipeInput` を当てる。問題があれば `Issue` に入れる
3. 問題の無い候補の名前を、登録済みのレシピ (`repo.List`、削除済みを除く) とほかの候補の名前と比べ、同じなら `Duplicate` を立てる
4. ファイル名の昇順に並べて返す

エラーのある候補は重複の判定に加えない。名前の読めないファイルと比べても意味が無いため。

### D4. 問題の伝え方

Go 側は日本語のメッセージを持たない (docs/design.md §7)。ファイルごとの問題は Go のエラーではなく、`ImportIssue` の文字列コードで返す。

| コード | 意味 | `IssueArg` |
| --- | --- | --- |
| `too_large` | 64 KB を超えている | なし |
| `no_frontmatter` | `---` で始まらない、または閉じる `---` が無い | なし |
| `invalid_yaml` | frontmatter を YAML として読めない | なし |
| `name_empty` | 名前が無い | なし |
| `name_too_long` / `servings_out_of_range` / `steps_too_long` / `notes_too_long` / `tag_too_long` | S04 と同じ制約の違反 | なし |
| `ingredient_name_empty` | 材料の名前が無い | 何番目の材料か |
| `amount_invalid` | 量を読み取れない | 材料名 |

`validateRecipeInput` のエラーから `ImportIssue` への対応は、`errors.Is` で引く対応表をサービス層に置く。
フロントは `pages.recipes.import.issue.<code>` の i18n key でメッセージを出す。

### D5. 量の表記ゆれを直す処理

**選択肢:**

- (a) AI のプロンプトで数値と単位に分けさせ、アプリは数値だけを受け付ける
  — Pros: アプリ側の処理が要らない
  / Cons: AI の出力は揺れる。1 か所でも「1/2」が混ざるとそのファイルが落ち、十数件を取り込むときに手で直す手間が出る
- (b) アプリ側で、よくある書き方を数値と単位に直す ← 採用
  — Pros: AI の出力の揺れと、人が手で書いた揺れの両方を吸収できる
  / Cons: 直す規則を持つことになり、想定外の書き方はエラーになる

**採用: (b)**。プロンプトでも数値と単位に分けるよう指示し、アプリ側はその取りこぼしを受け止める二段にする。
`recipe.ParseAmount(raw any, unit string, measuringUnits, unquantifiedWords []string) (amount float64, unitOut string, ok bool)` を domain に置く。
`raw` は YAML が数値を `int` / `float64`、文字列を `string` に読むのをそのまま受ける。

文字列の処理の順は次のとおり。

1. `golang.org/x/text/width` の `Fold` で全角の英数字と記号を半角に、半角カナを全角にし、NFC で濁点を合成してから trim する。
   「〜」「～」は `~` に揃える。
   `Narrow` は全角カタカナまで半角カナにして「カップ」が量る単位と一致しなくなるので使わない
2. 空なら量 0
3. 量を持たない言葉と一致すれば、量 0、単位は `unit` が空ならその言葉
4. 先頭が量る単位 (`大さじ` など) なら切り出して単位の候補にする
5. 数値部分を、帯分数 (`1と1/2`、`1 1/2`)、分数、範囲 (`a~b`、`a-b`)、小数の順に読む。範囲は大きいほうを取る
6. 残りの文字列を単位の候補にする。4 と 6 の候補が両方あればエラー
7. 単位の候補があり、`unit` が空ならそれを単位にする。`unit` と一致すればそのまま。食い違えばエラー
8. 量は小数第 2 位で丸める。負の数はエラー

範囲で大きいほうを取るのは、S07 で買い足りなくなるほうを避けるため。
量を持たない言葉の一覧は、量る単位 (`MeasuringUnits`) と同じくユーザーが S09 で足したくなる語彙なので、`internal/config/defaults.go` に
`UnquantifiedWords` として置く。domain が `config` を import しない方針 (add-measuring-units D1) に従い、一覧は引数で渡す。

### D6. 同名の判定

**選択肢:**

- (a) trim した名前の完全一致 ← 採用
  — Pros: 規則が単純で、S02 の見た目と判定が食い違わない
  / Cons: 「肉じゃが」と「肉ジャガ」は別物になる
- (b) 全角と半角、カタカナとひらがなをそろえてから比べる
  — Pros: 揺れた名前も拾える
  / Cons: そろえる処理は進行中の `add-ingredient-name-matching` が作る。先に別の実装を作ると二重になる

**採用: (a)**。そろえる処理が入ったあとで差し替える (proposal の後続への引き継ぎ)。

### D7. まとめての登録

**選択肢:**

- (a) フロントから `Create` を件数分呼ぶ
  — Pros: Go 側に手を入れない
  / Cons: 途中で失敗すると一部だけ登録され、どれが入ったかをユーザーが確かめることになる
- (b) `RecipeRepo.CreateMany` で 1 トランザクションにまとめる ← 採用
  — Pros: 全件が入るか、1 件も入らないかのどちらかになる。選び直してもう一度取り込めばよい
  / Cons: 既存の `Create` と挿入の処理を共有するよう整える必要がある

**採用: (b)**。`CreateMany` は `insertRecipeRow` と `insertRecipeIngredients` を件数分呼ぶ。
`created_at` と `updated_at` は全件で同じ時刻になるので、S02 の既定の並び (更新日 降順) では取り込んだレシピどうしの順は決まらない。
並びに意味を持たせる要求は無いので、このままにする。

### D8. ダイアログの構成

S02a は shadcn の `Dialog` を使い、`PreviewImport` の結果の件数で中身を切り替える。

- 1 件：`RecipeImportPreview`。S03 の材料、手順、メモの欄と同じ表示にするため、`recipe-detail.tsx` からこの 3 つの欄を
  `RecipeContentSections` として切り出し、S03 とプレビューの両方で使う。「編集」「削除」の操作は切り出さない
- 2 件以上：`RecipeImportList`。行のチェックの状態はダイアログの中の `Set<fileName>` で持つ。初期値は問題も重複も無い候補

選択肢のメニュー (「ファイルを選ぶ」「フォルダを選ぶ」) は既存の `dropdown-menu` を使い、各項目が隠した `input` の `click()` を呼ぶ。
同じファイルを選び直しても `change` が発火するよう、読み取りのあとで `input.value` を空にする。

React の型定義は `webkitdirectory` を持たないので、`{...{ webkitdirectory: "" }}` のように属性を広げて渡す。

### D9. 人数の既定値

S04 の新規登録は人数が空で始まるが、ファイルでは省略されることが多い (料理サイトの「作りやすい分量」など)。
省略したファイルをエラーにすると AI の出力を毎回直すことになるので、既定値で埋める。
既定値はユーザーの世帯の人数に合わせて変えたい値なので、`internal/config/defaults.go` に `RecipeImportDefaultServings` (既定 2) として置く。

## Risks / Trade-offs

- [Risk] WebView によっては `webkitdirectory` が効かず、フォルダを選べない。
  WSL (WebKitGTK) では実際に効かず、「フォルダを選ぶ」で通常のファイル選択ダイアログが開いてフォルダを選べないことを確認した。
  Windows (WebView2) では選べることを実機で確認した
  → Mitigation：「ファイルを選ぶ」は標準の `multiple` だけで動き、フォルダ内のファイルを全選択すれば同じ結果になる。
  Linux は配布先ではなく開発環境なので、Go 側でネイティブのダイアログを開く (D2 の (a)) ほどの手間はかけない
- [Risk] AI の YAML がコロンや `#` を含む値を引用符で囲まず、読めないファイルが出る
  → Mitigation：`docs/recipe-import.md` のプロンプトで、文字列の値を必ず引用符で囲むよう指示する。読めないファイルは `invalid_yaml` で名指しする
- [Risk] 100 件 × 64 KB を 1 回の IPC で送ると、WebView と Go の間の受け渡しが重くなる
  → Mitigation：正しいレシピのファイルは 1 件数 KB で、実際の合計は数百 KB に収まる。上限は異常なフォルダを選んだときの歯止めとして置く
- [Risk] 確認から取り込みまでの間に同じ名前のレシピが増え、警告なしに重複する
  → Mitigation：単一ユーザーのデスクトップアプリで、ダイアログを開いている間にほかの画面から登録する経路は無い。害も同名が 2 件並ぶだけで済む
- [Risk] 量の読み取りの規則が、想定外の書き方 (「1かけ」「半分」) をエラーにする
  → Mitigation：エラーに材料名を添えるので、元のファイルを直して選び直せる。頻出するものが見つかったら `UnquantifiedWords` や規則に足す

## Migration Plan

スキーマは変えない。既存のデータにも触れない。ロールバックはコードを戻すだけで済む。
