## Why

献立を組む作業のうち、「今週の在庫と最近作ったものを見て、どのレシピを置くか考える」部分は、利用者がいちばん面倒に感じているところである
([product.md §1](../../../docs/product.md) ペルソナ「1 週間の献立を勝手に考える、または、案を提示してほしい」)。
[design.md §1](../../../docs/design.md) は「全自動生成は AI 前提になるため採らない」としており、アプリ自身に AI を組み込む予定は無い。

一方で、利用者がすでに使っている AI エージェント (Claude Desktop など) は、MCP サーバーを通して外部のアプリを操作できる。
meal-cycle が MCP サーバーを持てば、アプリは AI を内蔵しないまま、利用者が自分で選んだエージェントに献立づくりを手伝わせられる。
Wails サービス (`internal/service/`) は UI から独立した API 層になっており、E2E サーバー (`cmd/e2e-server`) で別の入口から呼ぶ形もすでに動いている。

## What Changes

- MCP サーバーを、GUI とは別の実行ファイル `meal-cycle-mcp` として追加する
  - AI クライアントが子プロセスとして起動し、標準入出力でやり取りする。ネットワークの待ち受けはしない
  - GUI と同じ DB ファイル (OS 標準の場所) を開く。`-db` で別のファイルを指せる
  - DB が無いとき、スキーマが古いとき、このファイルより新しい版のときは、ツールを動かさずに「先に meal-cycle を起動する」などの理由を返す
- 参照のツールを公開する
  - レシピの一覧と 1 件の詳細、期間の献立、冷蔵庫の在庫、常備品、買い物リスト、献立候補 (3 軸と週の提案)
- 追加と状態の変更のツールを公開する
  - 献立へのレシピの割り当て、在庫の追加、常備品の追加と状態の変更、レシピの登録
- ツールの失敗は、エラーの種類を表すコードと英語の短い説明で返す
- GUI は、ウィンドウにフォーカスが戻ったときに表示中のデータを取り直す
  - エージェントが書き込んだ内容を、GUI を開き直さずに見られるようにする
- Windows ビルドで `meal-cycle-mcp.exe` も出力し、AI クライアントへの登録手順を `docs/mcp.md` に書く

### Non-goals

- **削除、書き換え、取り消しのツール**：献立の削除や空にする操作、在庫やレシピの編集と削除、作った印と在庫の消費、作り置きを広げる操作は公開しない。
  エージェントの誤操作で、利用者が気付かないうちにデータを失うのを避ける。必要になったら、取り消しの仕組みと一緒に後続の proposal で足す
- **設定と DB の操作**：S09 の値の変更、レシピの書き出しと取り込み、DB の書き出しと復元は公開しない
- **HTTP など、ネットワーク越しの MCP**：localhost でもポートを開くと、[design.md §2](../../../docs/design.md) の「ネットワーク通信は原則なし」から外れる
- **GUI 本体からの MCP サーバーの起動や設定画面**：登録は利用者が AI クライアント側の設定ファイルで行う
- **アプリへの AI の組み込み**：LLM を呼ぶのはエージェント側であり、meal-cycle は通信しない
- **MCP の resources と prompts**：tools だけを公開する

## Capabilities

### New Capabilities

- `agent-access`：AI エージェントが MCP 経由で献立、レシピ、在庫、常備品、買い物リスト、献立候補を参照し、追加と状態の変更を行う。
  GUI はエージェントの書き込みをフォーカスの復帰で表示に反映する

### Modified Capabilities

(なし)

## Impact

**新設ファイル**

- `cmd/mcp-server/main.go`：引数の解釈、DB を開いて版を確かめる、stdio で MCP サーバーを動かす
- `cmd/mcp-server/tools.go`：ツールの定義 (名前、説明、入力スキーマ) と、`internal/service` への転送
- `cmd/mcp-server/errors.go`：domain のエラーをツールのエラーコードに対応づける
- `cmd/mcp-server/tools_test.go`：in-memory の transport でツールを呼ぶテスト
- `internal/storage/sqlite/agent.go`：`OpenExisting` (ファイルの有無と版を確かめて開く。スキーマ更新はしない) と `ErrDatabaseMissing` / `ErrSchemaOutdated` / `ErrSchemaTooNew`
- `internal/storage/sqlite/agent_test.go`
- `frontend/e2e/agent-access.spec.ts`：フォーカスの復帰で GUI が取り直すことの E2E
- `docs/mcp.md`：ビルド、Claude Desktop などへの登録、公開するツールの一覧、注意点 (データがエージェントの LLM に送られること)

**更新ファイル**

- `go.mod` と `go.sum`：`github.com/modelcontextprotocol/go-sdk` を追加する
- `frontend/src/main.tsx`：TanStack Query のフォーカス検出を `window` の `focus` に合わせる
- `Taskfile.yml`：`build:mcp` を足し、`build:win` と `build:win:copy` で `meal-cycle-mcp.exe` も作ってコピーする
- `docs/design.md`：§1 想定シナリオと §6 アーキテクチャに MCP サーバーを足す。§2 の通信の行に MCP サーバーは通信しないことを添える
- `docs/product.md`：§1 ナラティブに、エージェントに献立を手伝わせる使い方を 1 段落足す
- `CLAUDE.md`：常用コマンドに `task build:mcp`、アーキテクチャに MCP の入口を足す
- `README.md`：MCP サーバーの節を足し、`docs/mcp.md` へリンクする

**生成物 (再生成のみ)**

- なし (Wails サービスのメソッドは変えないので `frontend/bindings/` は変わらない)

**migration**

- なし

**後続 proposal への引き継ぎ**

- 削除と書き換えのツールは、エージェントの操作を 1 回分まとめて取り消せる仕組み (操作の記録) と一緒に入れる
- エージェントが書き込んだ直後に GUI へ即時に知らせる仕組み (DB の変更の監視など) は、フォーカスの復帰で足りないと感じたら足す
- `add-settings-database-file` の復元を予約したまま MCP で書き込むと、次の起動で書き込みが消える。両方が入ったら、予約中は書き込みのツールを止めるかを決める
