// cmd/e2e-server は Playwright から駆動するための E2E 専用サーバ。
// frontend/dist を配信しつつ、Wails IPC の代わりに POST /api/{Service}/{Method}
// でサービスを呼べるようにする。本番配布物には含めない。
//
// webview (WebKit2GTK) は外部ドライバから掴めないため、IPC 層だけを HTTP に
// 差し替え、React / サービス / SQLite は本物のまま通す (design.md D1)。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yito0000/meal-cycle/internal/sampledata"
	"github.com/yito0000/meal-cycle/internal/service"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// fixture がこの行を待って baseURL を確定する。文言を変えるときは
// frontend/e2e/fixtures.ts も一緒に直すこと。
const listeningPrefix = "E2E_SERVER_LISTENING "

func main() {
	dbPath := flag.String("db", "", "SQLite ファイルパス (必須)")
	port := flag.Int("port", 0, "待受ポート (0 なら OS 任せ)")
	dist := flag.String("dist", "frontend/dist", "配信する frontend のビルド成果物ディレクトリ")
	seed := flag.Bool("seed", false, "起動時にサンプルレシピを投入する")
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("-db は必須")
	}

	db, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(db); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	recipeSvc := service.NewRecipeService(db)
	if *seed {
		if err := seedRecipes(recipeSvc); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle(apiPrefix, dispatchHandler(newRegistry(db, recipeSvc)))
	mux.Handle("/", spaHandler(*dist))

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	fmt.Printf("%shttp://%s\n", listeningPrefix, ln.Addr().String())
	os.Stdout.Sync()

	if err := http.Serve(ln, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// newRegistry は main.go の application.Options.Services と同じ 3 サービスを返す。
// 片方だけ増えると E2E が本番と食い違うので、dispatch_test.go で件数を突き合わせる。
func newRegistry(db *sql.DB, recipeSvc *service.RecipeService) serviceRegistry {
	return serviceRegistry{
		"HealthService":   service.NewHealthService(db),
		"MealPlanService": service.NewMealPlanService(),
		"RecipeService":   recipeSvc,
	}
}

func seedRecipes(svc *service.RecipeService) error {
	ctx := context.Background()
	for _, r := range sampledata.Recipes() {
		if _, err := svc.Create(ctx, r); err != nil {
			return fmt.Errorf("create %q: %w", r.Name, err)
		}
	}
	return nil
}

// spaHandler は dist を配信し、実ファイルの無いパスは index.html に落とす。
// ルータが memory history なので通常は / しか要求されないが、Playwright が
// 直接 /recipes/new を叩いたときに 404 にしないため。
func spaHandler(dist string) http.Handler {
	fileServer := http.FileServer(http.Dir(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if clean != "." && !strings.HasPrefix(clean, "..") {
			if st, err := os.Stat(filepath.Join(dist, clean)); err == nil && !st.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, filepath.Join(dist, "index.html"))
	})
}
