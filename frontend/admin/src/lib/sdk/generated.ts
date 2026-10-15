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
}

export interface Ruleset {
  id: string;
  name: string;
  version: string;
  description?: string;
  enabled: boolean;
  rules: Rule[];
}

export interface RulesetList {
  items: Ruleset[];
}

export interface EvaluateRequest {
  rulesetId: string;
  facts: Record<string, JSONValue>;
}

export interface MatchedRule {
  ruleId: string;
  name: string;
  priority: number;
  actions?: Action[];
}

export interface Decision {
  rulesetId: string;
  version: string;
  matched: boolean;
  hitRules: MatchedRule[];
}

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceList {
  items: Workspace[];
  active_workspace_id: string;
}

export interface WorkspaceInput {
  name: string;
  slug?: string;
  description?: string;
}

export interface ActiveWorkspaceRequest {
  workspace_id: string;
}

export interface ActiveWorkspaceResponse {
  active_workspace_id: string;
}

export class SnakeRuleApiError extends Error {
  constructor(public readonly status: number, message: string) { super(message); }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const multipart = typeof FormData !== 'undefined' && init?.body instanceof FormData;
  const response = await fetch(path, {
    ...init,
    headers: { ...(init?.body && !multipart ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },
  });
  if (!response.ok) {
    if (response.status === 401 && !path.startsWith('/api/v1/auth/')) window.dispatchEvent(new Event('snakerule:unauthorized'));
    const body = await response.json().catch(() => ({}));
    throw new SnakeRuleApiError(response.status, body?.error?.message ?? response.statusText);
  }
  return (response.status === 204 ? undefined : response.json()) as Promise<T>;
}

export const snakeRuleApi = {
  getSession: () => request<SessionResponse>('/api/v1/auth/session'),
  login: (input: LoginRequest) => request<SessionResponse>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(input) }),
  logout: () => request<{ authenticated: boolean }>('/api/v1/auth/logout', { method: 'POST' }),
  listRulesets: () => request<RulesetList>('/api/v1/rulesets'),
  getRuleset: (id: string) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(id)}`),
  saveRuleset: (ruleset: Ruleset) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(ruleset.id)}`, { method: 'PUT', body: JSON.stringify(ruleset) }),
  deleteRuleset: (id: string) => request<void>(`/api/v1/rulesets/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  evaluate: (input: EvaluateRequest) => request<Decision>('/api/v1/engine/evaluate', { method: 'POST', body: JSON.stringify(input) }),
  getLicense: () => request<LicenseStatus>('/api/v1/platform/license'),
  uploadLicense: (file: File) => { const form = new FormData(); form.append('file', file); return request<LicenseStatus>('/api/v1/platform/license', { method: 'POST', body: form }); },
  clearLicense: () => request<LicenseStatus>('/api/v1/platform/license', { method: 'DELETE' }),
  listWorkspaces: () => request<WorkspaceList>('/api/v1/workspaces'),
  createWorkspace: (input: WorkspaceInput) => request<Workspace>('/api/v1/workspaces', { method: 'POST', body: JSON.stringify(input) }),
  updateWorkspace: (id: string, input: WorkspaceInput) => request<Workspace>(`/api/v1/workspaces/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteWorkspace: (id: string) => request<void>(`/api/v1/workspaces/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  setActiveWorkspace: (workspace_id: string) => request<ActiveWorkspaceResponse>('/api/v1/workspaces/active', { method: 'PUT', body: JSON.stringify({ workspace_id }) }),
};
