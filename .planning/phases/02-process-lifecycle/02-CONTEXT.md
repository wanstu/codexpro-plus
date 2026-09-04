# Phase 2 Context — codexpro 进程生命周期

## Goal

让每个 Workspace 都能独立启动/停止 `codexpro-core.exe`，并在 UI 中明确看到运行状态、PID、端口、目录和日志；运行时状态绝不写回 Workspace 配置。

## Requirements

- PROC-01: 启动某个 Workspace 服务
- PROC-02: 停止某个正在运行的服务
- PROC-03: 查看运行实例（别名 / PID / 端口 / 目录）
- PROC-04: 运行实例可以点选
- CORE-01: 每个 Workspace 独立配置 Token / Bash Mode / Write Mode / Tool Mode / Inherit Env

## Locked Decisions

1. `Workspace` 继续代表持久化配置；`Instance` 只存在内存。
2. `codexpro-core.exe` 按 v1 约束从 `codexprov4.exe` 同目录解析。
3. 每个 Workspace 使用稳定 token。配置版本最终升级到 v3：v1/v2 Workspace 自动补齐 CodexPro 参数；token 属于连接配置，不属于运行时状态。
4. token 使用 32 字节随机数据的 hex 表示（64 个十六进制字符），用户也可以自定义，但至少 24 字节。
5. Workspace 独立持久化 `bash_mode`、`write_mode`、`tool_mode`、`inherit_env`；旧配置迁移默认保持现有行为：`full / workspace / full / true`。
6. 固定启动环境仍包括：`CODEXPRO_ROOT=<workspace>`、`CODEXPRO_ALLOWED_ROOTS=<workspace>`、`CODEXPRO_HOST=127.0.0.1`、`CODEXPRO_PORT=<port>`、`CODEXPRO_MODE=agent`；权限类 env 从 Workspace 配置读取。
7. 启动状态分为 `starting / running / stopped`；前后端都必须阻止同一 Workspace 在 `starting` 期间再次启动。
8. 删除或编辑正在启动/运行的 Workspace 时拒绝操作，避免持久化配置与正在运行的实例发生漂移。
9. 停止服务必须停止完整进程树；Windows 使用 `taskkill.exe /PID <pid> /T /F`。
10. 显式退出管理器时停止所有由本次管理器启动的实例，避免孤儿进程。

## Runtime Model

```go
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
```

内部对象额外持有 `*exec.Cmd`、done channel、stopping 标记等，不暴露给前端。

## Logs

- 日志目录：`~/.config/codexprov4/logs/`
- 每个 Workspace 一个 `<workspace-id>.log`
- core stdout/stderr 都追加到该文件
- 管理器也写入启动、PID、停止、异常退出等事件
- 前端通过 `GetWorkspaceLog(id)` 读取尾部，默认最多 64 KiB

## UI

- Workspace 卡片增加 `已停止 / 运行中` 状态。
- 已停止：显示“启动”。
- 运行中：显示 PID、端口、启动时间，并显示“停止”。
- 卡片提供“查看日志”。
- 页面增加“运行实例”区域，条目可点击并定位对应 Workspace。
- 前端定时轮询运行时状态，处理 core 自行退出的场景。

## Exit Criteria

1. 两个 Workspace 可同时启动且 PID/端口独立。
2. 停止一个实例不影响另一个。
3. core 异常退出后 UI 能恢复到停止状态并留下错误/日志。
4. 运行实例列表可点击。
5. 配置 JSON 中无 PID、StartedAt、Status 等运行时字段。
