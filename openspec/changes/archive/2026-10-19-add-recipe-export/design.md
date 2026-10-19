## Context

動機は [proposal.md](proposal.md) の Why を、受け入れ条件は [specs/recipe/spec.md](specs/recipe/spec.md) を参照。

この change に効く現状は次のとおり。

- インポートの形式は `internal/domain/recipe/importfile.go` の `ParseImportFile` が読む。
  本文は行頭が `## ` の行だけを節の区切りにし、「手順」「メモ」以外の見出しの節は捨てる (コードブロックの中は除く)
- インポートはファイルの選択をフロントの `<input type="file">` で行い、Go 側はファイルシステムに触れない (add-recipe-bulk-import D2)
- アプリはまだ Wails のネイティブのダイアログを使っていない。Wails v3 beta.12 には
  `application.Get().Dialog.OpenFile().CanChooseDirectories(true).PromptForSingleSelection()` があり、選んだフォルダの絶対パスを返す
- `RecipeRepo.List` は S02 向けの projection で、材料、手順、メモを返さない。全件を材料ごと読む口は無い
- E2E サーバ (`cmd/e2e-server`) はテストごとに一時ディレクトリを作り、そこに DB を置いて起動する。Wails の runtime は無い
- `gopkg.in/yaml.v3` はインポートで直接使っている
- Go 側は日本語の表示文言を持たない (docs/design.md §7)

## Goals / Non-Goals

**Goals:**

- 書き出したファイルを、手を加えずにインポートで読み戻せること。これを Go のテストで往復させて確かめる
- フォルダの選択から書き込みまでを E2E で通せること
- 書き出し先の既存のファイルを決して壊さないこと

**Non-Goals:**

- 書き出しの形式を利用者が選べるようにすること (JSON、CSV など)
- 書き出しの進み具合を表示すること。数百件でも 1 秒かからないため

## Decisions

### D1. ファイルの書き出し方

**選択肢:**

- (a) フロントで Blob を作り、`<a download>` で保存させる (1 件ずつ、または zip にまとめて)
  — Pros: Go 側がファイルシステムに触れない。インポートと対称になる
  / Cons: Wails の WebView がダウンロードをどう扱うか (保存先を聞くか、黙って Downloads に置くか、何もしないか) が WebView2 と WebKitGTK で違い、
  確かめていない。1 件ずつだと件数分のダウンロードが走り、zip にすると zip のライブラリが要る
- (b) File System Access API の `showDirectoryPicker` でフォルダを選び、フロントから書き込む
  — Pros: フォルダに直接書け、Go 側は変わらない
  / Cons: WebKitGTK に無い。Playwright でダイアログを操作できず、E2E で通せない
- (c) Go 側で Wails のフォルダ選択ダイアログを開き、Go 側で書き込む。ダイアログは interface の裏に置き、E2E では差し替える ← 採用
  — Pros: OS 標準のダイアログで、WebView の種類に左右されない。書き込みの失敗と後始末を Go で確実に扱える。
  E2E では決まったフォルダを返す実装に替えれば、書き込みまで本物を通せる
  / Cons: サービスがファイルシステムに触れる。ダイアログ自体は E2E で通らない

**採用: (c)**。インポートで Go 側をファイルシステムから切り離したのは、E2E でファイルの選択を通すためだった (add-recipe-bulk-import D2)。
書き出しでは、ダイアログを interface に切り出せば同じ目的を果たせ、(a) と (b) の WebView 依存を避けられる。

```go
// internal/service/folderpicker.go
type FolderPicker interface {
    // 閉じられたら "" と nil を返す。
    PickFolder(title string) (string, error)
}

type wailsFolderPicker struct{}

func NewWailsFolderPicker() FolderPicker { return wailsFolderPicker{} }
```

ダイアログのタイトルは i18n の文言なので、フロントから引数で渡す。
E2E サーバは `-export-dir` で受けたパスを返す `fixedFolderPicker` を使う。
`NewRecipeService(db, picker)` に picker を足し、`main.go` と `cmd/e2e-server` とテストを追従させる。
テストでは `t.TempDir()` を返す picker と、`""` を返す picker と、エラーを返す picker を使う。

### D2. サービスの API

```go
// internal/domain/recipe/exportfile.go
type ExportResult struct {
    Cancelled bool   `json:"cancelled"`
    Dir       string `json:"dir"`   // 作った書き出し用のフォルダの絶対パス
    Count     int    `json:"count"`
}

// internal/service/recipe.go
func (s *RecipeService) ExportToFolder(ctx context.Context, dialogTitle string) (recipe.ExportResult, error)
```

手順は次のとおり。

1. `repo.ListDetailed` で削除済みを除く全レシピを、材料、手順、メモごと読む。0 件なら `ErrRecipeExportEmpty` を返す (ボタンが押せないので通常は来ない)
2. `picker.PickFolder(dialogTitle)` を呼ぶ。`""` なら `Cancelled: true` を返す
3. 選んだフォルダの中に書き出し用のフォルダを `os.Mkdir` で作る (D5)
4. レシピごとに `ExportFileName` でファイル名を決め、`FormatExportFile` の結果を `os.WriteFile` で書く
5. 3 か 4 で失敗したら、3 で作ったフォルダを `os.RemoveAll` で消し、`ErrRecipeExportWrite` を包んで返す

ダイアログを先に開いてから DB を読む順にもできるが、0 件で空のフォルダを作らないよう、読むのを先にする。

`ListDetailed` は作成日の昇順で返す。D4 の番号の振り方を、作った順という安定した順にするため。
材料は既存の `loadIngredients` を件数分呼ぶ。ローカルの SQLite で数百件なら N+1 でも問題にならない。

### D3. ファイルの中身

**frontmatter の組み立て:**

- (a) 文字列を連結して YAML を手で書く
  — Pros: 項目の順と引用符の付け方を完全に決められる
  / Cons: `:` や `#`、改行、先頭の `-` を含むレシピ名を正しく引用する処理を自前で持つことになる
- (b) yaml タグ付きの struct を `yaml.v3` の `Marshal` に渡す ← 採用
  — Pros: 引用は yaml.v3 が正しく行う。読む側も yaml.v3 なので往復が崩れない
  / Cons: 引用符の付き方は yaml.v3 次第で、[docs/recipe-import.md](../../../docs/recipe-import.md) の例と見た目が少し違う

**採用: (b)**。項目は `name`、`servings`、`tags`、`ingredients` の順の struct にし、`tags` と `ingredients` と `unit` は `omitempty` にする。
`amount` は `float64` のまま書く。量 0 の材料も `amount: 0` を書き、読み戻したときに文字列の量として解釈される余地を作らない。
「少々」のように量を持たない言葉は単位として保存されているので、`amount: 0`、`unit: 少々` になり、そのまま読み戻せる。

**本文:** 手順が空でなければ `## 手順` の節を、メモが空でなければ `## メモ` の節を書く。両方空なら本文を書かない。
見出しは「手順」「メモ」で固定する。Go 側は表示文言を持たない方針だが、これは表示ではなくインポート形式のキーワードであり、
[docs/recipe-import.md](../../../docs/recipe-import.md) の例と揃える。英語の UI で使っていても、読み戻しには影響しない。

**レベル 2 の見出しの扱い:**

- (a) そのまま書く
  — Pros: 書き出したファイルが保存内容と一致する
  / Cons: 読み戻すと「## 下ごしらえ」以降の手順が「手順」以外の節として捨てられ、レシピの内容が欠ける
- (b) 行頭の `## ` を `### ` に下げる ← 採用
  — Pros: 読み戻しで内容が欠けない。S03 の描画は h2 を h4、h3 を h5 に下げているので、見た目の差は見出しの大きさ 1 段だけ
  / Cons: 往復で完全には一致しない
- (c) インポートの読み方を変え、「## 手順」の後のレベル 2 の見出しを節の区切りにしない
  — Pros: 往復が完全に一致する
  / Cons: 「## 材料」のような AI の出力の余計な節を捨てる既存の振る舞い (spec の Requirement「インポートするファイルの形式」) が壊れる

**採用: (b)**。見出しの判定は `splitImportBody` と同じく、コードブロック (` ``` ` と `~~~`) の中を除いて行う。
フェンスの判定を 2 か所で別々に書くと食い違うので、`importfile.go` の判定を小さな関数に切り出して共有する。

### D4. ファイル名

**選択肢:**

- (a) レシピ名をそのまま使い、使えない文字を置き換える ← 採用
  — Pros: ファイルを一覧したときにどのレシピか分かる
  / Cons: OS ごとの禁止文字と予約名に備える処理が要る
- (b) レシピの ID をファイル名にする
  — Pros: 衝突せず、処理も要らない
  / Cons: ファイル名から中身が分からず、手で直したいファイルを探せない

**採用: (a)**。配布先の Windows の規則に合わせ、どの OS でも同じ名前にする。`ExportFileName(name string, used map[string]bool) string` の手順は次のとおり。

1. `< > : " / \ | ? *` と制御文字を `_` に置き換える
2. 前後の空白と、末尾の `.` を除く
3. 100 文字 (rune) を超えたら 100 文字に切る。Windows のパス長の上限 (260) に、選んだフォルダのパスの分の余裕を残すため
4. 空になったら `recipe` にする
5. `CON`、`PRN`、`AUX`、`NUL`、`COM1`〜`COM9`、`LPT1`〜`LPT9` (大文字と小文字を区別しない) なら先頭に `_` を付ける
6. 小文字にした名前が `used` にあれば `-2`、`-3` と番号を付けて空くまで試す。Windows と macOS のファイルシステムは大文字と小文字を区別しないため
7. `.md` を付ける

### D5. 書き出し用のフォルダ

**選択肢:**

- (a) 選んだフォルダに直接書き、同名のファイルがあれば上書きする
  — Pros: 書き出すたびに同じ場所が最新になる
  / Cons: 無関係なファイルや、手で直した書き出し済みのファイルを黙って上書きしうる
- (b) 選んだフォルダに直接書き、同名のファイルがあれば番号を付ける
  — Pros: 上書きしない
  / Cons: 2 回目の書き出しで「肉じゃが-2.md」が並び、どれが新しいか分からなくなる
- (c) 選んだフォルダの中に、日時を含む名前のフォルダを毎回作る ← 採用
  — Pros: 上書きが起こらず、いつの書き出しかが名前で分かる。失敗したときはフォルダごと消せば後始末が済む
  / Cons: 書き出すたびにフォルダが増える

**採用: (c)**。フォルダ名は `meal-cycle-recipes-YYYYMMDD-HHMMSS` (ローカル時刻) にする。
`os.Mkdir` が既存で失敗したとき (同じ秒に 2 回書き出した場合) は `-2`、`-3` を付けて作り直す。
この名前はファイルシステム上の識別子で、表示文言ではないので Go 側に置く。

### D6. エラーの伝え方

型付きのエラーを 2 つ足し、フロントが i18n key に対応付ける。

| エラー | 意味 | トースト |
| --- | --- | --- |
| `ErrRecipeExportEmpty` | 書き出すレシピが無い | `pages.recipes.export.empty` |
| `ErrRecipeExportWrite` | フォルダの作成かファイルの書き込みに失敗した | `pages.recipes.export.failed` |

ダイアログ自体を開けなかったエラーも `ErrRecipeExportWrite` に寄せる。利用者から見て、どちらも「書き出せなかった」で足りるため。
Go のエラー文字列 (パスや OS の理由) は詳細としてトーストの説明欄に出す。権限が無いのか、ディスクが一杯なのかを利用者が見分けられるようにする。

### D7. フロントの構成

`RecipeExportButton` は `useMutation` で `RecipeService.ExportToFolder(t("pages.recipes.export.dialog_title"))` を呼ぶ。

- 実行中はボタンを押せなくする。ダイアログを開いている間も実行中に含まれるので、二重に開かない
- 成功：`Cancelled` なら何もしない。そうでなければ `pages.recipes.export.done` (件数) のトーストを出し、説明欄に `Dir` を出す
- 失敗：D6 のトーストを出す
- レシピの件数は S02 がすでに持っている `RecipeService.List` の結果から取り、0 件なら `disabled` にする

## Risks / Trade-offs

- [Risk] レシピが 100 件を超えると、書き出したフォルダをインポートの「フォルダを選ぶ」で 1 回では読めない (インポートの上限)
  → Mitigation：「ファイルを選ぶ」で 100 件ずつ選べば読める。この proposal では上限を変えず、proposal の後続への引き継ぎに残す
- [Risk] Wails のフォルダ選択ダイアログは E2E で通らず、キャンセル時の戻り値 (`""`) も実機でしか確かめられない
  → Mitigation：キャンセルは picker の差し替えでサービス層のテストに入れる。実機の挙動は Windows の確認でダイアログを閉じて確かめる
- [Risk] 手順とメモのレベル 2 の見出しを下げるので、往復で完全には一致しない
  → Mitigation：spec に明記し、[docs/recipe-import.md](../../../docs/recipe-import.md) にも書く。描画の差は見出しの大きさ 1 段だけ
- [Risk] 同じ名前のレシピの番号は作成日の順で振るので、レシピを削除すると次の書き出しで番号がずれる
  → Mitigation：番号はファイルを区別するためだけのもので、読み戻しには使わない。中身のレシピ名は元のまま
- [Risk] Linux の開発環境 (WebKitGTK) で Wails のフォルダ選択ダイアログが期待どおり開くかは確かめていない
  → Mitigation：配布先は Windows なので、Windows の実機で確かめる。開発時は E2E で書き込みまでを通す

## Migration Plan

スキーマは変えない。既存のデータにも触れない。ロールバックはコードを戻すだけで済む。
