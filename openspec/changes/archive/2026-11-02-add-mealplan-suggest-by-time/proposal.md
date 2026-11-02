## Why

[product.md §5](../../../docs/product.md) の献立候補の 3 軸のうち、「使い切り」と「久しぶりに」は実装した (add-mealplan-suggest-by-stock / -by-recency)。
残る「時短」は、平日の夜に手早く作れるレシピを選びたいときの軸である。
レシピに所要時間の項目が無いため、軸を作る前提が欠けていた ([design.md §1](../../../docs/design.md))。

本 proposal はレシピに所要時間を足し、レシピ選択ダイアログに「時短」の候補の欄を足して 3 軸をそろえる。

## What Changes

- レシピに所要時間 (分) を足す。任意の項目で、入れるなら 1〜999 の整数
  - S04 の基本情報の人数の隣に「所要時間 (分)」の欄を足す
  - S03 の基本情報に「約 N 分」を出す。未入力なら出さない
  - インポートのファイルの frontmatter で `cookMinutes` を読む。エクスポートは所要時間があれば `cookMinutes` を書く
- レシピ選択ダイアログの「久しぶりに」の欄の下に「時短」の欄を足す
  - 所要時間が S09 の「時短」とみなす分数以下のレシピを、所要時間の短い順、同じならレシピ名の昇順に上位 5 件出す
  - 根拠は「N 分」。所要時間が未入力のレシピと削除済みのレシピは候補にしない
  - 見出しの件数、該当なし、押したときの動作、検索中に隠すことは、ほかの 2 軸と同じ
- S09 の献立の節に「『時短』とみなす所要時間」(既定 20 分、5〜120 分) を足す

### Non-goals

- **材料の数による時短の推定**：[product.md §5](../../../docs/product.md) は「所要時間 or 材料数」としていたが、材料が少なくても煮込みで時間がかかるレシピは多い。所要時間を入れたレシピだけを対象にする
- **週の候補への時短の軸の追加**：週の候補は 3 種類の順位表で案を作っている。時短を加えるかは、セルの候補で使われ方を見てから決める
- **曜日ごとの時短の指定 (平日だけ時短など)**
- **S02 の一覧への所要時間の列と並び替え**
- **既存のレシピへの所要時間の一括入力**：S04 で 1 件ずつ入れるか、エクスポート → 編集 → インポートで入れる

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：レシピに所要時間を足し、S04 / S03 / インポート / エクスポートで扱う
- `suggestion`：レシピ選択ダイアログに「時短」の候補を足す
- `settings`：S09 の献立の節に「時短」とみなす所要時間を足し、検証の規則に範囲を足す

## Impact

**新設ファイル**

- `internal/storage/sqlite/migrations/00012_recipes_cook_minutes.sql`
- `internal/domain/suggestion/by_time.go`
- `internal/domain/suggestion/by_time_test.go`
- `frontend/e2e/suggestion-by-time.spec.ts`

**更新ファイル**

- `internal/domain/recipe/recipe.go`：`Recipe` に `CookMinutes int` (0 = 未入力) を足し、検証に 0〜999 を足す。`ErrCookMinutesOutOfRange` を足す
- `internal/domain/recipe/recipe_test.go`
- `internal/domain/recipe/importfile.go` / `importfile_test.go`：frontmatter の `cookMinutes` を読む。範囲外は `ImportIssueCookMinutesOutOfRange`
- `internal/domain/recipe/exportfile.go` / `exportfile_test.go`：`CookMinutes` が 0 でなければ `cookMinutes` を書く
- `internal/storage/sqlite/recipe_repo.go` / `recipe_repo_test.go`：`cook_minutes` 列の読み書き
- `internal/config/defaults.go` / `defaults_test.go`：`QuickCookMaxMinutes int` (既定 20)
- `internal/domain/settings/settings.go` / `validate.go` / `errors.go` / `settings_test.go`：`QuickCookMaxMinutes` と `ErrQuickCookMinutesOutOfRange` (5〜120)
- `internal/service/settings.go`：`settingKeys` に `quickCookMaxMinutes` を足す
- `internal/domain/suggestion/suggestion.go`：`Candidate` に `CookMinutes int` を、`Result` に `ByTime []Candidate` を足す
- `internal/storage/sqlite/suggestion_repo.go`：レシピの所要時間を読む
- `internal/service/suggestion.go` / `suggestion_test.go`：`List` で `ByTime` を埋める
- `frontend/src/components/recipe/recipe-edit-form.tsx` / `use-recipe-form.ts`：所要時間の欄
- `frontend/src/components/recipe/recipe-detail.tsx`：基本情報に「約 N 分」
- `frontend/src/components/recipe/recipe-import-preview.tsx`：プレビューに所要時間
- `frontend/src/components/mealplan/recipe-picker-dialog.tsx` / `suggestion-section.tsx`：「時短」の欄
- `frontend/src/components/settings/settings-form.tsx` / `error-messages.ts`：S09 の欄とエラー
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`
- `docs/recipe-import.md`：frontmatter の項目に `cookMinutes` を足し、「`cookTime` は読まない」の例を直す。AI 向けのプロンプトのひな形に足す
- `docs/product.md`：§1 可変値一覧に行を足す。§4 S03 / S04 / S09 と §5 の時短の行を直す
- `docs/design.md`：§1 と §9 の時短を実装済みに直す。§4 の Recipe に `cookMinutes` を足す
- `docs/wireframes/S01a_recipe_picker_dialog.drawio.svg` / `S04_recipe_edit.drawio.svg` / `S09_settings.drawio.svg`

**生成物 (再生成のみ)**

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`recipe.Recipe`、`suggestion.Result`、`settings.Values` の型が変わる

**migration**

- `00012_recipes_cook_minutes.sql`：`recipes` に `cook_minutes INTEGER NOT NULL DEFAULT 0` を足す。
  ほかの proposal が先に 00012 を使ったら、実装時に次の空き番号へ振り直す

**後続 proposal への引き継ぎ**

- 週の候補に時短の順位表を足すかは、セルの候補での使われ方を見てから決める
- `add-mcp-server` の `create_recipe` と `get_recipe` の入力と出力に `cookMinutes` が増える。MCP が先に入っていれば、ツールの説明文を直す
