# AI エージェントから meal-cycle を使う (MCP サーバー)

献立を考える手間のうち、在庫と最近作ったものを見比べてレシピを選ぶ部分は、利用者が普段使っている AI エージェントに任せられる。
meal-cycle 自体は AI を持たない。
代わりに、エージェントが meal-cycle のデータを読み書きするための MCP サーバー `meal-cycle-mcp` を同梱している。

`meal-cycle-mcp` は GUI とは別の実行ファイルである。
AI クライアント (Claude Desktop など) が子プロセスとして起動し、標準入出力だけでやり取りする。
ポートを開くことも、外部と通信することもない。

## ビルド

Windows 向けの配布物は、GUI と一緒に作られる。

```bash
task build:win WIN_DEST=/mnt/c/Users/<Windows のユーザー名>/Desktop/
```

`bin/meal-cycle.exe` と `bin/meal-cycle-mcp.exe` ができ、`WIN_DEST` を渡せば両方がそこへコピーされる。
いま開発している OS 向けに MCP サーバーだけを作るなら `task build:mcp` を使う (出力は `bin/meal-cycle-mcp`、Windows では `.exe` 付き)。

## AI クライアントへの登録

登録は利用者が AI クライアントの設定ファイルで行う。
meal-cycle の側に設定画面は無い。

Claude Desktop なら、`claude_desktop_config.json` (Windows では `%APPDATA%\Claude\claude_desktop_config.json`) に次を足して、Claude Desktop を再起動する。

```json
{
  "mcpServers": {
    "meal-cycle": {
      "command": "C:\\Users\\<ユーザー名>\\Desktop\\meal-cycle-mcp.exe"
    }
  }
}
```

データベースは GUI と同じ OS 標準の場所 (Windows なら `%LOCALAPPDATA%\meal-cycle\meal-cycle.db`) を開く。
別のファイルを使いたいときは `"args": ["-db", "D:\\path\\to\\meal-cycle.db"]` を足す。

## 公開しているツール

エージェントに開いているのは、参照と追加と常備品の状態の変更だけである。
日付は `YYYY-MM-DD`、時間帯は `breakfast` / `lunch` / `dinner` で渡す。

| ツール | できること |
|---|---|
| `list_recipes` | レシピの一覧 (ID、名前、人数、タグ) |
| `get_recipe` | レシピ 1 件の詳細 (材料、手順、メモ、所要時間 `cookMinutes`。0 は未入力) |
| `list_meal_plan` | 開始日から 1 / 2 / 4 週の献立 |
| `list_stock` | 冷蔵庫の在庫と期限の状態 |
| `list_staples` | 常備品と、ある / 少ない / ない の状態 |
| `get_shopping_list` | 開始日から指定した日数の買い物リスト。S09 で世帯人数を選んでいれば、1 回きりの献立の量は換算済み (`scaledServings`) |
| `list_suggestions` | 献立の候補 (使い切り、久しぶりに、時短) |
| `suggest_week_plans` | 週の空いたコマを埋める案 (書き込みはしない) |
| `assign_recipe` | 献立に 1 品足す (既存の品の後ろに並ぶ) |
| `add_stock_items` | 在庫をまとめて足す (1 件でも不正なら 1 件も足さない) |
| `add_staple` | 常備品を足す |
| `set_staple_state` | 常備品の状態を変える |
| `create_recipe` | レシピを登録する |

削除、編集、並べ替え、作った印、設定の変更、書き出しと取り込みは公開していない。
エージェントの誤操作で、利用者が気付かないうちにデータが消えるのを避けるためである。
エージェントが足した品や在庫が気に入らなければ、GUI のいつもの操作で消す。

ツールが失敗したときは、`{"code":"mealplan.invalid_date","message":"..."}` の形で、種類を表すコードと英語の説明を返す。

## 使うときの注意

**データはエージェントの LLM に送られる。**
meal-cycle-mcp 自体は通信しないが、エージェントが読んだレシピや在庫は、AI クライアントを通してその LLM (多くはクラウド) に渡る。
渡したデータの扱いは、AI クライアント側の規約に従う。
「完全ローカル」の前提を崩したくないなら、登録しなければよい。

**アプリを更新したら、先に GUI を一度起動する。**
スキーマの更新と更新前の控えは GUI の起動だけが行い、MCP サーバーはスキーマの版が一致するデータベースしか開かない。
版が合わないときは、ツールが `database.schema_outdated` (GUI を起動すれば直る) か `database.schema_too_new` (meal-cycle-mcp を新しくすれば直る) を返す。
データベースがまだ無いときの `database.missing` も、GUI を一度起動すれば直る。

**GUI への反映は、ウィンドウに戻ったときに起きる。**
エージェントが書き込んでも、開いている GUI の表示はすぐには変わらない。
GUI のウィンドウにフォーカスを戻すと、表示中の画面のデータを取り直す。

設定 (S09) の変更は、MCP サーバーを起動し直さなくても次のツール呼び出しから効く。
