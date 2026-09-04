# codexprov4

## What This Is

codexpro 的 Windows 原生封装。codexpro 是一个 MCP HTTP 服务，原生只支持单工作目录；本项目在其上包一层，让用户可以同时维护多个工作目录（例如 `D:\projects\foo` 和 `D:\projects\bar`），每个跑独立端口，通过系统托盘统一管理。

## Core Value

**让多个工作目录各自独立地跑起来，且持久化配置永不被运行时操作污染。**

## Business Context

个人工具，自用。无营收模型。

## Requirements

### Validated

(None yet — 这是重写工程，上一版 codexprov3 的代码已废弃不复用)

### Active

- [ ] **WORK-01**: 用户可以添加工作目录（别名 + 路径）
- [ ] **WORK-02**: 用户可以编辑已添加的工作目录
- [ ] **WORK-03**: 用户可以删除工作目录
- [ ] **WORK-04**: 每个工作目录有独立端口（从 8787 起自动递增）
- [ ] **WORK-05**: 工作目录配置持久化到 `~/.config/codexprov4/config.json`
- [ ] **PROC-01**: 用户可以启动某个工作目录的服务
- [ ] **PROC-02**: 用户可以停止某个正在运行的服务
- [ ] **PROC-03**: 用户可以查看正在运行的进程（别名 / PID / 端口 / 目录）
- [ ] **PROC-04**: 进程列表里的条目可以被点选
- [ ] **PROC-05**: 用户可以临时切换某个进程的工作目录（**仅影响运行时，不回写持久化配置**）
- [ ] **TRAY-01**: 程序常驻系统托盘
- [ ] **TRAY-02**: 左键点击托盘图标打开主面板
- [ ] **TRAY-03**: 右键托盘图标有菜单，含「退出」
- [ ] **AUTO-01**: 管理器本身可以设置为开机自启动
- [ ] **AUTO-02**: 每个工作目录可单独勾选是否随管理器自动拉起
- [ ] **UI-01**: 界面文案全部中文，且**不出现乱码**
- [ ] **UI-02**: 所有文字完整显示，**不被截断或遮挡**
- [ ] **UI-03**: 每个控件的作用**自解释**（不靠用户猜）
- [ ] **UI-04**: 列表为空时显示引导文案，不是空白
- [ ] **CLI-01**: 提供命令行接口供脚本调用（add / start / stop / status / url 等）

### Out of Scope

- 复用 codexprov3 的任何代码 — 该工程 UI 层缺陷过多，已整体废弃
- 跨平台支持 — 仅 Windows
- 服务自身的业务逻辑 — 本项目只做进程与配置管理，codexpro 的能力由 codexpro-core 提供

## Context

**为什么会重写：** 上一版 codexprov3（Rust + 手写 Win32 控件）在 UI 层连续翻车，三轮修复都没能解决根本问题：

1. 控件 ID 撞车（标题框用了 `hmenu(1)`，与 `ID_STATUS_TEXT=1` 冲突）→ 状态文本被写进 24px 高的标题框，第一行字被挡住一半
2. `LBS_OWNERDRAWFIXED` 的 listbox 未加 `LBS_HASSTRINGS` → `LB_GETTEXT` 永远返回 `LB_ERR`，列表从来没显示过任何文字，用户看到的是白板
3. 手算控件坐标，一处高度改动推歪后面所有控件
4. 中文乱码：schtasks 输出 GBK 被 `from_utf8_lossy` 读；控制台未用 `WriteConsoleW`
5. 复选框作用不明、进程条目无法点选、空列表是白板

**根因：** 手写 Win32 控件坐标 + 逐个调 API 的方式，每次改动都会引入新的布局/文字问题。本次改用 HTML/CSS 布局，从根本上消除这类问题。

**技术栈决策：** Go + Wails。
- 选 Wails 是因为：托盘内置（`SetSystemTray`）、布局用 HTML/CSS（flex，不用手算坐标）、中文天然 UTF-8 无编码问题、产物单文件约 10MB
- 曾评估并否决：fltk（**无系统托盘组件**，核心需求撑不起来）、PySide6（打包约 50MB）、Tauri（需 Node/bun 工具链，产物更大）、Go+原生 Win32（布局仍要手写，会重蹈覆辙）

**上一版积累的、与 UI 无关的经验（这些仍然是有效的）：**
- token 必须 ≥24 字节，否则 codexpro 直接退出（用 32 字节 hex）
- Job object 只能在常驻模式启用。CLI 一次性命令起进程后自身退出 → Job 句柄关闭 → 把刚起的 core 一起杀掉
- Go/Rust 默认不自动杀子进程，停止服务需要显式处理进程树
- 8787 端口被 Docker Desktop 占用，端口分配要能跳过已占用端口
- 端口释放需要等待（进程退出后立刻查端口会误判）

## Constraints

- **Tech stack**: Go 1.26.5 + Wails v2.15.0 — 托盘需原生支持、布局不能手写、中文不能乱码，三者同时满足
- **Platform**: Windows only（Wails 虽跨平台，本项目不投入跨平台验证）
- **Compatibility**: 必须与 `codexpro-core.exe`（bun 编译 codexpro 的产物）同目录部署
- **Data model**: Workspace（持久化）与 Instance（运行时）严格分离，见下方 Key Decisions

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + Wails 而非 Rust + Win32 | 上一版手写 Win32 控件坐标连续翻车；Wails 用 HTML/CSS 布局 + 内置托盘，从根本上消除布局/文字/编码三类问题 | — Pending |
| Workspace / Instance 严格分离 | 持久化配置（应该跑什么）与运行时状态（现在跑着什么）是两个语义。混在一起会导致一次临时操作永久改坏用户配置 | — Pending |
| 临时切换目录不回写配置 | 同上。进程列表里的操作是运行时行为，绝不回写 config.json | — Pending |
| 不复用 codexprov3 代码 | UI 层缺陷是结构性的，非局部 bug。带过来只会继承问题 | — Pending |
| 决策必须写入 .planning/ | 上一版所有设计讨论只存在于对话中，对话一断全部丢失，导致无法回溯「为什么这么写」 | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-09-04 after initialization*
