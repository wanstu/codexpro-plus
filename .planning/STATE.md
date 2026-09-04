---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: 可用托盘管理器
status: verifying
last_updated: "2026-09-04T20:55:00+08:00"
last_activity: 2026-09-04
progress:
  total_phases: 4
  completed_phases: 1
  total_plans: 3
  completed_plans: 1
  percent: 70
---

# Project State

## Current Position

Phase: 2 + 3 of 4 — 进程生命周期 + 托盘与两级自启
Plan: 02-01 + 03-01
Status: Release candidate; Go tests and Wails production build pass, final repeated-launch wake smoke test pending
Last activity: 2026-09-04 — Added manager single-instance wake behavior and prepared first release tag

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
- `codexpro-core.exe` is resolved next to the running `codexprov4.exe`.
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

## Verification Status / Risks

1. Repository static analysis recognizes all current Go/JS symbols without parser warnings.
2. Latest Windows verification passes: `go test ./...` = ok and Wails v2.15 production build succeeds with Go 1.26.5 after schema-v3, startup-lock, single-instance and wake-request changes.
3. Live `@codexprov4-1` verification passed earlier: root/allowed root = `D:\projects\work\wm_group`, port 8600, auth enabled, bash=full, write=workspace, tool=full, inheritEnv=true; CodexPro self-test reported 0 failures.
4. Runtime diagnosis previously found three simultaneously running managers and split runtime state; current observation now shows only one `codexprov4.exe`, consistent with MGR-01 single-instance behavior.
5. Wake-request file lifecycle has a Windows unit test. Final manual smoke check remains: hidden primary + repeated `--autostart` stays hidden; normal repeated launch restores/shows the existing window.
6. `frontend/wailsjs/` is generated output and is now ignored; Wails build regenerates it locally as needed.

## Next Action

Run locally from `D:\projects\codexprov4`:

1. `go test ./...`
2. `wails build`
3. Verify an existing v2 config migrates to v3 without changing existing tokens/ports.
4. Rapidly click Start on one Workspace; only one launch flow should occur and the button should immediately show `启动中…` disabled.
5. Edit a stopped Workspace and verify Token / Bash / Write / Tool / Inherit Env persist and are reflected by that instance's `server_config` after restart.
6. Re-run the Phase 2/3 tray/autostart smoke checks, then move PROC/TRAY/AUTO/CORE requirements to Validated.
