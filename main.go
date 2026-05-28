package main

import (
	"embed"
	"log"

	"clientsys/internal/store"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	path, err := store.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	database, err := store.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	app := NewApp(database)
	err = wails.Run(&options.App{
		Title:            "ClientSys",
		Width:            1440,
		Height:           900,
		MinWidth:         1120,
		MinHeight:        700,
		BackgroundColour: &options.RGBA{R: 244, G: 247, B: 252, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		Mac:              &mac.Options{TitleBar: mac.TitleBarHiddenInset()},
		Windows:          &windows.Options{WebviewIsTransparent: false, WindowIsTranslucent: false},
	})
	if err != nil {
		log.Fatal(err)
	}
}
