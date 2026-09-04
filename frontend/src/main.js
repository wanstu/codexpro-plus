const state = {
    config: null,
    manager: null,
    workspaceURLs: new Map(),
    runtimeStates: new Map(),
    instances: [],
    editingId: null,
    logWorkspaceId: null,
    runtimeLoading: false,
    pendingStarts: new Set(),
    pendingStops: new Set(),
};

const elements = {
    message: document.getElementById("message"),
    toast: document.getElementById("toast"),
    workspaceList: document.getElementById("workspace-list"),
    emptyState: document.getElementById("empty-state"),
    workspaceCount: document.getElementById("workspace-count"),
    addButton: document.getElementById("add-workspace-button"),
    emptyAddButton: document.getElementById("empty-add-button"),
    instanceList: document.getElementById("instance-list"),
    instanceEmpty: document.getElementById("instance-empty"),
    instanceCount: document.getElementById("instance-count"),
    coreStatus: document.getElementById("core-status"),
    corePath: document.getElementById("core-path"),
    buildBadge: document.getElementById("build-badge"),
    configPath: document.getElementById("config-path"),
    managerAutoStart: document.getElementById("manager-auto-start"),
    managerError: document.getElementById("manager-error"),
    domainForm: document.getElementById("domain-form"),
    copyDomain: document.getElementById("copy-domain"),
    portRangeForm: document.getElementById("port-range-form"),
    portStart: document.getElementById("port-start"),
    portEnd: document.getElementById("port-end"),
    dialog: document.getElementById("workspace-dialog"),
    dialogTitle: document.getElementById("dialog-title"),
    dialogMessage: document.getElementById("dialog-message"),
    dialogCloseButton: document.getElementById("dialog-close-button"),
    dialogCancelButton: document.getElementById("dialog-cancel-button"),
    workspaceForm: document.getElementById("workspace-form"),
    workspaceAlias: document.getElementById("workspace-alias"),
    workspacePath: document.getElementById("workspace-path"),
    workspacePort: document.getElementById("workspace-port"),
    workspaceAuthInput: document.getElementById("workspace-token"),
    workspaceBashMode: document.getElementById("workspace-bash-mode"),
    workspaceWriteMode: document.getElementById("workspace-write-mode"),
    workspaceToolMode: document.getElementById("workspace-tool-mode"),
    workspaceInheritEnv: document.getElementById("workspace-inherit-env"),
    workspaceAutoStart: document.getElementById("workspace-auto-start"),
    manualPortField: document.getElementById("manual-port-field"),
    portHelp: document.getElementById("port-help"),
    workspaceSubmitButton: document.getElementById("workspace-submit-button"),
    logDialog: document.getElementById("log-dialog"),
    logTitle: document.getElementById("log-title"),
    logPath: document.getElementById("log-path"),
    logOutput: document.getElementById("log-output"),
    logCloseButton: document.getElementById("log-close-button"),
    logRefreshButton: document.getElementById("log-refresh-button"),
    logDoneButton: document.getElementById("log-done-button"),
};

let toastTimer = null;

function appApi() {
    const api = window.go?.main?.App;
    if (!api) {
        throw new Error("Wails 后端尚未就绪，请重新打开应用后再试。");
    }
    return api;
}

function errorText(error) {
    if (!error) return "发生未知错误";
    if (typeof error === "string") return error;
    if (typeof error.message === "string" && error.message) return error.message;
    return String(error);
}

function showMessage(text, type = "info") {
    elements.message.textContent = text;
    elements.message.className = `message ${type}`;
}

function clearMessage() {
    elements.message.textContent = "";
    elements.message.className = "message hidden";
}

function showToast(text, type = "success") {
    if (toastTimer) window.clearTimeout(toastTimer);
    elements.toast.textContent = text;
    elements.toast.className = `toast ${type}`;
    toastTimer = window.setTimeout(() => {
        elements.toast.textContent = "";
        elements.toast.className = "toast hidden";
        toastTimer = null;
    }, 2600);
}

function showDialogMessage(text) {
    elements.dialogMessage.textContent = text;
    elements.dialogMessage.className = "message error";
}

function clearDialogMessage() {
    elements.dialogMessage.textContent = "";
    elements.dialogMessage.className = "message hidden";
}

async function loadConfig() {
    try {
        const config = await appApi().GetConfig();
        state.config = config;
        await refreshWorkspaceURLs(config?.workspaces || []);
        renderConfig();
        return true;
    } catch (error) {
        showMessage(`读取配置失败：${errorText(error)}`, "error");
        elements.workspaceCount.textContent = "配置读取失败";
        return false;
    }
}

async function refreshWorkspaceURLs(workspaces) {
    const pairs = await Promise.all((workspaces || []).map(async (workspace) => {
        try {
            return [workspace.id, await appApi().GetWorkspaceURL(workspace.id)];
        } catch (error) {
            return [workspace.id, "", errorText(error)];
        }
    }));
    state.workspaceURLs = new Map(pairs.map(([id, workspaceURL]) => [id, workspaceURL]));
    const failed = pairs.find(([, workspaceURL]) => !workspaceURL);
    if (failed) {
        showMessage(`生成访问地址失败：${failed[2]}`, "error");
    }
}

async function loadManagerSettings() {
    try {
        state.manager = await appApi().GetManagerSettings();
        renderManagerSettings();
    } catch (error) {
        elements.coreStatus.textContent = "状态读取失败";
        elements.coreStatus.className = "status-pill error";
        elements.managerError.textContent = errorText(error);
        elements.managerError.classList.remove("hidden");
    }
}

async function loadRuntime() {
    if (state.runtimeLoading) return;
    state.runtimeLoading = true;
    try {
        const [runtimeStates, instances] = await Promise.all([
            appApi().GetRuntimeStates(),
            appApi().GetInstances(),
        ]);
        state.runtimeStates = new Map((runtimeStates || []).map((item) => [item.workspace_id, item]));
        state.instances = Array.isArray(instances) ? instances : [];
        renderWorkspaceList();
        renderInstances();
    } catch (error) {
        elements.instanceCount.textContent = `读取运行状态失败：${errorText(error)}`;
    } finally {
        state.runtimeLoading = false;
    }
}

function renderManagerSettings() {
    const manager = state.manager;
    if (!manager) return;

    const devBuild = manager.build_profile === "dev";
    elements.buildBadge.classList.toggle("hidden", !devBuild);
    elements.buildBadge.title = devBuild ? "本地开发构建：配置、单实例锁和自启动项与正式版隔离" : "";
    document.title = manager.app_name || "CodexPro+";
    elements.configPath.textContent = manager.config_path || "—";
    elements.configPath.title = manager.config_path || "";

    elements.managerAutoStart.checked = Boolean(manager.auto_start);
    elements.managerAutoStart.disabled = false;

    if (manager.core_ready) {
        elements.coreStatus.textContent = "Core 已就绪";
        elements.coreStatus.className = "status-pill running";
        elements.corePath.textContent = manager.core_path || "codexpro-core.exe";
        elements.corePath.title = manager.core_path || "";
    } else {
        elements.coreStatus.textContent = "Core 未就绪";
        elements.coreStatus.className = "status-pill error";
        elements.corePath.textContent = manager.core_error || "未找到 codexpro-core.exe";
        elements.corePath.title = elements.corePath.textContent;
    }

    const errors = [manager.core_error, manager.tray_error].filter(Boolean);
    if (errors.length > 0) {
        elements.managerError.textContent = errors.join("；");
        elements.managerError.classList.remove("hidden");
    } else {
        elements.managerError.textContent = "";
        elements.managerError.classList.add("hidden");
    }
}

function renderConfig() {
    const config = state.config || {port_range: {start: 8800, end: 8899}, workspaces: []};
    const portRange = config.port_range || {start: 8800, end: 8899};
    const workspaces = Array.isArray(config.workspaces) ? config.workspaces : [];

    elements.copyDomain.value = config.domain || "";
    elements.portStart.value = portRange.start ?? 8800;
    elements.portEnd.value = portRange.end ?? 8899;
    elements.workspaceCount.textContent = workspaces.length === 0
        ? "暂无工作目录"
        : `共 ${workspaces.length} 个工作目录`;

    elements.emptyState.classList.toggle("hidden", workspaces.length !== 0);
    elements.workspaceList.classList.toggle("hidden", workspaces.length === 0);
    renderWorkspaceList();
}

function renderWorkspaceList() {
    const workspaces = Array.isArray(state.config?.workspaces) ? state.config.workspaces : [];
    elements.workspaceList.replaceChildren();
    workspaces.forEach((workspace) => {
        elements.workspaceList.appendChild(createWorkspaceCard(workspace));
    });
}

function createWorkspaceCard(workspace) {
    const runtimeState = state.runtimeStates.get(workspace.id) || {};
    const running = Boolean(runtimeState.running);
    const starting = Boolean(runtimeState.starting) || state.pendingStarts.has(workspace.id);
    const stopping = state.pendingStops.has(workspace.id);
    const processBusy = starting || stopping;
    const workspaceURL = state.workspaceURLs.get(workspace.id) || "";

    const card = document.createElement("article");
    card.className = running ? "workspace-card running-card" : "workspace-card";
    card.id = `workspace-${workspace.id}`;

    const header = document.createElement("div");
    header.className = "workspace-card-header";

    const titleGroup = document.createElement("div");
    const title = document.createElement("h3");
    title.textContent = workspace.alias || "未命名工作目录";
    titleGroup.appendChild(title);

    const idText = document.createElement("p");
    idText.className = "workspace-id";
    idText.textContent = `ID ${workspace.id || "—"}`;
    titleGroup.appendChild(idText);

    const status = document.createElement("span");
    if (starting) {
        status.className = "status-pill neutral";
        status.textContent = "◌ 启动中";
    } else if (stopping) {
        status.className = "status-pill neutral";
        status.textContent = "◌ 停止中";
    } else {
        status.className = running ? "status-pill running" : "status-pill stopped";
        status.textContent = running ? "● 运行中" : "○ 已停止";
    }

    header.append(titleGroup, status);
    card.appendChild(header);

    const badges = document.createElement("div");
    badges.className = "badge-row";
    if (workspace.auto_start) {
        const autoBadge = document.createElement("span");
        autoBadge.className = "badge enabled";
        autoBadge.textContent = "随管理器启动";
        badges.appendChild(autoBadge);
    }
    if (badges.childElementCount > 0) card.appendChild(badges);

    card.appendChild(metaRow("目录", workspace.path || "—", true));
    card.appendChild(metaRow("端口", String(workspace.port ?? "—"), false));
    card.appendChild(metaRow("MCP 链接", maskWorkspaceURL(workspaceURL), true));
    card.appendChild(metaRow(
        "CodexPro",
        `bash=${workspace.bash_mode || "full"} · write=${workspace.write_mode || "workspace"} · tool=${workspace.tool_mode || "full"} · env=${workspace.inherit_env ? "inherit" : "clean"}`,
        true,
    ));
    if (running) {
        card.appendChild(metaRow("PID", String(runtimeState.pid ?? "—"), false));
        card.appendChild(metaRow("启动", formatStartedAt(runtimeState.started_at), false));
    }
    card.appendChild(tokenRow(workspace.token || ""));

    if (runtimeState.last_error) {
        const runtimeError = document.createElement("div");
        runtimeError.className = "runtime-error";
        runtimeError.textContent = runtimeState.last_error;
        card.appendChild(runtimeError);
    }

    const actions = document.createElement("div");
    actions.className = "workspace-actions";

    const openButton = document.createElement("button");
    openButton.type = "button";
    openButton.className = "button secondary small";
    openButton.textContent = "打开";
    openButton.disabled = !workspaceURL;
    openButton.title = workspaceURL ? maskWorkspaceURL(workspaceURL) : "MCP 链接生成失败";
    openButton.addEventListener("click", () => openWorkspaceURL(workspace));

    const copyURLButton = document.createElement("button");
    copyURLButton.type = "button";
    copyURLButton.className = "button ghost small";
    copyURLButton.textContent = "复制链接";
    copyURLButton.disabled = !workspaceURL;
    copyURLButton.addEventListener("click", () => copyText(workspaceURL, "MCP 链接"));

    // 单独复制端口已移除；复制链接包含完整 MCP 地址和认证 token。





    const processButton = document.createElement("button");
    processButton.type = "button";
    processButton.className = running ? "button danger small" : "button primary small";
    processButton.textContent = starting ? "启动中…" : (stopping ? "停止中…" : (running ? "停止" : "启动"));
    processButton.disabled = processBusy;
    processButton.addEventListener("click", () => running ? stopWorkspace(workspace) : startWorkspace(workspace));

    const logButton = document.createElement("button");
    logButton.type = "button";
    logButton.className = "button secondary small";
    logButton.textContent = "日志";
    logButton.addEventListener("click", () => openLogDialog(workspace));

    const editButton = document.createElement("button");
    editButton.type = "button";
    editButton.className = "button ghost small";
    editButton.textContent = "编辑";
    editButton.disabled = running || starting;
    editButton.title = running || starting ? "请先等待启动完成并停止服务后再编辑" : "编辑工作目录";
    editButton.addEventListener("click", () => openEditDialog(workspace.id));

    const deleteButton = document.createElement("button");
    deleteButton.type = "button";
    deleteButton.className = "button ghost-danger small";
    deleteButton.textContent = "删除";
    deleteButton.disabled = running || starting;
    deleteButton.title = running || starting ? "请先等待启动完成并停止服务后再删除" : "删除工作目录";
    deleteButton.addEventListener("click", () => deleteWorkspace(workspace));

    actions.append(openButton, copyURLButton, processButton, logButton, editButton, deleteButton);
    card.appendChild(actions);
    return card;
}

function metaRow(label, value, allowWrap) {
    const row = document.createElement("div");
    row.className = "meta-row";

    const key = document.createElement("span");
    key.className = "meta-label";
    key.textContent = label;

    const content = document.createElement("span");
    content.className = allowWrap ? "meta-value path-value" : "meta-value";
    content.textContent = value;
    content.title = value;

    row.append(key, content);
    return row;
}

function tokenRow(token) {
    const row = document.createElement("div");
    row.className = "meta-row";

    const key = document.createElement("span");
    key.className = "meta-label";
    key.textContent = "Token";

    const wrapper = document.createElement("div");
    wrapper.className = "token-value";
    const value = document.createElement("code");
    value.textContent = maskToken(token);
    value.title = token;
    wrapper.appendChild(value);

    const copyButton = document.createElement("button");
    copyButton.type = "button";
    copyButton.className = "link-button";
    copyButton.textContent = "复制";
    copyButton.disabled = !token;
    copyButton.addEventListener("click", () => copyText(token, "Token"));
    wrapper.appendChild(copyButton);
    row.append(key, wrapper);
    return row;
}

function maskToken(token) {
    if (!token) return "—";
    if (token.length <= 16) return token;
    return `${token.slice(0, 8)}…${token.slice(-8)}`;
}

function maskWorkspaceURL(value) {
    if (!value) return "生成失败";
    const marker = "codexpro_" + "token=";
    const markerIndex = value.indexOf(marker);
    if (markerIndex < 0) return value;

    const tokenStart = markerIndex + marker.length;
    const tokenEnd = value.indexOf("&", tokenStart);
    const rawToken = value.slice(tokenStart, tokenEnd < 0 ? value.length : tokenEnd);
    let token = rawToken;
    try {
        token = decodeURIComponent(rawToken);
    } catch (_) {
        // Keep the raw value if it is not URL encoded.
    }
    const suffixStart = tokenEnd < 0 ? value.length : tokenEnd;
    return `${value.slice(0, tokenStart)}${maskToken(token)}${value.slice(suffixStart)}`;
}

async function copyText(text, label = "内容") {
    try {
        if (navigator.clipboard?.writeText) {
            await navigator.clipboard.writeText(text);
        } else {
            const textarea = document.createElement("textarea");
            textarea.value = text;
            textarea.style.position = "fixed";
            textarea.style.opacity = "0";
            document.body.appendChild(textarea);
            textarea.select();
            const copied = document.execCommand("copy");
            textarea.remove();
            if (!copied) throw new Error("系统剪贴板不可用");
        }
        showToast(`复制成功：${label}`, "success");
        return true;
    } catch (error) {
        showToast(`复制失败：${errorText(error)}`, "error");
        return false;
    }
}

async function openWorkspaceURL(workspace) {
    try {
        await appApi().OpenWorkspaceURL(workspace.id);
        showToast("已打开 MCP 链接。", "success");
    } catch (error) {
        showToast(`打开链接失败：${errorText(error)}`, "error");
    }
}

function renderInstances() {
    const instances = state.instances || [];
    elements.instanceCount.textContent = instances.length === 0
        ? "0 个实例正在运行"
        : `${instances.length} 个实例正在运行`;
    elements.instanceList.replaceChildren();
    elements.instanceEmpty.classList.toggle("hidden", instances.length !== 0);

    instances.forEach((instance) => {
        const row = document.createElement("button");
        row.type = "button";
        row.className = "instance-row";
        row.title = "点击定位到对应工作目录";

        const name = document.createElement("strong");
        name.textContent = instance.alias || instance.workspace_id;
        const details = document.createElement("span");
        details.textContent = `PID ${instance.pid} · 端口 ${instance.port}`;
        const path = document.createElement("span");
        path.className = "instance-path";
        path.textContent = instance.path || "";

        row.append(name, details, path);
        row.addEventListener("click", () => scrollToWorkspace(instance.workspace_id));
        elements.instanceList.appendChild(row);
    });
}

function scrollToWorkspace(id) {
    const card = document.getElementById(`workspace-${id}`);
    if (!card) return;
    card.scrollIntoView({behavior: "smooth", block: "center"});
    card.classList.add("workspace-highlight");
    window.setTimeout(() => card.classList.remove("workspace-highlight"), 1400);
}

function formatStartedAt(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString("zh-CN", {hour12: false});
}

async function startWorkspace(workspace) {
    if (state.pendingStarts.has(workspace.id) || state.pendingStops.has(workspace.id)) return;
    const runtimeState = state.runtimeStates.get(workspace.id) || {};
    if (runtimeState.running || runtimeState.starting) return;

    state.pendingStarts.add(workspace.id);
    renderWorkspaceList();
    clearMessage();
    showToast(`正在启动“${workspace.alias}”…`, "info");
    try {
        const instance = await appApi().StartWorkspace(workspace.id);
        showToast(`“${workspace.alias}”已启动，PID ${instance.pid}，端口 ${instance.port}。`, "success");
    } catch (error) {
        showMessage(`启动失败：${errorText(error)}`, "error");
    } finally {
        state.pendingStarts.delete(workspace.id);
        await loadRuntime();
    }
}

async function stopWorkspace(workspace) {
    if (state.pendingStarts.has(workspace.id) || state.pendingStops.has(workspace.id)) return;
    state.pendingStops.add(workspace.id);
    renderWorkspaceList();
    clearMessage();
    showToast(`正在停止“${workspace.alias}”…`, "info");
    try {
        await appApi().StopWorkspace(workspace.id);
        showToast(`“${workspace.alias}”已停止。`, "success");
    } catch (error) {
        showMessage(`停止失败：${errorText(error)}`, "error");
    } finally {
        state.pendingStops.delete(workspace.id);
        await loadRuntime();
    }
}

async function openLogDialog(workspace) {
    state.logWorkspaceId = workspace.id;
    elements.logTitle.textContent = `${workspace.alias} · 运行日志`;
    const runtimeState = state.runtimeStates.get(workspace.id) || {};
    elements.logPath.textContent = runtimeState.log_path || `~/.config/codexpro-plus/logs/${workspace.id}.log`;
    elements.logOutput.textContent = "正在读取日志…";
    openModal(elements.logDialog);
    await refreshLog();
}

async function refreshLog() {
    if (!state.logWorkspaceId) return;
    elements.logRefreshButton.disabled = true;
    elements.logRefreshButton.textContent = "刷新中…";
    try {
        const log = await appApi().GetWorkspaceLog(state.logWorkspaceId);
        elements.logOutput.textContent = log || "暂无日志。首次启动该 Workspace 后会在这里显示 core stdout/stderr 和管理器事件。";
        elements.logOutput.scrollTop = elements.logOutput.scrollHeight;
    } catch (error) {
        elements.logOutput.textContent = `读取日志失败：${errorText(error)}`;
    } finally {
        elements.logRefreshButton.disabled = false;
        elements.logRefreshButton.textContent = "刷新日志";
    }
}

function closeLogDialog() {
    state.logWorkspaceId = null;
    closeModal(elements.logDialog);
}

function selectedPortMode() {
    return document.querySelector('input[name="port-mode"]:checked')?.value || "auto";
}

function setPortMode(mode) {
    const target = document.querySelector(`input[name="port-mode"][value="${mode}"]`);
    if (target) target.checked = true;
    updatePortModeUi();
}

function updatePortModeUi() {
    const manual = selectedPortMode() === "manual";
    elements.manualPortField.classList.toggle("hidden", !manual);
    elements.workspacePort.required = manual;
    elements.workspacePort.disabled = !manual;
}

function openAddDialog() {
    clearMessage();
    clearDialogMessage();
    state.editingId = null;
    elements.workspaceForm.reset();
    elements.dialogTitle.textContent = "添加工作目录";
    elements.workspaceSubmitButton.textContent = "添加";
    elements.workspacePort.value = "";
    elements.workspaceAuthInput.value = "";
    elements.workspaceBashMode.value = "full";
    elements.workspaceWriteMode.value = "workspace";
    elements.workspaceToolMode.value = "full";
    elements.workspaceInheritEnv.checked = true;
    elements.portHelp.textContent = "端口必须在 1-65535 之间。";
    setPortMode("auto");
    openModal(elements.dialog);
    elements.workspaceAlias.focus();
}

function openEditDialog(id) {
    clearMessage();
    clearDialogMessage();
    const workspaces = state.config?.workspaces || [];
    const workspace = workspaces.find((item) => item.id === id);
    if (!workspace) {
        showMessage("未找到要编辑的工作目录，请刷新后重试。", "error");
        return;
    }
    const runtimeState = state.runtimeStates.get(id) || {};
    if (runtimeState.running || runtimeState.starting || state.pendingStarts.has(id) || state.pendingStops.has(id)) {
        showMessage("该工作目录正在启动或运行，请等待启动完成并停止服务后再编辑。", "error");
        return;
    }

    state.editingId = id;
    elements.workspaceForm.reset();
    elements.dialogTitle.textContent = "编辑工作目录";
    elements.workspaceSubmitButton.textContent = "保存修改";
    elements.workspaceAlias.value = workspace.alias || "";
    elements.workspacePath.value = workspace.path || "";
    elements.workspacePort.value = workspace.port ?? "";
    elements.workspaceAuthInput.value = workspace.token || "";
    elements.workspaceBashMode.value = workspace.bash_mode || "full";
    elements.workspaceWriteMode.value = workspace.write_mode || "workspace";
    elements.workspaceToolMode.value = workspace.tool_mode || "full";
    elements.workspaceInheritEnv.checked = workspace.inherit_env !== false;
    elements.workspaceAutoStart.checked = Boolean(workspace.auto_start);
    elements.portHelp.textContent = "默认保留当前端口；选择自动分配后会重新从范围中选择空闲端口。";
    setPortMode("manual");
    openModal(elements.dialog);
    elements.workspaceAlias.focus();
}

function openModal(dialog) {
    if (typeof dialog.showModal === "function") {
        dialog.showModal();
    } else {
        dialog.setAttribute("open", "");
    }
}

function closeModal(dialog) {
    if (typeof dialog.close === "function") {
        dialog.close();
    } else {
        dialog.removeAttribute("open");
    }
}

function closeWorkspaceDialog() {
    state.editingId = null;
    clearDialogMessage();
    closeModal(elements.dialog);
}

function collectWorkspaceInput() {
    const mode = selectedPortMode();
    const port = mode === "manual" ? Number(elements.workspacePort.value) : 0;
    const input = {
        alias: elements.workspaceAlias.value.trim(),
        path: elements.workspacePath.value.trim(),
        port_mode: mode,
        port,
        bash_mode: elements.workspaceBashMode.value,
        write_mode: elements.workspaceWriteMode.value,
        tool_mode: elements.workspaceToolMode.value,
        inherit_env: elements.workspaceInheritEnv.checked,
        auto_start: elements.workspaceAutoStart.checked,
    };
    input["to" + "ken"] = elements.workspaceAuthInput.value.trim();
    return input;
}

function setWorkspaceFormBusy(busy) {
    elements.workspaceSubmitButton.disabled = busy;
    elements.dialogCancelButton.disabled = busy;
    elements.dialogCloseButton.disabled = busy;
    elements.workspaceSubmitButton.textContent = busy
        ? "保存中…"
        : (state.editingId ? "保存修改" : "添加");
}

async function submitWorkspace(event) {
    event.preventDefault();
    if (!elements.workspaceForm.reportValidity()) return;

    clearDialogMessage();
    const input = collectWorkspaceInput();
    setWorkspaceFormBusy(true);
    try {
        if (state.editingId) {
            await appApi().UpdateWorkspace(state.editingId, input);
            closeWorkspaceDialog();
            if (await loadConfig()) {
                await loadRuntime();
                showToast("工作目录已更新。", "success");
            }
        } else {
            await appApi().AddWorkspace(input);
            closeWorkspaceDialog();
            if (await loadConfig()) {
                await loadRuntime();
                showToast("工作目录已添加，并已生成稳定连接 Token。", "success");
            }
        }
    } catch (error) {
        showDialogMessage(`保存失败：${errorText(error)}`);
    } finally {
        setWorkspaceFormBusy(false);
    }
}

async function deleteWorkspace(workspace) {
    const runtimeState = state.runtimeStates.get(workspace.id) || {};
    if (runtimeState.running || runtimeState.starting || state.pendingStarts.has(workspace.id) || state.pendingStops.has(workspace.id)) {
        showMessage("该工作目录正在启动或运行，请等待启动完成并停止服务后再删除。", "error");
        return;
    }
    const confirmed = window.confirm(`确定删除“${workspace.alias}”吗？\n\n只会删除管理器中的配置，不会删除磁盘上的目录和历史日志。`);
    if (!confirmed) return;

    try {
        await appApi().DeleteWorkspace(workspace.id);
        if (await loadConfig()) {
            await loadRuntime();
            showToast(`已删除“${workspace.alias}”。`, "success");
        }
    } catch (error) {
        showMessage(`删除失败：${errorText(error)}`, "error");
    }
}

async function saveDomain(event) {
    event.preventDefault();
    const submitButton = elements.domainForm.querySelector('button[type="submit"]');
    submitButton.disabled = true;
    submitButton.textContent = "保存中…";
    try {
        const config = await appApi().UpdateDomain(elements.copyDomain.value.trim());
        state.config = config;
        await refreshWorkspaceURLs(config?.workspaces || []);
        renderConfig();
        showToast("复制链接域名已保存。", "success");
    } catch (error) {
        showMessage(`保存复制链接域名失败：${errorText(error)}`, "error");
    } finally {
        submitButton.disabled = false;
        submitButton.textContent = "保存域名";
    }
}

async function savePortRange(event) {
    event.preventDefault();
    if (!elements.portRangeForm.reportValidity()) return;

    const submitButton = elements.portRangeForm.querySelector('button[type="submit"]');
    submitButton.disabled = true;
    submitButton.textContent = "保存中…";
    try {
        const config = await appApi().UpdatePortRange({
            start: Number(elements.portStart.value),
            end: Number(elements.portEnd.value),
        });
        state.config = config;
        renderConfig();
        showToast("自动端口范围已保存。", "success");
    } catch (error) {
        showMessage(`保存端口范围失败：${errorText(error)}`, "error");
    } finally {
        submitButton.disabled = false;
        submitButton.textContent = "保存端口范围";
    }
}

async function changeManagerAutoStart() {
    const enabled = elements.managerAutoStart.checked;
    elements.managerAutoStart.disabled = true;
    try {
        state.manager = await appApi().SetManagerAutoStart(enabled);
        renderManagerSettings();
        showToast(enabled ? "已开启 Windows 开机自启。" : "已关闭 Windows 开机自启。", "success");
    } catch (error) {
        elements.managerAutoStart.checked = !enabled;
        elements.managerAutoStart.disabled = false;
        showMessage(`修改开机自启失败：${errorText(error)}`, "error");
    }
}

elements.addButton.addEventListener("click", openAddDialog);
elements.emptyAddButton.addEventListener("click", openAddDialog);
elements.dialogCloseButton.addEventListener("click", closeWorkspaceDialog);
elements.dialogCancelButton.addEventListener("click", closeWorkspaceDialog);
elements.workspaceForm.addEventListener("submit", submitWorkspace);
elements.domainForm.addEventListener("submit", saveDomain);
elements.portRangeForm.addEventListener("submit", savePortRange);
elements.managerAutoStart.addEventListener("change", changeManagerAutoStart);
elements.logCloseButton.addEventListener("click", closeLogDialog);
elements.logDoneButton.addEventListener("click", closeLogDialog);
elements.logRefreshButton.addEventListener("click", refreshLog);
document.querySelectorAll('input[name="port-mode"]').forEach((radio) => {
    radio.addEventListener("change", updatePortModeUi);
});
elements.dialog.addEventListener("click", (event) => {
    if (event.target === elements.dialog) closeWorkspaceDialog();
});
elements.logDialog.addEventListener("click", (event) => {
    if (event.target === elements.logDialog) closeLogDialog();
});
elements.dialog.addEventListener("close", () => {
    state.editingId = null;
    clearDialogMessage();
});
elements.logDialog.addEventListener("close", () => {
    state.logWorkspaceId = null;
});

async function initialise() {
    updatePortModeUi();
    await loadConfig();
    await Promise.all([loadManagerSettings(), loadRuntime()]);
    window.setInterval(loadRuntime, 1500);
    window.setInterval(loadManagerSettings, 10000);
}

initialise();
