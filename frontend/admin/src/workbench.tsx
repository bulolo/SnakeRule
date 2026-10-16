import {
  useCallback,
  useEffect,
  useState,
  type DragEvent,
} from "react";
import { CustomSelect, LoginScreen } from "./components/common";
import { SettingsDialog } from "./components/settings";
import { Toaster, toast } from "sonner";
import { ProjectsPanel, type Project } from "./components/projects";
import {
  snakeRuleApi,
  type Condition,
  type Decision,
  type Rule,
  type Ruleset,
  type Organization,
  type InvocationLog,
  type PlatformSettings,
} from "./lib/sdk/generated";

const starter: Ruleset = {
  id: "new-ruleset",
  name: "新规则集",
  version: "1.0.0",
  description: "",
  enabled: true,
  rules: [],
};

const pretty = (value: unknown) => JSON.stringify(value, null, 2);
export function Workbench() {
  const [selected, setSelected] = useState<Ruleset>(starter);
  const [editor, setEditor] = useState(pretty(starter));
  const [facts, setFacts] = useState(
    pretty({ order: { amount: 12800 }, customer: { isNew: true } }),
  );
  const [decision, setDecision] = useState<Decision | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [version, setVersion] = useState("0.0.2");
  const [platformBrand, setPlatformBrand] = useState<PlatformSettings>({ name: "SnakeRule", tagline: "确定性规则求值平台", logo_url: "" });
  const [busy, setBusy] = useState(false);
  const [screen, setScreen] = useState<"overview" | "design" | "projects">(() => {
    const saved = sessionStorage.getItem("snakerule:screen");
    return saved === "projects" || saved === "design" ? "projects" : "overview";
  });
  const [designMode, setDesignMode] = useState<"canvas" | "json">("canvas");
  const [activeRuleId, setActiveRuleId] = useState("");
  const [dragRuleId, setDragRuleId] = useState("");
  const [authState, setAuthState] = useState<
    "checking" | "anonymous" | "authenticated"
  >("checking");
  const [email, setEmail] = useState("");
  const [isSuperAdmin, setIsSuperAdmin] = useState(false);
  const [organizationRole, setOrganizationRole] = useState("");
  const [settingsArea, setSettingsArea] = useState<
    "platform" | "system" | null
  >(null);
  const [alertChannels, setAlertChannels] = useState<{ id: string; name: string; type: string; enabled: boolean }[]>([]);
  const [notifications, setNotifications] = useState<{ id: string; title: string; message: string; created_at: string; read: boolean }[]>([]);
  const [ruleAlertModalOpen, setRuleAlertModalOpen] = useState(false);
  const [ruleAlertOverride, setRuleAlertOverride] = useState(false);
  const [ruleAlertEnabled, setRuleAlertEnabled] = useState(false);
  const [ruleAlertChannelIDs, setRuleAlertChannelIDs] = useState<string[]>([]);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [activeOrganizationID, setActiveOrganizationID] = useState("default");
  const [projects, setProjects] = useState<Project[]>([]);
  const [overviewLogs, setOverviewLogs] = useState<InvocationLog[]>([]);
  const [overviewLogsBusy, setOverviewLogsBusy] = useState(false);
  const [globalLogsOpen, setGlobalLogsOpen] = useState(false);
  const [globalLogPage, setGlobalLogPage] = useState(1);
  const [globalLogSelected, setGlobalLogSelected] = useState<InvocationLog | null>(null);
  const [globalLogTab, setGlobalLogTab] = useState<"summary" | "request" | "response">("summary");
  const [openProjectID, setOpenProjectID] = useState("");
  const [projectDetailOpen, setProjectDetailOpen] = useState(false);
  const [editorReturnScreen, setEditorReturnScreen] = useState<"projects">("projects");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  useEffect(() => {
    void snakeRuleApi.getPlatformSettings().then(setPlatformBrand).catch(() => undefined);
  }, []);
  useEffect(() => { document.title = platformBrand.name || "SnakeRule"; }, [platformBrand.name]);

  const refreshOrganizations = useCallback(async () => {
    const result = await snakeRuleApi.listOrganizations();
    setOrganizations(result.list);
    setActiveOrganizationID(result.active_organization_id);
  }, []);

  const refreshProjects = useCallback(async () => {
    const body = await snakeRuleApi.listProjects();
    setProjects(body.list ?? []);
    setOpenProjectID(body.active_project_id ?? "");
  }, []);

  useEffect(() => {
    if (screen !== "overview" || !projects.length) { setOverviewLogs([]); return; }
    let cancelled = false;
    setOverviewLogsBusy(true);
    Promise.all(projects.map((project) => snakeRuleApi.listProjectInvocations(project.id, { is_pager: 0 })))
      .then((responses) => { if (!cancelled) setOverviewLogs(responses.flatMap((response) => response.list ?? [])); })
      .catch((error: Error) => { if (!cancelled) toast.error("加载调用统计失败", { description: error.message }); })
      .finally(() => { if (!cancelled) setOverviewLogsBusy(false); });
    return () => { cancelled = true; };
  }, [screen, projects]);

  const overviewDays = Array.from({ length: 7 }, (_, index) => {
    const date = new Date(); date.setHours(0, 0, 0, 0); date.setDate(date.getDate() - (6 - index));
    const next = new Date(date); next.setDate(next.getDate() + 1);
    const dayLogs = overviewLogs.filter((item) => { const time = new Date(item.created_at); return time >= date && time < next; });
    return { label: `${date.getMonth() + 1}/${date.getDate()}`, total: dayLogs.length, matched: dayLogs.filter((item) => item.matched).length, failed: dayLogs.filter((item) => item.status_code >= 400).length };
  });
  const latestOverviewLogs = [...overviewLogs].sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at)).slice(0, 10);
  const sortedOverviewLogs = [...overviewLogs].sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at));
  const globalLogPageSize = 20;
  const globalLogPageCount = Math.max(1, Math.ceil(sortedOverviewLogs.length / globalLogPageSize));
  const pagedOverviewLogs = sortedOverviewLogs.slice((globalLogPage - 1) * globalLogPageSize, globalLogPage * globalLogPageSize);

  const refreshAlertChannels = useCallback(async () => {
    const body = await snakeRuleApi.listOrganizationAlertChannels(activeOrganizationID);
    setAlertChannels((body.list ?? []).filter((item: { enabled: boolean }) => item.enabled));
  }, [activeOrganizationID]);

  const refreshNotifications = useCallback(async () => {
    const body = await snakeRuleApi.listNotifications();
    setNotifications(body.list ?? []);
  }, []);

  async function markNotificationsRead(id?: string) {
    try {
      if (id) await snakeRuleApi.readNotification(id);
      else await snakeRuleApi.readAllNotifications();
      await refreshNotifications();
    } catch (error) {
      toast.error("更新通知状态失败", { description: error instanceof Error ? error.message : "请求失败" });
    }
  }

  const refresh = useCallback(
    async (id?: string) => {
      const [list, meta] = await Promise.all([
        snakeRuleApi.listRulesets(),
        snakeRuleApi.getMeta(),
      ]);
      setVersion(meta.version ?? "0.0.2");
      const item =
        list.list.find((candidate) => candidate.id === (id ?? selected.id)) ??
        list.list[0];
      if (item) {
        setSelected(item);
        setEditor(pretty(item));
      } else {
        setSelected(starter);
        setEditor(pretty(starter));
      }
    },
    [selected.id],
  );
  useEffect(() => {
    let active = true;
    snakeRuleApi
      .getSession()
      .then((session) => {
        if (!active) return;
        if (!session.authenticated) {
          setAuthState("anonymous");
          return;
        }
        setEmail(session.user?.email ?? "");
        setIsSuperAdmin(session.user?.is_superadmin ?? false);
        setOrganizationRole(session.user?.role ?? "");
        setAuthState("authenticated");
        Promise.all([refreshOrganizations(), refreshProjects(), refreshAlertChannels(), refreshNotifications()]).catch((e: Error) =>
          toast.error("工作台数据加载失败", { description: e.message }),
        );
      })
      .catch(() => {
        if (active) setAuthState("anonymous");
      });
    return () => {
      active = false;
    };
  }, []);
  useEffect(() => {
    if (authState !== "authenticated") return;
    const timer = window.setInterval(() => { void refreshNotifications().catch(() => undefined); }, 15000);
    return () => window.clearInterval(timer);
  }, [authState, refreshNotifications]);
  useEffect(() => {
    const closeMenus = (event: PointerEvent) => {
      document.querySelectorAll<HTMLElement>(".notification-menu[open], .user-menu[open]").forEach((menu) => {
        if (!menu.contains(event.target as Node)) menu.removeAttribute("open");
      });
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") document.querySelectorAll<HTMLElement>(".notification-menu[open], .user-menu[open]").forEach((menu) => menu.removeAttribute("open"));
    };
    document.addEventListener("pointerdown", closeMenus);
    document.addEventListener("keydown", closeOnEscape);
    return () => { document.removeEventListener("pointerdown", closeMenus); document.removeEventListener("keydown", closeOnEscape); };
  }, []);
  useEffect(() => {
    const onUnauthorized = () => {
      setAuthState("anonymous");
      setEmail("");
      setIsSuperAdmin(false);
      setOrganizationRole("");
      setDecision(null);
      setScreen("projects");
    };
    window.addEventListener("snakerule:unauthorized", onUnauthorized);
    return () =>
      window.removeEventListener("snakerule:unauthorized", onUnauthorized);
  }, []);

  useEffect(() => { sessionStorage.setItem("snakerule:screen", screen); }, [screen]);

  function open(item: Ruleset, returnScreen: "projects" = "projects") {
    setSelected(item);
    setEditor(pretty(item));
    setDecision(null);
    setError("");
    setNotice("");
    setDesignMode("canvas");
    setActiveRuleId(item.rules[0]?.id ?? "");
    setEditorReturnScreen(returnScreen);
    setScreen("design");
  }
  async function switchOrganization(organizationID: string) {
    if (!organizationID || organizationID === activeOrganizationID) return;
    setBusy(true);
    setError("");
    try {
      await snakeRuleApi.setActiveOrganization(organizationID);
      setActiveOrganizationID(organizationID);
      const session = await snakeRuleApi.getSession();
      setOrganizationRole(session.user?.role ?? "");
      setOpenProjectID("");
      setScreen("projects");
      setDecision(null);
      setActiveRuleId("");
      await refreshProjects();
    } catch (e) {
      setError(e instanceof Error ? e.message : "切换组织失败");
    } finally {
      setBusy(false);
    }
  }
  async function switchProject(id: string) {
    if (!id || id === openProjectID) return;
    setBusy(true); setError("");
    try {
      await snakeRuleApi.setActiveProject(id);
      setOpenProjectID(id); setDecision(null); setActiveRuleId(""); await refresh();
    } catch (e) { setError(e instanceof Error ? e.message : "切换项目失败"); }
    finally { setBusy(false); }
  }
  async function create(name = "新规则集"): Promise<Ruleset> {
    const item: Ruleset = { ...starter, id: `ruleset-${Date.now()}`, name };
    return snakeRuleApi.createRuleset(item);
  }
  function updateRuleset(change: (value: Ruleset) => Ruleset) {
    try {
      const value = change(JSON.parse(editor) as Ruleset);
      setEditor(pretty(value));
      setSelected(value);
    } catch {
      setError("规则定义格式有误，请切换到 JSON 视图修复后再继续。");
    }
  }
  function addRule() {
    const id = `rule-${Date.now()}`;
    const rule: Rule = {
      id,
      name: "新条件规则",
      priority: 100,
      enabled: true,
      conditions: { fact: "order.amount", operator: "greaterThan", value: 0 },
      actions: [],
    };
    updateRuleset((value) => ({ ...value, rules: [...value.rules, rule] }));
    setActiveRuleId(id);
  }
  function addAction(type: "tag" | "log") {
    const target =
      (JSON.parse(editor) as Ruleset).rules.find(
        (rule) => rule.id === activeRuleId,
      ) ?? (JSON.parse(editor) as Ruleset).rules[0];
    if (!target) {
      const id = `rule-${Date.now()}`;
      const rule: Rule = {
        id,
        name: "新条件规则",
        priority: 100,
        enabled: true,
        conditions: { fact: "order.amount", operator: "greaterThan", value: 0 },
        actions: [
          {
            type,
            config:
              type === "tag"
                ? { label: "REVIEW" }
                : { message: "rule matched" },
          },
        ],
      };
      updateRuleset((value) => ({ ...value, rules: [...value.rules, rule] }));
      setActiveRuleId(id);
      return;
    }
    updateRuleset((value) => ({
      ...value,
      rules: value.rules.map((rule) =>
        rule.id === target.id
          ? {
              ...rule,
              actions: [
                ...(rule.actions ?? []),
                {
                  type,
                  config:
                    type === "tag"
                      ? { label: "REVIEW" }
                      : { message: "rule matched" },
                },
              ],
            }
          : rule,
      ),
    }));
  }
  function dropComponent(event: DragEvent<HTMLElement>) {
    event.preventDefault();
    const type = event.dataTransfer.getData("application/snakerule-component");
    if (type === "condition") addRule();
    else if (type === "tag" || type === "log") addAction(type);
  }
  function reorderRule(targetId: string) {
    if (!dragRuleId || dragRuleId === targetId) return;
    updateRuleset((value) => {
      const rules = [...value.rules];
      const from = rules.findIndex((rule) => rule.id === dragRuleId);
      const to = rules.findIndex((rule) => rule.id === targetId);
      if (from < 0 || to < 0) return value;
      const [moved] = rules.splice(from, 1);
      rules.splice(to, 0, moved);
      return {
        ...value,
        rules: rules.map((rule, index) => ({
          ...rule,
          priority: (rules.length - index) * 10,
        })),
      };
    });
    setDragRuleId("");
  }
  function updateActiveRule(change: (rule: Rule) => Rule) {
    updateRuleset((value) => ({
      ...value,
      rules: value.rules.map((rule) =>
        rule.id === activeRuleId ? change(rule) : rule,
      ),
    }));
  }
  function openRuleAlertConfig(rule: Rule) {
    const policy = rule.alert ?? selected.alert;
    setRuleAlertOverride(Boolean(rule.alert));
    setRuleAlertEnabled(policy?.enabled ?? false);
    setRuleAlertChannelIDs(policy?.channel_ids ?? []);
    setRuleAlertModalOpen(true);
  }
  function saveRuleAlertConfig() {
    updateActiveRule((rule) => ({
      ...rule,
      alert: ruleAlertOverride
        ? { enabled: ruleAlertEnabled, channel_ids: ruleAlertEnabled ? ruleAlertChannelIDs : [] }
        : undefined,
    }));
    setRuleAlertModalOpen(false);
    toast.info("通知设置已应用到草稿，请保存规则集后生效");
  }
  function patchFirstCondition(
    condition: Condition,
    patch: Partial<Condition>,
  ): Condition {
    if (condition.fact) return { ...condition, ...patch };
    if (condition.all?.length)
      return {
        ...condition,
        all: condition.all.map((child, index) =>
          index === 0 ? patchFirstCondition(child, patch) : child,
        ),
      };
    if (condition.any?.length)
      return {
        ...condition,
        any: condition.any.map((child, index) =>
          index === 0 ? patchFirstCondition(child, patch) : child,
        ),
      };
    if (condition.not)
      return { ...condition, not: patchFirstCondition(condition.not, patch) };
    return { ...condition, ...patch };
  }
  const conditionOf = (condition: Condition): Condition =>
    condition.fact
      ? condition
      : (condition.all?.[0] ?? condition.any?.[0] ?? {});
  const activeRule = selected.rules?.find((rule) => rule.id === activeRuleId);
  const activeRuleAlert = activeRule?.alert ?? selected.alert;
  const activeCondition = activeRule
    ? conditionOf(activeRule.conditions)
    : undefined;
  const setPaletteDrag = (
    event: DragEvent<HTMLButtonElement>,
    component: string,
  ) => {
    event.dataTransfer.setData("application/snakerule-component", component);
    event.dataTransfer.effectAllowed = "copy";
  };
  async function save() {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const value = JSON.parse(editor) as Ruleset;
      if (!value.id?.trim() || !value.name?.trim())
        throw new Error("规则集 ID 和名称不能为空");
      const saved = await snakeRuleApi.updateRuleset(value);
      setSelected(saved);
      setEditor(pretty(saved));
      setNotice("规则集已保存");
      toast.success("规则集已保存");
    } catch (e) {
      const message = e instanceof Error ? e.message : "保存失败";
      setError(message);
      toast.error("保存规则集失败", { description: message });
    } finally {
      setBusy(false);
    }
  }
  async function removeRuleset(item: Ruleset) {
    setBusy(true);
    setError("");
    try {
      await snakeRuleApi.deleteRuleset(item.id);
      setDecision(null);
      await refresh();
      setNotice(`已删除规则集“${item.name}”`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "删除规则集失败");
    } finally {
      setBusy(false);
    }
  }
  async function evaluate() {
    setBusy(true);
    setError("");
    setDecision(null);
    try {
      const value = JSON.parse(editor) as Ruleset;
      const input = JSON.parse(facts);
      const result = await snakeRuleApi.evaluate({
        ruleset_id: value.id,
        facts: input,
        ruleset: value,
      }, "designer");
      setDecision(result);
      await refreshNotifications();
    } catch (e) {
      setError(e instanceof Error ? e.message : "运行失败");
    } finally {
      setBusy(false);
    }
  }

  async function logout() {
    setBusy(true);
    setError("");
    try {
      await snakeRuleApi.logout();
      setAuthState("anonymous");
      setEmail("");
      setIsSuperAdmin(false);
      setOrganizationRole("");
      setSettingsArea(null);
      setDecision(null);
      setScreen("overview");
    } catch (e) {
      setError(e instanceof Error ? e.message : "退出失败，请重试。");
    } finally {
      setBusy(false);
    }
  }

  if (authState === "checking")
    return (
      <main className="auth-loading">
        <span className="brand-icon">S</span>
        <b>正在验证登录状态…</b>
      </main>
    );
  if (authState === "anonymous")
    return (
      <LoginScreen
        brand={platformBrand}
        onSuccess={(email) => {
          setEmail(email);
          void snakeRuleApi.getSession().then((session) => {
            setIsSuperAdmin(session.user?.is_superadmin ?? false);
            setOrganizationRole(session.user?.role ?? "");
          });
          setAuthState("authenticated");
          Promise.all([refreshOrganizations(), refreshProjects()]).catch((e: Error) =>
            setError(`工作台数据加载失败：${e.message}`),
          );
        }}
      />
    );

  const currentRoleLabel = isSuperAdmin ? "超级管理员" : organizationRole === "organization_admin" ? "组织管理员" : "组织成员";

  return (
    <div className={`shell${sidebarCollapsed ? " sidebar-collapsed" : ""}`}>
      <aside className="sidebar">
        <div className="sidebar-brand-row">
          <a className="brand" href="/">
            <span className={`brand-icon${platformBrand.logo_url ? " has-logo" : ""}`}>{platformBrand.logo_url ? <img src={platformBrand.logo_url} alt="" /> : platformBrand.name.slice(0, 1)}</span>
            <span>{platformBrand.name}<small>{platformBrand.tagline || "RULE ENGINE"}</small></span>
          </a>
        </div>
        <button
          className={`side-nav ${screen === "overview" ? "side-nav-active" : ""}`}
          title={sidebarCollapsed ? "概览" : undefined}
          onClick={() => setScreen("overview")}
        >
          <span>
            <svg viewBox="0 0 20 20" aria-hidden="true">
              <path d="M3 3h5v6H3zM12 3h5v4h-5zM12 10h5v7h-5zM3 12h5v5H3z" />
            </svg>
          </span>{" "}
          概览
        </button>
        <button
          className={`side-nav ${screen === "projects" || screen === "design" ? "side-nav-active" : ""}`}
          title={sidebarCollapsed ? "项目" : undefined}
          onClick={() => { setProjectDetailOpen(false); setScreen("projects"); }}
        >
          <span><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 5.5h5l1.5 2H17v8H3zM3 5.5V4h5l1.5 1.5" /></svg></span>{" "}项目
        </button>
        <div className="sidebar-footer">
          <div className="sidebar-note">
            <span>⌘</span>
            <b>规则引擎</b>
            <small>确定性决策 · 可追溯执行</small>
            <em>v{version}</em>
          </div>
          <button
            className="sidebar-toggle"
            aria-label={sidebarCollapsed ? "展开侧边栏" : "收起侧边栏"}
            title={sidebarCollapsed ? "展开侧边栏" : "收起侧边栏"}
            onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
          >
            {sidebarCollapsed ? "›" : "‹"}
            <span>{sidebarCollapsed ? "展开侧边栏" : "收起侧边栏"}</span>
          </button>
        </div>
      </aside>
      <main className={`main ${screen === "projects" && projectDetailOpen ? "main-project-detail" : ""}`}>
        <header className="topbar">
          <div className="topbar-context">
            <CustomSelect
              variant="organization"
              ariaLabel="切换组织"
              value={activeOrganizationID}
              disabled={busy || !organizations.length}
              options={organizations.map((organization) => ({
                value: organization.id,
                label: organization.name,
              }))}
              onChange={(value) => void switchOrganization(value)}
            />
            <span>{screen === "overview" ? "概览" : screen === "projects" ? "项目" : selected.name}</span>
          </div>
          <div className="topbar-actions">
            {isSuperAdmin && (
              <button
                className="settings-entry platform-entry"
                aria-label="平台设置"
                title="平台设置"
                onClick={() => setSettingsArea("platform")}
              >
                <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3.5 5.5h13M3.5 10h13M3.5 14.5h13M7 3.5v4M13 8v4M9 12.5v4" /></svg>
                <span className="settings-entry-label">平台设置</span>
              </button>
            )}
            {(isSuperAdmin || organizationRole === "organization_admin") && <button
              className="settings-entry"
              aria-label="组织设置"
              title="组织设置"
              onClick={() => setSettingsArea("system")}
            >
              <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 17h14M5 17V6.5L10 4l5 2.5V17M8 9h.01M12 9h.01M8 12h.01M12 12h.01M9 17v-2.5h2V17" /></svg>
              <span className="settings-entry-label">组织设置</span>
            </button>}
            <details className="notification-menu">
              <summary aria-label="站内通知" title="站内通知"><svg viewBox="0 0 20 20" aria-hidden="true"><path d="M15.5 8a5.5 5.5 0 0 0-11 0c0 6-2 6-2 7.5h15C17.5 14 15.5 14 15.5 8ZM8 18h4"/></svg>{notifications.some((item) => !item.read) && <i>{notifications.filter((item) => !item.read).length}</i>}</summary>
              <section className="notification-popover"><header><b>站内通知</b><button className="quiet" disabled={!notifications.some((item) => !item.read)} onClick={(event) => { event.preventDefault(); void markNotificationsRead(); }}>全部已读</button></header>
                {notifications.length ? notifications.slice(0, 20).map((item) => <article key={item.id} className={item.read ? "is-read" : "is-unread"} onClick={() => !item.read && void markNotificationsRead(item.id)}><b>{item.title}</b><p>{item.message}</p><small>{new Date(item.created_at).toLocaleString("zh-CN")}</small></article>) : <p className="notification-empty">暂无通知</p>}
              </section>
            </details>
            <details className="user-menu">
              <summary className="topbar-user" title={`${email} · ${currentRoleLabel}`}>
                <span className="avatar">
                  {email.slice(0, 2).toUpperCase() || "SR"}
                </span>
                <span>
                  <b>{email}</b>
                  <small>{currentRoleLabel}</small>
                </span>
                <i>⌄</i>
              </summary>
              <div className="user-menu-popover">
                <div className="user-menu-identity">
                  <span className="avatar">
                    {email.slice(0, 2).toUpperCase() || "SR"}
                  </span>
                  <div>
                    <b>{currentRoleLabel}</b>
                    <small>{email}</small>
                  </div>
                </div>
                <button
                  className="user-menu-logout"
                  onClick={logout}
                  disabled={busy}
                >
                  ↪ <span>退出登录</span>
                </button>
              </div>
            </details>
          </div>
        </header>
        <div className={`content ${screen === "projects" && projectDetailOpen ? "content-project-detail" : ""}`}>
          {screen === "overview" ? (
            <>
              <section className="title-row">
                <div>
                  <small className="eyebrow">ORGANIZATION OVERVIEW</small>
                  <h1>概览</h1>
                  <p>查看规则调用趋势、命中情况与最近的求值记录。</p>
                </div>
              </section>
              <section className="metrics list-metrics">
                <div>
                  <small>近 7 天调用</small>
                  <b>{overviewDays.reduce((sum, day) => sum + day.total, 0)}</b>
                  <em>全部求值来源</em>
                </div>
                <div>
                  <small>规则命中</small>
                  <b>{overviewDays.reduce((sum, day) => sum + day.matched, 0)}</b>
                  <em>近 7 天</em>
                </div>
                <div>
                  <small>请求失败</small>
                  <b>{overviewDays.reduce((sum, day) => sum + day.failed, 0)}</b>
                  <em>HTTP 4xx / 5xx</em>
                </div>
              </section>
              <div className="overview-main-grid"><section className="overview-call-chart">
                <header><div><small className="eyebrow">RULE EVALUATIONS</small><h2>调用情况</h2><p>当前组织近 7 天的求值记录，包含模拟与真实调用</p></div><span>{overviewLogsBusy ? "统计中…" : `累计 ${overviewDays.reduce((sum, day) => sum + day.total, 0)} 次`}</span></header>
                <div className="overview-chart-legend"><span><i className="total" />调用次数</span><span><i className="matched" />规则命中</span><span><i className="failed" />请求失败</span></div>
                <div className="overview-chart-bars">{overviewDays.map((day) => {
                  const max = Math.max(1, ...overviewDays.map((item) => item.total));
                  return <div className="overview-chart-day" key={day.label}><div className="overview-chart-columns" title={`${day.total} 次调用 · 命中 ${day.matched} · 失败 ${day.failed}`}><i className="total" style={{ height: `${Math.max(day.total ? 5 : 0, day.total / max * 100)}%` }} /><i className="matched" style={{ height: `${Math.max(day.matched ? 5 : 0, day.matched / max * 100)}%` }} /><i className="failed" style={{ height: `${Math.max(day.failed ? 5 : 0, day.failed / max * 100)}%` }} /></div><small>{day.label}</small><b>{day.total}</b></div>;
                })}</div>
                {!overviewLogsBusy && overviewDays.every((day) => day.total === 0) && <div className="overview-chart-empty">近 7 天还没有求值记录。</div>}
              </section>
              <section className="overview-global-logs">
                <header><div><small className="eyebrow">INVOCATION LOGS</small><h2>调用日志</h2><p>当前组织各项目最近的求值记录</p></div><button className="overview-log-more" aria-label="查看全部调用日志" title="查看全部调用日志" onClick={() => { setGlobalLogPage(1); setGlobalLogSelected(null); setGlobalLogsOpen(true); }}><svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="4" cy="10" r="1.4"/><circle cx="10" cy="10" r="1.4"/><circle cx="16" cy="10" r="1.4"/></svg></button></header>
                {latestOverviewLogs.length ? <div className="overview-global-log-list">{latestOverviewLogs.slice(0, 8).map((log) => <article key={log.id}><div className="overview-global-log-top"><div className="overview-log-result"><i className={log.status_code >= 400 ? "failed" : log.matched ? "matched" : "ok"} /><span>{log.status_code >= 400 ? "失败" : log.matched ? `命中 ${log.hit_rules?.length ?? 0} 条` : "成功 · 未命中"}</span></div><time>{new Date(log.created_at).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" })}</time></div><b>{projects.find((project) => project.id === log.project_id)?.name ?? log.project_id}</b><small>{log.ruleset_id || "未记录规则集"} · {log.source === "designer_simulation" ? "设计器模拟" : log.source === "project_api" ? "真实 API" : "管理端调用"} · {log.duration_ms < 1 ? "<1" : log.duration_ms} ms</small></article>)}</div> : <div className="overview-chart-empty">暂无调用记录</div>}
              </section></div>
              {globalLogsOpen && <div className="overview-logs-overlay" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setGlobalLogsOpen(false); }}><section className="overview-logs-dialog" role="dialog" aria-modal="true" aria-labelledby="overview-logs-title"><header><div><small className="eyebrow">INVOCATION LOGS</small><h2 id="overview-logs-title">调用日志</h2><p>当前组织 · 共 {sortedOverviewLogs.length} 条记录</p></div><button className="quiet" onClick={() => setGlobalLogsOpen(false)} aria-label="关闭">×</button></header><div className="overview-logs-table-wrap"><table><thead><tr><th>时间</th><th>结果</th><th>项目 / 规则集</th><th>来源</th><th>状态</th><th>耗时</th></tr></thead><tbody>{pagedOverviewLogs.map((log) => <tr key={log.id} className={globalLogSelected?.id === log.id ? "selected" : ""} onClick={() => { setGlobalLogSelected(log); setGlobalLogTab("summary"); }}><td>{new Date(log.created_at).toLocaleString("zh-CN")}</td><td>{log.status_code >= 400 ? "失败" : log.matched ? `命中 ${log.hit_rules?.length ?? 0} 条` : "成功，未命中"}</td><td><b>{projects.find((project) => project.id === log.project_id)?.name ?? log.project_id}</b><small>{log.ruleset_id || "—"}</small></td><td>{log.source === "designer_simulation" ? "设计器模拟" : log.source === "project_api" ? "真实 API" : "管理端调用"}</td><td>{log.status_code}</td><td>{log.duration_ms < 1 ? "<1" : log.duration_ms} ms</td></tr>)}</tbody></table>{!pagedOverviewLogs.length && <div className="overview-chart-empty">暂无调用记录</div>}</div>{globalLogSelected && <section className="overview-log-detail"><header><div><b>{globalLogSelected.status_code >= 400 ? "调用失败" : globalLogSelected.matched ? `命中 ${globalLogSelected.hit_rules?.length ?? 0} 条规则` : "成功，未命中规则"}</b><small>{globalLogSelected.method || "POST"} {globalLogSelected.path || "/api/v1/engine/evaluate"} · {projects.find((project) => project.id === globalLogSelected.project_id)?.name ?? globalLogSelected.project_id}</small></div><nav><button className={globalLogTab === "summary" ? "active" : ""} onClick={() => setGlobalLogTab("summary")}>概览</button><button className={globalLogTab === "request" ? "active" : ""} onClick={() => setGlobalLogTab("request")}>请求</button><button className={globalLogTab === "response" ? "active" : ""} onClick={() => setGlobalLogTab("response")}>响应</button></nav></header>{globalLogTab === "summary" ? <div className="overview-log-summary"><span>来源：{globalLogSelected.source === "designer_simulation" ? "设计器模拟" : globalLogSelected.source === "project_api" ? "真实 API" : "管理端调用"}</span><span>凭据：{globalLogSelected.credential_type || "管理端会话"}</span><span>HTTP：{globalLogSelected.status_code}</span><span>耗时：{globalLogSelected.duration_ms < 1 ? "<1" : globalLogSelected.duration_ms} ms</span><span>时间：{new Date(globalLogSelected.created_at).toLocaleString("zh-CN")}</span></div> : <pre>{globalLogTab === "request" ? globalLogSelected.request_body || "此记录没有保存请求体。" : globalLogSelected.response_body || "此记录没有保存响应体。"}</pre>}</section>}<footer><span>每页 20 条</span><div><button className="quiet" disabled={globalLogPage <= 1} onClick={() => { setGlobalLogPage((page) => Math.max(1, page - 1)); setGlobalLogSelected(null); }}>上一页</button><span>{globalLogPage} / {globalLogPageCount}</span><button className="quiet" disabled={globalLogPage >= globalLogPageCount} onClick={() => { setGlobalLogPage((page) => Math.min(globalLogPageCount, page + 1)); setGlobalLogSelected(null); }}>下一页</button></div></footer></section></div>}
            </>
          ) : screen === "projects" ? (
            <ProjectsPanel projects={projects} projectID={openProjectID} activeOrganizationID={activeOrganizationID} alertChannels={alertChannels} projectDetailOpen={projectDetailOpen} setProjectDetailOpen={setProjectDetailOpen} onRefresh={refreshProjects} onSelect={switchProject} onCreateRuleset={create} onDeleteRuleset={removeRuleset} onOpenRuleset={(ruleset) => open(ruleset, "projects")} />
          ) : (
            <div className="designer-modal-overlay" onMouseDown={(event) => { if (event.target === event.currentTarget) setScreen(editorReturnScreen); }}>
              <div className="designer-modal" role="dialog" aria-modal="true" aria-label={`规则设计器：${selected.name || "未命名规则集"}`}>
              <div className="designer-modal-heading">
                <div className="crumb-row">
                  <span>规则设计器</span><span>/</span><b>{selected.name || "未命名规则集"}</b>
                  <span className={selected.enabled ? "tag-on" : "tag-off"}>{selected.enabled ? "启用中" : "已停用"}</span>
                </div>
                <button className="quiet" onClick={() => setScreen(editorReturnScreen)} aria-label="关闭规则设计器">×</button>
              </div>
              <section className="title-row design-title">
                <div>
                  <small className="eyebrow">RULE DESIGNER</small>
                  <h1>{selected.name || "未命名规则集"}</h1>
                  <p>
                    {selected.description ||
                      "编辑规则定义，并通过测试事实验证决策结果。"}
                  </p>
                </div>
                <div className="title-actions">
                  <button className="primary" disabled={busy} onClick={save}>
                    {busy ? "保存中…" : "保存规则集"}
                  </button>
                </div>
              </section>
              <section className="design-meta">
                <div>
                  <small>规则集 ID</small>
                  <b>{selected.id}</b>
                </div>
                <div>
                  <small>版本</small>
                  <b>{selected.version || "—"}</b>
                </div>
                <div>
                  <small>规则数量</small>
                  <b>{selected.rules?.length ?? 0}</b>
                </div>
                <div className="meta-note">
                  <span>ⓘ</span>{" "}
                  保存后规则定义立即更新；调试结果不会写入执行记录。
                </div>
              </section>
              <div className="designer-toolbar">
                <div>
                  <span className="canvas-live-dot" />
                  设计画布 <small>拖入组件构建规则流程</small>
                </div>
                <div className="mode-switch">
                  <button
                    className={designMode === "canvas" ? "selected" : ""}
                    onClick={() => setDesignMode("canvas")}
                  >
                    画布
                  </button>
                  <button
                    className={designMode === "json" ? "selected" : ""}
                    onClick={() => setDesignMode("json")}
                  >
                    JSON
                  </button>
                </div>
              </div>
              {designMode === "canvas" ? (
                <div className="canvas-layout">
                  <aside className="component-palette">
                    <div className="palette-title">
                      组件库 <small>拖拽到画布</small>
                    </div>
                    <div className="palette-group-label">规则节点</div>
                    <button
                      className="palette-component"
                      draggable
                      onDragStart={(e) => setPaletteDrag(e, "condition")}
                    >
                      <span className="component-icon condition-icon">◇</span>
                      <span>
                        <b>条件规则</b>
                        <small>字段与条件匹配</small>
                      </span>
                      <em>⠿</em>
                    </button>
                    <button
                      className="palette-component"
                      draggable
                      onDragStart={(e) => setPaletteDrag(e, "tag")}
                    >
                      <span className="component-icon output-icon">↗</span>
                      <span>
                        <b>输出动作</b>
                        <small>命中后添加标签</small>
                      </span>
                      <em>⠿</em>
                    </button>
                    <button
                      className="palette-component"
                      draggable
                      onDragStart={(e) => setPaletteDrag(e, "log")}
                    >
                      <span className="component-icon log-icon">≋</span>
                      <span>
                        <b>日志动作</b>
                        <small>记录规则命中信息</small>
                      </span>
                      <em>⠿</em>
                    </button>
                    <div className="palette-tip">
                      <b>使用提示</b>
                      <small>
                        拖动规则节点可调整优先级。点击节点后，在右侧编辑条件和动作。
                      </small>
                    </div>
                  </aside>
                  <section
                    className="flow-canvas"
                    onDragOver={(e) => {
                      e.preventDefault();
                      e.dataTransfer.dropEffect = "copy";
                    }}
                    onDrop={dropComponent}
                  >
                    <div className="canvas-caption">
                      <span>流程画布</span>
                      <small>由上至下按优先级求值</small>
                    </div>
                    <div className="flow-start">
                      <span>▶</span>
                      <div>
                        <b>开始</b>
                        <small>接收事实并开始匹配</small>
                      </div>
                    </div>
                    <div className="flow-connector">
                      <i />
                    </div>
                    {selected.rules?.map((rule, index) => (
                      <div className="flow-rule-wrap" key={rule.id}>
                        <article
                          className={`flow-rule-node ${activeRuleId === rule.id ? "node-selected" : ""}`}
                          draggable
                          onDragStart={() => setDragRuleId(rule.id)}
                          onDragEnd={() => setDragRuleId("")}
                          onDragOver={(e) => {
                            if (dragRuleId) e.preventDefault();
                          }}
                          onDrop={(e) => {
                            if (dragRuleId) {
                              e.preventDefault();
                              e.stopPropagation();
                              reorderRule(rule.id);
                            }
                          }}
                          onClick={() => setActiveRuleId(rule.id)}
                        >
                          <div className="node-topline">
                            <span className="node-type">
                              <i /> 条件规则
                            </span>
                            <span className="node-priority">
                              优先级 {rule.priority}
                            </span>
                            <button
                              className="node-menu"
                              aria-label="删除规则"
                              onClick={(e) => {
                                e.stopPropagation();
                                updateRuleset((value) => ({
                                  ...value,
                                  rules: value.rules.filter(
                                    (item) => item.id !== rule.id,
                                  ),
                                }));
                                if (activeRuleId === rule.id)
                                  setActiveRuleId("");
                              }}
                            >
                              ×
                            </button>
                          </div>
                          <b className="node-title">
                            {rule.name || "未命名规则"}
                          </b>
                          <div className="node-condition">
                            <span>IF</span>
                            <code>
                              {conditionOf(rule.conditions).fact || "配置条件"}{" "}
                              {conditionOf(rule.conditions).operator || ""}{" "}
                              {String(conditionOf(rule.conditions).value ?? "")}
                            </code>
                          </div>
                          <div className="node-actions">
                            {rule.actions?.length ? (
                              rule.actions.map((action, actionIndex) => (
                                <span key={`${action.type}-${actionIndex}`}>
                                  <i>↗</i>
                                  {action.type === "tag"
                                    ? `标签：${String(action.config?.label ?? "未命名")}`
                                    : action.type}
                                </span>
                              ))
                            ) : (
                              <span className="no-action">
                                尚未配置命中动作
                              </span>
                            )}
                          </div>
                        </article>
                        {index < selected.rules.length - 1 && (
                          <div className="flow-connector">
                            <i />
                          </div>
                        )}
                      </div>
                    ))}
                    {!selected.rules?.length && (
                      <div className="canvas-empty">
                        <span>＋</span>
                        <b>从左侧拖入第一个规则组件</b>
                        <small>也可以点击下方按钮添加条件规则</small>
                        <button className="secondary" onClick={addRule}>
                          ＋ 添加条件规则
                        </button>
                      </div>
                    )}
                    <div className="drop-hint">
                      ＋ 将组件拖到这里添加到规则流程
                    </div>
                  </section>
                  <aside className="node-inspector">
                    <div className="inspector-head">
                      <div>
                        <small>属性配置</small>
                        <b>{activeRule ? "条件规则" : "节点属性"}</b>
                      </div>
                      <span>⚙</span>
                    </div>
                    {activeRule && activeCondition ? (
                      <>
                        <label>
                          规则名称
                          <input
                            value={activeRule.name}
                            onChange={(e) =>
                              updateActiveRule((rule) => ({
                                ...rule,
                                name: e.target.value,
                              }))
                            }
                          />
                        </label>
                        <label>
                          优先级
                          <input
                            type="number"
                            value={activeRule.priority}
                            onChange={(e) =>
                              updateActiveRule((rule) => ({
                                ...rule,
                                priority: Number(e.target.value),
                              }))
                            }
                          />
                        </label>
                        <div className="inspector-divider" />
                        <div className="inspector-section-title">
                          条件配置 <small>满足条件时触发</small>
                        </div>
                        <label>
                          事实字段
                          <input
                            placeholder="例如 order.amount"
                            value={activeCondition.fact ?? ""}
                            onChange={(e) =>
                              updateActiveRule((rule) => ({
                                ...rule,
                                conditions: patchFirstCondition(
                                  rule.conditions,
                                  { fact: e.target.value },
                                ),
                              }))
                            }
                          />
                        </label>
                        <label>
                          匹配方式
                          <CustomSelect
                            ariaLabel="匹配方式"
                            value={activeCondition.operator ?? "equal"}
                            options={[
                              ["equal", "等于"],
                              ["notEqual", "不等于"],
                              ["greaterThan", "大于"],
                              ["greaterThanInclusive", "大于等于"],
                              ["lessThan", "小于"],
                              ["lessThanInclusive", "小于等于"],
                            ].map(([value, label]) => ({ value, label }))}
                            onChange={(value) =>
                              updateActiveRule((rule) => ({
                                ...rule,
                                conditions: patchFirstCondition(
                                  rule.conditions,
                                  { operator: value },
                                ),
                              }))
                            }
                          />
                        </label>
                        <label>
                          比较值
                          {typeof activeCondition.value === "boolean" ? (
                            <CustomSelect
                              ariaLabel="比较值"
                              value={String(activeCondition.value)}
                              options={[
                                { value: "true", label: "true" },
                                { value: "false", label: "false" },
                              ]}
                              onChange={(value) =>
                                updateActiveRule((rule) => ({
                                  ...rule,
                                  conditions: patchFirstCondition(
                                    rule.conditions,
                                    { value: value === "true" },
                                  ),
                                }))
                              }
                            />
                          ) : (
                            <input
                              type={
                                typeof activeCondition.value === "number"
                                  ? "number"
                                  : "text"
                              }
                              value={String(activeCondition.value ?? "")}
                              onChange={(e) =>
                                updateActiveRule((rule) => ({
                                  ...rule,
                                  conditions: patchFirstCondition(
                                    rule.conditions,
                                    {
                                      value:
                                        typeof activeCondition.value ===
                                        "number"
                                          ? Number(e.target.value)
                                          : e.target.value,
                                    },
                                  ),
                                }))
                              }
                            />
                          )}
                        </label>
                        <div className="inspector-divider" />
                        <div className="inspector-section-title">
                          命中动作{" "}
                          <small>
                            {activeRule.actions?.length ?? 0} 个动作
                          </small>
                        </div>
                        {(activeRule.actions ?? []).map((action, index) => (
                          <div
                            className="action-edit"
                            key={`${action.type}-${index}`}
                          >
                            <span>
                              {action.type === "tag" ? "标签输出" : action.type}
                            </span>
                            <input
                              value={String(
                                action.config?.label ??
                                  action.config?.message ??
                                  "",
                              )}
                              onChange={(e) =>
                                updateActiveRule((rule) => ({
                                  ...rule,
                                  actions: rule.actions?.map((item, i) =>
                                    i === index
                                      ? {
                                          ...item,
                                          config: {
                                            ...item.config,
                                            [item.type === "tag"
                                              ? "label"
                                              : "message"]: e.target.value,
                                          },
                                        }
                                      : item,
                                  ),
                                }))
                              }
                            />
                            <button
                              aria-label="移除动作"
                              onClick={() =>
                                updateActiveRule((rule) => ({
                                  ...rule,
                                  actions: rule.actions?.filter(
                                    (_, i) => i !== index,
                                  ),
                                }))
                              }
                            >
                              ×
                            </button>
                          </div>
                        ))}
                        <button
                          className="add-action"
                          onClick={() => addAction("tag")}
                        >
                          ＋ 添加标签动作
                        </button>
                        <div className="inspector-divider" />
                        <div className="inspector-section-title">
                          命中通知 <small>{activeRule.alert ? activeRule.alert.enabled ? "单条规则覆盖" : "单条规则关闭" : selected.alert?.enabled ? "继承规则集默认" : "未配置"}</small>
                        </div>
                        {activeRuleAlert?.enabled && <div className="rule-alert-summary"><b>{activeRule.alert ? "此规则的通知渠道" : "继承的默认渠道"}</b><div>{activeRuleAlert.channel_ids.map((id) => alertChannels.find((channel) => channel.id === id)?.name ?? id).map((name) => <span key={name}>{name}</span>)}</div></div>}
                        <button className="rule-alert-open" onClick={() => openRuleAlertConfig(activeRule)}>
                          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M5 14.5h10l-1.2-1.6V8.5a3.8 3.8 0 0 0-7.6 0v4.4L5 14.5Z"/><path d="M8 16.2h4"/></svg>
                          {activeRule.alert ? "编辑单条规则覆盖" : "设置单条规则覆盖"}
                        </button>
                      </>
                    ) : (
                      <div className="inspector-empty">
                        <span>◇</span>
                        <b>选择一个规则节点</b>
                        <small>在画布点击节点，编辑条件和动作。</small>
                      </div>
                    )}
                  </aside>
                </div>
              ) : (
                <section className="panel json-panel">
                  <div className="panel-head">
                    <span className="step">{`{ }`}</span>
                    <div>
                      <b>规则定义</b>
                      <small>JSON DSL · all / any / not 条件组</small>
                    </div>
                    <span className="json-tag">JSON</span>
                  </div>
                  <div className="editor-hint">
                    <span>✳</span> 高级编辑模式。规则按优先级从高到低求值。
                  </div>
                  <textarea
                    className="rule-editor"
                    aria-label="规则集 JSON 编辑器"
                    spellCheck={false}
                    value={editor}
                    onChange={(e) => {
                      setEditor(e.target.value);
                      try {
                        setSelected(JSON.parse(e.target.value) as Ruleset);
                      } catch {
                        /* keep partial input */
                      }
                    }}
                  />
                  <div className="panel-bottom">
                    <span>{notice || "修改后保存以更新规则集"}</span>
                  </div>
                </section>
              )}
              <div className="design-bottom">
                <section className="panel test-panel">
                  <div className="panel-head">
                    <span className="step blue">02</span>
                    <div>
                      <b>模拟决策</b>
                      <small>输入事实并查看命中规则</small>
                    </div>
                    <span className="json-tag">TEST</span>
                  </div>
                  <label className="facts-label" htmlFor="facts">
                    测试事实 <small>FACTS · JSON</small>
                  </label>
                  <textarea
                    id="facts"
                    className="facts-editor"
                    spellCheck={false}
                    value={facts}
                    onChange={(e) => setFacts(e.target.value)}
                  />
                  <button
                    className="run-button"
                    disabled={busy}
                    onClick={evaluate}
                  >
                    运行模拟 <span>→</span>
                  </button>
                  <div className="result-head">
                    <b>运行结果</b>
                    {decision && (
                      <span className={decision.matched ? "tag-on" : "tag-off"}>
                        {decision.matched
                          ? `命中 ${decision.hit_rules.length} 条`
                          : "无规则命中"}
                      </span>
                    )}
                  </div>
                  {decision ? (
                    <pre className="result">{pretty(decision)}</pre>
                  ) : (
                    <div className="empty-result">
                      ◎<small>模拟执行结果将在这里显示</small>
                    </div>
                  )}
                </section>
              </div>
              </div>
            </div>
          )}
          {error && (
            <div className="error-message" role="alert">
              {error}
            </div>
          )}
          <footer>{platformBrand.name}　·　{platformBrand.tagline}　·　v{version}</footer>
        </div>
      </main>
      {ruleAlertModalOpen && activeRule && <div className="project-modal-overlay" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setRuleAlertModalOpen(false); }}><section className="project-modal project-config-modal" role="dialog" aria-modal="true" aria-labelledby="rule-alert-title">
        <header><div><small>RULE NOTIFICATION</small><h2 id="rule-alert-title">命中通知</h2><p>「{activeRule.name}」默认继承规则集设置，也可以单独覆盖。</p></div><button className="quiet" onClick={() => setRuleAlertModalOpen(false)} aria-label="关闭">×</button></header>
        <div className="rule-alert-mode">
          <label><input type="radio" name="rule-alert-mode" checked={!ruleAlertOverride} onChange={() => setRuleAlertOverride(false)} /><span><b>继承规则集默认</b><small>{selected.alert?.enabled ? `已启用 · ${selected.alert.channel_ids.map((id) => alertChannels.find((channel) => channel.id === id)?.name ?? id).join("、")}` : "当前规则集未设置默认通知"}</small></span></label>
          <label><input type="radio" name="rule-alert-mode" checked={ruleAlertOverride} onChange={() => setRuleAlertOverride(true)} /><span><b>自定义此规则</b><small>使用单独的启用状态和通知渠道</small></span></label>
        </div>
        {ruleAlertOverride && <>
          <label className="project-toggle"><input type="checkbox" checked={ruleAlertEnabled} onChange={(event) => setRuleAlertEnabled(event.target.checked)} /><span><b>启用此规则的告警</b><small>关闭后，此规则不会触发通知</small></span></label>
          {ruleAlertEnabled && <div className="project-check-group"><b>通知渠道</b>{alertChannels.length ? alertChannels.map((channel) => <label key={channel.id}><input type="checkbox" checked={ruleAlertChannelIDs.includes(channel.id)} onChange={() => setRuleAlertChannelIDs((items) => items.includes(channel.id) ? items.filter((id) => id !== channel.id) : [...items, channel.id])} /><span>{channel.name}<small>{channel.type}</small></span></label>) : <small>暂无可用渠道，请先在「组织设置 → 告警渠道」添加并启用。</small>}</div>}
        </>}
        <footer><button className="quiet" onClick={() => setRuleAlertModalOpen(false)}>取消</button><button className="primary" disabled={ruleAlertOverride && ruleAlertEnabled && ruleAlertChannelIDs.length === 0} onClick={saveRuleAlertConfig}>应用设置</button></footer>
      </section></div>}
      {settingsArea && (
        <SettingsDialog
          area={settingsArea}
          isSuperAdmin={isSuperAdmin}
          organizationRole={organizationRole}
          version={version}
          platformBrand={platformBrand}
          onPlatformBrandChange={setPlatformBrand}
          organizations={organizations}
          activeOrganizationID={activeOrganizationID}
          onOrganizationRefresh={refreshOrganizations}
          onAlertChannelRefresh={refreshAlertChannels}
          onClose={() => setSettingsArea(null)}
        />
      )}
      <Toaster position="top-right" richColors duration={4500} closeButton />
    </div>
  );
}
