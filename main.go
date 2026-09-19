package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	desktopkit "github.com/wanstu/wails-desktop-kit"
	kitui "github.com/wanstu/wails-desktop-kit/ui"
)

//go:embed all:frontend/src
var assets embed.FS

//go:embed frontend/src/assets/images/tray-icon.png
var trayIcon []byte

func main() {
	launch, err := desktopkit.ParseLaunchOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "codexpro-plus:", err)
		os.Exit(1)
	}

	app := NewApp()
	if err := runDesktop(app, launch); err != nil {
		fmt.Fprintln(os.Stderr, "codexpro-plus:", err)
		os.Exit(1)
	}
}

func runDesktop(app *App, launch desktopkit.LaunchOptions) error {
	appAssets, err := fs.Sub(assets, "frontend/src")
	if err != nil {
		return err
	}

	window := desktopkit.DefaultWindowConfig()
	window.Width = 1120
	window.Height = 820
	window.MinWidth = 820
	window.MinHeight = 620
	window.HidePolicy = desktopkit.HideSafe
	window.StartHiddenOnAutoStart = true
	window.Background = desktopkit.Color{R: 244, G: 247, B: 251, A: 1}

	return desktopkit.Run(desktopkit.Config{
		ID:                   desktopAppID(),
		Title:                appDisplayName(),
		Assets:               kitui.Mount(appAssets),
		Bind:                 []interface{}{app},
		Theme:                desktopkit.DefaultThemeConfig(),
		Launch:               launch,
		Window:               window,
		SingleInstance:       true,
		SecondInstancePolicy: desktopkit.SecondInstanceWakeManual,
		Tray: desktopkit.TrayConfig{
			Enabled:            true,
			Icon:               trayIcon,
			Tooltip:            appDisplayName(),
			AutoStart:          app.launchAtLogin,
			ShowLabel:          "打开主面板",
			HideLabel:          "隐藏主面板",
			LaunchAtLoginLabel: "开机启动 " + appDisplayName(),
			QuitLabel:          "退出 " + appDisplayName(),
		},
		Hooks: desktopkit.Hooks{
			Ready:    app.setController,
			Startup:  app.startup,
			Shutdown: app.shutdown,
		},
	})
}
