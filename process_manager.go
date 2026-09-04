package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const logTailBytes int64 = 64 * 1024

type Instance struct {
	WorkspaceID string `json:"workspace_id"`
	Alias       string `json:"alias"`
	Path        string `json:"path"`
	PID         int    `json:"pid"`
	Port        int    `json:"port"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at"`
	LogPath     string `json:"log_path"`
}

type RuntimeState struct {
	WorkspaceID string `json:"workspace_id"`
	Starting    bool   `json:"starting"`
	Running     bool   `json:"running"`
	PID         int    `json:"pid"`
	Port        int    `json:"port"`
	StartedAt   string `json:"started_at"`
	LogPath     string `json:"log_path"`
	LastError   string `json:"last_error"`
}

type managedInstance struct {
	Instance
	cmd      *exec.Cmd
	done     chan struct{}
	stopping bool
}

type ProcessManager struct {
	service          *WorkspaceService
	mu               sync.Mutex
	instances        map[string]*managedInstance
	starting         map[string]bool
	lastErrors       map[string]string
	corePathResolver func() (string, error)
	portAvailable    portAvailabilityFunc
}

func NewProcessManager(service *WorkspaceService) *ProcessManager {
	return &ProcessManager{
		service:          service,
		instances:        make(map[string]*managedInstance),
		starting:         make(map[string]bool),
		lastErrors:       make(map[string]string),
		corePathResolver: defaultCorePath,
		portAvailable:    systemPortAvailable,
	}
}

func (m *ProcessManager) Start(workspaceID string) (Instance, error) {
	if m == nil || m.service == nil {
		return Instance{}, errors.New("进程管理器未初始化")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return Instance{}, errors.New("工作目录 ID 不能为空")
	}

	m.mu.Lock()
	if current, ok := m.instances[workspaceID]; ok {
		instance := current.Instance
		m.mu.Unlock()
		return instance, nil
	}
	if m.starting[workspaceID] {
		m.mu.Unlock()
		return Instance{}, errors.New("该工作目录正在启动")
	}
	m.starting[workspaceID] = true
	delete(m.lastErrors, workspaceID)
	m.mu.Unlock()

	started := false
	defer func() {
		if started {
			return
		}
		m.mu.Lock()
		delete(m.starting, workspaceID)
		m.mu.Unlock()
	}()

	workspace, err := m.service.GetWorkspace(workspaceID)
	if err != nil {
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}
	if workspace.Token == "" {
		err := errors.New("工作目录 token 为空，请重新保存配置")
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}
	checker := m.portAvailable
	if checker == nil {
		checker = systemPortAvailable
	}
	if !checker(workspace.Port) {
		err := fmt.Errorf("端口 %d 当前已被其他进程占用", workspace.Port)
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}

	corePath, err := m.corePath()
	if err != nil {
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}
	logPath, err := m.workspaceLogPath(workspaceID)
	if err != nil {
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		err = fmt.Errorf("创建日志目录失败: %w", err)
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		err = fmt.Errorf("打开日志文件失败: %w", err)
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}
	_, _ = fmt.Fprintf(logFile, "\n[%s] manager: 启动 %s，root=%s，port=%d\n", time.Now().Format(time.RFC3339), workspace.Alias, workspace.Path, workspace.Port)

	cmd := exec.Command(corePath)
	cmd.Dir = workspace.Path
	inheritEnv := "0"
	if workspace.InheritEnv {
		inheritEnv = "1"
	}
	cmd.Env = mergeEnvironment(os.Environ(), map[string]string{
		"CODEXPRO_ROOT":          workspace.Path,
		"CODEXPRO_ALLOWED_ROOTS": workspace.Path,
		"CODEXPRO_HOST":          "127.0.0.1",
		"CODEXPRO_PORT":          strconv.Itoa(workspace.Port),
		"CODEXPRO_MODE":          "agent",
		"CODEXPRO_WRITE_MODE":    workspace.WriteMode,
		"CODEXPRO_BASH_MODE":     workspace.BashMode,
		"CODEXPRO_TOOL_MODE":     workspace.ToolMode,
		"CODEXPRO_INHERIT_ENV":   inheritEnv,
		"CODEXPRO_HTTP_TOKEN":    workspace.Token,
	})
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	prepareChildCommand(cmd)

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		err = fmt.Errorf("启动 codexpro-core.exe 失败: %w", err)
		m.recordError(workspaceID, err.Error())
		return Instance{}, err
	}
	_ = logFile.Close()

	instance := Instance{
		WorkspaceID: workspace.ID,
		Alias:       workspace.Alias,
		Path:        workspace.Path,
		PID:         cmd.Process.Pid,
		Port:        workspace.Port,
		Status:      "running",
		StartedAt:   time.Now().Format(time.RFC3339),
		LogPath:     logPath,
	}
	managed := &managedInstance{
		Instance: instance,
		cmd:      cmd,
		done:     make(chan struct{}),
	}

	m.mu.Lock()
	delete(m.starting, workspaceID)
	delete(m.lastErrors, workspaceID)
	m.instances[workspaceID] = managed
	m.mu.Unlock()
	started = true

	_ = m.appendLog(workspaceID, fmt.Sprintf("manager: 进程已创建 PID=%d，等待端口 %d 就绪", instance.PID, instance.Port))
	go m.waitForExit(workspaceID, managed)

	if err := waitForPortListening(instance.Port, managed.done, 6*time.Second); err != nil {
		state := m.runtimeState(workspaceID)
		if state.LastError != "" {
			return Instance{}, errors.New(state.LastError)
		}
		_ = m.Stop(workspaceID)
		message := fmt.Sprintf("服务启动失败: %v", err)
		m.recordError(workspaceID, message)
		return Instance{}, errors.New(message)
	}

	_ = m.appendLog(workspaceID, fmt.Sprintf("manager: 服务已就绪 PID=%d，port=%d", instance.PID, instance.Port))
	return instance, nil
}

func (m *ProcessManager) Stop(workspaceID string) error {
	if m == nil {
		return errors.New("进程管理器未初始化")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return errors.New("工作目录 ID 不能为空")
	}

	m.mu.Lock()
	managed, ok := m.instances[workspaceID]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	managed.stopping = true
	pid := managed.PID
	port := managed.Port
	done := managed.done
	m.mu.Unlock()

	_ = m.appendLog(workspaceID, fmt.Sprintf("manager: 请求停止 PID=%d", pid))
	killErr := killProcessTree(pid)
	if killErr != nil {
		select {
		case <-done:
			return nil
		default:
		}
		msg := fmt.Sprintf("停止进程树失败: %v", killErr)
		m.recordError(workspaceID, msg)
		return errors.New(msg)
	}

	select {
	case <-done:
		checker := m.portAvailable
		if checker == nil {
			checker = systemPortAvailable
		}
		if err := waitForPortReleased(port, checker, 3*time.Second); err != nil {
			msg := fmt.Sprintf("进程已退出，但端口 %d 未及时释放: %v", port, err)
			m.recordError(workspaceID, msg)
			return errors.New(msg)
		}
		_ = m.appendLog(workspaceID, fmt.Sprintf("manager: 端口 %d 已释放", port))
		return nil
	case <-time.After(8 * time.Second):
		msg := fmt.Sprintf("等待 PID %d 退出超时", pid)
		m.recordError(workspaceID, msg)
		return errors.New(msg)
	}
}

func (m *ProcessManager) StopAll() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	ids := make([]string, 0, len(m.instances))
	for id := range m.instances {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	var messages []string
	for _, id := range ids {
		if err := m.Stop(id); err != nil {
			messages = append(messages, err.Error())
		}
	}
	if len(messages) > 0 {
		return errors.New(strings.Join(messages, "; "))
	}
	return nil
}

func (m *ProcessManager) IsRunning(workspaceID string) bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	_, running := m.instances[workspaceID]
	return running || m.starting[workspaceID]
}

func (m *ProcessManager) ListInstances() []Instance {
	if m == nil {
		return []Instance{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]Instance, 0, len(m.instances))
	for _, managed := range m.instances {
		result = append(result, managed.Instance)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Alias) < strings.ToLower(result[j].Alias)
	})
	return result
}

func (m *ProcessManager) RuntimeStates() []RuntimeState {
	if m == nil {
		return []RuntimeState{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	seen := make(map[string]bool, len(m.instances)+len(m.starting)+len(m.lastErrors))
	result := make([]RuntimeState, 0, len(m.instances)+len(m.starting)+len(m.lastErrors))
	for id, managed := range m.instances {
		seen[id] = true
		result = append(result, RuntimeState{
			WorkspaceID: id,
			Running:     true,
			PID:         managed.PID,
			Port:        managed.Port,
			StartedAt:   managed.StartedAt,
			LogPath:     managed.LogPath,
			LastError:   m.lastErrors[id],
		})
	}
	for id, starting := range m.starting {
		if seen[id] || !starting {
			continue
		}
		seen[id] = true
		result = append(result, RuntimeState{WorkspaceID: id, Starting: true})
	}
	for id, lastError := range m.lastErrors {
		if seen[id] {
			continue
		}
		result = append(result, RuntimeState{WorkspaceID: id, LastError: lastError})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].WorkspaceID < result[j].WorkspaceID })
	return result
}

func (m *ProcessManager) ReadLog(workspaceID string) (string, error) {
	path, err := m.workspaceLogPath(workspaceID)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("读取日志失败: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("读取日志信息失败: %w", err)
	}
	start := info.Size() - logTailBytes
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return "", fmt.Errorf("定位日志失败: %w", err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取日志失败: %w", err)
	}
	return string(data), nil
}

func (m *ProcessManager) StartConfiguredWorkspaces() {
	if m == nil || m.service == nil {
		return
	}
	cfg, err := m.service.GetConfig()
	if err != nil {
		return
	}
	for _, workspace := range cfg.Workspaces {
		if !workspace.AutoStart {
			continue
		}
		if _, err := m.Start(workspace.ID); err != nil {
			// Start 已记录 last error 和日志。单项失败不能阻断其他 Workspace。
			continue
		}
	}
}

func (m *ProcessManager) CorePath() (string, error) {
	return m.corePath()
}

func (m *ProcessManager) corePath() (string, error) {
	resolver := m.corePathResolver
	if resolver == nil {
		resolver = defaultCorePath
	}
	return resolver()
}

func (m *ProcessManager) waitForExit(workspaceID string, managed *managedInstance) {
	err := managed.cmd.Wait()

	m.mu.Lock()
	current, ok := m.instances[workspaceID]
	stopping := ok && current == managed && current.stopping
	if ok && current == managed {
		delete(m.instances, workspaceID)
		if stopping {
			delete(m.lastErrors, workspaceID)
		} else if err != nil {
			m.lastErrors[workspaceID] = fmt.Sprintf("服务异常退出: %v", err)
		} else {
			m.lastErrors[workspaceID] = "服务已退出"
		}
	}
	m.mu.Unlock()

	if stopping {
		_ = m.appendLog(workspaceID, fmt.Sprintf("manager: PID=%d 已停止", managed.PID))
	} else if err != nil {
		_ = m.appendLog(workspaceID, fmt.Sprintf("manager: PID=%d 异常退出: %v", managed.PID, err))
	} else {
		_ = m.appendLog(workspaceID, fmt.Sprintf("manager: PID=%d 已退出", managed.PID))
	}
	close(managed.done)
}

func (m *ProcessManager) runtimeState(workspaceID string) RuntimeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if managed, ok := m.instances[workspaceID]; ok {
		return RuntimeState{
			WorkspaceID: workspaceID,
			Running:     true,
			PID:         managed.PID,
			Port:        managed.Port,
			StartedAt:   managed.StartedAt,
			LogPath:     managed.LogPath,
			LastError:   m.lastErrors[workspaceID],
		}
	}
	return RuntimeState{WorkspaceID: workspaceID, LastError: m.lastErrors[workspaceID]}
}

func (m *ProcessManager) recordError(workspaceID, message string) {
	m.mu.Lock()
	m.lastErrors[workspaceID] = message
	m.mu.Unlock()
	_ = m.appendLog(workspaceID, "manager: "+message)
}

func (m *ProcessManager) workspaceLogPath(workspaceID string) (string, error) {
	if m == nil || m.service == nil || m.service.store == nil {
		return "", errors.New("配置服务未初始化")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return "", errors.New("工作目录 ID 不能为空")
	}
	return filepath.Join(filepath.Dir(m.service.store.Path()), "logs", workspaceID+".log"), nil
}

func (m *ProcessManager) appendLog(workspaceID, message string) error {
	path, err := m.workspaceLogPath(workspaceID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "[%s] %s\n", time.Now().Format(time.RFC3339), message)
	return err
}

func waitForPortReleased(port int, checker portAvailabilityFunc, timeout time.Duration) error {
	if checker == nil {
		checker = systemPortAvailable
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		if checker(port) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待 %s 后仍不可用", timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func waitForPortListening(port int, done <-chan struct{}, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-done:
			return errors.New("core 在端口就绪前已经退出")
		default:
		}

		connection, err := net.DialTimeout("tcp", address, 150*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("端口 %d 在 %s 内没有开始监听", port, timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func defaultCorePath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取 codexpro-plus.exe 路径: %w", err)
	}
	path := filepath.Join(filepath.Dir(executable), "codexpro-core.exe")
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("未找到 codexpro-core.exe，请将它放到 codexpro-plus.exe 同目录：%s", path)
		}
		return "", fmt.Errorf("无法访问 codexpro-core.exe: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("codexpro-core.exe 路径不是文件：%s", path)
	}
	return path, nil
}

func mergeEnvironment(base []string, overrides map[string]string) []string {
	keys := make(map[string]bool, len(overrides))
	for key := range overrides {
		keys[strings.ToLower(key)] = true
	}

	result := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		index := strings.IndexByte(entry, '=')
		if index <= 0 {
			result = append(result, entry)
			continue
		}
		if keys[strings.ToLower(entry[:index])] {
			continue
		}
		result = append(result, entry)
	}

	overrideKeys := make([]string, 0, len(overrides))
	for key := range overrides {
		overrideKeys = append(overrideKeys, key)
	}
	sort.Strings(overrideKeys)
	for _, key := range overrideKeys {
		result = append(result, key+"="+overrides[key])
	}
	return result
}
