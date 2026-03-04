async function fetchJSON(url, opts = {}) {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch (e) {
    throw new Error(`JSON 解析失败: ${text}`);
  }
  if (!res.ok) {
    const msg = (data && data.error) ? data.error : res.statusText;
    throw new Error(msg);
  }
  return data;
}

function el(id) { return document.getElementById(id); }
function direction() {
  const v = document.querySelector("input[name=direction]:checked");
  return v ? v.value : "ollama_to_lmstudio";
}

function badgeEl(status) {
  const cls = status === "ready" || status === "already_synced" || status === "already_linked"
    ? "ok" : (status === "conflict" || status === "symlink_mismatch" ? "bad" : "warn");
  const span = document.createElement("span");
  span.className = `badge ${cls}`;
  span.textContent = status;
  return span;
}

let lastScanItems = [];
let currentItems = [];
let viewMode = "all"; // all | supported | unsupported
let selectionById = {};
let modelNameById = {};
let busy = false;

function setActionHint(msg) {
  el("actionHint").textContent = msg || "";
}

function setBusy(v) {
  busy = !!v;
  const buttons = ["saveConfig", "reloadStatus", "scan", "apply", "selectAll", "selectNone"];
  for (const id of buttons) {
    const node = el(id);
    if (node) node.disabled = busy;
  }
  const radios = document.querySelectorAll("input[name=direction]");
  for (const r of radios) r.disabled = busy;
}

function setTabsVisible(visible) {
  const tabs = el("lmstudioTabs");
  if (!tabs) return;
  tabs.classList.toggle("hidden", !visible);
}

function setViewMode(mode) {
  viewMode = mode;
  const ids = ["tabSupported", "tabUnsupported", "tabAll"];
  for (const id of ids) {
    const b = el(id);
    if (!b) continue;
    b.classList.toggle("active", id === "tab" + mode[0].toUpperCase() + mode.slice(1));
  }
  renderFiltered();
}

function updateTabs(items, dir) {
  if (dir !== "lmstudio_to_ollama") return;
  const total = items.length;
  const unsupported = items.filter(i => i.status === "unsupported").length;
  const supported = total - unsupported;

  el("tabSupported").textContent = `可同步 (${supported})`;
  el("tabUnsupported").textContent = `不支持 (${unsupported})`;
  el("tabAll").textContent = `全部 (${total})`;
}

function filterItems(items, dir) {
  if (dir !== "lmstudio_to_ollama") return items;
  if (viewMode === "supported") return items.filter(i => i.status !== "unsupported");
  if (viewMode === "unsupported") return items.filter(i => i.status === "unsupported");
  return items;
}

function updateScanHint(dir) {
  const total = lastScanItems.length;
  const displayed = currentItems.length;
  const selected = lastScanItems.filter(i => i.selectable !== false && selectionById[i.id]).length;
  if (dir === "lmstudio_to_ollama") {
    el("scanHint").textContent = `显示 ${displayed}/${total} 条 | 已选 ${selected} 条`;
    return;
  }
  el("scanHint").textContent = `共 ${total} 条 | 已选 ${selected} 条`;
}

function renderFiltered() {
  const dir = direction();
  const filtered = filterItems(lastScanItems, dir);
  renderItems(filtered, dir);
  updateScanHint(dir);
}

function renderItems(items, dir) {
  currentItems = items;
  const tbody = el("items");
  tbody.innerHTML = "";

  for (const item of items) {
    const tr = document.createElement("tr");
    const disabled = item.selectable === false;

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    const checked = selectionById[item.id];
    checkbox.checked = checked === undefined ? !!item.selected : !!checked;
    checkbox.disabled = disabled;
    checkbox.dataset.id = item.id;
    checkbox.addEventListener("change", () => {
      selectionById[item.id] = checkbox.checked;
      updateScanHint(dir);
    });

    const td0 = document.createElement("td");
    td0.appendChild(checkbox);

    const td1 = document.createElement("td");
    const labelDiv = document.createElement("div");
    labelDiv.textContent = item.label;
    const detailDiv = document.createElement("div");
    detailDiv.className = "hint";
    detailDiv.textContent = item.detail || "";
    td1.appendChild(labelDiv);
    td1.appendChild(detailDiv);
    if (item.message) {
      const errorDiv = document.createElement("div");
      errorDiv.className = "hint";
      errorDiv.textContent = item.message;
      td1.appendChild(errorDiv);
    }

    const td2 = document.createElement("td");
    td2.appendChild(badgeEl(item.status));

    const td3 = document.createElement("td");
    if (dir === "lmstudio_to_ollama") {
      const input = document.createElement("input");
      input.type = "text";
      input.value = modelNameById[item.id] || item.modelName || "";
      input.disabled = disabled;
      input.dataset.id = item.id;
      input.className = "modelName";
      input.addEventListener("input", () => {
        modelNameById[item.id] = input.value;
      });
      td3.appendChild(input);
    } else {
      td3.textContent = "-";
    }

    tr.appendChild(td0);
    tr.appendChild(td1);
    tr.appendChild(td2);
    tr.appendChild(td3);
    tbody.appendChild(tr);
  }
}

async function loadConfigAndStatus() {
  const [cfg, st] = await Promise.all([
    fetchJSON("/api/config"),
    fetchJSON("/api/status"),
  ]);

  el("lmstudioModelsDir").value = cfg.lmStudioModelsDir || "";
  el("ollamaModelsDir").value = cfg.ollamaModelsDir || "";
  el("ollamaHost").value = cfg.ollamaHost || "";
  el("ollamaBin").value = cfg.ollamaBin || "";
  el("allowFixSymlink").checked = !!cfg.allowFixSymlink;
  el("allowRecreateBrokenSymlink").checked = !!cfg.allowRecreateBrokenSymlink;
  el("allowReplaceExistingBlob").checked = !!cfg.allowReplaceExistingBlob;

  const bin = st.binary && st.binary.found ? `${st.binary.path} (${st.binary.source})` : "未找到";
  const srv = st.server && st.server.reachable ? "可连接" : `不可连接${st.server && st.server.error ? `: ${st.server.error}` : ""}`;
  el("status").textContent = `Ollama: ${bin} | 服务: ${srv}`;
}

async function saveConfig() {
  el("saveHint").textContent = "";
  const cfg = {
    lmStudioModelsDir: el("lmstudioModelsDir").value,
    ollamaModelsDir: el("ollamaModelsDir").value,
    ollamaHost: el("ollamaHost").value,
    ollamaBin: el("ollamaBin").value,
    allowFixSymlink: el("allowFixSymlink").checked,
    allowRecreateBrokenSymlink: el("allowRecreateBrokenSymlink").checked,
    allowReplaceExistingBlob: el("allowReplaceExistingBlob").checked,
  };

  await fetchJSON("/api/config", {
    method: "PUT",
    body: JSON.stringify(cfg),
  });
  el("saveHint").textContent = "已保存";
  await loadConfigAndStatus();
}

async function scan() {
  el("scanHint").textContent = "扫描中…";
  const dir = direction();
  setBusy(true);
  try {
    const res = await fetchJSON("/api/scan", {
      method: "POST",
      body: JSON.stringify({ direction: dir }),
    });
    lastScanItems = res.items || [];
    selectionById = {};
    modelNameById = {};
    for (const it of lastScanItems) {
      selectionById[it.id] = !!it.selected;
      if (it.modelName) modelNameById[it.id] = it.modelName;
    }

    setTabsVisible(dir === "lmstudio_to_ollama");
    updateTabs(lastScanItems, dir);
    if (dir === "lmstudio_to_ollama") setViewMode("supported");
    else setViewMode("all");
  } finally {
    setBusy(false);
  }
}

function setSelection(value) {
  for (const it of lastScanItems) {
    if (it.selectable === false) {
      selectionById[it.id] = false;
      continue;
    }
    selectionById[it.id] = !!value;
  }
  renderFiltered();
}

async function apply() {
  el("results").textContent = "";
  setActionHint("");
  const dir = direction();

  const selectedIDs = lastScanItems
    .filter(i => i.selectable !== false && selectionById[i.id])
    .map(i => i.id);
  if (selectedIDs.length === 0) {
    setActionHint("未选择任何模型");
    return;
  }

  const payload = { direction: dir, selected: selectedIDs, imports: [] };
  if (dir === "lmstudio_to_ollama") {
    const byId = {};
    for (const it of lastScanItems) byId[it.id] = it;
    for (const id of selectedIDs) {
      const item = byId[id];
      payload.imports.push({
        ggufPath: item.ggufPath,
        modelName: modelNameById[id] || item.modelName || "",
      });
    }
  }

  setBusy(true);
  const started = Date.now();
  setActionHint(`同步中…（${selectedIDs.length} 项）`);
  try {
    const res = await fetchJSON("/api/apply", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    el("results").textContent = JSON.stringify(res, null, 2);
    const ms = Date.now() - started;
    if (res && res.error) {
      setActionHint(`同步失败（耗时 ${(ms / 1000).toFixed(1)}s）`);
    } else {
      setActionHint(`同步完成（耗时 ${(ms / 1000).toFixed(1)}s）`);
    }
    await scan();
  } catch (e) {
    setActionHint(`同步失败：${e.message}`);
    throw e;
  } finally {
    setBusy(false);
  }
}

document.addEventListener("DOMContentLoaded", async () => {
  el("saveConfig").addEventListener("click", () => saveConfig().catch(e => el("saveHint").textContent = e.message));
  el("reloadStatus").addEventListener("click", () => loadConfigAndStatus().catch(e => el("status").textContent = e.message));
  el("scan").addEventListener("click", () => scan().catch(e => el("scanHint").textContent = e.message));
  el("apply").addEventListener("click", () => apply().catch(e => el("results").textContent = e.message));
  el("selectAll").addEventListener("click", () => setSelection(true));
  el("selectNone").addEventListener("click", () => setSelection(false));
  el("tabSupported").addEventListener("click", () => setViewMode("supported"));
  el("tabUnsupported").addEventListener("click", () => setViewMode("unsupported"));
  el("tabAll").addEventListener("click", () => setViewMode("all"));

  await loadConfigAndStatus().catch(e => el("status").textContent = e.message);
  await scan().catch(e => el("scanHint").textContent = e.message);
});
