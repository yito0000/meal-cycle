package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dbPath, err := sqlite.DefaultPath()
	if err != nil {
		log.Fatalf("resolve db path: %v", err)
	}
	log.Printf("meal-cycle: database at %s", dbPath)

	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(db); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	app := application.New(application.Options{
		Name:        "meal-cycle",
		Description: "献立・冷蔵庫・レシピを一元管理するデスクトップアプリ",
		Services: []application.Service{
			application.NewService(NewHealthService(db)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "meal-cycle",
		Width:  1000,
		Height: 618,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
