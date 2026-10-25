package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Собранный фронтенд (Svelte) встраивается прямо в exe.
// Папку frontend/dist создает `wails build` / `wails dev` (npm run build).
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "golm",
		Width:     960,
		Height:    720,
		MinWidth:  720,
		MinHeight: 520,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 248, G: 249, B: 251, A: 1},
		OnStartup:        app.startup,
		// Методы App становятся доступны во фронтенде как JS-функции
		// (см. frontend/wailsjs/go/main/App.js — генерируется Wails).
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
