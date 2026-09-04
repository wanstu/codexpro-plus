# Roadmap — CodexPro+ v1.0 可用托盘管理器

## Milestone Goal

交付一个可日常使用的 Windows 托盘管理器：多个 codexpro 工作目录各自持久化、各自独立端口、各自独立进程，运行时状态不污染持久化配置。

## Phase 1 — Workspace 管理与持久化

**Goal:** 用户可以在中文界面中维护工作目录，并稳定持久化配置；端口既能自动分配也能手动指定。

**Requirements:** WORK-01, WORK-02, WORK-03, WORK-04, WORK-05, UI-04

**Scope:**
- 定义持久化 `Workspace` / `Config` 数据模型，保持与后续运行时 `Instance` 模型分离。
- 配置文件固定为 `~/.config/codexpro-plus/config.json`，首次启动自动创建；若只存在旧 `~/.config/codexprov4/config.json` 则自动复制迁移并保留旧文件。
- 默认端口范围 8800-8899；自动分配跳过配置中已占用端口和系统当前已监听端口。
- 单个 Workspace 支持手动端口；新增/编辑时做目录、别名、端口冲突与范围校验。
- 提供前端可调用的增删改查接口。
- 替换 Wails 默认模板页，完成 Workspace 列表、空状态、添加/编辑/删除和端口设置的第一版中文 UI。
- 为配置读写、校验、端口分配建立 Go 单元测试。

**Exit criteria:**
1. 重启应用后 Workspace 配置保持不变。
2. 自动端口不会选中已被其他 Workspace 或系统进程占用的端口。
3. 手动端口冲突会被明确拒绝并返回可读错误。
4. 用户能通过 UI 完成添加、编辑、删除。
5. 空列表有明确中文引导。
6. `go test ./...` 通过，Wails 项目可构建。

---

## Phase 2 — codexpro 进程生命周期

**Goal:** 每个 Workspace 可以独立启动/停止一个 codexpro-core 实例，并显示准确运行状态。

**Requirements:** PROC-01, PROC-02, PROC-03, PROC-04, CORE-01

**Scope:**
- 引入纯运行时 `Instance` 模型，不写回 Workspace 持久化字段。
- 每个 Workspace 持久化独立 CodexPro 启动参数：Token / Bash Mode / Write Mode / Tool Mode / Inherit Env；旧配置自动迁移为兼容当前行为的默认值。
- 按 Workspace 路径、端口和独立 token 拉起 `codexpro-core.exe`。
- token 使用满足 codexpro 要求的随机值（至少 24 字节；默认 32 字节随机数据编码）。
- 启动前确认端口可用；进程退出后等待端口释放，避免误判。
- 停止服务时显式处理完整进程树。
- UI 展示别名 / PID / 端口 / 目录 / 状态，运行条目可点选。
- 应用退出时有明确的子进程清理策略，并验证不会出现孤儿进程或误杀新进程。

**Exit criteria:**
1. 可独立启动两个不同 Workspace，端口和 PID 不互相污染。
2. 停止其中一个不会影响另一个。
3. 进程异常退出后 UI 状态能恢复为停止。
4. 运行列表可选择条目并显示对应 Workspace。
5. 配置文件不写入 PID、运行状态、临时 token 等运行时字段。

---

## Phase 3 — 托盘与两级自启

**Goal:** 应用成为真正的常驻托盘管理器，并支持管理器自启和 Workspace 自启。

**Requirements:** TRAY-01, TRAY-02, TRAY-03, AUTO-01, AUTO-02, MGR-01

**Scope:**
- Windows 系统托盘常驻。
- 左键托盘图标打开/聚焦主面板。
- 右键托盘菜单至少包含“打开”和“退出”。
- 关闭主窗口默认隐藏到托盘，不等价于退出。
- 管理器强制单实例；重复启动 CodexPro+ 时不得创建第二个 ProcessManager。
- 管理器开机自启开关。
- 每个 Workspace 持久化 `auto_start` 配置；管理器启动后按配置拉起实例。
- 自启失败时保留其他 Workspace 的启动流程，并在 UI 中显示失败原因。

**Exit criteria:**
1. 关闭窗口后托盘仍在，左键可恢复窗口。
2. 右键“退出”能真正退出并执行既定清理策略。
3. 管理器自启开关重启后状态正确。
4. 勾选 auto-start 的 Workspace 随管理器启动，未勾选的不启动。

---

## Phase 4 — UX / 稳定性 / 发布验证

**Goal:** 清掉上一版已知 UI 与编码坑，完成 v1.0 的可用性和发布级验证。

**Requirements:** UI-01, UI-02, UI-03

**Scope:**
- 全量中文文案检查，确保 UTF-8 路径与中文目录正常。
- 检查常见 Windows 缩放比例下的布局，不截断、不遮挡。
- 所有复选框、端口模式、启动/停止、删除等控件具备自解释标签或辅助说明。
- 校验错误、启动失败、端口冲突、core 不存在等异常路径给出用户可理解反馈。
- 验证 `codexpro-core.exe` 同目录部署约束。
- 完整回归 Workspace / Process / Tray / Auto-start。
- 生产构建与最小发布说明。

**Exit criteria:**
1. UI-01/02/03 全部人工验证通过。
2. 所有 v1.0 Active requirements 都有可复现验证结果。
3. 生产构建成功，干净环境启动路径已验证。
4. `.planning/PROJECT.md` 中完成的需求迁入 Validated，并记录对应 phase。

---

## Requirement Traceability

| Requirement | Phase |
|---|---|
| WORK-01 | Phase 1 |
| WORK-02 | Phase 1 |
| WORK-03 | Phase 1 |
| WORK-04 | Phase 1 |
| WORK-05 | Phase 1 |
| PROC-01 | Phase 2 |
| PROC-02 | Phase 2 |
| PROC-03 | Phase 2 |
| PROC-04 | Phase 2 |
| CORE-01 | Phase 2 |
| TRAY-01 | Phase 3 |
| TRAY-02 | Phase 3 |
| TRAY-03 | Phase 3 |
| AUTO-01 | Phase 3 |
| AUTO-02 | Phase 3 |
| MGR-01 | Phase 3 |
| UI-01 | Phase 4 |
| UI-02 | Phase 4 |
| UI-03 | Phase 4 |
| UI-04 | Phase 1 |

## Planning Notes

- `PROJECT.md` 约束写 Go 1.26.5，但当前 `go.mod` 声明 `go 1.25.0`。Phase 1 开工前先以本机实际 `go version` 和 Wails v2.15.0 构建结果为准，再决定是否需要调整 `go.mod`；不在规划阶段擅自升级工具链。
- `codexprov3` 仅作为历史问题来源，不复制其实现代码。
- `Workspace`（持久化）与 `Instance`（运行时）从 Phase 1 起就保持类型和存储边界分离。
- CLI 与运行时临时切换目录继续保持 Out of Scope。
