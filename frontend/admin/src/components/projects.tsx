import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { snakeRuleApi, type InvocationLog, type Rule, type RuleAlert, type Ruleset } from "../lib/sdk/generated";

export type Project = {
  id: string; name: string; slug: string; description?: string; icon?: string;
};
const projectIcons = ["◈", "🛒", "🛡️", "📦", "⚙️", "💳"];
type APIKey = { id: string; name: string; prefix: string; type?: "apikey" | "jwt"; created_at: string };

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, { ...init, headers: { ...(init?.body ? { "Content-Type": "application/json" } : {}), ...init?.headers } });
  const raw = await response.text();
  let body: any = {};
  try { body = raw ? JSON.parse(raw) : {}; } catch { throw new Error(`接口返回非 JSON 响应（HTTP ${response.status}）：${raw.slice(0, 120)}`); }
  if (!response.ok) throw new Error(body?.error?.message ?? `请求失败（HTTP ${response.status}）`);
  return body as T;
}

function effectiveRuleAlert(ruleset: Ruleset, rule: Rule): RuleAlert | undefined {
  return rule.alert ?? ruleset.alert;
}

export function ProjectsPanel({
  projects, projectID, activeWorkspaceID, alertChannels, projectDetailOpen, setProjectDetailOpen, onRefresh, onSelect, onCreateRuleset, onDeleteRuleset, onOpenRuleset,
}: {
  projects: Project[]; projectID: string; activeWorkspaceID: string; alertChannels: { id: string; name: string }[]; projectDetailOpen: boolean; setProjectDetailOpen: (open: boolean) => void;
  onRefresh: () => Promise<void>; onSelect: (id: string) => Promise<void>; onCreateRuleset: (name: string) => Promise<Ruleset>; onDeleteRuleset: (ruleset: Ruleset) => Promise<void>; onOpenRuleset: (ruleset: Ruleset) => void;
}) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [editingID, setEditingID] = useState("");
  const [editName, setEditName] = useState("");
  const [editSlug, setEditSlug] = useState("");
  const [editIcon, setEditIcon] = useState("◈");
  const [rulesets, setRulesets] = useState<Ruleset[]>([]);
  const [selectedRulesetID, setSelectedRulesetID] = useState("");
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [invocationLogs, setInvocationLogs] = useState<InvocationLog[]>([]);
  const [invocationLogsTotal, setInvocationLogsTotal] = useState(0);
  const [invocationPage, setInvocationPage] = useState(1);
  const [invocationLogsOpen, setInvocationLogsOpen] = useState(false);
  const [invocationLogsBusy, setInvocationLogsBusy] = useState(false);
  const [keyName, setKeyName] = useState("服务端调用");
  const [keyType, setKeyType] = useState<"apikey" | "jwt">("apikey");
  const [newSecret, setNewSecret] = useState("");
  const [busy, setBusy] = useState(false);
  const [toggleBusy, setToggleBusy] = useState("");
  const [, setError] = useState("");
  const [confirmProjectDelete, setConfirmProjectDelete] = useState("");
  const [projectModalOpen, setProjectModalOpen] = useState(false);
  const [projectTemplate, setProjectTemplate] = useState<"" | "ecommerce">("");
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [keyCreateMode, setKeyCreateMode] = useState(false);
  const [rulesetQuery, setRulesetQuery] = useState("");
  const [rulesetModalOpen, setRulesetModalOpen] = useState(false);
  const [guideOpen, setGuideOpen] = useState(false);
  const [rulesetName, setRulesetName] = useState("");
  const [confirmRulesetDelete, setConfirmRulesetDelete] = useState<Ruleset | null>(null);
  const [alertRuleset, setAlertRuleset] = useState<Ruleset | null>(null);
  const [defaultAlertEnabled, setDefaultAlertEnabled] = useState(false);
  const [defaultAlertChannelIDs, setDefaultAlertChannelIDs] = useState<string[]>([]);
  const [applyDefaultToAll, setApplyDefaultToAll] = useState(true);
  const visibleRulesets = rulesets.filter((item) => `${item.name} ${item.id}`.toLowerCase().includes(rulesetQuery.toLowerCase()));
  const selectedRuleset = visibleRulesets.find((item) => item.id === selectedRulesetID) ?? visibleRulesets[0];
  const activeProject = projects.find((item) => item.id === projectID);
  const previousWorkspaceID = useRef(activeWorkspaceID);
  useEffect(() => { if (previousWorkspaceID.current !== activeWorkspaceID) { previousWorkspaceID.current = activeWorkspaceID; setProjectDetailOpen(false); } }, [activeWorkspaceID, setProjectDetailOpen]);
  const refreshDetail = useCallback(async (): Promise<Ruleset[]> => {
    if (!activeProject) return [];
    let loadedRulesets: Ruleset[] = [];
    try {
      const result = await api<{ items: Ruleset[] }>("/api/v1/rulesets");
      loadedRulesets = result.items ?? [];
      setRulesets(loadedRulesets);
    } catch (e) { toast.error("读取项目规则集失败", { description: e instanceof Error ? e.message : "请求失败" }); }
    try {
      const result = await api<{ items: APIKey[] }>(`/api/v1/projects/${activeProject.id}/api-keys`);
      setKeys(result.items ?? []);
    } catch (e) { toast.error("读取项目 API Key 失败", { description: e instanceof Error ? e.message : "请求失败" }); }
    return loadedRulesets;
  }, [activeProject]);
  useEffect(() => { void refreshDetail(); }, [refreshDetail]);

  async function createProject(template: "" | "ecommerce") {
    setBusy(true); setError("");
    try {
      const result = await api<Project>("/api/v1/projects", { method: "POST", body: JSON.stringify({
        name: template ? "电商订单风控演示" : name.trim(), slug: slug.trim(), description: template ? "包含订单风控规则和项目 API Key 的完整演示项目。" : description.trim(), template,
      }) });
      await onRefresh(); await onSelect(result.id);
      setName(""); setSlug(""); setDescription("");
      setProjectModalOpen(false); setProjectDetailOpen(true);
      toast.success(template ? "电商演示项目已创建，内置订单风控规则集。" : "项目已创建。");
    } catch (e) { toast.error("创建项目失败", { description: e instanceof Error ? e.message : "请求失败" }); }
    finally { setBusy(false); }
  }
  async function saveProject() {
    const target = projects.find((project) => project.id === editingID);
    if (!target) return;
    setBusy(true); setError("");
    try {
      await api(`/api/v1/projects/${target.id}`, { method: "PUT", body: JSON.stringify({ name: editName.trim(), slug: editSlug.trim(), description: description.trim(), icon: editIcon }) });
      await onRefresh(); setEditingID(""); toast.success("项目资料已更新。");
    } catch (e) { toast.error("更新项目失败", { description: e instanceof Error ? e.message : "请求失败" }); }
    finally { setBusy(false); }
  }
  async function createKey() {
    if (!activeProject) return;
    setBusy(true); setError(""); setNewSecret("");
    try {
      const result = await api<{ item: APIKey; secret?: string; token?: string }>(`/api/v1/projects/${activeProject.id}/api-keys`, { method: "POST", body: JSON.stringify({ name: keyName.trim(), type: keyType }) });
      setNewSecret(result.token ?? result.secret ?? ""); setKeyCreateMode(true); await refreshDetail();
    } catch (e) { toast.error(`创建${keyType === "jwt" ? " JWT" : " API Key"}失败`, { description: e instanceof Error ? e.message : "请求失败" }); }
    finally { setBusy(false); }
  }
  async function revokeKey(key: APIKey) {
    setBusy(true); setError("");
    try { await api(`/api/v1/projects/${projectID}/api-keys/${key.id}`, { method: "DELETE" }); await refreshDetail(); }
    catch (e) { toast.error("撤销 API Key 失败", { description: e instanceof Error ? e.message : "请求失败" }); }
    finally { setBusy(false); }
  }
  async function openInvocationLogs(page = 1) {
    setInvocationLogsOpen(true);
    setInvocationLogsBusy(true);
    setInvocationPage(page);
    try {
      const result = await snakeRuleApi.listProjectInvocations(projectID, { page, size: 20, is_pager: 1 });
      setInvocationLogs(result.list ?? []);
      setInvocationLogsTotal(result.pagination?.total ?? 0);
    } catch (e) {
      toast.error("读取调用日志失败", { description: e instanceof Error ? e.message : "请求失败" });
    } finally { setInvocationLogsBusy(false); }
  }
  async function removeProject(project: Project) {
    setBusy(true); setError("");
    try { await api(`/api/v1/projects/${project.id}`, { method: "DELETE" }); setConfirmProjectDelete(""); await onRefresh(); setProjectDetailOpen(false); toast.success("项目已删除。"); }
    catch (e) { toast.error("删除项目失败", { description: e instanceof Error ? e.message : "请求失败" }); }
    finally { setBusy(false); }
  }
  async function toggleRuleset(set: Ruleset) {
    const key = `ruleset:${set.id}`;
    setToggleBusy(key); setError("");
    try {
      await api(`/api/v1/rulesets/${encodeURIComponent(set.id)}`, {
        method: "PUT", body: JSON.stringify({ ...set, enabled: !set.enabled }),
      });
      await refreshDetail();
      toast.success(`规则集已${set.enabled ? "停用" : "启用"}`);
    } catch (e) { const message = e instanceof Error ? e.message : "更新规则集状态失败"; setError(message); toast.error("更新规则集状态失败", { description: message }); }
    finally { setToggleBusy(""); }
  }
  async function toggleRule(set: Ruleset, ruleID: string) {
    const key = `rule:${set.id}:${ruleID}`;
    setToggleBusy(key); setError("");
    try {
      const updated = { ...set, rules: set.rules.map((rule) => rule.id === ruleID ? { ...rule, enabled: !rule.enabled } : rule) };
      await api(`/api/v1/rulesets/${encodeURIComponent(set.id)}`, { method: "PUT", body: JSON.stringify(updated) });
      await refreshDetail();
      const rule = set.rules.find((item) => item.id === ruleID);
      toast.success(`规则已${rule?.enabled ? "停用" : "启用"}`);
    } catch (e) { const message = e instanceof Error ? e.message : "更新规则状态失败"; setError(message); toast.error("更新规则状态失败", { description: message }); }
    finally { setToggleBusy(""); }
  }
  function openRulesetAlert(set: Ruleset) {
    setAlertRuleset(set);
    setDefaultAlertEnabled(set.alert?.enabled ?? false);
    setDefaultAlertChannelIDs(set.alert?.channel_ids ?? []);
    setApplyDefaultToAll(true);
  }
  async function saveRulesetAlert() {
    if (!alertRuleset) return;
    setBusy(true); setError("");
    try {
      const updated: Ruleset = {
        ...alertRuleset,
        alert: defaultAlertEnabled ? { enabled: true, channel_ids: defaultAlertChannelIDs } : undefined,
        rules: alertRuleset.rules.map((rule) => applyDefaultToAll ? { ...rule, alert: undefined } : rule),
      };
      await api(`/api/v1/rulesets/${encodeURIComponent(updated.id)}`, { method: "PUT", body: JSON.stringify(updated) });
      await refreshDetail();
      setAlertRuleset(null);
      toast.success(applyDefaultToAll ? "默认通知已保存并应用到全部规则" : "规则集默认通知已保存");
    } catch (e) {
      const message = e instanceof Error ? e.message : "保存规则集通知失败";
      setError(message); toast.error("保存规则集通知失败", { description: message });
    } finally { setBusy(false); }
  }
  async function createRuleset() {
    const title = rulesetName.trim();
    if (!title) return;
    setBusy(true);
    try {
      const created = await onCreateRuleset(title);
      const loaded = await refreshDetail();
      if (!loaded.some((item) => item.id === created.id)) {
        setRulesets((items) => items.some((item) => item.id === created.id) ? items : [...items, created]);
      }
      setSelectedRulesetID(created.id);
      setRulesetQuery("");
      setRulesetModalOpen(false);
      setRulesetName("");
      toast.success("规则集已创建，可点击设计图标添加规则。");
    } catch (e) {
      toast.error("创建规则集失败", { description: e instanceof Error ? e.message : "请求失败" });
    } finally { setBusy(false); }
  }
  const guideRequest = `curl -X POST http://localhost:8080/api/v1/engine/evaluate \\
  -H 'Content-Type: application/json' \\
  -H 'X-API-Key: YOUR_PROJECT_API_KEY' \\
  -d '{"rulesetId":"${selectedRuleset?.id ?? "your-ruleset-id"}","facts":{"order":{"amount":12800},"customer":{"isNew":true}}}'`;
  async function copyGuideRequest() {
    try { await navigator.clipboard.writeText(guideRequest); toast.success("调用示例已复制"); }
    catch { toast.error("复制失败", { description: "请检查浏览器剪贴板权限。" }); }
  }
  return <>
    <section className={`title-row project-page-heading ${projectDetailOpen ? "is-detail" : ""}`}><div><small className="eyebrow">PROJECTS</small><h1>{projectDetailOpen && activeProject ? activeProject.name : "项目"}</h1><p>{projectDetailOpen ? (activeProject?.description || `项目标识：${activeProject?.slug ?? ""}`) : "工作空间内的项目列表。选择项目进入项目空间并管理项目规则。"}</p></div>{!projectDetailOpen && <div className="project-title-actions"><button className="secondary" disabled={busy} onClick={() => { setProjectTemplate("ecommerce"); setProjectModalOpen(true); }}>创建演示项目</button><button className="primary" onClick={() => { setProjectTemplate(""); setProjectModalOpen(true); }}>＋ 新建项目</button></div>}</section>
    {!projectDetailOpen && <section className="project-card-grid">{projects.map((project) => <article className="project-card" key={project.id} role="button" tabIndex={0} onClick={(event) => { if ((event.target as HTMLElement).closest("button")) return; void onSelect(project.id).then(() => setProjectDetailOpen(true)); }} onKeyDown={(event) => { if (event.target === event.currentTarget && (event.key === "Enter" || event.key === " ")) { event.preventDefault(); void onSelect(project.id).then(() => setProjectDetailOpen(true)); } }}><div className="project-card-top"><span className="project-mark">{project.icon || project.name.slice(0,1)}</span><div className="project-card-actions"><button className="project-edit-icon" aria-label={`编辑项目 ${project.name}`} title="编辑项目" disabled={busy} onClick={(event) => { event.stopPropagation(); setEditName(project.name); setEditSlug(project.slug); setDescription(project.description ?? ""); setEditIcon(project.icon || "◈"); setEditingID(project.id); }}><svg viewBox="0 0 20 20" aria-hidden="true"><path d="m12.8 4.2 3 3M4 16l3.7-.8 8.7-8.7a2.1 2.1 0 0 0-3-3l-8.7 8.7L4 16Z"/></svg></button>{project.id !== "default" && <button className="project-delete-icon" aria-label={`删除项目 ${project.name}`} title="删除项目" disabled={busy} onClick={(event) => { event.stopPropagation(); setConfirmProjectDelete(project.id); }}><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4.5 6h11M8 6V4.5h4V6m2.5 0-.7 10h-7L6 6m2.5 2.5v5m3-5v5"/></svg></button>}</div></div><h3>{project.name}</h3><p>{project.description || "暂无项目说明"}</p><code>{project.slug}</code><div className="project-card-footer"><span>项目规则</span><b>进入项目　→</b></div></article>)}</section>}
    {activeProject && <>
      {projectDetailOpen && <div className="project-detail-toolbar"><button className="quiet" onClick={() => setProjectDetailOpen(false)}>← 返回项目列表</button><div><button className="secondary" onClick={() => void openInvocationLogs()}>调用日志</button><button className="secondary" onClick={() => { setEditName(activeProject.name); setEditSlug(activeProject.slug); setDescription(activeProject.description ?? ""); setEditIcon(activeProject.icon || "◈"); setEditingID(activeProject.id); }}>编辑项目</button><button className="project-icon-action" aria-label="配置项目鉴权" title="配置项目鉴权" onClick={() => { setNewSecret(""); setKeyCreateMode(false); setKeyModalOpen(true); }}><svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="7" cy="12" r="3.2"/><path d="m9.3 9.8 6.2-6.2 1.8 1.8-1.8 1.8-1.4-1.4-1.5 1.5 1.4 1.4-1.7 1.7"/></svg></button></div></div>}
      {projectDetailOpen && <>
      <section className="list-panel project-detail-panel"><div className="project-rules-toolbar"><div><b>项目规则</b><small>{rulesets.length} 个规则集 · {rulesets.reduce((count, item) => count + item.rules.length, 0)} 条规则 · {rulesets.filter((item) => item.enabled).length} 个启用</small></div><div><label className="project-rules-search"><span>⌕</span><input value={rulesetQuery} onChange={(event) => setRulesetQuery(event.target.value)} placeholder="搜索规则集" /></label><button className="primary" onClick={() => { setRulesetName(""); setRulesetModalOpen(true); }}>＋ 新建规则集</button><button className="secondary project-guide-button" onClick={() => setGuideOpen(true)}><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 3.5h8.5L16 7v9.5H4zM12 3.5V7h4M7 10h6M7 13h6"/></svg>使用指南</button></div></div>
        {!rulesets.length ? <div className="table-empty"><b>此项目还没有规则集</b><small>创建规则集，开始添加项目规则。</small><button className="secondary" onClick={() => setRulesetModalOpen(true)}>＋ 新建规则集</button></div> : !visibleRulesets.length ? <div className="table-empty"><b>没有匹配的规则集</b><small>调整搜索词后重试。</small></div> : <div className="project-rules-workspace">
          <nav className="project-ruleset-nav" aria-label="项目规则集">
            {visibleRulesets.map((set) => <button key={set.id} type="button" className={`project-ruleset-nav-item ${selectedRuleset?.id === set.id ? "selected" : ""}`} onClick={() => setSelectedRulesetID(set.id)}>
              <span className="project-ruleset-nav-title"><b>{set.name}</b><i className={set.enabled ? "is-enabled" : "is-disabled"}>{set.enabled ? "启用" : "停用"}</i></span>
              <small>{set.rules.length} 条规则 · v{set.version || "1.0.0"}</small>
              <code>{set.id}</code>
            </button>)}
          </nav>
          {selectedRuleset && <section className="project-ruleset-detail">
            <header className="project-ruleset-detail-header">
              <div><small>RULESET</small><h2>{selectedRuleset.name}</h2><p>{selectedRuleset.id} · v{selectedRuleset.version || "1.0.0"} · {selectedRuleset.rules.length} 条规则</p></div>
              <div className="project-ruleset-detail-actions">
                <button type="button" className={`project-status-toggle ${selectedRuleset.enabled ? "is-enabled" : "is-disabled"}`} disabled={Boolean(toggleBusy)} title={selectedRuleset.enabled ? "点击停用规则集" : "点击启用规则集"} onClick={() => void toggleRuleset(selectedRuleset)}>{toggleBusy === `ruleset:${selectedRuleset.id}` ? "保存中…" : selectedRuleset.enabled ? "已启用" : "已停用"}</button>
                <button type="button" className={`project-ruleset-alert-button ${selectedRuleset.alert?.enabled ? "is-configured" : ""}`} title="规则集默认通知" aria-label={`配置规则集 ${selectedRuleset.name} 的默认通知`} onClick={() => openRulesetAlert(selectedRuleset)}><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M15.5 8a5.5 5.5 0 0 0-11 0c0 6-2 6-2 7.5h15C17.5 14 15.5 14 15.5 8ZM8 18h4"/></svg></button>
                <button type="button" className="project-design-icon" title="打开设计器" aria-label={`设计规则集 ${selectedRuleset.name}`} onClick={() => onOpenRuleset(selectedRuleset)}><svg viewBox="0 0 20 20" aria-hidden="true"><rect x="2.5" y="3" width="6" height="4" rx="1"/><rect x="11.5" y="13" width="6" height="4" rx="1"/><rect x="2.5" y="13" width="6" height="4" rx="1"/><path d="M5.5 7v3h9v3M5.5 10v3"/></svg></button>
                <button className="project-ruleset-delete" aria-label={`删除规则集 ${selectedRuleset.name}`} onClick={() => setConfirmRulesetDelete(selectedRuleset)}>删除</button>
              </div>
            </header>
            {selectedRuleset.rules.length ? <div className="project-rule-list">{selectedRuleset.rules.map((rule) => {
              const alertPolicy = effectiveRuleAlert(selectedRuleset, rule);
              return <div className="project-rule-row" key={rule.id}><span className="project-rule-dot"/><b>{rule.name}</b><code>{rule.id}</code>{alertPolicy && <span className={`project-rule-alert ${alertPolicy.enabled ? "is-on" : "is-off"}`} title={`${rule.alert ? "单条规则覆盖" : "继承规则集默认"} · ${alertPolicy.enabled ? `通知渠道：${alertPolicy.channel_ids.map((id) => alertChannels.find((channel) => channel.id === id)?.name ?? id).join("、") || "未选渠道"}` : "命中通知已停用"}`}><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M15.5 8a5.5 5.5 0 0 0-11 0c0 6-2 6-2 7.5h15C17.5 14 15.5 14 15.5 8ZM8 18h4"/></svg><span>{alertPolicy.enabled ? `${rule.alert ? "单条" : "默认"} · ${alertPolicy.channel_ids.map((id) => alertChannels.find((channel) => channel.id === id)?.name ?? id).join("、") || "未选渠道"}` : "通知停用"}</span></span>}<small>优先级 {rule.priority}</small><button type="button" className={`project-status-toggle compact ${rule.enabled ? "is-enabled" : "is-disabled"}`} disabled={Boolean(toggleBusy)} title={rule.enabled ? "点击停用规则" : "点击启用规则"} onClick={() => void toggleRule(selectedRuleset, rule.id)}>{toggleBusy === `rule:${selectedRuleset.id}:${rule.id}` ? "保存中…" : rule.enabled ? "启用" : "停用"}</button></div>;
            })}</div> : <div className="table-empty"><b>这个规则集还没有规则</b><small>使用设计器添加第一条规则。</small></div>}
          </section>}
        </div>}
      </section>
      {alertRuleset && <div className="project-modal-overlay" role="presentation"><section className="project-modal project-config-modal ruleset-alert-modal" role="dialog" aria-modal="true" aria-labelledby="ruleset-alert-title"><header><div><small>RULESET NOTIFICATIONS</small><h2 id="ruleset-alert-title">规则集默认通知</h2><p>「{alertRuleset.name}」中的规则默认继承此设置；单条规则可单独覆盖。</p></div><button className="quiet" disabled={busy} onClick={() => setAlertRuleset(null)} aria-label="关闭">×</button></header><label className="project-toggle"><input type="checkbox" checked={defaultAlertEnabled} onChange={(event) => setDefaultAlertEnabled(event.target.checked)} /><span><b>启用默认命中通知</b><small>命中且未配置单条规则覆盖时，发送到所选渠道</small></span></label><div className="project-check-group"><b>默认通知渠道</b>{alertChannels.length ? alertChannels.map((channel) => <label key={channel.id}><input type="checkbox" checked={defaultAlertChannelIDs.includes(channel.id)} onChange={() => setDefaultAlertChannelIDs((ids) => ids.includes(channel.id) ? ids.filter((id) => id !== channel.id) : [...ids, channel.id])} /><span>{channel.name}</span></label>) : <small>暂无启用的渠道，请先在系统设置中添加告警渠道。</small>}</div><label className="project-toggle ruleset-alert-apply"><input type="checkbox" checked={applyDefaultToAll} onChange={(event) => setApplyDefaultToAll(event.target.checked)} /><span><b>应用到全部现有规则</b><small>清除单条规则的告警覆盖，让所有规则继承上面的默认设置</small></span></label><footer><button className="quiet" disabled={busy} onClick={() => setAlertRuleset(null)}>取消</button><button className="primary" disabled={busy || (defaultAlertEnabled && defaultAlertChannelIDs.length === 0)} onClick={() => void saveRulesetAlert()}>{busy ? "正在保存…" : "保存默认通知"}</button></footer></section></div>}
      {rulesetModalOpen && <div className="project-modal-overlay" role="presentation"><section className="project-modal" role="dialog" aria-modal="true" aria-labelledby="new-ruleset-title"><header><div><small>PROJECT RULES</small><h2 id="new-ruleset-title">新建规则集</h2><p>规则集会创建在「{activeProject.name}」项目内。</p></div><button className="quiet" disabled={busy} onClick={() => setRulesetModalOpen(false)} aria-label="关闭">×</button></header><label className="project-modal-field">规则集名称<input value={rulesetName} onChange={(event) => setRulesetName(event.target.value)} placeholder="例如：订单风控规则" /></label><footer><button className="quiet" disabled={busy} onClick={() => setRulesetModalOpen(false)}>取消</button><button className="primary" disabled={busy || !rulesetName.trim()} onClick={() => void createRuleset()}>{busy ? "正在创建…" : "创建规则集"}</button></footer></section></div>}
      {guideOpen && <div className="project-modal-overlay" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setGuideOpen(false); }}><section className="project-modal project-guide-modal" role="dialog" aria-modal="true" aria-labelledby="project-guide-title"><header><div><small>PROJECT GUIDE</small><h2 id="project-guide-title">规则集使用指南</h2><p>从创建规则到确认规则是否命中。</p></div><button className="quiet" onClick={() => setGuideOpen(false)} aria-label="关闭">×</button></header><ol className="project-guide-steps"><li><b>创建规则集</b><span>点击“新建规则集”并填写名称，创建后会显示在左侧列表。</span></li><li><b>设计并保存</b><span>在规则集右侧点击设计图标，添加条件和动作，再保存规则集。只有启用的规则会参与求值。</span></li><li><b>创建项目凭据</b><span>点击项目右上角钥匙图标，可选择创建 API Key 或 JWT。凭据只显示一次，请妥善保存；JWT 有效期为 30 天。</span></li><li><b>发送能满足条件的 facts</b><span>此示例用于订单风控演示：订单金额大于 10000 且 customer.isNew 为 true 时，应命中“大额新客订单审核”。字段路径和类型都要与规则条件一致。</span></li></ol><div className="project-guide-code-head"><b>API Key 调用示例</b><button className="quiet" onClick={() => void copyGuideRequest()}>复制</button></div><pre className="project-guide-code"><code>{guideRequest}</code></pre><div className="project-guide-result"><b>怎样算成功？</b><p>HTTP 200 表示求值请求成功；响应里的 <code>matched: true</code> 且 <code>hitRules</code> 有规则，表示规则确实触发。HTTP 200 但 <code>matched: false</code> 表示调用成功、没有规则条件匹配；HTTP 非 2xx 表示请求或鉴权失败。</p><pre>{`{\n  "rulesetId": "${selectedRuleset?.id ?? "order-risk"}",\n  "matched": true,\n  "hitRules": [{ "ruleId": "large-new-order", "name": "大额新客订单审核" }]\n}`}</pre><p>每次求值也会进入项目的“调用日志”，可查看命中、未命中、HTTP 状态和耗时；facts 内容不会写入日志。</p></div><footer><button className="primary" onClick={() => setGuideOpen(false)}>知道了</button></footer></section></div>}
      {confirmRulesetDelete && <div className="project-modal-overlay" role="presentation"><section className="project-modal project-delete-modal" role="dialog" aria-modal="true" aria-labelledby="delete-ruleset-title"><header><div><small>DELETE RULESET</small><h2 id="delete-ruleset-title">删除规则集？</h2><p>“{confirmRulesetDelete.name}”及其中的规则将被删除。此操作无法撤销。</p></div></header><footer><button className="quiet" onClick={() => setConfirmRulesetDelete(null)}>取消</button><button className="workspace-confirm-delete" disabled={busy} onClick={() => { void onDeleteRuleset(confirmRulesetDelete).then(() => refreshDetail()).then(() => setConfirmRulesetDelete(null)); }}>确认删除</button></footer></section></div>}
      </>}
    </>}
    {editingID && <div className="project-modal-overlay" role="presentation"><section className="project-modal project-edit-modal" role="dialog" aria-modal="true" aria-labelledby="edit-project-title"><header><div><small>PROJECT SETTINGS</small><h2 id="edit-project-title">编辑项目</h2><p>更新项目名称、Slug 和说明。</p></div><button className="quiet" onClick={() => setEditingID("")} aria-label="关闭">×</button></header><div className="project-create-fields"><label>项目图标<div className="project-icon-picker">{projectIcons.map((icon) => <button type="button" key={icon} className={editIcon === icon ? "selected" : ""} onClick={() => setEditIcon(icon)}>{icon}</button>)}</div></label><label>项目名称<input value={editName} onChange={(event) => setEditName(event.target.value)} placeholder="项目名称" /></label><label>Slug<input value={editSlug} onChange={(event) => setEditSlug(event.target.value.toLowerCase())} placeholder="project-slug" /></label><label>说明<input value={description} onChange={(event) => setDescription(event.target.value)} placeholder="项目说明" /></label></div><footer><button className="quiet" onClick={() => setEditingID("")}>取消</button><button className="primary" disabled={busy || !editName.trim() || !editSlug.trim()} onClick={() => void saveProject()}>{busy ? "保存中…" : "保存项目"}</button></footer></section></div>}
    {invocationLogsOpen && <div className="project-modal-overlay" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setInvocationLogsOpen(false); }}><section className="project-modal project-invocation-modal" role="dialog" aria-modal="true" aria-labelledby="invocation-log-title"><header><div><small>PROJECT INVOCATIONS</small><h2 id="invocation-log-title">调用日志</h2><p>显示第 {invocationLogs.length ? (invocationPage - 1) * 20 + 1 : 0}–{(invocationPage - 1) * 20 + invocationLogs.length} 条，共 {invocationLogsTotal} 次求值。标记“设计器模拟”的记录来自设计器试运行；“真实 API 调用”来自项目 API Key 或 JWT。日志不保存 facts 内容。</p></div><button className="quiet" onClick={() => setInvocationLogsOpen(false)} aria-label="关闭">×</button></header>{invocationLogsBusy ? <div className="table-empty"><b>正在读取调用日志…</b></div> : invocationLogs.length ? <div className="project-invocation-list">{invocationLogs.map((item) => { const ok = item.status_code >= 200 && item.status_code < 300; return <article className="project-invocation-row" key={item.id}><div className="project-invocation-result"><i className={ok ? item.matched ? "matched" : "unmatched" : "failed"} /><div><div className="project-invocation-badges"><b>{!ok ? `调用失败 · HTTP ${item.status_code}` : item.matched ? `规则命中 · ${item.hit_rules?.length ?? 0} 条` : "执行成功 · 无规则命中"}</b><em className={item.source === "designer_simulation" ? "simulation" : item.source === "project_api" ? "real" : "management"}>{item.source === "designer_simulation" ? "设计器模拟" : item.source === "project_api" ? "真实 API 调用" : "管理端调用"}</em></div><small>{item.hit_rules?.map((hit) => hit.name).join("、") || (ok ? "检查请求的 facts 是否符合已启用规则条件" : "检查规则集 ID、请求参数和鉴权凭据")}</small></div></div><div className="project-invocation-meta"><code>{item.ruleset_id || "未解析规则集"}</code><span>{item.credential_type === "jwt" ? "JWT" : item.credential_type === "apikey" ? "API Key" : "管理端会话"} · {item.duration_ms} ms</span><time>{new Date(item.created_at).toLocaleString("zh-CN")}</time></div></article>; })}</div> : <div className="table-empty project-invocation-empty"><b>暂无调用记录</b><small>使用设计器运行模拟，或通过项目 API 调用规则集后，记录会显示在这里。</small></div>}{invocationLogsTotal > 20 && <div className="project-invocation-pagination"><span>第 {invocationPage} / {Math.ceil(invocationLogsTotal / 20)} 页</span><button className="secondary" disabled={invocationLogsBusy || invocationPage <= 1} onClick={() => void openInvocationLogs(invocationPage - 1)}>上一页</button><button className="secondary" disabled={invocationLogsBusy || invocationPage >= Math.ceil(invocationLogsTotal / 20)} onClick={() => void openInvocationLogs(invocationPage + 1)}>下一页</button></div>}<footer><button className="secondary" disabled={invocationLogsBusy} onClick={() => void openInvocationLogs(invocationPage)}>刷新</button><button className="primary" onClick={() => setInvocationLogsOpen(false)}>关闭</button></footer></section></div>}
    {keyModalOpen && <div className="project-modal-overlay" role="presentation"><section className="project-modal project-config-modal" role="dialog" aria-modal="true" aria-labelledby="project-key-title"><header><div><small>PROJECT AUTHENTICATION</small><h2 id="project-key-title">{newSecret ? (keyType === "jwt" ? "保存 JWT" : "保存 API Key") : keyCreateMode ? (keyType === "jwt" ? "创建 JWT" : "创建 API Key") : "项目鉴权"}</h2><p>{newSecret ? "凭据只显示这一次，请立即复制并妥善保存。" : "使用 API Key 或 JWT 调用此项目内已保存的规则。"}</p></div><button className="quiet" onClick={() => { setKeyModalOpen(false); setNewSecret(""); setKeyCreateMode(false); }} aria-label="关闭">×</button></header>
      {newSecret ? <div className="project-secret"><code>{newSecret}</code><p>{keyType === "jwt" ? "JWT 有效期为 30 天；令牌只显示这一次。" : "API Key 只显示这一次。"}</p><pre>{`curl -X POST http://localhost:8080/api/v1/engine/evaluate \
  -H 'Content-Type: application/json' \
  -H '${keyType === "jwt" ? `Authorization: Bearer ${newSecret}` : `X-API-Key: ${newSecret}`}' \
  -d '{"rulesetId":"order-risk","facts":{"order":{"amount":12800},"customer":{"isNew":true}}}'`}</pre><footer><button className="quiet" onClick={() => { setNewSecret(""); setKeyCreateMode(false); }}>← 返回凭据列表</button><button className="primary" onClick={() => { setNewSecret(""); setKeyCreateMode(false); setKeyModalOpen(false); }}>已保存，关闭</button></footer></div> : keyCreateMode ? <><div className="project-auth-type-picker"><button type="button" className={keyType === "apikey" ? "selected" : ""} onClick={() => setKeyType("apikey")}><b>API Key</b><small>固定密钥，使用 X-API-Key 请求头</small></button><button type="button" className={keyType === "jwt" ? "selected" : ""} onClick={() => setKeyType("jwt")}><b>JWT</b><small>签发 30 天令牌，使用 Bearer 认证</small></button></div><label className="project-modal-field">{keyType === "jwt" ? "令牌名称" : "密钥名称"}<input value={keyName} onChange={(e) => setKeyName(e.target.value)} placeholder={keyType === "jwt" ? "例如 order-service-token" : "例如 checkout-service"} /></label><footer><button className="quiet" onClick={() => setKeyCreateMode(false)}>← 返回凭据列表</button><button className="primary" disabled={busy || !keyName.trim()} onClick={() => void createKey()}>{busy ? "正在生成…" : keyType === "jwt" ? "创建 JWT" : "生成 API Key"}</button></footer></> : <><div className="project-key-list">{keys.length ? keys.map((key) => <article className="project-key-row" key={key.id}><div><b>{key.name}</b><code>{key.prefix}••••••••</code><small>{key.type === "jwt" ? "JWT · 30 天有效" : "API Key"}</small></div><small>创建于 {new Date(key.created_at).toLocaleString("zh-CN")}</small><button className="quiet workspace-delete" disabled={busy} onClick={() => void revokeKey(key)}>撤销</button></article>) : <div className="table-empty"><b>暂无项目凭据</b><small>创建 API Key 或 JWT 以调用此项目的已保存规则。</small></div>}</div><footer><button className="quiet" onClick={() => setKeyModalOpen(false)}>关闭</button><button className="secondary" onClick={() => { setKeyName("服务端调用"); setKeyType("apikey"); setKeyCreateMode(true); }}>＋ 创建 API Key</button><button className="primary" onClick={() => { setKeyName("服务端 JWT"); setKeyType("jwt"); setKeyCreateMode(true); }}>＋ 创建 JWT</button></footer></>}
    </section></div>}
    {projectModalOpen && <div className="project-modal-overlay" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !busy) setProjectModalOpen(false); }}><section className="project-modal" role="dialog" aria-modal="true" aria-labelledby="new-project-title"><header><div><small>PROJECT SETUP</small><h2 id="new-project-title">新建项目</h2><p>项目属于当前工作空间，可在项目内创建规则集。</p></div><button className="quiet" disabled={busy} onClick={() => setProjectModalOpen(false)} aria-label="关闭">×</button></header><div className="project-template-choice"><button type="button" className={!projectTemplate ? "selected" : ""} onClick={() => setProjectTemplate("")}><b>空项目</b><small>从零开始创建规则集。</small></button><button type="button" className={projectTemplate === "ecommerce" ? "selected" : ""} onClick={() => setProjectTemplate("ecommerce")}><b>电商订单风控演示</b><small>预置黑名单拦截、大额新客审核和 VIP 标记规则。</small></button></div>{!projectTemplate && <div className="project-create-fields"><label>项目名称<input value={name} onChange={(e) => setName(e.target.value)} placeholder="例如：订单策略" /></label><label>Slug<input value={slug} onChange={(e) => setSlug(e.target.value.toLowerCase())} placeholder="留空自动生成" /></label><label>说明<input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="可选" /></label></div>}<footer><button className="quiet" disabled={busy} onClick={() => setProjectModalOpen(false)}>取消</button><button className="primary" disabled={busy || (!projectTemplate && !name.trim())} onClick={() => void createProject(projectTemplate)}>{busy ? "正在创建…" : projectTemplate ? "创建演示项目" : "创建空项目"}</button></footer></section></div>}
    {confirmProjectDelete && <div className="project-modal-overlay" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !busy) setConfirmProjectDelete(""); }}><section className="project-modal project-delete-modal" role="dialog" aria-modal="true" aria-labelledby="delete-project-title"><header><div><small>DELETE PROJECT</small><h2 id="delete-project-title">删除项目？</h2><p>项目中的规则集和 API Key 会一并删除。此操作无法撤销。</p></div></header><footer><button className="quiet" disabled={busy} onClick={() => setConfirmProjectDelete("")}>取消</button><button className="workspace-confirm-delete" disabled={busy} onClick={() => { const target = projects.find((item) => item.id === confirmProjectDelete); if (target) void removeProject(target); }}>{busy ? "正在删除…" : "确认删除"}</button></footer></section></div>}
  </>;
}
