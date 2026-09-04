# CodexPro+

CodexPro+ 是一个面向 Windows 的 CodexPro Workspace / Runtime 管理器。

它使用 Wails 提供原生桌面界面，在一个常驻托盘程序里管理多个独立工作目录，并为每个 Workspace 启动一个独立的 `codexpro-core.exe` 实例。

> CodexPro+ 不是 CodexPro 的 fork，也不修改 CodexPro 的业务逻辑。它负责 Workspace、配置、端口、进程、托盘、自启动和运行状态；实际 MCP 能力由 CodexPro 提供。

## 功能

- 多 Workspace 管理：添加、编辑、删除、持久化。
- 一个 Workspace 对应一个独立 CodexPro 实例和端口。
- 自动端口分配，也可以手动指定端口。
- 每个 Workspace 独立配置 CodexPro 权限参数。
- 启动 / 停止 `codexpro-core.exe`，显示 PID、端口、启动时间和运行日志。
- Windows 托盘常驻。
- 关闭窗口时隐藏到托盘，不退出服务。
- 普通重复启动 `codexpro-plus.exe` 时唤醒已经运行的主窗口，不创建第二个管理器。
- `--autostart` 重复启动时静默退出，不自动弹出窗口。
- 管理器本身支持 Windows 开机自启动。
- 每个 Workspace 可单独设置“随管理器启动”。
- 配置版本迁移和旧 `codexprov4` 配置兼容。
- Workspace 卡片可复制完整 MCP 链接，并用默认浏览器打开该地址；链接包含 `/mcp` endpoint 和 Workspace Token。
- 管理器可配置“复制链接域名”；留空默认使用 `127.0.0.1`，且不会改变 core 实际监听地址。
- GitHub Actions 自动测试、构建 Windows 发布包，并在 tag 时自动创建 Release。

## 项目关系

CodexPro+ 的运行结构：

```text
CodexPro+
    |
    +-- Workspace A
    |      +-- codexpro-core.exe : 8800
    |
    +-- Workspace B
    |      +-- codexpro-core.exe : 8801
    |
    +-- Workspace C
           +-- codexpro-core.exe : 8802
```

控制链路：

```text
Workspace
    ↓
Persistent Config
    ↓
ProcessManager
    ↓
codexpro-core.exe
    ↓
CodexPro MCP tools
```

CodexPro 上游项目：

```text
https://github.com/rebel0789/codexpro
```

## Release 包

正常的 Windows Release ZIP 包包含两个文件：

```text
CodexProPlus-windows-amd64.zip
├── codexpro-plus.exe
└── codexpro-core.exe
```

两个 exe 必须放在同一个目录。

启动：

```powershell
.\codexpro-plus.exe
```

CodexPro+ 会根据 Workspace 配置为每个工作目录启动独立的 `codexpro-core.exe`。

## Workspace 的 CodexPro 参数

每个 Workspace 可以配置：

| 配置 | 可选值 | 作用 |
| --- | --- | --- |
| HTTP Token | 至少 24 字节 | 该 CodexPro MCP 实例的认证 Token；留空时 CodexPro+ 自动生成 32 字节随机值 |
| Bash Mode | `off` / `safe` / `full` | 控制 CodexPro 是否可以执行 shell，以及 shell 权限范围 |
| Write Mode | `off` / `handoff` / `workspace` | 控制 CodexPro 的源码写入模式 |
| Tool Mode | `minimal` / `standard` / `full` | 控制向 MCP 客户端暴露的工具集合 |
| Inherit Env | 开 / 关 | 是否让 core 继承本机 PATH 等环境变量 |

对于可信的本地开发工程，CodexPro+ 默认使用：

```text
Bash Mode    full
Write Mode   workspace
Tool Mode    full
Inherit Env  true
Token        自动生成
```

CodexPro+ 自动管理以下参数，不在 Workspace 中重复配置：

```text
CODEXPRO_ROOT
CODEXPRO_ALLOWED_ROOTS
CODEXPRO_HOST=127.0.0.1
CODEXPRO_PORT
CODEXPRO_MODE=agent
```

其中 `ROOT / ALLOWED_ROOTS` 来自 Workspace 路径，`PORT` 来自 Workspace 端口。

## 配置和日志

当前配置目录：

```text
~/.config/codexpro-plus/
├── config.json
└── logs/
```

Workspace 日志：

```text
~/.config/codexpro-plus/logs/<workspace-id>.log
```

### 复制链接域名

`config.json` 中的 manager-level `domain` 只用于生成 Workspace 的访问/复制链接。

```text
domain 为空                 -> http://127.0.0.1:8800/mcp?<token-query>
domain = localhost          -> http://localhost:8800/mcp?<token-query>
domain = dev.example.com    -> http://dev.example.com:8800/mcp?<token-query>
domain = https://dev.example.com -> https://dev.example.com:8800/mcp?<token-query>
```

`<token-query>` 代表 CodexPro 的 Workspace Token 查询参数；“复制链接”会生成可直接给 MCP 客户端使用的完整 URL。

不要在 `domain` 中填写端口；Workspace 自己的 `port` 会被追加到链接中。该设置不会改变 `CODEXPRO_HOST=127.0.0.1`，因此它不会让 CodexPro 自动监听局域网或公网地址。

## 本地开发环境

当前开发和 CI 基线：

```text
Windows amd64
Go 1.26.5
Wails CLI 2.15.0
Node.js 24
Bun 1.4.0
```

需要：

- Windows 10/11。
- Go。
- Wails CLI。
- WebView2 Runtime。
- 如果需要自己构建 `codexpro-core.exe`：Node.js、npm、Bun、Git。

安装 Wails CLI：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
```

检查环境：

```powershell
go version
wails version
node --version
npm --version
bun --version
```

## 构建 CodexPro+ 管理器

克隆：

```powershell
git clone https://github.com/wanstu/codexpro-plus.git
cd codexpro-plus
```

运行测试：

```powershell
go test ./...
```

构建：

```powershell
wails build
```

输出：

```text
build/bin/codexpro-plus.exe
```

开发模式：

```powershell
wails dev
```

## 构建 codexpro-core.exe

CodexPro 官方源码是 Node.js / TypeScript 项目。上游标准源码构建流程是：

```powershell
npm ci
npm run build
```

它会生成 JavaScript 构建结果，例如：

```text
dist/http.js
```

CodexPro+ 需要一个可直接启动的 Windows 单文件 `codexpro-core.exe`。本项目在官方 TypeScript build 之后，再使用 Bun standalone executable 编译：

```powershell
bun build --compile .\dist\http.js --outfile .\codexpro-core.exe
```

为了让这个过程可复现，仓库提供：

```text
scripts/build-codexpro-core.ps1
```

### 一键从固定上游版本构建

在 CodexPro+ 仓库根目录运行：

```powershell
.\scripts\build-codexpro-core.ps1
```

脚本会：

1. 临时 clone CodexPro 官方仓库。
2. checkout CodexPro+ 当前固定的 upstream commit。
3. 执行 `npm ci --ignore-scripts --no-audit --no-fund`。
4. 执行 `npm run build`。
5. 使用 Bun 编译 `dist/http.js`。
6. 输出 `build/bin/codexpro-core.exe`。
7. 删除临时源码目录。

当前固定的 CodexPro upstream commit：

```text
587f7fd3a4644a847bba13aeb49336056052e1f6
```

指定其他 CodexPro ref：

```powershell
.\scripts\build-codexpro-core.ps1 -Ref <commit-or-tag>
```

### 使用本机已有 CodexPro 源码

如果已经 clone 了 CodexPro：

```powershell
.\scripts\build-codexpro-core.ps1 `
  -Source D:\projects\codexpro
```

传入 `-Source` 时脚本不会替你切换分支或 commit，会直接使用该目录当前代码。

指定输出路径：

```powershell
.\scripts\build-codexpro-core.ps1 `
  -Source D:\projects\codexpro `
  -Output D:\output\codexpro-core.exe
```

> `bun build --compile` 是 CodexPro+ 的 Windows 打包约定，不是 CodexPro 上游官方 npm 发布流程。CodexPro 官方开发流程仍以 Node.js / npm build 为准。

## 完整本地 Release 构建

从仓库根目录：

```powershell
go test ./...

.\scripts\build-codexpro-core.ps1

wails build
```

完成后确认：

```powershell
Get-ChildItem .\build\bin\
```

应至少包含：

```text
codexpro-plus.exe
codexpro-core.exe
```

手工打 ZIP：

```powershell
New-Item -ItemType Directory -Force .\dist | Out-Null

Compress-Archive -Force `
  -Path .\build\bin\codexpro-plus.exe,.\build\bin\codexpro-core.exe `
  -DestinationPath .\dist\CodexProPlus-windows-amd64.zip
```

## GitHub Actions CI

工作流：

```text
.github/workflows/ci.yml
```

### Push / Pull Request

自动执行：

```text
Checkout
Setup Go 1.26.5
Setup Node.js 24
Setup Bun 1.4.0
Install Wails 2.15.0
go test ./...
Build pinned CodexPro core
wails build
Package ZIP
Upload Actions artifact
```

Actions artifact：

```text
CodexProPlus-windows-amd64.zip
```

### Tag Release

推送 `v*` tag，例如：

```powershell
git tag -a v1.0.0-rc1 -m "CodexPro+ v1.0.0-rc1"
git push origin v1.0.0-rc1
```

GitHub Actions 会自动：

1. 重新运行测试。
2. 构建固定版本的 `codexpro-core.exe`。
3. 构建 `codexpro-plus.exe`。
4. 生成 `CodexProPlus-windows-amd64.zip`。
5. 创建 GitHub Release。
6. 把 ZIP 上传到 Release Assets。

带 `-rc`、`-beta` 等 `-` 后缀的 tag 会被标记为 prerelease。

## 单实例和启动行为

CodexPro+ 本身强制单实例。

普通启动：

```powershell
.\codexpro-plus.exe
```

如果已有实例：

```text
第二个进程不创建新的 ProcessManager
        ↓
通知已有实例
        ↓
恢复 / 显示已有主窗口
        ↓
第二个进程退出
```

Windows 自启动使用：

```text
"<path>\codexpro-plus.exe" --autostart
```

`--autostart` 行为：

- 首次启动时主窗口保持隐藏，只显示托盘。
- 如果已经有 CodexPro+ 实例，则新进程静默退出。
- 不会因为系统自启动重复触发而把窗口自动弹到前台。

## 停止进程

Windows 下停止 Workspace 使用进程树终止，而不是只杀父进程：

```text
taskkill.exe /PID <pid> /T /F
```

停止后 CodexPro+ 还会等待对应端口真正释放，避免立即重新启动时撞到 Windows 的端口释放窗口。

## 源码结构

```text
.
├── app.go                     Wails API / 生命周期
├── config.go                  配置、迁移、持久化
├── workspace_service.go       Workspace CRUD / 端口 / CodexPro 参数
├── process_manager.go         core 进程生命周期、状态、日志
├── process_windows.go         Windows 进程树处理
├── tray_manager.go            系统托盘
├── autostart_windows.go       Windows 开机自启动
├── single_instance_windows.go 单实例和重复启动窗口唤醒
├── frontend/src/              Wails 前端
├── scripts/
│   └── build-codexpro-core.ps1
├── .github/workflows/
│   └── ci.yml
└── .planning/                 GSD 项目规划和状态
```

## 安全说明

`Bash Mode = full` 会让 CodexPro 获得更高的本地命令执行能力。只应对你信任的本地 Workspace 使用 `full`。

如果希望更严格：

```text
Bash Mode   safe 或 off
Write Mode  handoff 或 off
Tool Mode   standard 或 minimal
```

CodexPro+ 只负责把这些选项传给 CodexPro；实际文件访问、shell、安全策略和 MCP 工具行为由 CodexPro 实现。

## 当前范围

- Windows only。
- Wails v2。
- GUI / Tray manager，不提供独立 CLI 管理接口。
- 一个 Workspace = 一个 CodexPro runtime instance。
- 不在运行中的实例里临时切换 Workspace。
