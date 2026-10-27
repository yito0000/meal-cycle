## Context

動機は proposal.md の Why を参照。

- Wails サービス (`internal/service/`) は `*sql.DB` と `configSource` を受け取るコンストラクタで作れ、Wails に依存しないメソッドとして呼べる。
  `cmd/e2e-server` が同じサービスを HTTP から reflect で呼んでおり、Wails の外から使う前例がある
- `RecipeService` だけはフォルダ選択の `FolderPicker` を要求する。MCP では書き出しを公開しないので、選択しない実装を渡せば足りる
- DB は WAL / `foreign_keys` / `busy_timeout(5s)` で開いている (`sqlite.Open`)。複数プロセスが同じファイルを開くこと自体は SQLite の想定内
- 起動時のスキーマ更新は `sqlite.OpenAndMigrate` が控えを取ってから行う。`pendingMigrations` は「未適用があるか」だけを返し、DB がバイナリより新しい場合は判定しない
- `SettingsService` は作った時点で DB から設定を読み、以後はメモリの値を使う
- フロントの `QueryClient` は既定設定 (`new QueryClient()`)。TanStack Query v5 のフォーカス検出は `visibilitychange` だけを見る

## Goals / Non-Goals

**Goals:**

- 既存のサービスとドメインの検証をそのまま通し、GUI とエージェントで同じ規則が効くようにする
- MCP サーバーがスキーマや控えを触らず、GUI の起動時の安全策 (`add-shell-startup-safety`) を迂回しないようにする
- ツールの名前と入力の形を、エージェントが説明文だけで使える粒度にする

**Non-Goals:**

- サービス層の公開メソッドを MCP のために変えること (bindings を変えない)
- MCP の SDK を抽象化する層を作ること。差し替えの予定が無い

## Decisions

### D1. 配布形態: 別の実行ファイル

**選択肢:**

- (a) GUI 本体のサブコマンド (`meal-cycle mcp`) — Pros: 配る exe が 1 本。DB の場所の解決を共有しやすい /
  Cons: Windows の GUI ビルドは `-H windowsgui` で、標準入出力の扱いが親プロセスの起動方法に左右される。
  サブコマンドでも Wails の初期化を避ける分岐が `main.go` に入る
- (b) 別の実行ファイル `cmd/mcp-server` → `meal-cycle-mcp(.exe)` ← 採用 — Pros: コンソールのサブシステムで素直に標準入出力を使える。
  GUI の起動経路に手を入れない / Cons: 配る exe が 2 本になる
- (c) GUI に HTTP の MCP エンドポイントを埋め込む — Pros: GUI のキャッシュを直接更新できる /
  Cons: localhost でもポートを開く (design.md §2 に反する)。GUI を起動していないと使えない

**採用: (b)**。理由: (a) の Windows の標準入出力の不確かさと、(c) の通信の方針違反を避けられる。exe が増える手間は `task build:win` に含めて吸収する。

### D2. MCP の実装: 公式 Go SDK

**選択肢:**

- (a) `github.com/modelcontextprotocol/go-sdk` (v1 系) ← 採用 — Pros: 仕様の公式実装。型付きの入力から JSON Schema を作れる。
  in-memory の transport があり、テストでクライアントとサーバーを同じプロセスで繋げる / Cons: 依存が増える
- (b) `mark3labs/mcp-go` — Pros: 利用例が多い / Cons: 公式 SDK が v1 になった今、選ぶ理由が薄い
- (c) JSON-RPC を自前で書く — Pros: 依存なし / Cons: 初期化やスキーマ生成など、仕様の追従を自分で持つことになる

**採用: (a)**。理由: 公式で、テスト用の transport まで揃っている。`go get github.com/modelcontextprotocol/go-sdk@latest` で入れる (2026-10 時点で v1.8.0)。

### D3. DB の開き方: 版が一致するときだけ開き、更新しない

`internal/storage/sqlite/agent.go` に `OpenExisting(path)` を足す。

1. ファイルが無ければ `ErrDatabaseMissing` (ファイルを作らないよう、`Open` の前に `os.Stat` で判定する)
2. `Open` で開き、goose の現在の版と、埋め込んだ migration の最大の版を比べる
3. 現在 < 最大 なら `ErrSchemaOutdated`、現在 > 最大 なら `ErrSchemaTooNew`。どちらも閉じて返す

**選択肢:**

- (a) `OpenAndMigrate` を使う — Pros: 実装が要らない / Cons: GUI と MCP が同時に起動したとき、両方が控えと更新を走らせうる。
  古い MCP の exe が新しい DB を開いた場合に気付けない
- (b) 版が一致するときだけ開く ← 採用 — Pros: スキーマの変更は GUI の起動だけが行う、という不変条件を保てる /
  Cons: 更新後、GUI を一度起動するまで MCP が使えない

**採用: (b)**。理由: 控えと更新を 1 か所に保つほうが、利用者が気付かないうちに DB が変わる経路を作らずに済む。

版の確認はツールを呼ばれるたびに行う (D4)。MCP サーバーは AI クライアントの起動と同時に立ち上がって長く居続けるので、
起動時に 1 回だけ確かめると、その後に GUI が更新した DB を古い前提のまま触ることになる。

### D4. サービスはツール呼び出しごとに作る

ツールを呼ばれるたびに `OpenExisting` → `NewSettingsService(db)` → 必要なサービス、の順で作り、呼び出しの終わりに DB を閉じる。

**選択肢:**

- (a) 起動時に 1 回だけ作り、使い回す — Pros: 速い / Cons: GUI で変えた設定が MCP に届かない (`SettingsService` はメモリの値を使う)。
  D3 の版の確認も起動時だけになる
- (b) 呼び出しごとに作る ← 採用 — Pros: 設定と版をいつも DB から読む / Cons: 呼び出しごとに DB を開く分だけ遅い
- (c) 使い回しつつ、呼び出しごとに設定だけ読み直す — Pros: (b) より速い / Cons: `SettingsService` に再読み込みのメソッドを足すと Wails の bindings に出る

**採用: (b)**。理由: エージェントの呼び出しは人の操作の間隔で来るので、SQLite を開く数ミリ秒は問題にならない。サービス層を変えずに済む。

### D5. ツールの一覧と粒度

ツール名は英語の snake_case にし、説明文も英語で書く (エージェントの多くが英語の説明で最もよく動くため。UI の文字列ではないので i18n の対象外)。
入力の日付は `YYYY-MM-DD`、時間帯は `breakfast` / `lunch` / `dinner`。

| ツール | 呼ぶサービス | 備考 |
|---|---|---|
| `list_recipes` | `RecipeService.List` | 材料と手順は省き、ID / 名前 / 人数 / タグだけ返す |
| `get_recipe` | `RecipeService.GetByID` | |
| `list_meal_plan` | `MealPlanService.ListPeriod` | `start`, `weeks` (1 / 2 / 4) |
| `list_stock` | `FridgeService.List` | |
| `list_staples` | `FridgeService.ListStaples` | |
| `get_shopping_list` | `ShoppingService.List` | `start`, `days` |
| `list_suggestions` | `SuggestionService.List` | |
| `suggest_week_plans` | `SuggestionService.WeekPlans` | `weekStart` |
| `assign_recipe` | `MealPlanService.AssignRecipe` | `date`, `meal`, `recipeId` |
| `add_stock_items` | `FridgeService.AddMany` | 全件検証してから 1 トランザクションで入れる既存の挙動をそのまま使う |
| `add_staple` | `FridgeService.AddStaple` | |
| `set_staple_state` | `FridgeService.SetStapleState` | `state` は `enough` / `low` / `out` |
| `create_recipe` | `RecipeService.Create` | |

**選択肢:**

- (a) サービスの公開メソッドを reflect で全部ツールにする (e2e-server と同じ) — Pros: メソッドを足すと自動で増える /
  Cons: 削除系まで出る。入力の型が Go の struct そのままで、説明が付かない
- (b) 公開するものを手で選び、入力の型と説明を MCP 用に書く ← 採用 — Pros: 公開範囲を明示できる。エージェント向けの説明を付けられる /
  Cons: サービスを足しても自動では増えない

**採用: (b)**。理由: 公開範囲 (spec の「エージェントに公開しない操作」) をコードで固定できる。
`tools_test.go` でツール名の一覧を固定値と突き合わせ、意図せず増えたら落ちるようにする。

ツールの戻り値は、構造化した結果 (SDK の structured content) と、同じ内容の JSON テキストの両方で返す。
structured content を読まないクライアントがあるため。

### D6. エラーの返し方

`cmd/mcp-server/errors.go` に、domain と `internal/storage/sqlite` の sentinel error (`sqlite.ErrRecipeNotFound` など) とコードの対応表を置く。
`errors.Is` で引き、見つかったコードと `err.Error()` (domain のエラー文は英語) を、ツールの失敗 (`isError: true`) として返す。

- コードは `<capability>.<snake_case>` (例: `fridge.stock_name_empty`, `mealplan.invalid_date`, `database.schema_outdated`)
- 表に無いエラーは `internal` とし、説明に `err.Error()` をそのまま入れる
- 入力の JSON Schema で弾かれた場合は SDK が返すエラーに任せる (コードは付かない)。日付の書式など、スキーマで表せない検証はサービスの検証に任せる

**選択肢:**

- (a) `err.Error()` をそのまま返す — Pros: 対応表が要らない / Cons: `row 1: fridge: name empty` のように包まれた形で、エージェントが種類を判定しにくい
- (b) コードと説明を返す ← 採用 — Pros: design.md §7 の「Go はコードで返す」に揃う / Cons: domain のエラーを足したら表も直す必要がある

**採用: (b)**。表の漏れは `internal` に落ちるだけで動作は止まらないので、漏れのリスクは小さい。

### D7. GUI の取り直し: フォーカスの検出を `focus` イベントに合わせる

`frontend/src/main.tsx` で TanStack Query の `focusManager.setEventListener` を使い、`window` の `focus` で `handleFocus(true)` を呼ぶ。
既定の `refetchOnWindowFocus` (stale なクエリを取り直す) はそのまま効く。

**選択肢:**

- (a) 何もしない — Pros: 変更なし / Cons: v5 は `visibilitychange` だけを見るので、別ウィンドウから戻っただけでは取り直さない
- (b) `focus` を検出に加える ← 採用 — Pros: 数行で済む。既定のキャッシュ戦略を変えない / Cons: フォーカスのたびに表示中のクエリを取り直す
- (c) 一定間隔で取り直す — Pros: 戻らなくても反映される / Cons: 操作していない間も DB を読み続ける
- (d) DB の変更を Go 側で監視して Wails のイベントで知らせる — Pros: 即時 / Cons: 別プロセスの書き込みを検出する仕組みが要り、規模が大きい

**採用: (b)**。理由: エージェントの操作中、利用者は AI クライアントのウィンドウにいるので、GUI に戻った時点で揃えば足りる。

### D8. ビルドと配布

- `Taskfile.yml` に `build:mcp` を足す: `go build -trimpath -o bin/meal-cycle-mcp ./cmd/mcp-server`
- `build:win` の後に `GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/meal-cycle-mcp.exe ./cmd/mcp-server` を足す
  (`-H windowsgui` は付けない。付けると標準入出力を持てない)
- `build:win:copy` は `meal-cycle-mcp.exe` も `WIN_DEST` へコピーする
- AI クライアントへの登録例 (Claude Desktop の `claude_desktop_config.json`) は `docs/mcp.md` に書く

## Risks / Trade-offs

- [Risk] GUI と MCP が同時に書き込むと、片方が `SQLITE_BUSY` になる → `busy_timeout(5s)` で待つ。
  各ツールの書き込みは 1 トランザクションで短いので、5 秒を超える競合は起きにくい。超えたら `internal` で返り、エージェントが再試行できる
- [Risk] エージェントの誤った追加 (違う日への割り当て、重複した在庫) を、利用者がすぐ取り消せない →
  追加したものは GUI の既存の操作 (削除と取り消し) で消せる。公開を追加系に絞り、データが消える操作は出さない
- [Risk] データがエージェントの LLM (多くはクラウド) に送られ、「完全ローカル」の印象と食い違う → MCP サーバー自体は通信しないことと、
  エージェントに渡したデータの扱いは AI クライアント側の規約に従うことを `docs/mcp.md` と design.md §2 に明記する。登録は利用者が自分で行う opt-in にする
- [Risk] `internal/service` が Wails の `application` パッケージを import しており (`folderpicker.go`)、MCP の exe にも Wails が入る。
  Windows で Wails の初期化が副作用を持つと、MCP サーバーの起動に影響しうる → e2e-server で Linux では問題が無いことを確認済み。
  Windows では実機で `tools/list` が返ることを確かめる (tasks の手動確認)。問題があれば `folderpicker.go` の Wails 実装を `main` 側へ移す
- [Risk] Windows の WebView2 と WebKit2GTK で、ウィンドウの切り替え時に `focus` が発火しない場合がある → Windows 実機で D7 の挙動を確かめる。
  発火しなければ Wails のウィンドウイベント (`WindowFocus`) から同じ処理を呼ぶ
- [Trade-off] D3 により、アプリを更新したあとは GUI を一度起動するまで MCP が使えない → エラーのコードと説明で GUI の起動を促す

## Migration Plan

DB の変更は無い。新しい exe を配り、利用者が AI クライアントに登録したときから使える。
やめるときは AI クライアントの設定から外せばよく、DB には何も残らない。
