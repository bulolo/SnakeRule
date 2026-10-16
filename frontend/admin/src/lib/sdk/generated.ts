/* eslint-disable */
// Generated from the OpenAPI contract. Do not edit by hand.

export type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue };

export interface LoginRequest {
  email: string;
  password: string;
}

export interface SessionUser {
  email: string;
  name: string;
  is_superadmin: boolean;
  role?: "superadmin" | "organization_admin" | "organization_member" | "project_admin" | "project_member";
}

export interface SessionResponse {
  authenticated: boolean;
  user?: SessionUser;
}

export interface LicenseStatus {
  configured: boolean;
  edition: string;
  is_licensed: boolean;
  customer?: string;
  license_type?: string;
  expires_at?: string;
  features?: Record<string, JSONValue>;
  feature_keys?: string[];
  installation_id: string;
  error?: string;
}

export interface Action {
  type: string;
  config?: Record<string, JSONValue>;
}

export interface Condition {
  all?: Condition[];
  any?: Condition[];
  not?: Condition;
  fact?: string;
  operator?: string;
  value?: JSONValue;
}

export interface Rule {
  id: string;
  name: string;
  description?: string;
  priority: number;
  enabled: boolean;
  conditions: Condition;
  actions?: Action[];
  alert?: RuleAlert;
}

export interface Ruleset {
  id: string;
  name: string;
  version: string;
  description?: string;
  enabled: boolean;
  alert?: RuleAlert;
  rules: Rule[];
}

export interface RulesetList {
  list: Ruleset[];
  pagination: Pagination;
}

export interface EvaluateRequest {
  ruleset_id: string;
  facts: Record<string, JSONValue>;
  ruleset?: Ruleset;
}

export interface MatchedRule {
  rule_id: string;
  name: string;
  priority: number;
  actions?: Action[];
}

export interface Decision {
  ruleset_id: string;
  version: string;
  matched: boolean;
  hit_rules: MatchedRule[];
}

export interface AlertChannel {
  id: string;
  name: string;
  type: "webhook" | "wecom" | "dingtalk" | "feishu" | "internal";
  url: string;
  enabled: boolean;
  has_secret: boolean;
  organization_id?: string;
}

export interface AlertChannelInput {
  name: string;
  type: "webhook" | "wecom" | "dingtalk" | "feishu";
  url: string;
  secret?: string;
  enabled?: boolean;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  description?: string;
  created_at: string;
  updated_at: string;
  admin_user_id?: string;
}

export interface OrganizationList {
  list: Organization[];
  pagination: Pagination;
  active_organization_id: string;
}

export interface OrganizationInput {
  name: string;
  slug?: string;
  description?: string;
  admin_user_id?: string;
}

export interface ActiveOrganizationRequest {
  organization_id: string;
}

export interface ActiveOrganizationResponse {
  active_organization_id: string;
}

export interface Project {
  id: string;
  name: string;
  slug: string;
  description?: string;
  created_at?: string;
  updated_at?: string;
  icon?: string;
}

export interface ProjectInput {
  name: string;
  slug?: string;
  description?: string;
  template?: "" | "ecommerce";
  icon?: string;
}

export interface ProjectKey {
  id: string;
  name: string;
  prefix: string;
  created_at: string;
  type?: "apikey" | "jwt";
  credential_kind?: "api_key" | "access_token" | "signing_key";
  expires_at?: string;
}

export interface CreateProjectKeyInput {
  name: string;
  type?: "apikey" | "jwt";
  signing_key?: boolean;
  expires_at?: string;
}

export interface ActiveProjectRequest {
  project_id: string;
}

export interface ActiveProjectResponse {
  active_project_id: string;
}

export interface RuleAlert {
  enabled: boolean;
  channel_ids: string[];
}

export interface Notification {
  id: string;
  organization_id: string;
  title: string;
  message: string;
  project: string;
  ruleset: string;
  created_at: string;
  read: boolean;
}

export interface InvocationLog {
  id: string;
  organization_id: string;
  project_id: string;
  ruleset_id?: string;
  credential_type: "session" | "apikey" | "jwt";
  status_code: number;
  matched: boolean;
  hit_rules?: InvocationHitRule[];
  duration_ms: number;
  created_at: string;
  source: "designer_simulation" | "project_api" | "management";
  method?: string;
  path?: string;
  request_body?: string;
  response_body?: string;
}

export interface Pagination {
  is_pager: 0 | 1;
  page: number;
  size: number;
  total: number;
}

export interface InvocationListResponse {
  list: InvocationLog[];
  pagination: Pagination;
}

export interface InvocationListQuery {
  page?: number;
  size?: number;
  is_pager?: 0 | 1;
  ruleset_id?: string;
}

export interface InvocationHitRule {
  rule_id?: string;
  name?: string;
  priority?: number;
  actions?: string[];
}

export interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: "organization_admin" | "organization_member" | "superadmin";
  organization_ids: string[];
  created_at: string;
  organization_roles?: Record<string, JSONValue>;
  project_roles?: Record<string, JSONValue>;
}

export interface SaveUserInput {
  id?: string;
  email: string;
  name: string;
  role: "organization_admin" | "organization_member";
  password?: string;
  organization_ids?: string[];
}

export interface UserList {
  list: AdminUser[];
  pagination: Pagination;
}

export interface ProjectMember {
  user: ProjectMemberUser;
  role: "project_admin" | "project_member";
}

export interface ProjectMemberList {
  list: ProjectMember[];
  pagination: Pagination;
  can_manage: boolean;
}

export interface ProjectMemberInput {
  user_id: string;
  role: "project_admin" | "project_member";
}

export interface ProjectMemberRoleInput {
  role: "project_admin" | "project_member";
}

export interface ProjectMemberUser {
  id: string;
  name: string;
  email: string;
}

export interface PlatformSettings {
  name: string;
  tagline: string;
  logo_url: string;
}

export interface APIResponse {
  code: number;
  msg: string;
  error_code?: string;
  data?: JSONValue;
}

export interface APIErrorResponse {
  code: number;
  msg: string;
  error_code: string;
}

export interface ProjectList {
  list: Project[];
  pagination: Pagination;
  active_project_id: string;
}

export interface AlertChannelList {
  list: AlertChannel[];
  pagination: Pagination;
}

export interface NotificationList {
  list: Notification[];
  pagination: Pagination;
}

export interface ProjectKeyList {
  list: ProjectKey[];
  pagination: Pagination;
}

export class SnakeRuleApiError extends Error {
  constructor(public readonly status: number, message: string, public readonly errorCode?: string) { super(message); }
}

export async function readAPIResponse<T>(response: Response): Promise<T> {
  const raw = await response.text();
  let body: any;
  try { body = raw ? JSON.parse(raw) : {}; } catch { throw new SnakeRuleApiError(response.status, `接口返回非 JSON 响应（HTTP ${response.status}）：${raw.slice(0, 160)}`); }
  if (!response.ok || body?.code !== 0) throw new SnakeRuleApiError(response.status, body?.msg ?? response.statusText, body?.error_code);
  return body.data as T;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const multipart = typeof FormData !== 'undefined' && init?.body instanceof FormData;
  const response = await fetch(path, {
    ...init,
    headers: { ...(init?.body && !multipart ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },
  });
  if (response.status === 401 && !path.startsWith('/api/v1/auth/')) window.dispatchEvent(new Event('snakerule:unauthorized'));
  if (response.status === 204) return undefined as T;
  return readAPIResponse<T>(response);
}

export const snakeRuleApi = {
  getPlatformSettings: () => request<PlatformSettings>('/api/v1/platform/settings'),
  updatePlatformSettings: (input: PlatformSettings) => request<PlatformSettings>('/api/v1/platform/settings', { method: 'PUT', body: JSON.stringify(input) }),
  getSession: () => request<SessionResponse>('/api/v1/auth/session'),
  getMeta: () => request<{ version: string }>('/api/v1/meta'),
  login: (input: LoginRequest) => request<SessionResponse>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(input) }),
  logout: () => request<{ authenticated: boolean }>('/api/v1/auth/logout', { method: 'POST' }),
  listRulesets: () => request<RulesetList>('/api/v1/rulesets?is_pager=0'),
  getRuleset: (id: string) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(id)}`),
  createRuleset: (ruleset: Ruleset) => request<Ruleset>('/api/v1/rulesets', { method: 'POST', body: JSON.stringify(ruleset) }),
  updateRuleset: (ruleset: Ruleset) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(ruleset.id)}`, { method: 'PUT', body: JSON.stringify(ruleset) }),
  deleteRuleset: (id: string) => request<void>(`/api/v1/rulesets/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  evaluate: (input: EvaluateRequest, source?: 'designer') => request<Decision>('/api/v1/engine/evaluate', { method: 'POST', body: JSON.stringify(input), headers: source ? { 'X-SnakeRule-Source': source } : undefined }),
  listProjectInvocations: (id: string, query: InvocationListQuery = {}) => { const params = new URLSearchParams(); for (const [key, value] of Object.entries(query)) if (value !== undefined) params.set(key, String(value)); return request<InvocationListResponse>(`/api/v1/projects/${encodeURIComponent(id)}/invocations?${params.toString()}`); },
  getLicense: () => request<LicenseStatus>('/api/v1/platform/license'),
  uploadLicense: (file: File) => { const form = new FormData(); form.append('file', file); return request<LicenseStatus>('/api/v1/platform/license', { method: 'POST', body: form }); },
  clearLicense: () => request<LicenseStatus>('/api/v1/platform/license', { method: 'DELETE' }),
  listOrganizations: () => request<OrganizationList>('/api/v1/organizations?is_pager=0'),
  listProjects: () => request<ProjectList>('/api/v1/projects?is_pager=0'),
  createProject: (input: ProjectInput) => request<Project>('/api/v1/projects', { method: 'POST', body: JSON.stringify(input) }),
  updateProject: (id: string, input: ProjectInput) => request<Project>(`/api/v1/projects/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteProject: (id: string) => request<void>(`/api/v1/projects/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  setActiveProject: (project_id: string) => request<ActiveProjectResponse>('/api/v1/projects/active', { method: 'PUT', body: JSON.stringify({ project_id }) }),
  createOrganization: (input: OrganizationInput) => request<Organization>('/api/v1/organizations', { method: 'POST', body: JSON.stringify(input) }),
  updateOrganization: (id: string, input: OrganizationInput) => request<Organization>(`/api/v1/organizations/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteOrganization: (id: string) => request<void>(`/api/v1/organizations/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  setActiveOrganization: (organization_id: string) => request<ActiveOrganizationResponse>('/api/v1/organizations/active', { method: 'PUT', body: JSON.stringify({ organization_id }) }),
  listOrganizationUsers: (organizationID: string) => request<UserList>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users?is_pager=0`),
  createOrganizationUser: (organizationID: string, input: SaveUserInput) => request<{ item: AdminUser }>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users`, { method: 'POST', body: JSON.stringify(input) }),
  updateOrganizationUser: (organizationID: string, id: string, input: SaveUserInput) => request<{ item: AdminUser }>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteOrganizationUser: (organizationID: string, id: string) => request<void>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  listOrganizationAlertChannels: (organizationID: string) => request<AlertChannelList>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels?is_pager=0`),
  createOrganizationAlertChannel: (organizationID: string, input: AlertChannelInput) => request<AlertChannel>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels`, { method: 'POST', body: JSON.stringify(input) }),
  updateOrganizationAlertChannel: (organizationID: string, id: string, input: AlertChannelInput) => request<AlertChannel>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteOrganizationAlertChannel: (organizationID: string, id: string) => request<void>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  testOrganizationAlertChannel: (organizationID: string, id: string) => request<{ status: string }>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels/${encodeURIComponent(id)}/test`, { method: 'POST' }),
  listProjectMembers: (id: string) => request<ProjectMemberList>(`/api/v1/projects/${encodeURIComponent(id)}/members?is_pager=0`),
  listProjectMemberCandidates: (id: string) => request<UserList>(`/api/v1/projects/${encodeURIComponent(id)}/member-candidates?is_pager=0`),
  listProjectCredentials: (id: string) => request<ProjectKeyList>(`/api/v1/projects/${encodeURIComponent(id)}/credentials?is_pager=0`),
  createProjectCredential: (id: string, input: CreateProjectKeyInput) => request<{ item: ProjectKey; secret?: string; token?: string; signing_secret?: string }>(`/api/v1/projects/${encodeURIComponent(id)}/credentials`, { method: 'POST', body: JSON.stringify(input) }),
  revokeProjectCredential: (id: string, keyId: string) => request<void>(`/api/v1/projects/${encodeURIComponent(id)}/credentials/${encodeURIComponent(keyId)}`, { method: 'DELETE' }),
  listNotifications: () => request<NotificationList>('/api/v1/notifications?is_pager=0'),
  readNotification: (id: string) => request<{ read: boolean }>(`/api/v1/notifications/${encodeURIComponent(id)}/read`, { method: 'POST' }),
  readAllNotifications: () => request<{ read: boolean }>('/api/v1/notifications/read-all', { method: 'POST' }),
  addProjectMember: (id: string, input: ProjectMemberInput) => request<{ user: ProjectMemberUser; role: string }>(`/api/v1/projects/${encodeURIComponent(id)}/members`, { method: 'POST', body: JSON.stringify(input) }),
  updateProjectMember: (id: string, userId: string, role: 'project_admin' | 'project_member') => request<void>(`/api/v1/projects/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`, { method: 'PUT', body: JSON.stringify({ role }) }),
  removeProjectMember: (id: string, userId: string) => request<void>(`/api/v1/projects/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`, { method: 'DELETE' }),
};
