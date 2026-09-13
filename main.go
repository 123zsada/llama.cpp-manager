package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"llama-manager/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	store, err := config.NewStore()
	if err != nil {
		showFatalError("llama.cpp Manager 启动失败", err.Error())
		os.Exit(1)
	}

	app := NewApp(store)

	err = wails.Run(&options.App{
		Title:     "llama.cpp Manager",
		Width:     1280,
		Height:    820,
		MinWidth:  1024,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
