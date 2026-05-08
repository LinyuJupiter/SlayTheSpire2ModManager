package main

import (
	"embed"

	"ModManager/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/sts2-heybox-support.zip
var heyboxSupportZip []byte

func main() {
	application := app.New(heyboxSupportZip)

	err := wails.Run(&options.App{
		Title:  "杀戮尖塔2 Mod 管理器",
		Width:  1580,
		Height: 980,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        application.Startup,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			IsZoomControlEnabled: false,
			DisablePinchZoom:     true,
		},
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
