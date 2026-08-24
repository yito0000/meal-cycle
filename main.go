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
			application.NewService(NewMealPlanService()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Size the initial window from the primary screen's work area (in DIP,
	// so it is already correct for the OS's display scale) rather than a
	// fixed pixel count. Take ~85% of the work area so the taskbar and
	// window chrome do not clip, and cap the width so the app never
	// launches wider than S01's calendar actually needs (≈2400 DIP).
	//
	// Falls back to a conservative 1440 × 900 (fits 1920 × 1080 with
	// margin, still usable on 1366 × 768) when screen detection is
	// unavailable — e.g. a headless CI run.
	//
	// Zoom stays at 1.5 so text/UI stay readable on high-DPI monitors;
	// CSS font-size is deliberately untouched so dev/preview builds
	// render identically.
	winWidth, winHeight := 1440, 900
	if primary := app.Screen.GetPrimary(); primary != nil {
		area := primary.WorkArea
		w := int(float64(area.Width) * 0.85)
		h := int(float64(area.Height) * 0.85)
		if w > 2400 {
			w = 2400
		}
		if h > 1500 {
			h = 1500
		}
		if w >= 900 && h >= 560 {
			winWidth, winHeight = w, h
		}
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "meal-cycle",
		Width:     winWidth,
		Height:    winHeight,
		MinWidth:  900,
		MinHeight: 560,
		Zoom:      1.5,
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
