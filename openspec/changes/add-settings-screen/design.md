## Context

可変値は `internal/config/defaults.go` の `Defaults()` に集めてあり、各サービスはコンストラクタで `config.Defaults()` を読んで
`cfg` フィールドに持つ (`fridge.go` / `mealplan.go` / `recipe.go` / `shopping.go`)。
値を変える経路は無く、`Config` は「不変な既定値」の層として扱ってきた
(`add-mealplan-ui-preferences-persistence` design.md D1 の (c) 案を退けた理由)。

フロントは、サービスを通さずに既定値を使う箇所で Go の値を手で写している。

- `frontend/src/components/mealplan/mealplan-config.ts` の `MEALPLAN_DEFAULTS.leftoverCopyPresets` (S01 のコピー日数と、作り置きを広げる格子の大きさ)
- `frontend/src/components/shopping/shopping-config.ts` の `SHOPPING_DEFAULTS` (S07 の日数プリセットと既定日数)
- `frontend/src/lib/format-quantity.ts` の `MEASURING_UNITS` (量を単位の後ろに置くか前に置くか)

起動の流れは `add-shell-startup-safety` で決めたとおりで、DB を開けないときは `HealthService` だけを登録し、
フロントの `RootLayout` は `useHealth` の結果を待ってから各画面を描く。
表示言語は `frontend/src/i18n.ts` で `lng: "ja"` に固定している。

## Goals / Non-Goals

**Goals:**

- 保存した値を、再起動せずにサービスとフロントの両方へ同じ値で行き渡らせる
- 既定値の source of truth を `internal/config/defaults.go` に残したまま、利用者が変えた値だけを DB に持つ
- フロントの既定値の写しを、本 proposal で扱う 3 か所についてなくす

**Non-Goals:**

- 設定値の変更履歴や、複数の設定プロファイルの切替
- S01 のセルサイズのプリセット表 (`CELL_SIZE_PRESETS`) と `MEALPLAN_DEFAULTS.defaultVisibleMeals` の写しの解消。
  利用者が変える値ではないので、写しのまま残す

## Decisions

### D1. 保存先

**選択肢:**

- (a) SQLite に `settings` テーブルを足し、項目ごとに 1 行 (key / JSON の value) で持つ ← 採用 —
  Pros: 在庫やレシピと同じ DB に入るので、起動時の控え (`backups/`) にも含まれる。項目ごとに読めないときの扱いを決められる。
  Cons: migration / repo を 1 組足す
- (b) SQLite に 1 行だけのテーブルを作り、全項目を 1 つの JSON で持つ — Pros: 読み書きが 1 文で済む。
  Cons: JSON の一部が壊れると全項目を捨てるか、自前で部分的に読み解く処理が要る
- (c) DB と同じフォルダに `settings.json` を置く — Pros: テキストエディタで直せる。
  Cons: 控えに含まれない。書き込み途中の破損対策 (一時ファイルからの改名) を自前で書く必要がある
- (d) localStorage — Pros: Go 側の変更が要らない。
  Cons: サービスが値を読めないので、S05 の期限状態や S07 の合算に反映できない

**採用: (a)**。サービスが読める必要があるので (d) は使えない。
(b) と (c) は「壊れた項目だけ既定値に戻す」(spec「保存した値を読めないときの扱い」) を実現する手間が (a) より大きい。

### D2. 保存する範囲と項目の束ね方

既定値から変えた項目だけを保存し、読むときは `Defaults()` に重ねる。
全項目を保存すると、将来のリリースで既定値を見直したときに、値を一度も触っていない利用者にも古い既定値が残り続けるため。
「既定に戻す」は `settings` テーブルの全行を消す操作になる。

互いに条件を持つ値は 1 つの key に束ねる。

| key | 中身 |
| --- | --- |
| `language` | `"ja"` / `"en"` |
| `expiryAlert` | `{ "redDays": 3, "yellowDays": 7 }` |
| `leftoverCopyPresets` | `[2, 3, 4]` |
| `shoppingPeriod` | `{ "dayPresets": [3, 4, 7, 14, 28], "defaultDays": 7 }` |
| `measuringUnits` | `["大さじ", "小さじ", "カップ"]` |
| `unquantifiedWords` | `["少々", ...]` |
| `recipeImportDefaultServings` | `2` |

**選択肢:**

- (a) 互いに条件を持つ値を 1 key に束ねる ← 採用 — Pros: どの key も単独で検証できる。
  壊れた key を既定値に戻しても、別の key と矛盾しない。Cons: `Config` のフィールドと key が 1 対 1 にならない
- (b) `Config` のフィールドごとに 1 key — Pros: 対応が素直。
  Cons: 🔴 だけが壊れて既定値 3 に戻ると、保存済みの 🟡 = 2 と矛盾する。矛盾したときにどちらを捨てるかの規則が要る

**採用: (a)**。検証の単位と保存の単位がそろうので、読めないときの扱いが「その key を捨てる」の 1 規則で済む。

### D3. サービスへの反映

**選択肢:**

- (a) 保存したら再起動を求める — Pros: サービスの構造を変えない。Cons: spec の「再起動せずに反映」を満たせない
- (b) 各サービスが呼び出しのたびに DB から設定を読む — Pros: キャッシュの整合を考えなくてよい。
  Cons: 全メソッドに設定の読み込み失敗の分岐が入る
- (c) `SettingsService` が現在値をメモリに持ち、他サービスはそれを参照する ← 採用 —
  Pros: 読み込みは起動時と保存時だけ。他サービスのメソッドに失敗の分岐が増えない。
  Cons: `SettingsService` の作成順に依存する (main.go で先に作る)

**採用: (c)**。他サービスのコンストラクタは `config.Config` の代わりに、次の非公開 interface を受け取る。

```go
type configSource interface {
    current() config.Config
}
```

メソッドを非公開にするのは、Wails が公開メソッドをすべて bindings に出すため。
`Current()` にすると、フロントから `Config` 全体 (セルサイズの表など) が見えてしまう。
`SettingsService` は `sync.RWMutex` で現在値を守り、`Save` / `Reset` が DB への書き込みに成功してから差し替える。
サービスのテストでは `staticConfig` (`config.Config` をそのまま返す型) を渡す。

### D4. 公開 API

`SettingsService` は次の 3 つを公開する。どれも保存後の現在値を `settings.Values` で返す。

- `Get(ctx) (settings.Values, error)`
- `Save(ctx, values settings.Values) (settings.Values, error)`：正規化 → 検証 → 既定値との差分を `ReplaceAll` → 現在値の差し替え
- `Reset(ctx) (settings.Values, error)`：`DeleteAll` → 現在値を `Defaults()` に戻す

`settings.Values` は S09 で変えられる項目だけを持つ平らな構造体にし、`Config` 全体は返さない。
変換は `settings.FromConfig(config.Config) Values` と `Values.ApplyTo(config.Config) config.Config` の 2 つにする。

他案として、項目ごとに `SetExpiryAlert` のような setter を並べる形も考えた。
S09 は全項目を 1 つの「保存」で送る画面なので、setter を並べても呼び出しが増えるだけで得るものが無い。

### D5. 検証の置き場所とエラーの返し方

**選択肢:**

- (a) Go の `internal/domain/settings` だけで検証し、フロントは返ってきたエラーを欄に出す — Pros: 規則が 1 か所。
  Cons: 1 回の保存で分かるエラーが 1 つだけになる
- (b) Go で検証し、フロントは候補と言葉の「追加」の時点で重複、空白、文字数だけを確かめる ← 採用 —
  Pros: 入力中に気付けるエラー (重複など) はその場で出せる。保存の可否は Go が決める。
  Cons: 重複、空白、文字数の規則がフロントにも現れる
- (c) フロントと Go の両方で全規則を検証する — Pros: 保存前に全エラーを出せる。Cons: 規則の二重管理になり、今回なくしたい写しが別の形で戻る

**採用: (b)**。フロントが持つのは「1 つの値を追加できるか」の規則だけにし、件数と項目間の条件は Go に任せる。
Go は既存の `fridge` と同じく sentinel error (`settings: red days not below yellow days` など) を返し、
`frontend/src/components/settings/error-messages.ts` がメッセージから i18n key と欄を引く。

上限と下限 (🟡 は 60 日以下、候補は 5 個以下など) は `internal/domain/settings/validate.go` の定数に置く。
利用者が変える値ではなく実装上の上限なので、`defaults.go` には置かない (`fridge.go` の上限と同じ扱い)。
作り置きの日数の上限 7 は、「作り置きを広げる」の格子の行数と「他の作り置きにまとめる」の探索範囲がこの値で広がるため、
1 週間に収まる大きさで止める。

### D6. 保存した値を読めないとき

`SettingsService` の作成時と `Get` の前提になる読み込みは、key ごとに JSON の解釈と検証を行い、
失敗した key は捨てて既定値を使う。捨てた key は DB からは消さず、次の `Save` が `ReplaceAll` で上書きする。
未知の key (将来の版で足した key を古い版で読んだ場合) も同じく無視する。

`settings` テーブル自体を読めない (SQL のエラー) ときも、既定値で動かしてログに残す。
起動時のエラー画面は DB を開けない場合などに限っており (shell の「起動時の失敗の通知」)、
設定が読めないだけで献立や在庫まで使えなくするのは釣り合わないため。

### D7. 表示言語を Config に持つか

**選択肢:**

- (a) `config.Config` に `Language` を足し、他の項目と同じく `settings` テーブルに保存する ← 採用 —
  Pros: 保存と「既定に戻す」の経路が 1 本で済む。Cons: Go は表示言語を使わないのに値を持つ
- (b) フロントの localStorage だけに持つ — Pros: 起動時に Go を待たずに言語を決められる。
  Cons: S09 の保存と既定に戻す操作が 2 系統になる。DB の控えに含まれない

**採用: (a)**。起動時のちらつきは D8 の待ち合わせで防げるので、(b) の利点は残らない。

### D8. フロントでの受け取り

`RootLayout` の `useHealth` の待ち合わせの内側に `SettingsGate` を置き、`Outlet` を包む。

1. `useSettings()` (TanStack Query、query key `["settings"]`、`staleTime: Infinity`) で `SettingsService.Get` を呼ぶ
2. 取得できたら `i18n.changeLanguage(values.language)` を待ってから子を描く。待つ間は `useHealth` の待ち合わせと同じ空の背景を出す
3. 取得に失敗したら、再試行ボタン付きのメッセージを出す (i18n は既定の日本語)

`SettingsGate` の内側の部品は `useSettingsValues()` で取得済みの値を受け取る。
`SettingsGate` が取得前に子を描かないので、`useSettingsValues()` は `undefined` を返さない型にできる。

保存と既定に戻すの成功時は、返ってきた値で `["settings"]` を `setQueryData` し、表示言語を切り替え、
`["settings"]` 以外の query をすべて無効にする。
S05 の期限状態や S07 の合算はサービスが計算した結果なので、取り直さないと古い閾値のまま表示されるため。

起動時のエラー画面は `SettingsGate` の外側にあるので、日本語のまま表示する (spec「表示言語の切替」)。

### D9. フロントの写しの置き換え

- `formatQuantity(amount, unit, measuringUnits)`：純関数のまま、第 3 引数で量る単位を受け取る。
  呼び出し側 (S03 / S05 / S07 の部品、`shopping-text.ts`) が `useSettingsValues().measuringUnits` を渡す
- `resolveShoppingPeriod(search, today, period)` と `isPresetDays(days, presets)`：候補と既定日数を引数で受け取る。
  `shoppingSearchSchema` は型だけを検査するので変えない
- `week-calendar.tsx` の `SPREAD_INITIAL_FUTURE_DAYS` / `SPREAD_PRESET_DAYS`：モジュールの定数をやめ、部品の中で設定から求める

他案として、フロント用の React Context に設定を流し込み、`formatQuantity` の中で Context を読む形も考えた。
純関数でなくなり、`shopping-text.ts` (クリップボード用の文字列を作る) から呼べなくなるので採らない。

### D10. S09 のフォーム

S09 は保存済みの値をフォームの初期値にし、フォーム内の値と保存済みの値が違う間だけ「保存」を押せるようにする。
比較は正規化 (数値の並べ替え、前後の空白除去) をしてから行い、並べ替えただけの変更では押せるようにしない。

- 日数の候補：値ごとのチップに「×」を付け、末尾に数値の入力欄と「追加」を置く。買い物リストの 7 / 14 / 28 のチップは「×」を出さない
- 単位と言葉：同じくチップと入力欄。Enter でも追加でき、IME の変換確定の Enter は `frontend/src/lib/ime.ts` で除外する
- サイドバーから開いたときの日数：フォーム内の買い物リストの候補を選択肢にする select。選択中の候補を消したら、7 を選び直す
- 「既定に戻す」：既存の `dialog.tsx` で確認を出す

ワイヤーは `docs/wireframes/S09_settings.drawio.svg` に描き、product.md §4 には絵から読めない相互作用 (保存の可否、7 / 14 / 28 を消せない理由) だけを書く。

## Risks / Trade-offs

- [量る単位を消すと、登録済みのレシピの表示が変わる (「大さじ 1」が「1 大さじ」になり、S07 で量のある材料として合算される)]
  → spec で「保存の前に登録したレシピは書き換えない」と決め、表示は常に現在の設定に従うものとして扱う。
  消した単位を戻せば表示も戻るので、データは失われない
- [作り置きの日数の候補を小さくすると、既存の作り置きが「他の作り置きにまとめる」の候補に出にくくなる]
  → 探索範囲が狭まるだけで、作り置き自体は残る。spec の反映先の一覧に明記する
- [`SettingsService` を他サービスより先に作る順序に依存する]
  → 他サービスのコンストラクタの引数を `configSource` にし、渡し忘れはコンパイルエラーになるようにする。
  `cmd/e2e-server` の `newRegistry` も同じ組み立てにし、`dispatch_test.go` の登録数の突き合わせで食い違いを検出する
- [起動時にもう 1 回 IPC の往復を待つので、最初の画面が出るまでが延びる]
  → ローカルの SQLite を 1 回読むだけなので数ミリ秒で済む。`useHealth` と同じ空の背景を出し、文言のちらつきを出さない
- [フロントの「追加」時の検証と Go の検証で、重複、空白、文字数の規則が二重になる]
  → フロントの規則は追加の可否だけに絞り、件数と項目間の条件は持たない。Go の検証を最終判定にし、ずれたときは Go のエラーが欄に出る
- [進行中の `add-frontend-unit-tests` が `formatQuantity(amount, unit)` の形でテストを書く予定]
  → 後から入る側が第 3 引数に量る単位を渡す形に合わせる (proposal の後続への引き継ぎに記載)

## Migration Plan

- `00010_settings.sql` を足す。既存の DB は起動時のスキーマ更新でテーブルが増えるだけで、行は空のまま (すべて既定値) 動く
- スキーマ更新の前の控えは `add-shell-startup-safety` の仕組みで自動的に作られる
- 戻すときは goose の Down で `settings` テーブルを消す。消しても他のテーブルに影響はなく、値は既定値に戻る
