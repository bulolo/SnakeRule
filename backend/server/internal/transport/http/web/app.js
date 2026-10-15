const sample = {
  id: "order-risk",
  name: "订单风险规则",
  version: "1.0.0",
  description: "演示 all 条件组和事实路径",
  enabled: true,
  rules: [
    {
      id: "large-new-order",
      name: "大额新客订单",
      description: "大额订单且来自新客户时进入人工复核",
      priority: 100,
      enabled: true,
      conditions: {
        all: [
          { fact: "order.amount", operator: "greaterThan", value: 10000 },
          { fact: "customer.isNew", operator: "equal", value: true }
        ]
      },
      actions: [{ type: "tag", config: { label: "MANUAL_REVIEW" } }]
    },
    {
      id: "vip-order",
      name: "VIP 订单标记",
      priority: 50,
      enabled: true,
      conditions: { fact: "customer.level", operator: "equal", value: "VIP" },
      actions: [{ type: "tag", config: { label: "VIP" } }]
    }
  ]
};

const $ = (id) => document.getElementById(id);
const picker = $("ruleset-picker");
const editor = $("ruleset-json");
const facts = $("facts-json");
const state = { currentID: "" };

function showError(id, message = "") { $(id).textContent = message; }
function pretty(value) { return JSON.stringify(value, null, 2); }
function setEditor(value) { editor.value = pretty(value); state.currentID = value.id || ""; }

async function request(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { ...(options.body ? { "Content-Type": "application/json" } : {}), ...options.headers }
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data?.error?.message || `请求失败 (${response.status})`);
  }
  return response.status === 204 ? null : response.json();
}

async function refreshRulesets(selectID = state.currentID) {
  const data = await request("/api/v1/rulesets");
  const items = data.items || [];
  $("ruleset-count").textContent = `${items.length} 个`;
  picker.replaceChildren();
  const list = $("ruleset-list");
  list.replaceChildren();
  if (!items.length) {
    list.innerHTML = '<div class="empty">还没有规则集，编辑左侧 JSON 并保存即可开始。</div>';
    setEditor(sample);
    return;
  }
  for (const item of items) {
    const option = document.createElement("option");
    option.value = item.id;
    option.textContent = `${item.name} · ${item.id}`;
    picker.append(option);
    const card = document.createElement("button");
    card.type = "button";
    card.className = `ruleset-card${item.id === selectID ? " selected" : ""}`;
    card.innerHTML = `<div class="ruleset-card-top"><span class="ruleset-name"></span><span class="${item.enabled ? "enabled" : "disabled"}">${item.enabled ? "已启用" : "已停用"}</span></div><div class="ruleset-meta"></div>`;
    card.querySelector(".ruleset-name").textContent = item.name;
    card.querySelector(".ruleset-meta").textContent = `${item.id} · ${item.rules?.length || 0} 条规则 · v${item.version}`;
    card.addEventListener("click", () => loadRuleset(item.id));
    list.append(card);
  }
  const chosen = items.find((item) => item.id === selectID) || items[0];
  picker.value = chosen.id;
  await loadRuleset(chosen.id, false);
}

async function loadRuleset(id, updateList = true) {
  if (!id) return;
  const item = await request(`/api/v1/rulesets/${encodeURIComponent(id)}`);
  setEditor(item);
  if (updateList) await refreshRulesets(id);
}

$("refresh").addEventListener("click", async () => {
  try { await refreshRulesets(); showError("editor-error"); }
  catch (error) { showError("editor-error", error.message); }
});

picker.addEventListener("change", async () => {
  try { await loadRuleset(picker.value); showError("editor-error"); }
  catch (error) { showError("editor-error", error.message); }
});

$("new-rule").addEventListener("click", () => {
  const fresh = structuredClone(sample);
  fresh.id = `ruleset-${Date.now()}`;
  fresh.name = "新规则集";
  setEditor(fresh);
  showError("editor-error");
});

$("save").addEventListener("click", async () => {
  showError("editor-error");
  try {
    const item = JSON.parse(editor.value);
    if (!item.id) throw new Error("规则集 id 必填");
    await request(`/api/v1/rulesets/${encodeURIComponent(item.id)}`, { method: "PUT", body: JSON.stringify(item) });
    state.currentID = item.id;
    await refreshRulesets(item.id);
    showError("save-state", "已保存到当前服务");
    setTimeout(() => { $("save-state").textContent = "内存模式 · 重启后数据会重置"; }, 2200);
  } catch (error) { showError("editor-error", error.message); }
});

$("evaluate").addEventListener("click", async () => {
  showError("run-error");
  try {
    const item = JSON.parse(editor.value);
    const input = JSON.parse(facts.value);
    if (!item.id) throw new Error("规则集 id 必填");
    await request(`/api/v1/rulesets/${encodeURIComponent(item.id)}`, { method: "PUT", body: JSON.stringify(item) });
    state.currentID = item.id;
    await refreshRulesets(item.id);
    const result = await request("/api/v1/engine/evaluate", {
      method: "POST",
      body: JSON.stringify({ rulesetId: item.id, facts: input })
    });
    $("result").textContent = pretty(result);
    const badge = $("result-badge");
    badge.textContent = result.matched ? `命中 ${result.hitRules.length} 条` : "没有命中";
    badge.className = `result-badge ${result.matched ? "hit" : "miss"}`;
  } catch (error) { showError("run-error", error.message); }
});

facts.value = pretty({ order: { amount: 12800 }, customer: { isNew: true, level: "STANDARD" } });
setEditor(sample);
request("/api/v1/meta").then((meta) => { $("app-version").textContent = `v${meta.version}`; }).catch(() => {});
refreshRulesets().catch((error) => showError("editor-error", error.message));
