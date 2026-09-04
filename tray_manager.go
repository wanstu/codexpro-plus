package main

import (
	"fmt"
	"sync"

	"github.com/gogpu/systray"
)

type TrayManager struct {
	app *App

	icon []byte

	mu        sync.Mutex
	started   bool
	stopping  bool
	stopFunc  func()
	lastError string
}

func NewTrayManager(app *App, icon []byte) *TrayManager {
	return &TrayManager{app: app, icon: icon}
}

func (t *TrayManager) Start() {
	if t == nil || t.app == nil {
		return
	}

	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return
	}
	t.started = true
	t.stopping = false
	t.lastError = ""
	t.mu.Unlock()

	go func() {
		tray := systray.New()
		menu := systray.NewMenu()
		menu.Add("打开主面板", func() {
			t.app.showMainWindow()
		})
		menu.AddSeparator()
		menu.Add("退出", func() {
			t.app.quitApplication()
		})

		tray.SetIcon(t.icon).
			SetTooltip(appDisplayName()).
			SetMenu(menu)
		tray.OnClick(func() {
			t.app.showMainWindow()
		})
		tray.Show()

		t.mu.Lock()
		if t.stopping {
			t.mu.Unlock()
			tray.Remove()
			return
		}
		t.stopFunc = func() { tray.Remove() }
		t.mu.Unlock()

		if err := tray.Run(); err != nil {
			t.mu.Lock()
			t.lastError = fmt.Sprintf("系统托盘运行失败: %v", err)
			t.mu.Unlock()
		}

		t.mu.Lock()
		t.started = false
		t.stopFunc = nil
		t.mu.Unlock()
	}()
}

func (t *TrayManager) Stop() {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.stopping = true
	stop := t.stopFunc
	t.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func (t *TrayManager) LastError() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastError
}
