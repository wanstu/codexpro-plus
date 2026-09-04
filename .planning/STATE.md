---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: 可用托盘管理器
status: release_candidate
last_updated: "2026-09-04T23:42:00+08:00"
last_activity: 2026-09-04
progress:
  total_phases: 4
  completed_phases: 1
  total_plans: 4
  completed_plans: 1
  percent: 82
---

# Project State

## Current Position

Phase: 4 of 4 — UX / 稳定性 / 发布验证
Plan: 04-01
Status: Release candidate; Phase 4 UX is user-verified and the dedicated icon set passes production build; ready for RC2 tag
Last activity: 2026-09-04 — User manually verified the Phase 4 UX batch and confirmed it works well

## Planning Artifacts

- `.planning/PROJECT.md` — v1.0 scope, requirements, constraints, corrected tray decision
- `.planning/ROADMAP.md` — 4-phase v1.0 roadmap and requirement traceability
- `.planning/phases/01-workspace-management/01-CONTEXT.md` — completed Phase 1 context
- `.planning/phases/01-workspace-management/01-01-PLAN.md` — completed Phase 1 plan
- `.planning/phases/02-process-lifecycle/02-CONTEXT.md` — Phase 2 runtime model and boundaries
- `.planning/phases/02-process-lifecycle/02-01-PLAN.md` — Phase 2 implementation/verification plan
- `.planning/phases/03-tray-autostart/03-CONTEXT.md` — Phase 3 tray/autostart decisions
- `.planning/phases/03-tray-autostart/03-01-PLAN.md` — Phase 3 implementation/verification plan

## Completed Phase 1

- Workspace CRUD and persistent config are manually verified.
- Auto/manual port allocation and Chinese UI are manually verified.
- `go test ./...` and `wails build` passed before Phase 2/3 changes.

## Implemented for Phase 2

- Config schema is now v3; each Workspace receives a stable token and independent CodexPro permission settings.
- Existing v1/v2 config is migrated in place without changing Workspace ID/path/port/auto-start or rotating an existing token.
- Added runtime-only `Instance` / `RuntimeState`; PID/status/start time are never persisted.
- Added `ProcessManager` with start, stop, stop-all, abnormal-exit tracking and per-Workspace log tail.
- `codexpro-core.exe` is resolved next to the running `codexpro-plus.exe`.
- Core launch passes Workspace root, allowed roots, port, stable token, full tool/bash mode and inherited environment.
- Windows stop uses `taskkill.exe /PID <pid> /T /F` to terminate the process tree.
- Workspace UI now shows running/stopped status, PID, startup time, token, start/stop and logs.
- Added selectable running-instance list; clicking an instance locates its Workspace card.
- Editing/deleting a running Workspace is blocked in both UI and backend.
- Config schema is now v3: each Workspace can configure Token / Bash Mode / Write Mode / Tool Mode / Inherit Env; v1/v2 configs migrate to compatible defaults without rotating existing tokens.
- Frontend pending state + backend `starting` runtime state immediately lock Start/Edit/Delete during startup, preventing repeated clicks from creating overlapping start requests.
- Stop now waits for the service port to become available before reporting success, preventing an immediate restart from racing Windows port release.
- Windows `taskkill` output is not decoded into UTF-8 UI/log text, avoiding localized-console encoding garbage on stop failures.

## Implemented for Phase 3

- Corrected earlier assumption: Wails v2.15 has no supported built-in system tray API.
- Added `github.com/gogpu/systray` as the Windows tray implementation while keeping Wails v2 for the main UI.
- Window close now hides to tray (`HideWindowOnClose=true`).
- Tray left click opens the main panel; right-click menu contains “打开主面板” and “退出”.
- Explicit quit triggers Wails shutdown and stops all managed core instances.
- Added manager auto-start through `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` using `"<exe>" --autostart`.
- `--autostart` starts Wails hidden.
- Workspace `auto_start=true` is now executed at manager startup; one failure does not block other Workspaces.
- UI now exposes core readiness and Windows manager auto-start state.
- Manager auto-start state now verifies that the Run-key command points to the current executable, so a stale entry from a moved/old build is not reported as enabled.
- Added Windows manager single-instance locking (`manager.lock` exclusive handle); a second launch never creates another App/ProcessManager. Normal repeat launch writes `show-window.request` so the primary restores/shows its window, while `--autostart` repeat launch exits silently.
- Product/repository identity is now `CodexPro+` / `codexpro-plus`; executable output is `codexpro-plus.exe`.
- New config root is `~/.config/codexpro-plus`; first launch copies legacy `~/.config/codexprov4/config.json` without deleting the legacy file.
- Added reproducible `scripts/build-codexpro-core.ps1`, pinned to CodexPro upstream commit `587f7fd3a4644a847bba13aeb49336056052e1f6` by default.
- Added Windows GitHub Actions CI: Go tests, pinned CodexPro core build, Wails build, ZIP artifact, and automatic GitHub Release for `v*` tags.
- README replaced with full architecture, build, core packaging, configuration, CI and release documentation.

## Implemented for Phase 4

- Config schema is now v4 with manager-level `domain`; v1/v2/v3 configs remain compatible and an empty value intentionally means `127.0.0.1` for copied/access URLs.
- Workspace URL generation is centralized in Go: missing scheme -> `http://`, explicit `http://` / `https://` preserved, Workspace port appended.
- Domain validation rejects embedded ports, credentials, paths, query strings and fragments to avoid ambiguous URLs.
- `domain` is UI/copy-only. `ProcessManager` remains unchanged with `CODEXPRO_HOST=127.0.0.1`.
- Workspace cards now show a masked full MCP URL and provide “打开” + “复制链接”; the generated URL includes `/mcp` plus the Workspace Token query, and the standalone “复制端口” action has been removed.
- Token and MCP-link copy actions share one helper and show a transient “复制成功” toast; routine start/stop/save success feedback also uses transient Toast, while the top message area is reserved for persistent errors/warnings/notices.
- Manager panel now has a “复制链接域名” setting with an explicit note that it does not change core listening behavior.
- Replaced the temporary `C+` artwork with the dedicated icon set from `assets/icons/`: app branding/favicon uses `codexpro-plus-app.png`, the system tray embeds the optimized transparent `codexpro-plus-tray.png`, and Wails native window/taskbar/exe resources derive from `codexpro-plus-window.png`.
- README and Phase 4 planning document the new URL behavior and boundary.

## Verification Status / Risks

1. Repository static analysis recognizes all current Go/JS symbols without parser warnings.
2. Current UX/MCP-link/icon changes pass `go test ./...`, `node --check frontend/src/main.js`, and a production `wails build` using the new icon set; the produced exe was not launched, so the active CodexPro+ instance was not disturbed. The user had already manually verified the Phase 4 UX and confirmed it works well.
3. Live `@codexprov4-1` verification passed earlier: root/allowed root = `D:\projects\work\wm_group`, port 8600, auth enabled, bash=full, write=workspace, tool=full, inheritEnv=true; CodexPro self-test reported 0 failures.
4. Runtime diagnosis previously found three simultaneously running managers and split runtime state; current observation now shows only one `codexprov4.exe`, consistent with MGR-01 single-instance behavior.
5. Wake-request file lifecycle has a Windows unit test. Final manual smoke check remains: hidden primary + repeated `--autostart` stays hidden; normal repeated launch restores/shows the existing window.
6. `frontend/wailsjs/` is generated output and is now ignored; Wails build regenerates it locally as needed.

## Next Action

1. Push `master` and `v1.0.0-rc2`.
2. Verify the GitHub Actions Windows release build and `CodexProPlus-windows-amd64.zip` artifact.
3. Use RC2 in real work without adding new features.
4. If RC2 remains stable, promote the release line to final `v1.0.0`.

