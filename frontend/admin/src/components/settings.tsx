import { useCallback, useEffect, useRef, useState } from "react";
import {
  snakeRuleApi,
  type LicenseStatus,
  type Workspace,
} from "../lib/sdk/generated";

function LicenseSettings({ version }: { version: string }) {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const input = useRef<HTMLInputElement>(null);
  const refreshStatus = useCallback(async (showBusy = false) => {
    if (showBusy) setBusy(true);
    setError("");
    try {
      setStatus(await snakeRuleApi.getLicense());
    } catch (e) {
      setError(e instanceof Error ? e.message : "读取授权状态失败");
    } finally {
      if (showBusy) setBusy(false);
    }
  }, []);
  useEffect(() => {
    void refreshStatus();
  }, [refreshStatus]);
  async function upload(file?: File) {
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      setStatus(await snakeRuleApi.uploadLicense(file));
    } catch (e) {
      setError(e instanceof Error ? e.message : "授权校验失败");
    } finally {
      setBusy(false);
      if (input.current) input.current.value = "";
    }
  }
  async function clear() {
    setBusy(true);
    setError("");
    try {
      setStatus(await snakeRuleApi.clearLicense());
    } catch (e) {
      setError(e instanceof Error ? e.message : "清除授权失败");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <div className="settings-hero">
        <div>
          <small>PLATFORM SYSTEM</small>
          <h2>系统信息</h2>
          <p>运行版本与当前产品授权状态。</p>
        </div>
      </div>
      <section className="license-summary-card">
        <header className="license-card-heading">
          <span className="settings-icon">⌘</span>
          <div>
            <b>系统</b>
            <small>SnakeRule 平台授权与版本信息</small>
          </div>
          <button
            className="quiet"
            disabled={busy}
            onClick={() => void refreshStatus(true)}
          >
            {busy ? "正在刷新…" : "↻ 刷新"}
          </button>
        </header>
        <div className="license-system-grid">
          <div>
            <small>版本</small>
            <b>v{version}</b>
          </div>
          <div>
            <small>版本类型</small>
            <b>{status?.is_licensed ? "Enterprise" : "Community"}</b>
          </div>
          <div>
            <small>授权状态</small>
            <b>
              <span
                className={`license-status-pill ${status?.is_licensed ? "licensed" : ""}`}
              >
                <i />
                {status?.is_licensed
                  ? "已授权"
                  : status?.configured
                    ? "授权不可用"
                    : "未授权"}
              </span>
            </b>
          </div>
        </div>
        {status?.is_licensed ? (
          <div className="license-facts-grid">
            <div>
              <small>授权客户</small>
              <b>{status.customer || "—"}</b>
            </div>
            <div>
              <small>授权类型</small>
              <b>{status.license_type || status.edition}</b>
            </div>
            <div>
              <small>授权到期</small>
              <b>
                {status.expires_at
                  ? new Date(status.expires_at).toLocaleDateString("zh-CN")
                  : "长期有效"}
              </b>
            </div>
          </div>
        ) : (
          <p className="license-empty-copy">
            上传授权文件后，Pulse Agent
            将校验签名和实例绑定，通过后立即更新授权状态。
          </p>
        )}
        {status?.feature_keys?.length ? (
          <div className="license-feature-list">
            <small>授权功能</small>
            <div>
              {status.feature_keys.map((feature) => (
                <span key={feature}>{feature}</span>
              ))}
            </div>
          </div>
        ) : null}
        <div className="license-card-footer">
          <div>
            <small>实例 ID</small>
            <code>{status?.installation_id ?? "加载中…"}</code>
          </div>
          <div className="license-actions">
            <input
              className="license-file-input"
              ref={input}
              type="file"
              accept=".json,.license,.txt,application/json,text/plain"
              onChange={(e) => void upload(e.target.files?.[0])}
            />
            <button
              className="secondary"
              disabled={busy || !status}
              onClick={() => input.current?.click()}
            >
              {busy
                ? "正在校验…"
                : status?.configured
                  ? "替换授权"
                  : "上传授权"}
            </button>
            {status?.configured && (
              <button
                className="quiet license-clear"
                disabled={busy}
                onClick={() => void clear()}
              >
                清除授权
              </button>
            )}
          </div>
        </div>
      </section>
      <p className="settings-readonly-note">
        授权凭证只在服务端加密保存，不会返回到浏览器。授权功能必须由后端按功能点校验；前端状态仅用于展示。
      </p>
      {error && (
        <div className="login-error license-error" role="alert">
          {error}
        </div>
      )}
    </>
  );
}

function WorkspaceSettings({
  items,
  activeWorkspaceID,
  onRefresh,
}: {
  items: Workspace[];
  activeWorkspaceID: string;
  onRefresh: () => Promise<void>;
}) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [editingID, setEditingID] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [confirmDeleteID, setConfirmDeleteID] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const input = {
        name: name.trim(),
        slug: slug.trim(),
        description: description.trim(),
      };
      if (editingID) await snakeRuleApi.updateWorkspace(editingID, input);
      else await snakeRuleApi.createWorkspace(input);
      setName("");
      setSlug("");
      setDescription("");
      setEditingID("");
      await onRefresh();
      setFormOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "保存工作空间失败");
    } finally {
      setBusy(false);
    }
  }
  function edit(item: Workspace) {
    setEditingID(item.id);
    setName(item.name);
    setSlug(item.slug ?? "");
    setDescription(item.description ?? "");
    setError("");
    setFormOpen(true);
  }
  async function remove(item: Workspace) {
    setBusy(true);
    setError("");
    try {
      await snakeRuleApi.deleteWorkspace(item.id);
      setConfirmDeleteID("");
      await onRefresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "删除工作空间失败");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <div className="settings-hero">
        <div>
          <small>WORKSPACE</small>
          <h2>工作空间</h2>
          <p>为不同项目或团队隔离规则集，并在管理台快速切换。</p>
        </div>
      </div>
      {formOpen && <section className="workspace-form-view"><header><div><small>WORKSPACE</small><h2>{editingID ? "编辑工作空间" : "新建工作空间"}</h2><p>{editingID ? "更新名称、slug 与说明" : "工作空间中的规则集互相隔离"}</p></div><button className="quiet" onClick={() => { setFormOpen(false); setEditingID(""); }}>← 返回列表</button></header>
      <form className="workspace-form" onSubmit={(event) => void save(event)}>
        <div className="workspace-form-fields">
          <label>
            名称
            <input
              required
              maxLength={80}
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="例如：支付风控"
            />
          </label>
          <label>
            Slug
            <input
              maxLength={32}
              pattern="[a-z0-9][a-z0-9_-]{0,31}"
              disabled={editingID === "default"}
              value={slug}
              onChange={(event) => setSlug(event.target.value.toLowerCase())}
              placeholder="留空时自动生成"
            />
            <small>
              {editingID === "default"
                ? "默认空间的 slug 固定为 default"
                : "小写字母、数字、连字符或下划线"}
            </small>
          </label>
          <label>
            说明
            <input
              maxLength={200}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="可选，简要说明用途"
            />
          </label>
        </div>
        <div className="workspace-form-actions">
          <button className="primary" disabled={busy}>
            {busy ? "正在保存…" : editingID ? "保存修改" : "创建工作空间"}
          </button>
        </div>
      </form>
      </section>}
      {error && (
        <div className="login-error license-error" role="alert">
          {error}
        </div>
      )}
      {!formOpen && <section className="workspace-list-card">
        <header>
          <div>
            <b>工作空间</b>
            <small>{items.length} 个空间 · 默认空间不可删除</small>
          </div>
          <button className="primary" onClick={() => { setName(""); setSlug(""); setDescription(""); setEditingID(""); setFormOpen(true); }}>＋ 新建工作空间</button>
        </header>
        {items.map((item) => (
          <article className="workspace-row" key={item.id}>
            <span className="workspace-avatar">
              {item.name.slice(0, 1).toUpperCase()}
            </span>
            <div className="workspace-row-info">
              <b>
                {item.name}
                {item.id === activeWorkspaceID && (
                  <span className="workspace-current-pill">当前</span>
                )}
              </b>
              <small>
                {item.description ||
                  (item.id === "default" ? "初始工作空间" : "暂无说明")}
              </small>
              <code>{item.slug}</code>
            </div>
            {confirmDeleteID === item.id ? (
              <div className="workspace-confirm">
                <span>删除此空间及其中的规则集？</span>
                <button
                  className="quiet"
                  disabled={busy}
                  onClick={() => setConfirmDeleteID("")}
                >
                  取消
                </button>
                <button
                  className="workspace-confirm-delete"
                  disabled={busy}
                  onClick={() => void remove(item)}
                >
                  {busy ? "删除中…" : "确认删除"}
                </button>
              </div>
            ) : (
              <div className="workspace-row-actions">
                <button
                  className="quiet"
                  disabled={busy}
                  onClick={() => edit(item)}
                >
                  编辑
                </button>
                <button
                  className="quiet workspace-delete"
                  disabled={
                    busy ||
                    item.id === "default" ||
                    item.id === activeWorkspaceID
                  }
                  title={
                    item.id === activeWorkspaceID
                      ? "请先切换到其他工作空间"
                      : item.id === "default"
                        ? "默认工作空间不能删除"
                        : "删除工作空间"
                  }
                  onClick={() => setConfirmDeleteID(item.id)}
                >
                  删除
                </button>
              </div>
            )}
          </article>
        ))}
      </section>}
      <p className="settings-readonly-note">
        当前开发版的工作空间与规则集保存在内存中，服务重启后会恢复为初始空间和演示规则。
      </p>
    </>
  );
}

type AlertChannel = {
  id: string;
  name: string;
  type: "webhook" | "wecom" | "dingtalk" | "feishu" | "internal";
  url: string;
  enabled: boolean;
  has_secret: boolean;
};

async function alertAPIJSON(response: Response): Promise<any> {
  const raw = await response.text();
  let body: any;
  try { body = raw ? JSON.parse(raw) : {}; }
  catch { throw new Error(`告警渠道接口返回非 JSON 响应（HTTP ${response.status}）：${raw.slice(0, 160)}`); }
  if (!response.ok) throw new Error(body?.error?.message ?? `请求失败（HTTP ${response.status}）`);
  return body;
}

function AlertChannelSettings({ onRefresh }: { onRefresh: () => Promise<void> }) {
  const [items, setItems] = useState<AlertChannel[]>([]);
  const [editing, setEditing] = useState("");
  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [secret, setSecret] = useState("");
  const [channelType, setChannelType] = useState<AlertChannel["type"]>("webhook");
  const [enabled, setEnabled] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [confirmDelete, setConfirmDelete] = useState("");
  const [notice, setNotice] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const refresh = useCallback(async () => {
    const response = await fetch("/api/v1/system/alert-channels");
    const body = await alertAPIJSON(response);
    setItems(body.items ?? []);
  }, []);
  useEffect(() => { void refresh().catch((e: Error) => setError(e.message)); }, [refresh]);
  function startEdit(item: AlertChannel) {
    setEditing(item.id); setName(item.name); setURL(item.url); setSecret(""); setChannelType(item.type); setEnabled(item.enabled); setError(""); setFormOpen(true);
  }
  function reset() { setEditing(""); setName(""); setURL(""); setSecret(""); setChannelType("webhook"); setEnabled(true); setFormOpen(false); }
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const response = await fetch(editing ? `/api/v1/system/alert-channels/${editing}` : "/api/v1/system/alert-channels", {
        method: editing ? "PUT" : "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, type: channelType, url, secret, enabled }),
      });
      await alertAPIJSON(response);
      reset(); await refresh(); await onRefresh();
    } catch (e) { setError(e instanceof Error ? e.message : "保存告警渠道失败"); }
    finally { setBusy(false); }
  }
  async function remove(item: AlertChannel) {
    setBusy(true); setError("");
    try {
      const response = await fetch(`/api/v1/system/alert-channels/${item.id}`, { method: "DELETE" });
      if (!response.ok) await alertAPIJSON(response);
      setConfirmDelete(""); await refresh(); await onRefresh();
    } catch (e) { setError(e instanceof Error ? e.message : "删除告警渠道失败"); }
    finally { setBusy(false); }
  }
  async function sendTest(item: AlertChannel) {
    setBusy(true); setError(""); setNotice("");
    try {
      const response = await fetch(`/api/v1/system/alert-channels/${item.id}/test`, { method: "POST" });
      await alertAPIJSON(response);
      setNotice(`测试通知已发送到「${item.name}」。`);
    } catch (e) { setError(e instanceof Error ? e.message : "测试通知发送失败"); }
    finally { setBusy(false); }
  }
  return <>
    <div className="settings-hero"><div><small>SYSTEM ALERTS</small><h2>告警渠道</h2><p>在系统设置中维护可供项目告警策略引用的消息渠道。</p></div></div>
    {formOpen && <section className="workspace-form-view"><header><div><small>SYSTEM ALERTS</small><h2>{editing ? "编辑告警渠道" : "添加告警渠道"}</h2><p>选择机器人类型并配置 Webhook 地址。</p></div><button className="quiet" onClick={reset}>← 返回列表</button></header>
    <form className="workspace-form" onSubmit={(event) => void save(event)}>
      <div className="alert-channel-types" role="group" aria-label="告警渠道类型">
        {([["webhook", "自定义 Webhook"], ["wecom", "企业微信"], ["dingtalk", "钉钉"], ["feishu", "飞书"]] as const).map(([id, label]) => <button key={id} type="button" className={channelType === id ? "active" : ""} onClick={() => setChannelType(id)}>{label}</button>)}
      </div>
      <div className="workspace-form-fields">
        <label>渠道名称<input required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} placeholder="例如：订单风控通知" /></label>
        <label>{channelType === "wecom" ? "机器人 Webhook 地址" : "Webhook 地址"}<input required type="url" value={url} onChange={(e) => setURL(e.target.value)} placeholder={channelType === "wecom" ? "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=…" : channelType === "dingtalk" ? "https://oapi.dingtalk.com/robot/send?access_token=…" : channelType === "feishu" ? "https://open.feishu.cn/open-apis/bot/v2/hook/…" : "https://example.com/hooks/…"} /></label>
        {channelType !== "wecom" && <label>{channelType === "dingtalk" ? "加签密钥（可选）" : channelType === "feishu" ? "签名密钥（可选）" : "Bearer 密钥（可选）"}<input type="password" autoComplete="new-password" value={secret} onChange={(e) => setSecret(e.target.value)} placeholder={editing ? "留空表示保留现有密钥" : "输入机器人密钥"} /></label>}
      </div>
      <div className="workspace-form-actions">
        <button type="button" className={enabled ? "secondary" : "quiet"} onClick={() => setEnabled(!enabled)}>{enabled ? "已启用" : "已停用"}</button>
        <button className="primary" disabled={busy}>{busy ? "正在保存…" : editing ? "保存修改" : "添加渠道"}</button>
      </div>
    </form>
    </section>}
    {error && <div className="login-error license-error" role="alert">{error}</div>}
    {notice && <div className="settings-readonly-note" role="status">{notice}</div>}
    {!formOpen && <section className="workspace-list-card"><header><div><b>已配置渠道</b><small>{items.length} 个平台级渠道</small></div><button className="primary" onClick={() => { reset(); setFormOpen(true); }}>＋ 添加告警渠道</button></header>
      {items.length === 0 && <p className="settings-readonly-note">还没有配置告警渠道。添加后，项目告警策略即可选择该渠道。</p>}
      {items.map((item) => <article className="workspace-row" key={item.id}>
        <span className="workspace-avatar">{item.type === "internal" ? "◉" : "↗"}</span><div className="workspace-row-info"><b>{item.name}<span className="workspace-current-pill">{item.enabled ? "启用" : "停用"}</span></b><small>{item.type === "internal" ? "规则命中后显示在工作台通知中心" : item.url}</small><code>{{webhook:"自定义 Webhook",wecom:"企业微信",dingtalk:"钉钉",feishu:"飞书",internal:"站内通知"}[item.type]}{item.type === "internal" ? " · 系统默认" : item.has_secret ? " · 密钥已保存" : " · 无密钥"}</code></div>
        {item.type === "internal" ? <div className="workspace-row-actions"><span className="settings-readonly-note">不可删除</span></div> : confirmDelete === item.id ? <div className="workspace-confirm"><span>确认删除此渠道？</span><button className="quiet" disabled={busy} onClick={() => setConfirmDelete("")}>取消</button><button className="workspace-confirm-delete" disabled={busy} onClick={() => void remove(item)}>{busy ? "删除中…" : "确认删除"}</button></div> : <div className="workspace-row-actions"><button className="quiet" disabled={busy || !item.enabled} onClick={() => void sendTest(item)}>测试</button><button className="quiet" disabled={busy} onClick={() => startEdit(item)}>编辑</button><button className="quiet workspace-delete" disabled={busy} onClick={() => setConfirmDelete(item.id)}>删除</button></div>}
      </article>)}
    </section>}
    {!formOpen && <p className="settings-readonly-note">密钥不会通过读取接口返回；编辑时留空可保留现有密钥。</p>}
  </>;
}

export function SettingsDialog({
  area,
  onClose,
  email,
  version,
  workspaces,
  activeWorkspaceID,
  onWorkspaceRefresh,
  onAlertChannelRefresh,
}: {
  area: "platform" | "system";
  onClose: () => void;
  email: string;
  version: string;
  workspaces: Workspace[];
  activeWorkspaceID: string;
  onWorkspaceRefresh: () => Promise<void>;
  onAlertChannelRefresh: () => Promise<void>;
}) {
  const platform = area === "platform";
  const [tab, setTab] = useState(platform ? "overview" : "general");
  const tabs = platform
    ? [
        { id: "overview", label: "概览", group: "平台" },
        { id: "license", label: "系统信息", group: "平台管理" },
        { id: "workspaces", label: "工作空间", group: "工作空间" },
        { id: "config", label: "全局配置", group: "配置管理" },
      ]
    : [
        { id: "general", label: "通用设置", group: "工作区" },
        { id: "alerts", label: "告警渠道", group: "通知管理" },
        { id: "security", label: "安全设置", group: "账号安全" },
      ];
  const host = typeof window === "undefined" ? "—" : window.location.host;
  const heading = (
    eyebrow: string,
    title: string,
    description: string,
    badge?: string,
  ) => (
    <div className="settings-hero">
      <div>
        <small>{eyebrow}</small>
        <h2>{title}</h2>
        <p>{description}</p>
      </div>
      {badge && (
        <span className="settings-version-badge">
          <i />
          版本 <b>v{badge}</b>
        </span>
      )}
    </div>
  );
  const group = (
    title: string,
    hint: string,
    rows: [string, React.ReactNode][],
  ) => (
    <section className="settings-detail-section">
      <header>
        <div>
          <h3>{title}</h3>
          <small>{hint}</small>
        </div>
      </header>
      <div className="settings-detail-grid">
        {rows.map(([label, value]) => (
          <div className="settings-detail-row" key={label}>
            <span>{label}</span>
            <b>{value}</b>
          </div>
        ))}
      </div>
    </section>
  );
  return (
    <div
      className="settings-overlay"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <section
        className="settings-dialog"
        role="dialog"
        aria-modal="true"
        aria-label={platform ? "平台设置" : "系统设置"}
      >
        <header className="settings-dialog-header">
          <div>
            <span
              className={
                platform ? "settings-icon platform-icon" : "settings-icon"
              }
            >
              {platform ? "◆" : "⚙"}
            </span>
            <div>
              <b>{platform ? "平台设置" : "系统设置"}</b>
              <small>
                {platform ? "平台级信息与运行概况" : "当前管理端与规则引擎配置"}
              </small>
            </div>
          </div>
          <button onClick={onClose} aria-label="关闭设置">
            ×
          </button>
        </header>
        <div className="settings-dialog-body">
          <nav className="settings-tabs">
            {tabs.map((item, index) => (
              <div key={item.id}>
                {(index === 0 || item.group !== tabs[index - 1].group) && (
                  <small className="settings-nav-label">{item.group}</small>
                )}
                <button
                  className={tab === item.id ? "active" : ""}
                  onClick={() => setTab(item.id)}
                >
                  <i />
                  {item.label}
                </button>
              </div>
            ))}
            <div className="settings-nav-foot">
              <span className="settings-nav-mark">S</span>
              <span>
                <b>SnakeRule</b>
                <small>规则引擎平台</small>
              </span>
            </div>
          </nav>
          <main className="settings-content">
            {platform && tab === "overview" && (
              <>
                {heading(
                  "PLATFORM CONSOLE",
                  "平台概览",
                  "平台运行状态、版本信息与管理员入口。",
                  version,
                )}
                <div className="settings-overview-grid">
                  <article className="settings-overview-brand">
                    <span className="settings-brand-mark">S</span>
                    <div>
                      <small>产品名称</small>
                      <b>SnakeRule</b>
                      <p>规则引擎管理平台</p>
                    </div>
                  </article>
                  <article className="settings-overview-status">
                    <span className="settings-status-icon">✓</span>
                    <div>
                      <small>服务状态</small>
                      <b>运行正常</b>
                      <p>
                        <i />
                        管理端与 API 已连接
                      </p>
                    </div>
                  </article>
                  <article className="settings-overview-edition">
                    <small>产品版本</small>
                    <b>社区版</b>
                    <span>单实例管理</span>
                  </article>
                </div>
                {group("平台信息", "访问入口与当前管理员", [
                  ["访问地址", host],
                  ["管理员账号", email],
                  [
                    "管理员级别",
                    <>
                      <i className="superadmin-dot" />
                      超级管理员
                    </>,
                  ],
                ])}
              </>
            )}
            {platform && tab === "config" && (
              <>
                {heading(
                  "GLOBAL CONFIGURATION",
                  "全局配置",
                  "当前开发版本启用的基础运行参数。",
                  version,
                )}
                <div className="settings-config-grid">
                  {group("规则服务", "规则匹配与决策执行", [
                    ["规则存储", "进程内存"],
                    ["条件表达式", "all / any / not"],
                    ["求值策略", "按优先级从高到低"],
                  ])}
                  {group("身份验证", "控制台登录与会话", [
                    ["登录方式", "邮箱 + 密码"],
                    ["会话有效期", "24 小时"],
                    ["会话存储", "服务内存，重启后失效"],
                  ])}
                </div>
                <div className="settings-readonly-note">
                  运行参数来自服务端环境配置，需在 backend/.env 中调整。
                </div>
              </>
            )}
            {platform && tab === "license" && (
              <>
                <LicenseSettings version={version} />
              </>
            )}
            {platform && tab === "workspaces" && (
              <>
                <WorkspaceSettings
                  items={workspaces}
                  activeWorkspaceID={activeWorkspaceID}
                  onRefresh={onWorkspaceRefresh}
                />
              </>
            )}
            {!platform && tab === "general" && (
              <>
                {heading(
                  "WORKSPACE",
                  "通用设置",
                  "工作空间与规则引擎的基础运行信息。",
                  version,
                )}
                <div className="settings-config-grid">
                  {group("工作空间", "管理端显示与本地环境", [
                    ["产品名称", "SnakeRule"],
                    ["当前版本", `v${version}`],
                    [
                      "浏览器时区",
                      Intl.DateTimeFormat().resolvedOptions().timeZone,
                    ],
                    ["界面语言", "简体中文"],
                  ])}
                  {group("规则引擎", "当前规则执行方式", [
                    ["规则集存储", "进程内存"],
                    ["默认决策模式", "同步求值"],
                    ["模拟运行", "已启用，不写执行日志"],
                  ])}
                </div>
              </>
            )}
            {!platform && tab === "alerts" && <AlertChannelSettings onRefresh={onAlertChannelRefresh} />}
            {!platform && tab === "security" && (
              <>
                {heading(
                  "ACCOUNT & SECURITY",
                  "安全设置",
                  "当前管理员身份与登录会话保护状态。",
                  version,
                )}
                <div className="settings-account-banner">
                  <span className="settings-account-avatar">
                    {email.slice(0, 1).toUpperCase() || "A"}
                  </span>
                  <div>
                    <small>当前登录账号</small>
                    <b>{email}</b>
                  </div>
                  <span className="settings-role-pill">
                    <i />
                    超级管理员
                  </span>
                </div>
                <div className="settings-config-grid">
                  {group("身份认证", "控制台账号权限", [
                    ["账号角色", "超级管理员"],
                    ["身份认证", "已启用"],
                  ])}
                  {group("会话保护", "浏览器登录会话策略", [
                    ["Cookie", "HttpOnly"],
                    ["同站策略", "SameSite Lax"],
                    ["会话时长", "24 小时"],
                  ])}
                </div>
              </>
            )}
          </main>
        </div>
      </section>
    </div>
  );
}
