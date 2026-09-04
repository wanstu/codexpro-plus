package main

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ManagerSettings struct {
	AutoStart    bool   `json:"auto_start"`
	CoreReady    bool   `json:"core_ready"`
	CorePath     string `json:"core_path"`
	CoreError    string `json:"core_error"`
	TrayError    string `json:"tray_error"`
	BuildProfile string `json:"build_profile"`
	AppName      string `json:"app_name"`
	ConfigPath   string `json:"config_path"`
}

type App struct {
	ctx       context.Context
	service   *WorkspaceService
	processes *ProcessManager
	tray      *TrayManager
	initErr   error
}

func NewApp() *App {
	store, err := NewDefaultConfigStore()
	app := &App{initErr: err}
	if err == nil {
		app.service = NewWorkspaceService(store)
		app.processes = NewProcessManager(app.service)
	}
	return app
}

func (a *App) attachTray(icon []byte) {
	if a == nil {
		return
	}
	a.tray = NewTrayManager(a, icon)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go watchSingleInstanceWake(ctx, a.showMainWindow)
	if a.processes != nil {
		go a.processes.StartConfiguredWorkspaces()
	}
}

func (a *App) domReady(ctx context.Context) {
	a.ctx = ctx
	if a.tray != nil {
		a.tray.Start()
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.processes != nil {
		_ = a.processes.StopAll()
	}
	if a.tray != nil {
		a.tray.Stop()
	}
}

func (a *App) GetConfig() (Config, error) {
	if err := a.ready(); err != nil {
		return Config{}, err
	}
	return a.service.GetConfig()
}

func (a *App) AddWorkspace(input WorkspaceInput) (Workspace, error) {
	if err := a.ready(); err != nil {
		return Workspace{}, err
	}
	return a.service.AddWorkspace(input)
}

func (a *App) UpdateWorkspace(id string, input WorkspaceInput) (Workspace, error) {
	if err := a.ready(); err != nil {
		return Workspace{}, err
	}
	if a.processes.IsRunning(id) {
		return Workspace{}, errors.New("该工作目录正在运行，请先停止服务再编辑")
	}
	return a.service.UpdateWorkspace(id, input)
}

func (a *App) DeleteWorkspace(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	if a.processes.IsRunning(id) {
		return errors.New("该工作目录正在运行，请先停止服务再删除")
	}
	return a.service.DeleteWorkspace(id)
}

func (a *App) UpdatePortRange(portRange PortRange) (Config, error) {
	if err := a.ready(); err != nil {
		return Config{}, err
	}
	return a.service.UpdatePortRange(portRange)
}

func (a *App) UpdateDomain(domain string) (Config, error) {
	if err := a.ready(); err != nil {
		return Config{}, err
	}
	return a.service.UpdateDomain(domain)
}

func (a *App) GetWorkspaceURL(id string) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.service.GetWorkspaceURL(id)
}

func (a *App) OpenWorkspaceURL(id string) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	if a.ctx == nil {
		return "", errors.New("应用窗口尚未就绪")
	}
	workspaceURL, err := a.service.GetWorkspaceURL(id)
	if err != nil {
		return "", err
	}
	runtime.BrowserOpenURL(a.ctx, workspaceURL)
	return workspaceURL, nil
}

func (a *App) StartWorkspace(id string) (Instance, error) {
	if err := a.ready(); err != nil {
		return Instance{}, err
	}
	return a.processes.Start(id)
}

func (a *App) StopWorkspace(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.processes.Stop(id)
}

func (a *App) GetInstances() ([]Instance, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.processes.ListInstances(), nil
}

func (a *App) GetRuntimeStates() ([]RuntimeState, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.processes.RuntimeStates(), nil
}

func (a *App) GetWorkspaceLog(id string) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.processes.ReadLog(id)
}

func (a *App) GetManagerSettings() (ManagerSettings, error) {
	if err := a.ready(); err != nil {
		return ManagerSettings{}, err
	}
	autoStart, err := managerAutoStartEnabled()
	if err != nil {
		return ManagerSettings{}, err
	}
	settings := ManagerSettings{
		AutoStart:    autoStart,
		BuildProfile: currentBuildProfile(),
		AppName:      appDisplayName(),
		ConfigPath:   a.service.store.Path(),
	}
	if path, err := a.processes.CorePath(); err != nil {
		settings.CoreError = err.Error()
	} else {
		settings.CoreReady = true
		settings.CorePath = path
	}
	if a.tray != nil {
		settings.TrayError = a.tray.LastError()
	}
	return settings, nil
}

func (a *App) SetManagerAutoStart(enabled bool) (ManagerSettings, error) {
	if err := a.ready(); err != nil {
		return ManagerSettings{}, err
	}
	if err := setManagerAutoStart(enabled); err != nil {
		return ManagerSettings{}, err
	}
	return a.GetManagerSettings()
}

func (a *App) showMainWindow() {
	if a == nil || a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
}

func (a *App) quitApplication() {
	if a == nil || a.ctx == nil {
		return
	}
	runtime.Quit(a.ctx)
}

func (a *App) ready() error {
	if a == nil {
		return errors.New("应用未初始化")
	}
	if a.initErr != nil {
		return a.initErr
	}
	if a.service == nil {
		return errors.New("配置服务未初始化")
	}
	if a.processes == nil {
		return errors.New("进程管理器未初始化")
	}
	return nil
}
