package main

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/src
var assets embed.FS

//go:embed frontend/src/assets/images/tray-icon.png
var trayIcon []byte

func main() {
	releaseInstance, primary, err := acquireSingleInstance()
	if err != nil {
		println("Error:", err.Error())
		return
	}
	if !primary {
		if !launchedFromAutoStart() {
			_ = requestExistingInstanceWindow()
		}
		return
	}
	defer releaseInstance()
	if err := prepareSingleInstanceWake(); err != nil {
		println("Warning:", err.Error())
	}

	app := NewApp()
	app.attachTray(trayIcon)

	err = wails.Run(&options.App{
		Title:             "CodexPro+",
		Width:             1120,
		Height:            820,
		MinWidth:          820,
		MinHeight:         620,
		StartHidden:       launchedFromAutoStart(),
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 244, G: 247, B: 251, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func launchedFromAutoStart() bool {
	for _, arg := range os.Args[1:] {
		if strings.EqualFold(strings.TrimSpace(arg), "--autostart") {
			return true
		}
	}
	return false
}
