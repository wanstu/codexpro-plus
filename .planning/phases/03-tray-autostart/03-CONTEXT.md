# Phase 3 Context — 托盘与两级自启

## Goal

把 codexprov4 变成真正的 Windows 常驻托盘管理器，并实现管理器自启 + Workspace 随管理器启动。

## Requirements

- TRAY-01: 常驻系统托盘
- TRAY-02: 左键托盘图标打开主面板
- TRAY-03: 右键菜单包含“打开”和“退出”
- AUTO-01: 管理器可设置 Windows 开机自启
- AUTO-02: 每个 Workspace 可单独随管理器自动拉起
- MGR-01: codexprov4 管理器强制单实例，避免多个 ProcessManager 分裂运行状态

## Corrected Technical Decision

`.planning/PROJECT.md` 原先记录“Wails v2 内置 SetSystemTray”并不成立。Wails v2.15 官方没有受支持的托盘 runtime API，因此本阶段不迁移 Wails v3，而是在 Windows 下独立接入 Go system-tray 实现。

选型：`github.com/gogpu/systray`。

原因：
- 纯 Go，Windows 下不要求 CGO；
- 支持左键点击、右键菜单；
- 与当前 Go 1.26.5 / 单 EXE 目标兼容；
- 托盘只承担原生 notification-area 交互，主窗口仍完全由 Wails v2 管理。

## Window / Tray Lifecycle

- 正常启动：显示主窗口 + 托盘。
- 点击窗口关闭：隐藏窗口，不退出。
- 左键托盘：显示主窗口。
- 右键菜单“打开主面板”：显示主窗口。
- 右键菜单“退出”：调用 Wails Quit，随后停止所有受管实例并清理托盘。
- Windows 自启启动：注册命令带 `--autostart`，此时主窗口初始隐藏，只显示托盘。

## Manager Single-instance

- Windows 启动入口先获取用户配置目录下 `manager.lock` 的独占句柄。
- 同一用户已有 codexprov4 运行时，普通重复启动通过 `show-window.request` 通知主实例恢复/显示窗口，然后新进程退出。
- `--autostart` 重复启动只静默退出，不发送显示窗口请求，避免 Windows 登录时把托盘管理器自动弹到前台。
- 锁由 Windows 文件句柄持有；进程退出或崩溃时由 OS 自动释放，磁盘上的零字节锁文件可以保留。
- 这是 Workspace 运行状态一致性的前置条件：同一时刻只能有一份 runtime-only Instance map。

## Manager Auto-start

使用当前用户注册表：

`HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run`

值名：`CodexProV4`

值内容：`"<exe path>" --autostart`

优点：无需管理员权限、无需解析 `schtasks` 本地化输出，也避免上一版 GBK 编码问题。

## Workspace Auto-start

- `Workspace.AutoStart` 已在 Phase 1 持久化。
- 每次管理器启动后读取配置，逐个启动 `auto_start=true` 的 Workspace。
- 某个 Workspace 启动失败不能阻断其他 Workspace。
- 启动失败记录到 ProcessManager runtime error/log，UI 加载后可见。

## Exit Criteria

1. 关闭窗口后进程仍在且托盘可恢复窗口。
2. 托盘左键能打开主面板。
3. 托盘右键菜单有打开/退出。
4. 显式退出会停止受管 core 实例并结束管理器。
5. 管理器自启注册可开关且状态可读取。
6. `auto_start=true` Workspace 在管理器启动后自动拉起，单个失败不影响其他项。
