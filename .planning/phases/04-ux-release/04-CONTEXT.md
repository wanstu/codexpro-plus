# Phase 4 Context — UX / 链接 / 发布验证

## 目标

在 v1.0.0-rc1 基础上完成一批高频 UX 能力，并把它们纳入 Phase 4 发布验收：复制反馈、Workspace 访问地址、管理器级复制域名，以及最终生产构建回归。

## 已确认决策

1. `Config.domain` 是 Manager 级配置，不属于单个 Workspace。
2. `domain` 只用于生成 UI 展示/复制/打开的访问地址，绝不能传入 CodexPro 进程环境，也不能改变 `CODEXPRO_HOST=127.0.0.1`。
3. `domain` 留空时，访问地址使用 `127.0.0.1`。
4. 未写 scheme 时默认 `http://`；显式 `http://` / `https://` 时保留。
5. Workspace URL 统一由后端生成，避免复制与“打开链接”形成两套拼接规则。
6. 所有复制动作使用统一 helper；成功反馈至少明确出现“复制成功”，失败则给出可理解错误。
7. Workspace 卡片展示访问地址，并提供“打开”“复制链接”。
8. 管理器设置区域增加“复制链接域名”，明确说明它不会改变 CodexPro 实际监听地址。

## URL 示例

- domain 为空 + port 8800 -> `http://127.0.0.1:8800`
- `localhost` + 8800 -> `http://localhost:8800`
- `192.168.1.10` + 8800 -> `http://192.168.1.10:8800`
- `dev.example.com` + 8800 -> `http://dev.example.com:8800`
- `https://dev.example.com` + 8800 -> `https://dev.example.com:8800`

## 验证重点

- 配置升级/旧配置迁移不丢 Workspace 数据。
- domain 保存后重启可恢复。
- URL formatter 的默认域名和 scheme 规则有单元测试。
- ProcessManager 仍固定 `CODEXPRO_HOST=127.0.0.1`。
- 前端所有复制动作成功后均有明确反馈。
- `go test ./...` 与 Wails production build 通过。
