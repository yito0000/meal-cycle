package main

import (
	"embed"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/yito0000/meal-cycle/internal/service"
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
			application.NewService(service.NewHealthService(db)),
			application.NewService(service.NewMealPlanService(sqlite.NewMealPlanRepo(db))),
			application.NewService(service.NewRecipeService(db)),
			application.NewService(service.NewFridgeService(db)),
			application.NewService(service.NewShoppingService(db)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 初期ウィンドウサイズは primary screen の作業領域 (DIP) の約 85% を採用する。
	// タスクバーやウィンドウ枠のクリップを避けつつ、S01 カレンダーが必要とする
	// 範囲 (≒ 2400 DIP) を上限にキャップ。
	//
	// ウィンドウ生成を Common.ApplicationStarted まで遅延するのは、
	// app.Screen.GetPrimary() が app.Run() の中でプラットフォーム層が screen list を
	// キャッシュするまで nil を返すため (Linux は Linux.ApplicationStartup、Windows は
	// newPlatformApp、macOS は初回参照時)。Run 前に呼ぶと必ず fallback サイズになる。
	//
	// screen 検出が使えないケース (ヘッドレス CI 等) は 1440 × 900 に fallback する
	// (1920 × 1080 に余裕、1366 × 768 でも操作可)。
	//
	// Zoom は 1.5 のまま。高 DPI モニタで UI を読みやすくするため、CSS の font-size は
	// 意図的に触らず、dev / preview / packaged で見た目を揃える。
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		// GetPrimary() を短くポーリングする。Wails v3 beta.12 は application-event を
		// 並列 goroutine で dispatch するため、Linux では本ハンドラが
		// processAndCacheScreens リスナー (Linux.ApplicationStartup) とレースする。
		// WSLg ではさらに GDK が最初の activate tick で 0 monitor を返すことがある。
		// 数百 ms の待機で両方を吸収でき、起動体感は変わらない。
		primary := app.Screen.GetPrimary()
		deadline := time.Now().Add(500 * time.Millisecond)
		for primary == nil && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
			primary = app.Screen.GetPrimary()
		}

		winWidth, winHeight := 1440, 900
		if primary != nil {
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
			log.Printf("meal-cycle: primary work area %dx%d → window %dx%d",
				area.Width, area.Height, winWidth, winHeight)
		} else {
			log.Printf("meal-cycle: primary screen unavailable, using fallback %dx%d",
				winWidth, winHeight)
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
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
