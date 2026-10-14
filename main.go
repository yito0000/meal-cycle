package main

import (
	"database/sql"
	"embed"
	"errors"
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
	db, startup := openDatabase()
	if db != nil {
		defer db.Close()
	}

	// 配布用ビルドにはコンソールが無いので、起動に失敗しても終了せずエラー画面を出す (design.md D2)。
	// DB が無いとほかのサービスは作れないため HealthService だけを登録する。
	services := []application.Service{
		application.NewService(service.NewHealthService(db, startup)),
	}
	if startup.Stage == "" {
		services = append(services,
			application.NewService(service.NewMealPlanService(sqlite.NewMealPlanRepo(db))),
			application.NewService(service.NewRecipeService(db)),
			application.NewService(service.NewFridgeService(db)),
			application.NewService(service.NewShoppingService(db)),
		)
	}

	app := application.New(application.Options{
		Name:        "meal-cycle",
		Description: "献立・冷蔵庫・レシピを一元管理するデスクトップアプリ",
		Services:    services,
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
	// Zoom は 1.5 のまま。高 DPI モニタで UI を読みやすくするため WebView zoom で拡大し、
	// dev / preview / packaged で見た目を揃える。CSS 側の基準は index.css の html { font-size: 18px }。
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

func openDatabase() (*sql.DB, service.StartupState) {
	dbPath, err := sqlite.DefaultPath()
	if err != nil {
		log.Printf("meal-cycle: %v", err)
		return nil, service.StartupState{Stage: string(sqlite.StageOpen), Detail: err.Error()}
	}
	log.Printf("meal-cycle: database at %s", dbPath)

	state := service.StartupState{DBPath: dbPath, BackupDir: sqlite.BackupDir(dbPath)}
	db, err := sqlite.OpenAndMigrate(dbPath)
	if err != nil {
		log.Printf("meal-cycle: startup failed: %v", err)
		state.Stage = string(sqlite.StageOpen)
		var se *sqlite.StartupError
		if errors.As(err, &se) {
			state.Stage = string(se.Stage)
			err = se.Err
		}
		state.Detail = err.Error()
		return nil, state
	}
	return db, state
}
