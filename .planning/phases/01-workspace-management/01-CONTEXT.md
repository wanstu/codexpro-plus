# Phase 1 Context — Workspace 管理与持久化

## Phase Goal

完成 CodexPro+ 的第一条闭环：用户能在中文 UI 中添加、编辑、删除 Workspace，配置可靠写入 `~/.config/codexpro-plus/config.json`，并为每个 Workspace 分配独立端口；旧 `codexprov4` 配置由后续 rename migration 兼容迁移。

## Requirements

- WORK-01: 添加工作目录（别名 + 路径）
- WORK-02: 编辑工作目录
- WORK-03: 删除工作目录
- WORK-04: 独立端口；默认范围自动分配，也可手动指定
- WORK-05: 配置持久化
- UI-04: 空列表有引导文案

## Locked Decisions

这些来自 `.planning/PROJECT.md`，Phase 1 不重新讨论：

1. 技术栈：Go + Wails，Windows only。
2. 不复用 `codexprov3` 代码。
3. `Workspace` 是持久化配置；未来的 `Instance` 是运行时状态。两者必须严格分离。
4. 默认端口范围 8800-8899，范围可配置。
5. 单个 Workspace 可手动指定端口。
6. 运行时临时切换目录、CLI 都不做。
7. 配置路径固定为 `~/.config/codexpro-plus/config.json`；若仅存在旧 `codexprov4` 配置则复制迁移并保留旧文件。

## Proposed Persistent Model

```go
type Config struct {
    Version   int         `json:"version"`
    PortRange PortRange   `json:"port_range"`
    Workspaces []Workspace `json:"workspaces"`
}

type PortRange struct {
    Start int `json:"start"`
    End   int `json:"end"`
}

type Workspace struct {
    ID        string `json:"id"`
    Alias     string `json:"alias"`
    Path      string `json:"path"`
    Port      int    `json:"port"`
    AutoStart bool   `json:"auto_start"`
}
```

Notes:
- `ID` 必须稳定，避免前端用数组下标或可编辑 Alias/Path 作为主键。
- `AutoStart` 在 Phase 1 只做持久化字段；真正自动拉起在 Phase 3。
- PID、运行状态、临时 token、启动时间等绝不能出现在该模型或配置 JSON 中。
- `Version` 用于未来配置迁移；v1 初始值为 1。

## Config Rules

### First run

- 若配置文件不存在，返回默认配置并确保父目录可创建。
- 默认端口范围 8800-8899。
- 首次没有 Workspace 时 UI 显示明确引导。

### Save semantics

- 保存前完成完整校验。
- 使用同目录临时文件 + rename 的原子写入方式，避免进程中断留下半截 JSON。
- JSON 使用 UTF-8，中文 Alias/Path 不做额外编码转换。
- 文件权限采用当前用户可读写的合理默认值；Windows 上不依赖 Unix 权限作为安全边界。

### Workspace validation

- Alias trim 后不能为空。
- Path trim 后不能为空，必须是绝对路径且目录存在。
- Path 做 Windows 语义下的规范化后禁止重复（至少大小写不敏感比较）。
- Port 必须在 1-65535。
- 手动端口必须未被其他 Workspace 配置占用；新增/编辑时还要检查系统当前是否可监听。
- 自动端口必须从当前 PortRange 内挑选，同时跳过已配置端口和系统当前占用端口。

### Port range validation

- `1 <= start <= end <= 65535`。
- 修改范围后，已有手动/已分配 Workspace 端口可以保留，即范围只控制“后续自动分配”；不要因为调小范围就静默改写已有 Workspace 端口。

## Backend API Shape

Wails 绑定层建议保持薄，核心逻辑放到独立 store/service 中，避免所有逻辑继续堆进 `app.go`。

建议公开：

- `GetConfig() (Config, error)`
- `AddWorkspace(input WorkspaceInput) (Workspace, error)`
- `UpdateWorkspace(id string, input WorkspaceInput) (Workspace, error)`
- `DeleteWorkspace(id string) error`
- `UpdatePortRange(input PortRange) (Config, error)`

`WorkspaceInput` 建议明确表达端口模式，例如：

```go
type WorkspaceInput struct {
    Alias    string `json:"alias"`
    Path     string `json:"path"`
    PortMode string `json:"port_mode"` // "auto" | "manual"
    Port     int    `json:"port"`
    AutoStart bool  `json:"auto_start"`
}
```

持久化模型不需要保存 `PortMode`：只保存最终分配后的 Port。编辑时前端可以默认展示当前端口，并允许用户选择“重新自动分配”或手动输入。

## UI Direction

第一阶段只追求结构正确、中文可用，不在此阶段做最终视觉打磨。

主界面建议：

- 顶部：标题“CodexPro+” + “端口设置”。
- 主区：Workspace 卡片/表格，每项至少展示 Alias、Path、Port、Auto-start 状态。
- 操作：添加、编辑、删除。
- 空状态：说明“还没有工作目录”，并给出“添加工作目录”入口。
- 添加/编辑表单：别名、目录路径、端口模式（自动/手动）、端口、随管理器启动。
- 删除必须二次确认。
- 后端校验错误直接转成中文可理解提示，不仅写 console。

Phase 1 不实现：启动/停止按钮、PID、运行状态、托盘、自启执行逻辑。

## Test Matrix

### Config store

- 缺少配置文件 -> 默认值。
- 保存后重新加载完全一致。
- 中文 Alias/Path round trip。
- JSON 损坏 -> 返回明确错误，不覆盖原文件。
- 原子写入失败 -> 原配置尽量保持完整。

### Validation

- 空 Alias。
- 相对路径。
- 不存在目录。
- 重复路径（含 Windows 大小写差异）。
- 非法端口。
- 重复手动端口。
- 非法端口范围。

### Port allocation

- 从 8800 开始找到第一个可用端口。
- 跳过其他 Workspace 已配置端口。
- 跳过系统监听端口。
- 范围耗尽时返回明确错误。
- 编辑 Workspace 时允许保留自己的当前端口。

## Known Environment Gap

- `.planning/PROJECT.md` 写 Go 1.26.5；当前 `go.mod` 是 `go 1.25.0`。
- 当前 `@pj` 的 bash 执行环境运行 `go test ./...` 时提示 `go: command not found`，因此此 MCP 会话暂时无法完成 Go/Wails 编译验证。
- 不因此修改 `go.mod`；等可执行环境可用后再以实际 `go version` / `wails build` 结果决定。
