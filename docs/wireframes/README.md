# ワイヤーフレーム

meal-cycle の各画面のワイヤーフレーム。ここに置いた drawio ファイルが**ワイヤーの正**。
言葉での相互作用ルール（クリックすると何が起きるか等）は [../product.md](../product.md) #4 側に書く。

## ファイル形式

**`.drawio.svg` に統一する。**

- 拡張子は `.drawio.svg`（drawio 独自の「編集情報埋め込み SVG」形式）
- ダブルクリックで drawio が開いて再編集できる
- 同時に SVG としてブラウザ / GitHub / Markdown プレビューでそのまま表示される
- Git 上ではテキストとして扱われ、差分もある程度読める（レイアウト大改造は読みにくいが、小さな修正なら把握可能）

`.drawio`（XML のみ、画像として表示できない）や `.png`（編集不可）は使わない。

## ファイル命名

`S<番号>_<スネークケースの短い名前>.drawio.svg`

例:

- `S00_shell.drawio.svg` — 共通シェル（サイドバー + メインエリア）
- `S01_meal_calendar.drawio.svg`
- `S01a_recipe_picker_dialog.drawio.svg` — レシピ選択ダイアログ（**候補セクション付き / MVP 後拡張**、[../product.md](../product.md) #5）
- `S02_recipe_list.drawio.svg`
- `S03_recipe_detail.drawio.svg`
- `S04_recipe_edit.drawio.svg`
- `S05_fridge.drawio.svg`
- `S06_fridge_item_inline.drawio.svg`
- `S07_shopping_list.drawio.svg`

サブ画面（ダイアログやポップオーバー等、S01 に紐づく副次的なビュー）は `S01a`, `S01b`, ... のように英小文字サフィックスで区別する。

番号は [../product.md](../product.md) #2「画面一覧」の ID と一致させる。

## 編集方法

いずれの方法でもファイル内容は互換。好きなものを選ぶ。

- **VS Code / Cursor / IntelliJ**: 「Draw.io Integration」拡張を入れ、`.drawio.svg` を開くと GUI エディタが立ち上がる
- **ブラウザ**: <https://app.diagrams.net/> にファイルをドラッグ&ドロップ。編集後 `File → Save` で SVG として保存
- **デスクトップ版**: <https://github.com/jgraph/drawio-desktop/releases>

## ワイヤーの粒度・見た目のルール

「どこに何が置かれているか」が伝わればよい。ピクセルパーフェクトは目指さない。

- **色は使いすぎない**: 基本は白 / グレー / 黒。「押せる要素」「アラート状態」など意味のある区別だけ色を使う
- **shadcn/ui の実物と一致させようとしない**: あくまで論理レイアウトのスケッチ。細かい角丸やパディングは実装時に決める
- **文字は日本語で OK**: 実装時に i18n キーに置き換える前提なので、ワイヤー段階では自然な日本語で書く
- **注釈は drawio 側でも書いてよい**（吹き出し / メモ図形）が、遷移ルールなど文章化に向く情報は product.md に寄せる

## アイコン: Lucide

ワイヤー内のアイコンは **[Lucide](https://lucide.dev/)** に統一する。

- ライセンスは ISC（表記義務なし、商用・OSS 公開ともに完全に問題なし）
- 実装時は `lucide-react` パッケージをそのまま使える（shadcn/ui のデフォルト）
- 既に使っているアイコン: `calendar` / `book-open` / `refrigerator` / `shopping-cart` / `settings` / `search` / `chevron-left` / `chevron-right` / `plus` / `x`
- 新しく必要になったら <https://lucide.dev/icons/> でアイコン名を確認して追加する

⋯（U+22EF）や ☐ ☑（U+2610 / U+2611）は絵文字ではなく Unicode 幾何文字なのでそのまま使ってよい。実装段階で `more-horizontal` や `square` / `check-square` に差し替える予定。

## 初期生成スクリプトについて

これらの `.drawio.svg` は最初 `scratchpad/gen_wireframes.py` から一括生成した。以降は drawio エディタで直接編集する運用（スクリプトは残していない）。

新しい画面を追加する場合は drawio で新規作成し、既存の見た目を参考にする（配色トークン: 通常 = 白 + `#333`、プライマリボタン = `#dbeafe` + `#3b82f6`、期限アラート = 赤 `#ef4444` / 橙 `#f59e0b` / 緑 `#10b981`）。

### スクリプト生成したもの

以下は初期生成後にスクリプトから追加 / 再生成したもの。drawio 埋め込み XML と可視 SVG の両方を書き出しており、他ファイル同様に drawio エディタでセル単位の編集ができる。

- `S01a_recipe_picker_dialog.drawio.svg` — 新規生成
- `S01_meal_calendar.drawio.svg` の下部 CTA 「✨ 翌週の候補を出す（MVP 後）」 — 既存ファイルへの追記

## Markdown からの参照

product.md からは相対リンクで参照する。GitHub / VS Code プレビュー / drawio 拡張が入ったエディタ、いずれでも SVG が表示される。

```markdown
![S01 献立カレンダー](wireframes/S01_meal_calendar.drawio.svg)
```

または、単にリンクだけで済ませる場合：

```markdown
ワイヤー: [wireframes/S01_meal_calendar.drawio.svg](wireframes/S01_meal_calendar.drawio.svg)
```

現状 product.md #4 は後者を採用している（画面内容は文章で読める形になっているため、常時プレビューする必要が薄い）。

## ライセンスメモ

- drawio 本体は Apache License 2.0
- drawio 同梱のアイコン / シェイプは CC BY 4.0
- 本プロジェクトのワイヤーで drawio 標準アイコンを多用する場合、リポジトリ公開時に `NOTICE` に一行入れておくと安全（例: `Wireframes in docs/wireframes/ use icons from draw.io, licensed under CC BY 4.0.`）
