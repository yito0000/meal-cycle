// cmd/mcp-server は AI クライアントが子プロセスとして起動する MCP サーバー (meal-cycle-mcp)。
// 標準入出力だけでやり取りし、ネットワークの待ち受けはしない (add-mcp-server design.md D1)。
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pressly/goose/v3"

	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

func main() {
	dbPath := flag.String("db", "", "SQLite file path (default: the same database as the meal-cycle app)")
	flag.Parse()

	// 標準出力は MCP のメッセージ専用なので、ログが混ざるとクライアントが読めなくなる。
	log.SetOutput(os.Stderr)
	log.SetPrefix("meal-cycle-mcp: ")
	goose.SetLogger(log.New(os.Stderr, "meal-cycle-mcp: goose: ", log.LstdFlags))

	path := *dbPath
	if path == "" {
		p, err := sqlite.DefaultPath()
		if err != nil {
			log.Fatalf("resolve db path: %v", err)
		}
		path = p
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := newServer(path).Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		log.Fatalf("serve: %v", err)
	}
}
