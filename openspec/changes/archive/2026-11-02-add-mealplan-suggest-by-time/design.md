## Context

動機は proposal.md の Why を参照。

- `SuggestionService.List` はレシピ選択ダイアログを開いたときに 1 回呼ばれ、`Result{ByStock, ByRecency}` で全軸を返す (add-mealplan-suggest-by-stock design.md D2)
- 根拠の文言は Go で組まず、フロントが i18n で組む。候補は `Candidate` に軸ごとの根拠のフィールドを持たせている
- レシピは `recipes` テーブルに 1 行で、人数 `servings` は NOT NULL の整数
- インポートは YAML の frontmatter を `importfile.go` が読み、知らない項目 (`cookTime` など) は無視している。エクスポートは同じ形式で書く (`exportfile.go`)

## Goals / Non-Goals

**Goals:**

- 3 軸 (使い切り / 久しぶりに / 時短) をレシピ選択ダイアログにそろえる
- 所要時間を入れていないレシピが多くても、既存の動きを壊さない

**Non-Goals:**

- 材料数による推定と、週の候補への追加 (proposal.md の Non-goals)

## Decisions

### D1. 時短の判定は所要時間だけで行う

**選択肢:**

- (a) 所要時間を足し、それだけで判定する ← 採用 — Pros: 根拠が「N 分」と明確 / Cons: 所要時間を入れないと候補に出ない
- (b) 材料の数で判定する — Pros: 既存のデータで動く / Cons: 材料が 3 つの煮込みも時短になる。根拠として「材料 3 つ」は時短の理由にならない
- (c) 所要時間があればそれ、無ければ材料の数で判定する — Pros: 入力が無くても候補が出る /
  Cons: 2 種類の根拠が同じ欄に混ざり、並び順の基準が決まらない

**採用: (a)**。理由: 候補は根拠を添えて利用者に選ばせる方針 ([product.md §5](../../../docs/product.md)) で、根拠の確からしさを優先する。
所要時間の無いレシピばかりの間は「該当なし」が出るので、入れる動機にもなる。

### D2. 所要時間の持ち方: 0 を未入力とする整数

`recipes.cook_minutes INTEGER NOT NULL DEFAULT 0`、Go は `CookMinutes int`。0 は未入力、1〜999 は分。

**選択肢:**

- (a) NULL を許す列と `*int` — Pros: 未入力が型に出る / Cons: bindings で `number | null` になり、`.ts` と `.js` の生成で nullable の付き方が違う (CLAUDE.md の E2E shim の注意)
- (b) 0 を未入力とする ← 採用 — Pros: 既存の行に DEFAULT 0 で入り、migration が 1 文で済む。型が単純 / Cons: 0 分のレシピは表せない (意味が無いので困らない)

**採用: (b)**。

### D3. 欄の置き場所と並び

「久しぶりに」の下に「時短」を置く。並びは所要時間の昇順、同じならレシピ名の昇順。上位 5 件。
閾値 (既定 20 分) を超えるレシピは出さない。閾値で切らずに短い順に 5 件だけ出す案も考えたが、
40 分のレシピが「時短」として出ると根拠と欄の名前が合わないので見送る。

### D4. インポートの key 名: `cookMinutes`

**選択肢:**

- (a) `cookTime` を読み、「20分」「1時間」などの表記を解釈する — Pros: AI が出しがちな形をそのまま読める / Cons: 時間の表記の揺れを解釈する規則が要る
- (b) `cookMinutes` に分の整数で書かせる ← 採用 — Pros: 解釈の規則が要らない。プロンプトのひな形で整数を指示すれば AI はそのとおり出す /
  Cons: 既存のひな形で作ったファイルは所要時間を持たない

**採用: (b)**。理由: `recipe-import.md` はすでに AI に決まった形を出させる前提で書いてある。`cookTime` は今までどおり無視する。
範囲外 (0 以下、1000 以上、整数でない) はほかの項目と同じくそのファイルをエラーにする。

### D5. S09 の値

`QuickCookMaxMinutes int`、既定 20、5〜120。献立の節の「久しぶりに」とみなす日数の下に置く。

## Risks / Trade-offs

- [Risk] 所要時間を入れたレシピが少なく、欄がいつも「該当なし」になる → 欄は他の軸と同じく隠さない。入れる動機として残す
- [Risk] `add-shopping-household-scaling` も `S09 設定画面の表示` と `設定値の検証` を MODIFIED にする
  → 後に archive する change の delta を、先に archive した後の spec.md に合わせて書き直す (openspec-update-change)
- [Risk] migration の番号が `add-mealplan-template` と衝突する → 実装した順に番号を振り、後の change の proposal.md と tasks.md を直す
