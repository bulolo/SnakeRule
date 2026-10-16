import { useCallback, useEffect, useRef, useState } from "react";
import {
  type LicenseStatus,
  type PlatformSettings,
  type SystemInfo,
  type AlertChannel as APIAlertChannel,
  type AdminUser as APIAdminUser,
  type Organization,
} from "../lib/sdk";
import * as sdk from "../lib/sdk";
import { CustomSelect } from "./common";
import { toast } from "sonner";

type AdminUser = APIAdminUser;

function formatEnvironment(environment?: string, unavailable = false) {
  if (!environment) return unavailable ? "暂不可用" : "读取中…";
  const labels: Record<string, string> = {
    development: "开发",
    local: "本地",
    staging: "预发布",
    production: "生产",
    test: "测试",
  };
  return labels[environment] ?? environment;
}

function LicenseSettings({ version }: { version: string }) {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [systemInfoError, setSystemInfoError] = useState("");
  const input = useRef<HTMLInputElement>(null);
  const refreshStatus = useCallback(async (showBusy = false) => {
    if (showBusy) setBusy(true);
    setError("");
    try {
      const [licenseResult, infoResult] = await Promise.allSettled([
        sdk.getLicense(),
        sdk.getPlatformSystemInfo(),
      ]);
      if (licenseResult.status === "fulfilled") {
        setStatus(licenseResult.value);
      } else {
        const reason = licenseResult.reason;
        setError(reason instanceof Error ? reason.message : "读取授权状态失败");
      }
      if (infoResult.status === "fulfilled") {
        setSystemInfo(infoResult.value);
        setSystemInfoError("");
      } else {
        const reason = infoResult.reason;
        setSystemInfoError(reason instanceof Error ? reason.message : "读取运行状态失败");
      }
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
      setStatus(await sdk.uploadLicense({ file }));
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
      setStatus(await sdk.clearLicense());
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
          <small>SYSTEM INFORMATION</small>
          <h2>授权与版本</h2>
          <p>查看运行环境、数据库连接和产品授权状态。</p>
        </div>
      </div>
      <section className="license-summary-card">
        <header className="license-card-heading">
          <span className="settings-icon"><svg viewBox="0 0 20 20" aria-hidden="true"><ellipse cx="10" cy="5" rx="6.5" ry="2.5"/><path d="M3.5 5v10c0 1.4 2.9 2.5 6.5 2.5s6.5-1.1 6.5-2.5V5M3.5 10c0 1.4 2.9 2.5 6.5 2.5s6.5-1.1 6.5-2.5"/></svg></span>
          <div>
            <b>运行状态</b>
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
            <b>v{systemInfo?.version || version}</b>
          </div>
          <div>
            <small>运行环境</small>
            <b>{formatEnvironment(systemInfo?.environment, Boolean(systemInfoError))}</b>
          </div>
          <div>
            <small>版本类型</small>
            <b>{status?.edition ? status.edition[0].toUpperCase() + status.edition.slice(1) : status?.is_licensed ? "Enterprise" : "Community"}</b>
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
              <small>授权到期</small>
              <b>
                {status.expires_at
                  ? new Date(status.expires_at).toLocaleDateString("zh-CN")
                  : "长期有效"}
              </b>
            </div>
          </div>
        ) : (
          <p className="license-empty-copy">上传授权文件以启用产品授权。</p>
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
      <section className="system-service-card">
        <div>
          <b>数据库</b>
          <small>{systemInfo?.database.backend || "PostgreSQL"}</small>
        </div>
        <span className={`system-service-status ${systemInfo?.database.connected ? "is-connected" : systemInfoError ? "is-unavailable" : ""}`}>
          <i />{systemInfo ? (systemInfo.database.connected ? "连接正常" : "连接异常") : systemInfoError ? "状态不可用" : "检查中…"}
        </span>
      </section>
      {systemInfoError && <p className="system-info-error">无法读取服务状态：{systemInfoError}</p>}
      {error && (
        <div className="login-error license-error" role="alert">
          {error}
        </div>
      )}
    </>
  );
}

function PlatformBrandSettings({ value, onSave }: { value: PlatformSettings; onSave: (value: PlatformSettings) => void }) {
  const [name, setName] = useState(value.name);
  const [tagline, setTagline] = useState(value.tagline);
  const [logoURL, setLogoURL] = useState(value.logo_url);
  const [busy, setBusy] = useState(false);
  useEffect(() => { setName(value.name); setTagline(value.tagline); setLogoURL(value.logo_url); }, [value]);
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true);
    try {
      const result = await sdk.updatePlatformSettings({ name: name.trim(), tagline: tagline.trim(), logo_url: logoURL.trim() });
      onSave(result); toast.success("平台品牌配置已保存");
    } catch (e) { toast.error("保存平台品牌配置失败", { description: e instanceof Error ? e.message : "请稍后重试" }); }
    finally { setBusy(false); }
  }
  return <section className="settings-brand-config">
    <div className="settings-hero"><div><small>PLATFORM BRAND</small><h2>平台品牌</h2><p>设置登录页与管理端展示的产品名称、标语和 Logo。</p></div></div>
    <form className="workspace-form" onSubmit={(event) => void save(event)}>
      <div className="workspace-form-fields">
        <label>产品名称<input required maxLength={60} value={name} onChange={(event) => setName(event.target.value)} /></label>
        <label>产品标语<input maxLength={120} value={tagline} onChange={(event) => setTagline(event.target.value)} placeholder="例如：确定性规则求值平台" /></label>
        <label>Logo 图片地址<input type="url" value={logoURL} onChange={(event) => setLogoURL(event.target.value)} placeholder="https://example.com/logo.svg" /><small>留空时使用默认标识；仅支持公开 HTTP(S) 图片地址。</small></label>
      </div>
      <div className="settings-brand-preview">
        <small>预览</small>
        <div className="settings-brand-preview-content">
          <span className={`settings-brand-preview-mark${logoURL.trim() ? " has-logo" : ""}`}>
            {logoURL.trim() ? <img src={logoURL.trim()} alt={`${name || "SnakeRule"} Logo`} /> : (name.trim()[0] || "S")}
          </span>
          <div>
            <b>{name || "SnakeRule"}</b>
            <span>{tagline || "规则引擎管理控制台"}</span>
          </div>
        </div>
      </div>
      <div className="workspace-form-actions"><button className="primary" disabled={busy}>{busy ? "正在保存…" : "保存品牌配置"}</button></div>
    </form>
  </section>;
}

function OrganizationSettings({
  items,
  users,
  activeOrganizationID,
  onRefresh,
}: {
  items: Organization[];
  users: AdminUser[];
  activeOrganizationID: string;
  onRefresh: () => Promise<void>;
}) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [adminUserID, setAdminUserID] = useState("");
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
        admin_user_id: editingID ? undefined : adminUserID || undefined,
      };
      if (editingID) await sdk.updateOrganization(editingID, input);
      else await sdk.createOrganization(input);
      setName("");
      setSlug("");
      setDescription("");
      setAdminUserID("");
      setEditingID("");
      await onRefresh();
      setFormOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "保存组织失败");
    } finally {
      setBusy(false);
    }
  }
  function edit(item: Organization) {
    setEditingID(item.id);
    setName(item.name);
    setSlug(item.slug ?? "");
    setDescription(item.description ?? "");
    setError("");
    setFormOpen(true);
  }
  async function remove(item: Organization) {
    setBusy(true);
    setError("");
    try {
      await sdk.deleteOrganization(item.id);
      setConfirmDeleteID("");
      await onRefresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "删除组织失败");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <div className="settings-hero">
        <div>
          <small>ORGANIZATION</small>
          <h2>组织</h2>
          <p>为不同项目或团队隔离规则集，并在管理台快速切换。</p>
        </div>
      </div>
      {formOpen && <section className="workspace-form-view"><header><div><small>ORGANIZATION</small><h2>{editingID ? "编辑组织" : "新建组织"}</h2><p>{editingID ? "更新名称、slug 与说明" : "组织中的规则集互相隔离"}</p></div><button className="quiet" onClick={() => { setFormOpen(false); setEditingID(""); }}>← 返回列表</button></header>
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
                ? "默认组织的 slug 固定为 default"
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
          {!editingID && <div className="workspace-admin-picker"><label>初始组织管理员</label><CustomSelect value={adminUserID} options={[{value:"",label:"暂不指定"},...users.filter((user)=>user.role!=="superadmin").map((user)=>({value:user.id,label:`${user.name} · ${user.email}`}))]} onChange={setAdminUserID} ariaLabel="初始组织管理员" /><small>可指定现有账号作为新组织管理员。</small></div>}
        </div>
        <div className="workspace-form-actions">
          <button className="primary" disabled={busy}>
            {busy ? "正在保存…" : editingID ? "保存修改" : "创建组织"}
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
            <b>组织</b>
            <small>{items.length} 个组织 · 默认组织不可删除</small>
          </div>
          <button className="primary" onClick={() => { setName(""); setSlug(""); setDescription(""); setAdminUserID(""); setEditingID(""); setFormOpen(true); }}>＋ 新建组织</button>
        </header>
        {items.map((item) => (
          <article className="workspace-row" key={item.id}>
            <span className="workspace-avatar">
              {item.name.slice(0, 1).toUpperCase()}
            </span>
            <div className="workspace-row-info">
              <b>
                {item.name}
                {item.id === activeOrganizationID && (
                  <span className="workspace-current-pill">当前</span>
                )}
              </b>
              <small>
                {item.description ||
                  (item.id === "default" ? "初始组织" : "暂无说明")}
              </small>
              <code>{item.slug}</code>
            </div>
            {confirmDeleteID === item.id ? (
              <div className="workspace-confirm">
                <span>删除此组织及其中的项目？</span>
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
                    item.id === activeOrganizationID
                  }
                  title={
                    item.id === activeOrganizationID
                      ? "请先切换到其他组织"
                      : item.id === "default"
                        ? "默认组织不能删除"
                        : "删除组织"
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
    </>
  );
}

type AlertChannel = APIAlertChannel;
type AlertChannelType = Exclude<AlertChannel["type"], "internal">;

function AlertChannelSettings({ organizationID, onRefresh }: { organizationID: string; onRefresh: () => Promise<void> }) {
  const [items, setItems] = useState<AlertChannel[]>([]);
  const [editing, setEditing] = useState("");
  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [secret, setSecret] = useState("");
  const [channelType, setChannelType] = useState<AlertChannelType>("webhook");
  const [enabled, setEnabled] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [confirmDelete, setConfirmDelete] = useState("");
  const [notice, setNotice] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const refresh = useCallback(async () => {
    const body = await sdk.listOrganizationAlertChannels(organizationID);
    setItems(body.list ?? []);
  }, [organizationID]);
  useEffect(() => { void refresh().catch((e: Error) => setError(e.message)); }, [refresh]);
  function startEdit(item: AlertChannel) {
    if (item.type === "internal") return;
    setEditing(item.id); setName(item.name); setURL(item.url); setSecret(""); setChannelType(item.type); setEnabled(item.enabled); setError(""); setFormOpen(true);
  }
  function reset() { setEditing(""); setName(""); setURL(""); setSecret(""); setChannelType("webhook"); setEnabled(true); setFormOpen(false); }
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const input = { name, type: channelType, url, secret: secret || undefined, enabled };
      if (editing) await sdk.updateOrganizationAlertChannel(organizationID, editing, input);
      else await sdk.createOrganizationAlertChannel(organizationID, input);
      reset(); await refresh(); await onRefresh();
    } catch (e) { setError(e instanceof Error ? e.message : "保存告警渠道失败"); }
    finally { setBusy(false); }
  }
  async function remove(item: AlertChannel) {
    setBusy(true); setError("");
    try {
      await sdk.deleteOrganizationAlertChannel(organizationID, item.id);
      setConfirmDelete(""); await refresh(); await onRefresh();
    } catch (e) { setError(e instanceof Error ? e.message : "删除告警渠道失败"); }
    finally { setBusy(false); }
  }
  async function sendTest(item: AlertChannel) {
    setBusy(true); setError(""); setNotice("");
    try {
      await sdk.testOrganizationAlertChannel(organizationID, item.id);
      setNotice(`测试通知已发送到「${item.name}」。`);
    } catch (e) { setError(e instanceof Error ? e.message : "测试通知发送失败"); }
    finally { setBusy(false); }
  }
  return <>
    <div className="settings-hero"><div><small>ORGANIZATION ALERTS</small><h2>告警渠道</h2><p>维护当前组织的通知渠道，供组织内项目和规则告警策略引用。</p></div></div>
    {formOpen && <section className="workspace-form-view"><header><div><small>ORGANIZATION ALERTS</small><h2>{editing ? "编辑告警渠道" : "添加告警渠道"}</h2><p>选择机器人类型并配置 Webhook 地址。</p></div><button className="quiet" onClick={reset}>← 返回列表</button></header>
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
    {!formOpen && <section className="workspace-list-card"><header><div><b>已配置渠道</b><small>{items.length} 个组织渠道</small></div><button className="primary" onClick={() => { reset(); setFormOpen(true); }}>＋ 添加告警渠道</button></header>
      {items.length === 0 && <p className="settings-readonly-note">还没有配置告警渠道。添加后，项目告警策略即可选择该渠道。</p>}
      {items.map((item) => <article className="workspace-row" key={item.id}>
        <span className="workspace-avatar">{item.type === "internal" ? "◉" : "↗"}</span><div className="workspace-row-info"><b>{item.name}<span className="workspace-current-pill">{item.enabled ? "启用" : "停用"}</span></b><small>{item.type === "internal" ? "规则命中后显示在工作台通知中心" : item.url}</small><code>{{webhook:"自定义 Webhook",wecom:"企业微信",dingtalk:"钉钉",feishu:"飞书",internal:"站内通知"}[item.type]}{item.type === "internal" ? " · 系统默认" : item.has_secret ? " · 密钥已保存" : " · 无密钥"}</code></div>
        {item.type === "internal" ? <div className="workspace-row-actions"><span className="settings-readonly-note">不可删除</span></div> : confirmDelete === item.id ? <div className="workspace-confirm"><span>确认删除此渠道？</span><button className="quiet" disabled={busy} onClick={() => setConfirmDelete("")}>取消</button><button className="workspace-confirm-delete" disabled={busy} onClick={() => void remove(item)}>{busy ? "删除中…" : "确认删除"}</button></div> : <div className="workspace-row-actions"><button className="quiet" disabled={busy || !item.enabled} onClick={() => void sendTest(item)}>测试</button><button className="quiet" disabled={busy} onClick={() => startEdit(item)}>编辑</button><button className="quiet workspace-delete" disabled={busy} onClick={() => setConfirmDelete(item.id)}>删除</button></div>}
      </article>)}
    </section>}
  </>;
}

function UserManagement({ users, onRefresh, activeOrganizationID }: { users: AdminUser[]; onRefresh: () => Promise<void>; activeOrganizationID: string }) {
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<AdminUser | null>(null);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<"organization_admin" | "organization_member">("organization_member");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [confirmDelete, setConfirmDelete] = useState("");

  function openForm(user?: AdminUser) {
    setEditing(user ?? null);
    setName(user?.name ?? "");
    setEmail(user?.email ?? "");
    setPassword("");
    setRole((user?.organization_roles?.[activeOrganizationID] as "organization_admin" | "organization_member") ?? (user?.role === "organization_admin" ? "organization_admin" : "organization_member"));
    setError("");
    setFormOpen(true);
  }

  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const input = { email: email.trim(), name: name.trim(), password: password || undefined, role };
      if (editing) await sdk.updateOrganizationUser(activeOrganizationID, editing.id, input);
      else await sdk.createOrganizationUser(activeOrganizationID, input);
      setFormOpen(false);
      toast.success(editing ? "用户资料已更新" : "用户已创建");
      await onRefresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "保存用户失败");
    } finally {
      setBusy(false);
    }
  }

  async function remove(userID: string) {
    setBusy(true);
    setError("");
    try {
      await sdk.deleteOrganizationUser(activeOrganizationID, userID);
      setConfirmDelete("");
      toast.success("用户已从当前组织移除");
      await onRefresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "删除用户失败");
    } finally {
      setBusy(false);
    }
  }

  if (formOpen) {
    return <section className="user-form-view">
      <header>
        <div><small>ORGANIZATION USERS</small><h2>{editing ? "编辑用户账号" : "添加用户账号"}</h2><p>为当前组织创建账号并设置组织管理员或成员身份。</p></div>
        <button className="quiet" disabled={busy} onClick={() => setFormOpen(false)}>← 返回用户列表</button>
      </header>
      <form className="user-account-form" onSubmit={(event) => void save(event)}>
        <label>姓名<input required maxLength={80} value={name} onChange={(event) => setName(event.target.value)} /></label>
        <label>邮箱<input required type="email" value={email} onChange={(event) => setEmail(event.target.value)} /></label>
        <label>组织角色<CustomSelect value={role} options={[{ value: "organization_admin", label: "组织管理员" }, { value: "organization_member", label: "组织成员" }]} onChange={(value) => setRole(value as typeof role)} ariaLabel="组织角色" /></label>
        <label>{editing ? "重置密码（留空保持不变）" : "初始密码"}<input required={!editing} minLength={10} type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="至少 10 位" /></label>
        {error && <div className="login-error" role="alert">{error}</div>}
        <footer><button type="button" className="quiet" disabled={busy} onClick={() => setFormOpen(false)}>返回</button><button className="primary" disabled={busy}>{busy ? "保存中…" : "保存用户"}</button></footer>
      </form>
    </section>;
  }

  return <>
    <div className="settings-hero"><div><small>ORGANIZATION USERS</small><h2>用户管理</h2><p>管理当前组织的用户账号；项目内可分别授予项目管理员或项目成员角色。</p></div><button className="primary" onClick={() => openForm()}>＋ 添加用户</button></div>
    <section className="workspace-list-card">
      <header><div><b>当前组织账号</b><small>{users.length} 个账号</small></div></header>
      {users.length === 0 && <div className="table-empty"><b>暂无用户</b><small>添加账号后，可以在项目的“成员”选项卡分配项目角色。</small></div>}
      {users.map((user) => <article className="workspace-row" key={user.id}>
        <span className="workspace-avatar">{user.name.slice(0, 1)}</span>
        <div className="workspace-row-info"><b>{user.name}<span className="workspace-current-pill">{(user.organization_roles?.[activeOrganizationID] ?? (user.role === "organization_admin" ? "organization_admin" : "organization_member")) === "organization_admin" ? "组织管理员" : "组织成员"}</span></b><small>{user.email}</small><code>{user.organization_ids?.length ? `${user.organization_ids.length} 个组织` : "尚未分配组织"}</code></div>
        {confirmDelete === user.id ? <div className="workspace-confirm"><span>确认删除账号？</span><button className="quiet" disabled={busy} onClick={() => setConfirmDelete("")}>取消</button><button className="workspace-confirm-delete" disabled={busy} onClick={() => void remove(user.id)}>确认删除</button></div> : <div className="workspace-row-actions"><button className="quiet" onClick={() => openForm(user)}>编辑</button>{user.role !== "superadmin" && <button className="quiet workspace-delete" onClick={() => setConfirmDelete(user.id)}>删除</button>}</div>}
      </article>)}
    </section>
    {error && <div className="login-error license-error" role="alert">{error}</div>}
  </>;
}

export function SettingsDialog({
  area,
  onClose,
  isSuperAdmin,
  organizationRole,
  version,
  platformBrand,
  onPlatformBrandChange,
  organizations,
  activeOrganizationID,
  onOrganizationRefresh,
  onAlertChannelRefresh,
}: {
  area: "platform" | "system";
  onClose: () => void;
  isSuperAdmin: boolean;
  organizationRole: string;
  version: string;
  platformBrand: PlatformSettings;
  onPlatformBrandChange: (value: PlatformSettings) => void;
  organizations: Organization[];
  activeOrganizationID: string;
  onOrganizationRefresh: () => Promise<void>;
  onAlertChannelRefresh: () => Promise<void>;
}) {
  const platform = area === "platform";
  const canManageOrganization = isSuperAdmin || organizationRole === "organization_admin";
  const [tab, setTab] = useState(platform ? "system" : canManageOrganization ? "users" : "alerts");
  const [users,setUsers]=useState<AdminUser[]>([]);
  const refreshUsers=useCallback(async()=>{const body=await sdk.listOrganizationUsers(activeOrganizationID);setUsers(body.list??[])},[activeOrganizationID]);
  useEffect(()=>{if(!platform && canManageOrganization)void refreshUsers().catch((error:Error)=>toast.error("读取用户失败",{description:error.message}))},[platform,canManageOrganization,refreshUsers]);
  const tabs = platform
      ? [
        { id: "system", label: "系统信息", group: "系统" },
        { id: "brand", label: "平台品牌", group: "配置" },
        { id: "organizations", label: "组织管理", group: "资源" },
      ]
    : [
        ...(canManageOrganization ? [{ id: "users", label: "用户管理", group: "用户" }] : []),
        ...(canManageOrganization ? [{ id: "alerts", label: "告警渠道", group: "配置" }] : []),
      ];
  if (platform && !isSuperAdmin) return null;
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
        aria-label={platform ? "平台设置" : "组织设置"}
      >
        <header className="settings-dialog-header">
          <div>
            <span
              className={
                platform ? "settings-icon platform-icon" : "settings-icon"
              }
            >
              {platform ? <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3.5 5.5h13M3.5 10h13M3.5 14.5h13M7 3.5v4M13 8v4M9 12.5v4" /></svg> : <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 17h14M5 17V6.5L10 4l5 2.5V17M8 9h.01M12 9h.01M8 12h.01M12 12h.01M9 17v-2.5h2V17" /></svg>}
            </span>
            <div>
              <b>{platform ? "平台设置" : "组织设置"}</b>
              <small>
                {platform ? "平台授权、组织与品牌配置" : "当前组织的成员和告警配置"}
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
                {item.group && (index === 0 || item.group !== tabs[index - 1].group) && (
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
            {platform && tab === "system" && <LicenseSettings version={version} />}
            {platform && tab === "brand" && <PlatformBrandSettings value={platformBrand} onSave={onPlatformBrandChange} />}
            {platform && tab === "organizations" && (
              <>
                <OrganizationSettings
                  items={organizations}
                  users={users}
                  activeOrganizationID={activeOrganizationID}
                  onRefresh={onOrganizationRefresh}
                />
              </>
            )}
            {!platform && tab === "alerts" && <AlertChannelSettings organizationID={activeOrganizationID} onRefresh={onAlertChannelRefresh} />}
            {!platform && tab === "users" && <UserManagement users={users} onRefresh={refreshUsers} activeOrganizationID={activeOrganizationID} />}
          </main>
        </div>
      </section>
    </div>
  );
}
