// cmd/seed は動作確認用のサンプルレシピを DB に投入する開発用ツール。
// 本番配布物には含めない (task seed からのみ呼ぶ想定)。
//
// RecipeService 経由で Create するので、通常の入力と同じ validation を通る。
// 既に同名レシピが存在すれば黙ってスキップするので複数回実行しても安全。
package main

import (
	"context"
	"flag"
	"log"

	"github.com/yito0000/meal-cycle/internal/sampledata"
	"github.com/yito0000/meal-cycle/internal/service"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

func main() {
	dbPath := flag.String("db", "", "SQLite ファイルパス (未指定なら OS 既定の場所)")
	flag.Parse()

	path := *dbPath
	if path == "" {
		p, err := sqlite.DefaultPath()
		if err != nil {
			log.Fatalf("resolve db path: %v", err)
		}
		path = p
	}
	log.Printf("seed: database at %s", path)

	db, err := sqlite.Open(path)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(db); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	ctx := context.Background()
	svc := service.NewRecipeService(db, service.NewSettingsService(db), nil)

	existing, err := svc.List(ctx)
	if err != nil {
		log.Fatalf("list existing recipes: %v", err)
	}
	existingNames := make(map[string]bool, len(existing))
	for _, r := range existing {
		existingNames[r.Name] = true
	}

	inserted, skipped := 0, 0
	for _, r := range sampledata.Recipes() {
		if existingNames[r.Name] {
			log.Printf("seed: skip %q (already exists)", r.Name)
			skipped++
			continue
		}
		if _, err := svc.Create(ctx, r); err != nil {
			log.Fatalf("create %q: %v", r.Name, err)
		}
		log.Printf("seed: inserted %q", r.Name)
		inserted++
	}
	log.Printf("seed: done (inserted=%d, skipped=%d)", inserted, skipped)
}
