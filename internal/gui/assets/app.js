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

let currentItems = [];

function renderItems(items, dir) {
  currentItems = items;
  const tbody = el("items");
  tbody.innerHTML = "";

  for (const item of items) {
    const tr = document.createElement("tr");
    const disabled = item.selectable === false;

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = !!item.selected;
    checkbox.disabled = disabled;
    checkbox.dataset.id = item.id;

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
      input.value = item.modelName || "";
      input.disabled = disabled;
      input.dataset.id = item.id;
      input.className = "modelName";
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
  el("scanHint").textContent = "";
  el("results").textContent = "";
  const dir = direction();
  const res = await fetchJSON("/api/scan", {
    method: "POST",
    body: JSON.stringify({ direction: dir }),
  });
  renderItems(res.items || [], dir);
  el("scanHint").textContent = `共 ${res.items ? res.items.length : 0} 条`;
}

function setSelection(value) {
  const boxes = document.querySelectorAll("#items input[type=checkbox]");
  for (const b of boxes) {
    if (!b.disabled) b.checked = value;
  }
}

async function apply() {
  el("results").textContent = "";
  const dir = direction();

  const selectedIDs = [];
  const boxes = document.querySelectorAll("#items input[type=checkbox]");
  for (const b of boxes) {
    if (b.checked && !b.disabled) selectedIDs.push(b.dataset.id);
  }

  const payload = { direction: dir, selected: selectedIDs, imports: [] };
  if (dir === "lmstudio_to_ollama") {
    const inputs = document.querySelectorAll("#items input.modelName");
    const mapName = {};
    for (const i of inputs) mapName[i.dataset.id] = i.value;

    for (const item of currentItems) {
      if (!selectedIDs.includes(item.id)) continue;
      payload.imports.push({
        ggufPath: item.ggufPath,
        modelName: mapName[item.id] || item.modelName || "",
      });
    }
  }

  const res = await fetchJSON("/api/apply", {
    method: "POST",
    body: JSON.stringify(payload),
  });

  el("results").textContent = JSON.stringify(res, null, 2);
  await scan();
}

document.addEventListener("DOMContentLoaded", async () => {
  el("saveConfig").addEventListener("click", () => saveConfig().catch(e => el("saveHint").textContent = e.message));
  el("reloadStatus").addEventListener("click", () => loadConfigAndStatus().catch(e => el("status").textContent = e.message));
  el("scan").addEventListener("click", () => scan().catch(e => el("scanHint").textContent = e.message));
  el("apply").addEventListener("click", () => apply().catch(e => el("results").textContent = e.message));
  el("selectAll").addEventListener("click", () => setSelection(true));
  el("selectNone").addEventListener("click", () => setSelection(false));

  await loadConfigAndStatus().catch(e => el("status").textContent = e.message);
  await scan().catch(e => el("scanHint").textContent = e.message);
});
