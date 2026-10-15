import { useCallback, useEffect, useMemo, useRef, useState, type DragEvent } from 'react';
import { snakeRuleApi, type Condition, type Decision, type LicenseStatus, type Rule, type Ruleset, type Workspace } from './lib/sdk/generated';

const starter: Ruleset = { id: 'new-ruleset', name: '新规则集', version: '1.0.0', description: '', enabled: true, rules: [] };
const pretty = (value: unknown) => JSON.stringify(value, null, 2);
const dateLabel = () => new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit' }).format(new Date());

function CustomSelect({ value, options, onChange, disabled = false, ariaLabel }: { value: string; options: { value: string; label: string }[]; onChange: (value: string) => void; disabled?: boolean; ariaLabel: string }) {
  const root = useRef<HTMLDetailsElement>(null);
  const selected = options.find((option) => option.value === value);
  return <details className="custom-select" ref={root}>
    <summary aria-label={ariaLabel} aria-disabled={disabled} className={disabled ? 'custom-select-trigger disabled' : 'custom-select-trigger'}>{selected?.label ?? value}</summary>
    {!disabled && <div className="custom-select-menu" role="listbox" aria-label={ariaLabel}>{options.map((option) => <button type="button" role="option" aria-selected={option.value === value} key={option.value} onClick={() => { onChange(option.value); if (root.current) root.current.open = false; }}>{option.label}</button>)}</div>}
  </details>;
}

function LoginScreen({ onSuccess }: { onSuccess: (email: string) => void }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError('');
    try { const session = await snakeRuleApi.login({ email: email.trim(), password }); onSuccess(session.user?.email ?? email.trim()); }
    catch (e) { setError(e instanceof Error ? e.message : '登录失败，请重试。'); }
    finally { setBusy(false); }
  }
  return <main className="login-page"><div className="login-card"><a className="brand login-brand" href="/"><span className="brand-icon">S</span><span>SnakeRule<small>RULE ENGINE</small></span></a><div className="login-heading"><small className="eyebrow">SECURE WORKSPACE</small><h1>欢迎回来</h1><p>登录以管理规则集并运行决策测试。</p></div><form onSubmit={submit}><label>邮箱<input type="email" autoComplete="email" autoFocus required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="输入管理员邮箱"/></label><label>密码<input type="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} placeholder="输入密码"/></label>{error && <div className="login-error" role="alert">{error}</div>}<button className="primary login-submit" disabled={busy}>{busy ? '正在登录…' : '登录工作台'} <span>→</span></button></form><div className="login-foot"><i/> 安全会话连接 <span>·</span> SnakeRule</div></div><small className="login-version">规则引擎管理控制台</small></main>;
}

function LicenseSettings({ version }: { version: string }) {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const input = useRef<HTMLInputElement>(null);
  useEffect(() => { snakeRuleApi.getLicense().then(setStatus).catch((e: Error) => setError(e.message)); }, []);
  async function upload(file?: File) {
    if (!file) return;
    setBusy(true); setError('');
    try { setStatus(await snakeRuleApi.uploadLicense(file)); }
    catch (e) { setError(e instanceof Error ? e.message : '授权校验失败'); }
    finally { setBusy(false); if (input.current) input.current.value = ''; }
  }
  async function clear() {
    setBusy(true); setError('');
    try { setStatus(await snakeRuleApi.clearLicense()); }
    catch (e) { setError(e instanceof Error ? e.message : '清除授权失败'); }
    finally { setBusy(false); }
  }
  return <>
    <div className="settings-hero"><div><small>PLATFORM SYSTEM</small><h2>系统信息</h2><p>运行版本与当前产品授权状态。</p></div></div>
    <section className="license-summary-card"><header className="license-card-heading"><span className="settings-icon">⌘</span><div><b>系统</b><small>SnakeRule 平台授权与版本信息</small></div><button className="quiet" disabled={busy} onClick={() => { setError(''); snakeRuleApi.getLicense().then(setStatus).catch((e: Error) => setError(e.message)); }}>↻ 刷新</button></header>
      <div className="license-system-grid"><div><small>版本</small><b>v{version}</b></div><div><small>版本类型</small><b>{status?.is_licensed ? 'Enterprise' : 'Community'}</b></div><div><small>授权状态</small><b><span className={`license-status-pill ${status?.is_licensed ? 'licensed' : ''}`}><i/>{status?.is_licensed ? '已授权' : status?.configured ? '授权不可用' : '未授权'}</span></b></div></div>
      {status?.is_licensed ? <div className="license-facts-grid"><div><small>授权客户</small><b>{status.customer || '—'}</b></div><div><small>授权类型</small><b>{status.license_type || status.edition}</b></div><div><small>授权到期</small><b>{status.expires_at ? new Date(status.expires_at).toLocaleDateString('zh-CN') : '长期有效'}</b></div></div> : <p className="license-empty-copy">上传授权文件后，Pulse Agent 将校验签名和实例绑定，通过后立即更新授权状态。</p>}
      {status?.feature_keys?.length ? <div className="license-feature-list"><small>授权功能</small><div>{status.feature_keys.map((feature) => <span key={feature}>{feature}</span>)}</div></div> : null}
      <div className="license-card-footer"><div><small>实例 ID</small><code>{status?.installation_id ?? '加载中…'}</code></div><div className="license-actions"><input className="license-file-input" ref={input} type="file" accept=".json,.license,.txt,application/json,text/plain" onChange={(e) => void upload(e.target.files?.[0])}/><button className="secondary" disabled={busy || !status} onClick={() => input.current?.click()}>{busy ? '正在校验…' : status?.configured ? '替换授权' : '上传授权'}</button>{status?.configured && <button className="quiet license-clear" disabled={busy} onClick={() => void clear()}>清除授权</button>}</div></div>
    </section>
    <p className="settings-readonly-note">授权凭证只在服务端加密保存，不会返回到浏览器。授权功能必须由后端按功能点校验；前端状态仅用于展示。</p>
    {error && <div className="login-error license-error" role="alert">{error}</div>}
  </>;
}

function WorkspaceSettings({ items, activeWorkspaceID, onRefresh }: { items: Workspace[]; activeWorkspaceID: string; onRefresh: () => Promise<void> }) {
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [editingID, setEditingID] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError('');
    try {
      const input = { name: name.trim(), slug: slug.trim(), description: description.trim() };
      if (editingID) await snakeRuleApi.updateWorkspace(editingID, input);
      else await snakeRuleApi.createWorkspace(input);
      setName(''); setSlug(''); setDescription(''); setEditingID(''); await onRefresh();
    } catch (e) { setError(e instanceof Error ? e.message : '保存工作空间失败'); }
    finally { setBusy(false); }
  }
  function edit(item: Workspace) { setEditingID(item.id); setName(item.name); setSlug(item.slug ?? ''); setDescription(item.description ?? ''); setError(''); }
  async function remove(item: Workspace) {
    if (!window.confirm(`确认删除工作空间“${item.name}”？该空间中的规则集也会一并删除。`)) return;
    setBusy(true); setError('');
    try { await snakeRuleApi.deleteWorkspace(item.id); await onRefresh(); }
    catch (e) { setError(e instanceof Error ? e.message : '删除工作空间失败'); }
    finally { setBusy(false); }
  }
  return <>
    <div className="settings-hero"><div><small>WORKSPACE</small><h2>工作空间</h2><p>为不同项目或团队隔离规则集，并在管理台快速切换。</p></div></div>
    <form className="workspace-form" onSubmit={(event) => void save(event)}><div className="workspace-form-heading"><div><b>{editingID ? '编辑工作空间' : '新建工作空间'}</b><small>{editingID ? '更新名称、slug 与说明' : '工作空间中的规则集互相隔离'}</small></div></div><div className="workspace-form-fields"><label>名称<input required maxLength={80} value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：支付风控"/></label><label>Slug<input maxLength={32} pattern="[a-z0-9][a-z0-9_-]{0,31}" value={slug} onChange={(event) => setSlug(event.target.value.toLowerCase())} placeholder="留空时自动生成"/><small>小写字母、数字、连字符或下划线</small></label><label>说明<input maxLength={200} value={description} onChange={(event) => setDescription(event.target.value)} placeholder="可选，简要说明用途"/></label></div><div className="workspace-form-actions">{editingID && <button type="button" className="quiet" disabled={busy} onClick={() => { setEditingID(''); setName(''); setSlug(''); setDescription(''); }}>取消</button>}<button className="primary" disabled={busy}>{busy ? '正在保存…' : editingID ? '保存修改' : '创建工作空间'}</button></div></form>
    {error && <div className="login-error license-error" role="alert">{error}</div>}
    <section className="workspace-list-card"><header><div><b>工作空间</b><small>{items.length} 个空间 · 默认空间不可删除</small></div></header>{items.map((item) => <article className="workspace-row" key={item.id}><span className="workspace-avatar">{item.name.slice(0, 1).toUpperCase()}</span><div className="workspace-row-info"><b>{item.name}{item.id === activeWorkspaceID && <span className="workspace-current-pill">当前</span>}</b><small>{item.description || (item.id === 'default' ? '初始工作空间' : '暂无说明')}</small><code>{item.slug}</code></div><div className="workspace-row-actions"><button className="quiet" disabled={busy} onClick={() => edit(item)}>编辑</button><button className="quiet workspace-delete" disabled={busy || item.id === 'default' || item.id === activeWorkspaceID} title={item.id === activeWorkspaceID ? '请先切换到其他工作空间' : item.id === 'default' ? '默认工作空间不能删除' : '删除工作空间'} onClick={() => void remove(item)}>删除</button></div></article>)}</section>
    <p className="settings-readonly-note">当前开发版的工作空间与规则集保存在内存中，服务重启后会恢复为初始空间和演示规则。</p>
  </>;
}

function SettingsDialog({ area, onClose, email, version, workspaces, activeWorkspaceID, onWorkspaceRefresh }: { area: 'platform' | 'system'; onClose: () => void; email: string; version: string; workspaces: Workspace[]; activeWorkspaceID: string; onWorkspaceRefresh: () => Promise<void> }) {
  const platform = area === 'platform';
  const [tab, setTab] = useState(platform ? 'overview' : 'general');
  const tabs = platform ? [{ id: 'overview', label: '概览', group: '平台' }, { id: 'license', label: '系统信息', group: '平台管理' }, { id: 'workspaces', label: '工作空间', group: '工作空间' }, { id: 'config', label: '全局配置', group: '配置管理' }] : [{ id: 'general', label: '通用设置', group: '工作区' }, { id: 'security', label: '安全设置', group: '账号安全' }];
  const host = typeof window === 'undefined' ? '—' : window.location.host;
  const heading = (eyebrow: string, title: string, description: string, badge?: string) => <div className="settings-hero"><div><small>{eyebrow}</small><h2>{title}</h2><p>{description}</p></div>{badge && <span className="settings-version-badge"><i/>版本 <b>v{badge}</b></span>}</div>;
  const group = (title: string, hint: string, rows: [string, React.ReactNode][]) => <section className="settings-detail-section"><header><div><h3>{title}</h3><small>{hint}</small></div></header><div className="settings-detail-grid">{rows.map(([label, value]) => <div className="settings-detail-row" key={label}><span>{label}</span><b>{value}</b></div>)}</div></section>;
  return <div className="settings-overlay" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}><section className="settings-dialog" role="dialog" aria-modal="true" aria-label={platform ? '平台设置' : '系统设置'}>
    <header className="settings-dialog-header"><div><span className={platform ? 'settings-icon platform-icon' : 'settings-icon'}>{platform ? '◆' : '⚙'}</span><div><b>{platform ? '平台设置' : '系统设置'}</b><small>{platform ? '平台级信息与运行概况' : '当前管理端与规则引擎配置'}</small></div></div><button onClick={onClose} aria-label="关闭设置">×</button></header>
    <div className="settings-dialog-body"><nav className="settings-tabs">{tabs.map((item, index) => <div key={item.id}>{(index === 0 || item.group !== tabs[index - 1].group) && <small className="settings-nav-label">{item.group}</small>}<button className={tab === item.id ? 'active' : ''} onClick={() => setTab(item.id)}><i/>{item.label}</button></div>)}<div className="settings-nav-foot"><span className="settings-nav-mark">S</span><span><b>SnakeRule</b><small>规则引擎平台</small></span></div></nav>
      <main className="settings-content">{platform && tab === 'overview' && <>
        {heading('PLATFORM CONSOLE', '平台概览', '平台运行状态、版本信息与管理员入口。', version)}
        <div className="settings-overview-grid"><article className="settings-overview-brand"><span className="settings-brand-mark">S</span><div><small>产品名称</small><b>SnakeRule</b><p>规则引擎管理平台</p></div></article><article className="settings-overview-status"><span className="settings-status-icon">✓</span><div><small>服务状态</small><b>运行正常</b><p><i/>管理端与 API 已连接</p></div></article><article className="settings-overview-edition"><small>产品版本</small><b>社区版</b><span>单实例管理</span></article></div>
        {group('平台信息', '访问入口与当前管理员', [['访问地址', host], ['管理员账号', email], ['管理员级别', <><i className="superadmin-dot"/>超级管理员</>]])}
      </>}{platform && tab === 'config' && <>
        {heading('GLOBAL CONFIGURATION', '全局配置', '当前开发版本启用的基础运行参数。', version)}
        <div className="settings-config-grid">{group('规则服务', '规则匹配与决策执行', [['规则存储', '进程内存'], ['条件表达式', 'all / any / not'], ['求值策略', '按优先级从高到低']])}{group('身份验证', '控制台登录与会话', [['登录方式', '邮箱 + 密码'], ['会话有效期', '24 小时'], ['会话存储', '服务内存，重启后失效']])}</div>
        <div className="settings-readonly-note">运行参数来自服务端环境配置，需在 backend/.env 中调整。</div>
      </>}{platform && tab === 'license' && <>
        <LicenseSettings version={version}/>
      </>}{platform && tab === 'workspaces' && <>
        <WorkspaceSettings items={workspaces} activeWorkspaceID={activeWorkspaceID} onRefresh={onWorkspaceRefresh}/>
      </>}{!platform && tab === 'general' && <>
        {heading('WORKSPACE', '通用设置', '工作空间与规则引擎的基础运行信息。', version)}
        <div className="settings-config-grid">{group('工作空间', '管理端显示与本地环境', [['产品名称', 'SnakeRule'], ['当前版本', `v${version}`], ['浏览器时区', Intl.DateTimeFormat().resolvedOptions().timeZone], ['界面语言', '简体中文']])}{group('规则引擎', '当前规则执行方式', [['规则集存储', '进程内存'], ['默认决策模式', '同步求值'], ['模拟运行', '已启用，不写执行日志']])}</div>
      </>}{!platform && tab === 'security' && <>
        {heading('ACCOUNT & SECURITY', '安全设置', '当前管理员身份与登录会话保护状态。', version)}
        <div className="settings-account-banner"><span className="settings-account-avatar">{email.slice(0, 1).toUpperCase() || 'A'}</span><div><small>当前登录账号</small><b>{email}</b></div><span className="settings-role-pill"><i/>超级管理员</span></div>
        <div className="settings-config-grid">{group('身份认证', '控制台账号权限', [['账号角色', '超级管理员'], ['身份认证', '已启用']])}{group('会话保护', '浏览器登录会话策略', [['Cookie', 'HttpOnly'], ['同站策略', 'SameSite Lax'], ['会话时长', '24 小时']])}</div>
      </>}</main>
    </div>
  </section></div>;
}

export function Workbench() {
  const [items, setItems] = useState<Ruleset[]>([]);
  const [selected, setSelected] = useState<Ruleset>(starter);
  const [editor, setEditor] = useState(pretty(starter));
  const [facts, setFacts] = useState(pretty({ order: { amount: 12800 }, customer: { isNew: true } }));
  const [decision, setDecision] = useState<Decision | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [version, setVersion] = useState('0.1.0');
  const [busy, setBusy] = useState(false);
  const [screen, setScreen] = useState<'overview' | 'list' | 'design'>('overview');
  const [designMode, setDesignMode] = useState<'canvas' | 'json'>('canvas');
  const [activeRuleId, setActiveRuleId] = useState('');
  const [dragRuleId, setDragRuleId] = useState('');
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('全部状态');
  const [authState, setAuthState] = useState<'checking' | 'anonymous' | 'authenticated'>('checking');
  const [email, setEmail] = useState('');
  const [isSuperAdmin, setIsSuperAdmin] = useState(false);
  const [settingsArea, setSettingsArea] = useState<'platform' | 'system' | null>(null);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [activeWorkspaceID, setActiveWorkspaceID] = useState('default');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [rulesNavExpanded, setRulesNavExpanded] = useState(true);

  const refreshWorkspaces = useCallback(async () => {
    const result = await snakeRuleApi.listWorkspaces();
    setWorkspaces(result.items);
    setActiveWorkspaceID(result.active_workspace_id);
  }, []);

  const refresh = useCallback(async (id?: string) => {
    const [list, metaResponse] = await Promise.all([snakeRuleApi.listRulesets(), fetch('/api/v1/meta')]);
    const meta = await metaResponse.json();
    setVersion(meta.version ?? '0.1.0'); setItems(list.items);
    const item = list.items.find((candidate) => candidate.id === (id ?? selected.id)) ?? list.items[0];
    if (item) { setSelected(item); setEditor(pretty(item)); }
    else { setSelected(starter); setEditor(pretty(starter)); }
  }, [selected.id]);
  useEffect(() => {
    let active = true;
    snakeRuleApi.getSession().then((session) => {
      if (!active) return;
      if (!session.authenticated) { setAuthState('anonymous'); return; }
      setEmail(session.user?.email ?? ''); setIsSuperAdmin(session.user?.is_superadmin ?? false); setAuthState('authenticated');
      Promise.all([refresh(), refreshWorkspaces()]).catch((e: Error) => setError(`无法连接 API：${e.message}`));
    }).catch(() => { if (active) setAuthState('anonymous'); });
    return () => { active = false; };
  }, []);
  useEffect(() => {
    const onUnauthorized = () => { setAuthState('anonymous'); setEmail(''); setIsSuperAdmin(false); setItems([]); setDecision(null); setScreen('list'); };
    window.addEventListener('snakerule:unauthorized', onUnauthorized);
    return () => window.removeEventListener('snakerule:unauthorized', onUnauthorized);
  }, []);

  const filtered = useMemo(() => items.filter((item) => {
    const matches = `${item.name} ${item.id}`.toLowerCase().includes(query.toLowerCase());
    return matches && (statusFilter === '全部状态' || (statusFilter === '启用中' ? item.enabled : !item.enabled));
  }), [items, query, statusFilter]);

  function open(item: Ruleset) { setSelected(item); setEditor(pretty(item)); setDecision(null); setError(''); setNotice(''); setDesignMode('canvas'); setActiveRuleId(item.rules[0]?.id ?? ''); setScreen('design'); }
  async function switchWorkspace(workspaceID: string) {
    if (!workspaceID || workspaceID === activeWorkspaceID) return;
    setBusy(true); setError('');
    try {
      await snakeRuleApi.setActiveWorkspace(workspaceID);
      setActiveWorkspaceID(workspaceID); setScreen('list'); setDecision(null); setActiveRuleId('');
      await refresh();
    } catch (e) { setError(e instanceof Error ? e.message : '切换工作空间失败'); }
    finally { setBusy(false); }
  }
  function create() {
    const item: Ruleset = { ...starter, id: `ruleset-${Date.now()}` };
    setSelected(item); setEditor(pretty(item)); setDecision(null); setNotice('新规则集尚未保存'); setError(''); setDesignMode('canvas'); setActiveRuleId(''); setScreen('design');
  }
  function updateRuleset(change: (value: Ruleset) => Ruleset) {
    try { const value = change(JSON.parse(editor) as Ruleset); setEditor(pretty(value)); setSelected(value); }
    catch { setError('规则定义格式有误，请切换到 JSON 视图修复后再继续。'); }
  }
  function addRule() {
    const id = `rule-${Date.now()}`;
    const rule: Rule = { id, name: '新条件规则', priority: 100, enabled: true, conditions: { fact: 'order.amount', operator: 'greaterThan', value: 0 }, actions: [] };
    updateRuleset((value) => ({ ...value, rules: [...value.rules, rule] })); setActiveRuleId(id);
  }
  function addAction(type: 'tag' | 'log') {
    const target = (JSON.parse(editor) as Ruleset).rules.find((rule) => rule.id === activeRuleId) ?? (JSON.parse(editor) as Ruleset).rules[0];
    if (!target) {
      const id = `rule-${Date.now()}`;
      const rule: Rule = { id, name: '新条件规则', priority: 100, enabled: true, conditions: { fact: 'order.amount', operator: 'greaterThan', value: 0 }, actions: [{ type, config: type === 'tag' ? { label: 'REVIEW' } : { message: 'rule matched' } }] };
      updateRuleset((value) => ({ ...value, rules: [...value.rules, rule] })); setActiveRuleId(id); return;
    }
    updateRuleset((value) => ({ ...value, rules: value.rules.map((rule) => rule.id === target.id ? { ...rule, actions: [...(rule.actions ?? []), { type, config: type === 'tag' ? { label: 'REVIEW' } : { message: 'rule matched' } }] } : rule) }));
  }
  function dropComponent(event: DragEvent<HTMLElement>) {
    event.preventDefault(); const type = event.dataTransfer.getData('application/snakerule-component');
    if (type === 'condition') addRule(); else if (type === 'tag' || type === 'log') addAction(type);
  }
  function reorderRule(targetId: string) {
    if (!dragRuleId || dragRuleId === targetId) return;
    updateRuleset((value) => {
      const rules = [...value.rules]; const from = rules.findIndex((rule) => rule.id === dragRuleId); const to = rules.findIndex((rule) => rule.id === targetId);
      if (from < 0 || to < 0) return value;
      const [moved] = rules.splice(from, 1); rules.splice(to, 0, moved);
      return { ...value, rules: rules.map((rule, index) => ({ ...rule, priority: (rules.length - index) * 10 })) };
    }); setDragRuleId('');
  }
  function updateActiveRule(change: (rule: Rule) => Rule) {
    updateRuleset((value) => ({ ...value, rules: value.rules.map((rule) => rule.id === activeRuleId ? change(rule) : rule) }));
  }
  function patchFirstCondition(condition: Condition, patch: Partial<Condition>): Condition {
    if (condition.fact) return { ...condition, ...patch };
    if (condition.all?.length) return { ...condition, all: condition.all.map((child, index) => index === 0 ? patchFirstCondition(child, patch) : child) };
    if (condition.any?.length) return { ...condition, any: condition.any.map((child, index) => index === 0 ? patchFirstCondition(child, patch) : child) };
    if (condition.not) return { ...condition, not: patchFirstCondition(condition.not, patch) };
    return { ...condition, ...patch };
  }
  const conditionOf = (condition: Condition): Condition => condition.fact ? condition : condition.all?.[0] ?? condition.any?.[0] ?? {};
  const activeRule = selected.rules?.find((rule) => rule.id === activeRuleId);
  const activeCondition = activeRule ? conditionOf(activeRule.conditions) : undefined;
  const setPaletteDrag = (event: DragEvent<HTMLButtonElement>, component: string) => { event.dataTransfer.setData('application/snakerule-component', component); event.dataTransfer.effectAllowed = 'copy'; };
  async function save() {
    setBusy(true); setError(''); setNotice('');
    try {
      const value = JSON.parse(editor) as Ruleset;
      if (!value.id?.trim() || !value.name?.trim()) throw new Error('规则集 ID 和名称不能为空');
      const saved = await snakeRuleApi.saveRuleset(value);
      setSelected(saved); setEditor(pretty(saved)); setNotice('规则集已保存'); await refresh(saved.id);
    } catch (e) { setError(e instanceof Error ? e.message : '保存失败'); }
    finally { setBusy(false); }
  }
  async function evaluate() {
    setBusy(true); setError(''); setDecision(null);
    try { const value = JSON.parse(editor) as Ruleset; const input = JSON.parse(facts); await snakeRuleApi.saveRuleset(value); setDecision(await snakeRuleApi.evaluate({ rulesetId: value.id, facts: input })); }
    catch (e) { setError(e instanceof Error ? e.message : '运行失败'); }
    finally { setBusy(false); }
  }

  async function logout() {
    setBusy(true); setError('');
    try { await snakeRuleApi.logout(); setAuthState('anonymous'); setEmail(''); setIsSuperAdmin(false); setSettingsArea(null); setItems([]); setDecision(null); setScreen('list'); }
    catch (e) { setError(e instanceof Error ? e.message : '退出失败，请重试。'); }
    finally { setBusy(false); }
  }

  if (authState === 'checking') return <main className="auth-loading"><span className="brand-icon">S</span><b>正在验证登录状态…</b></main>;
  if (authState === 'anonymous') return <LoginScreen onSuccess={(email) => { setEmail(email); setIsSuperAdmin(true); setAuthState('authenticated'); Promise.all([refresh(), refreshWorkspaces()]).catch((e: Error) => setError(`无法连接 API：${e.message}`)); }} />;

  return <div className={`shell${sidebarCollapsed ? ' sidebar-collapsed' : ''}`}>
    <aside className="sidebar">
      <div className="sidebar-brand-row"><a className="brand" href="/"><span className="brand-icon">S</span><span>SnakeRule<small>RULE ENGINE</small></span></a><button className="sidebar-toggle" aria-label={sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'} title={sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'} onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}>{sidebarCollapsed ? '›' : '‹'}</button></div>
      <div className="side-label">工作空间</div>
      <button className={`side-nav ${screen === 'overview' ? 'side-nav-active' : ''}`} title={sidebarCollapsed ? '概览' : undefined} onClick={() => setScreen('overview')}><span><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 3h5v6H3zM12 3h5v4h-5zM12 10h5v7h-5zM3 12h5v5H3z"/></svg></span> 概览</button>
      <button className={`side-nav ${screen !== 'overview' ? 'side-nav-active' : ''}`} title={sidebarCollapsed ? '规则管理' : undefined} aria-expanded={rulesNavExpanded} onClick={() => setRulesNavExpanded((expanded) => !expanded)}><span><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M4 4.5h12M4 10h12M4 15.5h12"/><circle cx="2.5" cy="4.5" r=".6"/><circle cx="2.5" cy="10" r=".6"/><circle cx="2.5" cy="15.5" r=".6"/></svg></span> 规则管理 <b className={rulesNavExpanded ? 'nav-chevron expanded' : 'nav-chevron'}>⌄</b></button>
      {rulesNavExpanded && <div className="subnav"><button className="subnav-active" onClick={() => setScreen('list')}><i/>规则集</button><button onClick={() => setScreen('list')}><i/>决策规则</button><button onClick={() => setScreen('list')}><i/>执行记录</button></div>}
      <div className="sidebar-note"><span>⌘</span><b>规则引擎</b><small>确定性决策 · 可追溯执行</small><em>v{version}</em></div>
    </aside>
    <main className="main">
      <header className="topbar"><span>{screen === 'overview' ? '概览' : screen === 'list' ? '规则管理　/　规则集' : `规则管理　/　${selected.name}`} </span><div className="topbar-actions"><label className="workspace-switcher"><small>工作空间</small><CustomSelect ariaLabel="切换工作空间" value={activeWorkspaceID} disabled={busy || !workspaces.length} options={workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name }))} onChange={(value) => void switchWorkspace(value)}/></label>{isSuperAdmin && <button className="settings-entry platform-entry" onClick={() => setSettingsArea('platform')}>◆ 平台设置</button>}<button className="settings-entry" onClick={() => setSettingsArea('system')}>⚙ 系统设置</button><details className="user-menu"><summary className="topbar-user" title={`${email} · 超级管理员`}><span className="avatar">{email.slice(0, 2).toUpperCase() || 'SR'}</span><span><b>{email}</b><small>超级管理员</small></span><i>⌄</i></summary><div className="user-menu-popover"><div className="user-menu-identity"><span className="avatar">{email.slice(0, 2).toUpperCase() || 'SR'}</span><div><b>超级管理员</b><small>{email}</small></div></div><button className="user-menu-logout" onClick={logout} disabled={busy}>↪ <span>退出登录</span></button></div></details></div></header>
      <div className="content">
        {screen === 'overview' ? <>
          <section className="title-row"><div><small className="eyebrow">WORKSPACE OVERVIEW</small><h1>概览</h1><p>当前工作空间的规则资产、运行状态与快捷入口。</p></div><button className="primary create-button" onClick={create}>＋ 新建规则集</button></section>
          <section className="metrics list-metrics"><div><small>当前工作空间</small><b>{workspaces.find((workspace) => workspace.id === activeWorkspaceID)?.name ?? '默认工作空间'}</b><em>Slug · {workspaces.find((workspace) => workspace.id === activeWorkspaceID)?.slug ?? 'default'}</em></div><div><small>规则集</small><b>{items.length}</b><em>当前空间中的全部规则集</em></div><div><small>启用规则集</small><b>{items.filter((item) => item.enabled).length}</b><em>可参与决策求值</em></div><div><small>规则总数</small><b>{items.reduce((sum, item) => sum + item.rules.length, 0)}</b><em>引擎版本 · v{version}</em></div></section>
          <section className="overview-shortcuts"><button onClick={() => setScreen('list')}><span className="shortcut-icon rules-shortcut">↗</span><span><b>规则管理</b><small>查看、编辑和调试规则集</small></span><i>→</i></button><button onClick={() => setSettingsArea('platform')}><span className="shortcut-icon workspace-shortcut">⌘</span><span><b>工作空间</b><small>创建空间并管理空间信息</small></span><i>→</i></button><div className="overview-health"><span className="health-dot"/><span><b>平台服务正常</b><small>管理端已连接规则引擎 API</small></span></div></section>
          <section className="list-panel"><div className="list-toolbar"><div><b>最近规则集</b><small>共 {items.length} 个</small></div><button className="quiet" onClick={() => setScreen('list')}>查看全部规则集　→</button></div><div className="table-wrap"><table><thead><tr><th>规则集名称</th><th>规则数</th><th>状态</th><th>版本</th><th/></tr></thead><tbody>{items.slice(0, 5).map((item) => <tr key={item.id} onClick={() => open(item)}><td><div className="table-name"><span>{(item.name || 'R')[0]}</span><div><b>{item.name}</b><small>{item.id}</small></div></div></td><td>{item.rules.length} 条</td><td><span className={item.enabled ? 'status-pill enabled' : 'status-pill'}><i/>{item.enabled ? '启用中' : '已停用'}</span></td><td><code>{item.version || '—'}</code></td><td><button className="row-action">设计 <span>→</span></button></td></tr>)}{!items.length && <tr><td colSpan={5}><div className="table-empty"><b>当前空间还没有规则集</b><small>新建规则集，开始配置决策规则。</small><button className="secondary" onClick={create}>＋ 新建规则集</button></div></td></tr>}</tbody></table></div></section>
        </> : screen === 'list' ? <>
          <section className="title-row"><div><small className="eyebrow">RULE MANAGEMENT</small><h1>规则集</h1><p>集中管理规则定义，检查状态并进入设计与调试。</p></div><button className="primary create-button" onClick={create}>＋ 新建规则集</button></section>
          <section className="metrics list-metrics"><div><small>规则集总数</small><b>{items.length}</b><em>当前工作空间</em></div><div><small>启用中</small><b>{items.filter((item) => item.enabled).length}</b><em>可参与决策</em></div><div><small>规则总数</small><b>{items.reduce((sum, item) => sum + item.rules.length, 0)}</b><em>全部规则集</em></div><div><small>引擎版本</small><b className="version-metric">v{version}</b><em>当前服务</em></div></section>
          <section className="list-panel"><div className="list-toolbar"><div><b>全部规则集</b><small>共 {filtered.length} 个</small></div><div className="filters"><label className="search-box"><span>⌕</span><input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="搜索名称或 ID"/></label><CustomSelect ariaLabel="状态筛选" value={statusFilter} options={['全部状态', '启用中', '已停用'].map((label) => ({ value: label, label }))} onChange={setStatusFilter}/><button className="quiet refresh-button" onClick={() => refresh().catch((e: Error) => setError(e.message))}>↻ 刷新</button></div></div>
            <div className="table-wrap"><table><thead><tr><th>规则集名称</th><th>规则数</th><th>状态</th><th>版本</th><th>最近更新</th><th/></tr></thead><tbody>
              {filtered.map((item) => <tr key={item.id} onClick={() => open(item)}><td><div className="table-name"><span>{(item.name || 'R')[0]}</span><div><b>{item.name}</b><small>{item.id}</small></div></div></td><td>{item.rules.length} 条</td><td><span className={item.enabled ? 'status-pill enabled' : 'status-pill'}><i/>{item.enabled ? '启用中' : '已停用'}</span></td><td><code>{item.version || '—'}</code></td><td>{dateLabel()}</td><td><button className="row-action" onClick={(e) => { e.stopPropagation(); open(item); }}>设计 <span>→</span></button></td></tr>)}
              {!filtered.length && <tr><td colSpan={6}><div className="table-empty"><span>⌕</span><b>{items.length ? '没有匹配的规则集' : '还没有规则集'}</b><small>{items.length ? '调整搜索词或状态筛选。' : '新建规则集，开始配置你的第一条决策规则。'}</small>{!items.length && <button className="secondary" onClick={create}>＋ 新建规则集</button>}</div></td></tr>}
            </tbody></table></div>
          </section>
        </> : <>
          <div className="crumb-row"><button className="back-link" onClick={() => setScreen('list')}>← 返回规则集</button><span>/</span><span>{selected.name || '未命名规则集'}</span><span className={selected.enabled ? 'tag-on' : 'tag-off'}>{selected.enabled ? '启用中' : '已停用'}</span></div>
          <section className="title-row design-title"><div><small className="eyebrow">RULE DESIGNER</small><h1>{selected.name || '未命名规则集'}</h1><p>{selected.description || '编辑规则定义，并通过测试事实验证决策结果。'}</p></div><div className="title-actions"><button className="secondary" onClick={() => setScreen('list')}>取消</button><button className="primary" disabled={busy} onClick={save}>保存规则</button></div></section>
          <section className="design-meta"><div><small>规则集 ID</small><b>{selected.id}</b></div><div><small>版本</small><b>{selected.version || '—'}</b></div><div><small>规则数量</small><b>{selected.rules?.length ?? 0}</b></div><div className="meta-note"><span>ⓘ</span> 保存后规则定义立即更新；调试结果不会写入执行记录。</div></section>
          <div className="designer-toolbar"><div><span className="canvas-live-dot"/>设计画布 <small>拖入组件构建规则流程</small></div><div className="mode-switch"><button className={designMode === 'canvas' ? 'selected' : ''} onClick={() => setDesignMode('canvas')}>画布</button><button className={designMode === 'json' ? 'selected' : ''} onClick={() => setDesignMode('json')}>JSON</button></div></div>
          {designMode === 'canvas' ? <div className="canvas-layout">
            <aside className="component-palette"><div className="palette-title">组件库 <small>拖拽到画布</small></div><div className="palette-group-label">规则节点</div>
              <button className="palette-component" draggable onDragStart={(e) => setPaletteDrag(e, 'condition')}><span className="component-icon condition-icon">◇</span><span><b>条件规则</b><small>字段与条件匹配</small></span><em>⠿</em></button>
              <button className="palette-component" draggable onDragStart={(e) => setPaletteDrag(e, 'tag')}><span className="component-icon output-icon">↗</span><span><b>输出动作</b><small>命中后添加标签</small></span><em>⠿</em></button>
              <button className="palette-component" draggable onDragStart={(e) => setPaletteDrag(e, 'log')}><span className="component-icon log-icon">≋</span><span><b>日志动作</b><small>记录规则命中信息</small></span><em>⠿</em></button>
              <div className="palette-tip"><b>使用提示</b><small>拖动规则节点可调整优先级。点击节点后，在右侧编辑条件和动作。</small></div>
            </aside>
            <section className="flow-canvas" onDragOver={(e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'copy'; }} onDrop={dropComponent}>
              <div className="canvas-caption"><span>流程画布</span><small>由上至下按优先级求值</small></div>
              <div className="flow-start"><span>▶</span><div><b>开始</b><small>接收事实并开始匹配</small></div></div><div className="flow-connector"><i/></div>
              {selected.rules?.map((rule, index) => <div className="flow-rule-wrap" key={rule.id}>
                <article className={`flow-rule-node ${activeRuleId === rule.id ? 'node-selected' : ''}`} draggable onDragStart={() => setDragRuleId(rule.id)} onDragEnd={() => setDragRuleId('')} onDragOver={(e) => { if (dragRuleId) e.preventDefault(); }} onDrop={(e) => { if (dragRuleId) { e.preventDefault(); e.stopPropagation(); reorderRule(rule.id); } }} onClick={() => setActiveRuleId(rule.id)}>
                  <div className="node-topline"><span className="node-type"><i/> 条件规则</span><span className="node-priority">优先级 {rule.priority}</span><button className="node-menu" aria-label="删除规则" onClick={(e) => { e.stopPropagation(); updateRuleset((value) => ({ ...value, rules: value.rules.filter((item) => item.id !== rule.id) })); if (activeRuleId === rule.id) setActiveRuleId(''); }}>×</button></div>
                  <b className="node-title">{rule.name || '未命名规则'}</b><div className="node-condition"><span>IF</span><code>{conditionOf(rule.conditions).fact || '配置条件'} {conditionOf(rule.conditions).operator || ''} {String(conditionOf(rule.conditions).value ?? '')}</code></div>
                  <div className="node-actions">{rule.actions?.length ? rule.actions.map((action, actionIndex) => <span key={`${action.type}-${actionIndex}`}><i>↗</i>{action.type === 'tag' ? `标签：${String(action.config?.label ?? '未命名')}` : action.type}</span>) : <span className="no-action">尚未配置命中动作</span>}</div>
                </article>
                {index < selected.rules.length - 1 && <div className="flow-connector"><i/></div>}
              </div>)}
              {!selected.rules?.length && <div className="canvas-empty"><span>＋</span><b>从左侧拖入第一个规则组件</b><small>也可以点击下方按钮添加条件规则</small><button className="secondary" onClick={addRule}>＋ 添加条件规则</button></div>}
              <div className="drop-hint">＋ 将组件拖到这里添加到规则流程</div>
            </section>
            <aside className="node-inspector"><div className="inspector-head"><div><small>属性配置</small><b>{activeRule ? '条件规则' : '节点属性'}</b></div><span>⚙</span></div>{activeRule && activeCondition ? <>
              <label>规则名称<input value={activeRule.name} onChange={(e) => updateActiveRule((rule) => ({ ...rule, name: e.target.value }))}/></label>
              <label>优先级<input type="number" value={activeRule.priority} onChange={(e) => updateActiveRule((rule) => ({ ...rule, priority: Number(e.target.value) }))}/></label>
              <div className="inspector-divider"/><div className="inspector-section-title">条件配置 <small>满足条件时触发</small></div>
              <label>事实字段<input placeholder="例如 order.amount" value={activeCondition.fact ?? ''} onChange={(e) => updateActiveRule((rule) => ({ ...rule, conditions: patchFirstCondition(rule.conditions, { fact: e.target.value }) }))}/></label>
              <label>匹配方式<CustomSelect ariaLabel="匹配方式" value={activeCondition.operator ?? 'equal'} options={[['equal','等于'],['notEqual','不等于'],['greaterThan','大于'],['greaterThanInclusive','大于等于'],['lessThan','小于'],['lessThanInclusive','小于等于']].map(([value,label]) => ({value, label}))} onChange={(value) => updateActiveRule((rule) => ({ ...rule, conditions: patchFirstCondition(rule.conditions, { operator: value }) }))}/></label>
              <label>比较值{typeof activeCondition.value === 'boolean' ? <CustomSelect ariaLabel="比较值" value={String(activeCondition.value)} options={[{value:'true',label:'true'},{value:'false',label:'false'}]} onChange={(value) => updateActiveRule((rule) => ({ ...rule, conditions: patchFirstCondition(rule.conditions, { value: value === 'true' }) }))}/> : <input type={typeof activeCondition.value === 'number' ? 'number' : 'text'} value={String(activeCondition.value ?? '')} onChange={(e) => updateActiveRule((rule) => ({ ...rule, conditions: patchFirstCondition(rule.conditions, { value: typeof activeCondition.value === 'number' ? Number(e.target.value) : e.target.value }) }))}/>}</label>
              <div className="inspector-divider"/><div className="inspector-section-title">命中动作 <small>{activeRule.actions?.length ?? 0} 个动作</small></div>
              {(activeRule.actions ?? []).map((action, index) => <div className="action-edit" key={`${action.type}-${index}`}><span>{action.type === 'tag' ? '标签输出' : action.type}</span><input value={String(action.config?.label ?? action.config?.message ?? '')} onChange={(e) => updateActiveRule((rule) => ({ ...rule, actions: rule.actions?.map((item, i) => i === index ? { ...item, config: { ...item.config, [item.type === 'tag' ? 'label' : 'message']: e.target.value } } : item) }))}/><button aria-label="移除动作" onClick={() => updateActiveRule((rule) => ({ ...rule, actions: rule.actions?.filter((_, i) => i !== index) }))}>×</button></div>)}
              <button className="add-action" onClick={() => addAction('tag')}>＋ 添加标签动作</button>
            </> : <div className="inspector-empty"><span>◇</span><b>选择一个规则节点</b><small>在画布点击节点，编辑条件和动作。</small></div>}
              <div className="inspector-save"><span>{notice || '更改尚未保存'}</span><button className="primary" disabled={busy} onClick={save}>保存规则</button></div>
            </aside>
          </div> : <section className="panel json-panel"><div className="panel-head"><span className="step">{`{ }`}</span><div><b>规则定义</b><small>JSON DSL · all / any / not 条件组</small></div><span className="json-tag">JSON</span></div><div className="editor-hint"><span>✳</span> 高级编辑模式。规则按优先级从高到低求值。</div><textarea className="rule-editor" aria-label="规则集 JSON 编辑器" spellCheck={false} value={editor} onChange={(e) => { setEditor(e.target.value); try { setSelected(JSON.parse(e.target.value) as Ruleset); } catch { /* keep partial input */ } }}/><div className="panel-bottom"><span>{notice || '修改后保存以更新规则集'}</span><button className="primary" disabled={busy} onClick={save}>保存规则集</button></div></section>}
          <div className="design-bottom"><section className="panel test-panel"><div className="panel-head"><span className="step blue">02</span><div><b>模拟决策</b><small>输入事实并查看命中规则</small></div><span className="json-tag">TEST</span></div><label className="facts-label" htmlFor="facts">测试事实 <small>FACTS · JSON</small></label><textarea id="facts" className="facts-editor" spellCheck={false} value={facts} onChange={(e) => setFacts(e.target.value)}/><button className="run-button" disabled={busy} onClick={evaluate}>运行模拟 <span>→</span></button><div className="result-head"><b>运行结果</b>{decision && <span className={decision.matched ? 'tag-on' : 'tag-off'}>{decision.matched ? `命中 ${decision.hitRules.length} 条` : '无规则命中'}</span>}</div>{decision ? <pre className="result">{pretty(decision)}</pre> : <div className="empty-result">◎<small>模拟执行结果将在这里显示</small></div>}</section></div>
        </>}
        {error && <div className="error-message" role="alert">{error}</div>}
        <footer>SnakeRule　·　确定性规则求值　·　v{version}</footer>
      </div>
    </main>
    {settingsArea && <SettingsDialog area={settingsArea} email={email} version={version} workspaces={workspaces} activeWorkspaceID={activeWorkspaceID} onWorkspaceRefresh={refreshWorkspaces} onClose={() => setSettingsArea(null)} />}
  </div>;
}
