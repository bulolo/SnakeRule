#!/usr/bin/env python3
"""Generate TypeScript API schema types from the checked-in OpenAPI document."""

import json
import pathlib
import sys


def type_for(schema: dict) -> str:
    if "$ref" in schema:
        return schema["$ref"].rsplit("/", 1)[-1]
    if "enum" in schema:
        return " | ".join(json.dumps(value) for value in schema["enum"])
    kind = schema.get("type")
    if kind == "array":
        return f"{type_for(schema.get('items', {}))}[]"
    if kind == "string":
        return "string"
    if kind == "integer" or kind == "number":
        return "number"
    if kind == "boolean":
        return "boolean"
    if kind == "object":
        return "Record<string, JSONValue>"
    return "JSONValue"


def main() -> int:
    if len(sys.argv) != 3:
        print("Usage: python3 scripts/generate_sdk.py <openapi.json> <output.ts>", file=sys.stderr)
        return 2
    spec_path, output_path = map(pathlib.Path, sys.argv[1:])
    spec = json.loads(spec_path.read_text())
    schemas = spec["components"]["schemas"]
    lines = [
        "/* eslint-disable */",
        "// Generated from the OpenAPI contract. Do not edit by hand.",
        "",
        "export type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue };",
        "",
    ]
    for name, schema in schemas.items():
        required = set(schema.get("required", []))
        lines.append(f"export interface {name} {{")
        for field, prop in schema.get("properties", {}).items():
            optional = "" if field in required else "?"
            lines.append(f"  {field}{optional}: {type_for(prop)};")
        lines.extend(["}", ""])
    lines.extend([
        "export class SnakeRuleApiError extends Error {",
        "  constructor(public readonly status: number, message: string, public readonly errorCode?: string) { super(message); }",
        "}",
        "",
        "export async function readAPIResponse<T>(response: Response): Promise<T> {",
        "  const raw = await response.text();",
        "  let body: any;",
        "  try { body = raw ? JSON.parse(raw) : {}; } catch { throw new SnakeRuleApiError(response.status, `接口返回非 JSON 响应（HTTP ${response.status}）：${raw.slice(0, 160)}`); }",
        "  if (!response.ok || body?.code !== 0) throw new SnakeRuleApiError(response.status, body?.msg ?? response.statusText, body?.error_code);",
        "  return body.data as T;",
        "}",
        "",
        "async function request<T>(path: string, init?: RequestInit): Promise<T> {",
        "  const multipart = typeof FormData !== 'undefined' && init?.body instanceof FormData;",
        "  const response = await fetch(path, {",
        "    ...init,",
        "    headers: { ...(init?.body && !multipart ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },",
        "  });",
        "  if (response.status === 401 && !path.startsWith('/api/v1/auth/')) window.dispatchEvent(new Event('snakerule:unauthorized'));",
        "  if (response.status === 204) return undefined as T;",
        "  return readAPIResponse<T>(response);",
        "}",
        "",
        "export const snakeRuleApi = {",
        "  getPlatformSettings: () => request<PlatformSettings>('/api/v1/platform/settings'),",
        "  updatePlatformSettings: (input: PlatformSettings) => request<PlatformSettings>('/api/v1/platform/settings', { method: 'PUT', body: JSON.stringify(input) }),",
        "  getSession: () => request<SessionResponse>('/api/v1/auth/session'),",
        "  getMeta: () => request<{ version: string }>('/api/v1/meta'),",
        "  login: (input: LoginRequest) => request<SessionResponse>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(input) }),",
        "  logout: () => request<{ authenticated: boolean }>('/api/v1/auth/logout', { method: 'POST' }),",
        "  listRulesets: () => request<RulesetList>('/api/v1/rulesets?is_pager=0'),",
        "  getRuleset: (id: string) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(id)}`),",
        "  createRuleset: (ruleset: Ruleset) => request<Ruleset>('/api/v1/rulesets', { method: 'POST', body: JSON.stringify(ruleset) }),",
        "  updateRuleset: (ruleset: Ruleset) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(ruleset.id)}`, { method: 'PUT', body: JSON.stringify(ruleset) }),",
        "  deleteRuleset: (id: string) => request<void>(`/api/v1/rulesets/${encodeURIComponent(id)}`, { method: 'DELETE' }),",
        "  evaluate: (input: EvaluateRequest, source?: 'designer') => request<Decision>('/api/v1/engine/evaluate', { method: 'POST', body: JSON.stringify(input), headers: source ? { 'X-SnakeRule-Source': source } : undefined }),",
        "  listProjectInvocations: (id: string, query: InvocationListQuery = {}) => { const params = new URLSearchParams(); for (const [key, value] of Object.entries(query)) if (value !== undefined) params.set(key, String(value)); return request<InvocationListResponse>(`/api/v1/projects/${encodeURIComponent(id)}/invocations?${params.toString()}`); },",
        "  getLicense: () => request<LicenseStatus>('/api/v1/platform/license'),",
        "  uploadLicense: (file: File) => { const form = new FormData(); form.append('file', file); return request<LicenseStatus>('/api/v1/platform/license', { method: 'POST', body: form }); },",
        "  clearLicense: () => request<LicenseStatus>('/api/v1/platform/license', { method: 'DELETE' }),",
        "  listOrganizations: () => request<OrganizationList>('/api/v1/organizations?is_pager=0'),",
        "  listProjects: () => request<ProjectList>('/api/v1/projects?is_pager=0'),",
        "  createProject: (input: ProjectInput) => request<Project>('/api/v1/projects', { method: 'POST', body: JSON.stringify(input) }),",
        "  updateProject: (id: string, input: ProjectInput) => request<Project>(`/api/v1/projects/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),",
        "  deleteProject: (id: string) => request<void>(`/api/v1/projects/${encodeURIComponent(id)}`, { method: 'DELETE' }),",
        "  setActiveProject: (project_id: string) => request<ActiveProjectResponse>('/api/v1/projects/active', { method: 'PUT', body: JSON.stringify({ project_id }) }),",
        "  createOrganization: (input: OrganizationInput) => request<Organization>('/api/v1/organizations', { method: 'POST', body: JSON.stringify(input) }),",
        "  updateOrganization: (id: string, input: OrganizationInput) => request<Organization>(`/api/v1/organizations/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),",
        "  deleteOrganization: (id: string) => request<void>(`/api/v1/organizations/${encodeURIComponent(id)}`, { method: 'DELETE' }),",
        "  setActiveOrganization: (organization_id: string) => request<ActiveOrganizationResponse>('/api/v1/organizations/active', { method: 'PUT', body: JSON.stringify({ organization_id }) }),",
        "  listOrganizationUsers: (organizationID: string) => request<UserList>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users?is_pager=0`),",
        "  createOrganizationUser: (organizationID: string, input: SaveUserInput) => request<{ item: AdminUser }>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users`, { method: 'POST', body: JSON.stringify(input) }),",
        "  updateOrganizationUser: (organizationID: string, id: string, input: SaveUserInput) => request<{ item: AdminUser }>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),",
        "  deleteOrganizationUser: (organizationID: string, id: string) => request<void>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/users/${encodeURIComponent(id)}`, { method: 'DELETE' }),",
        "  listOrganizationAlertChannels: (organizationID: string) => request<AlertChannelList>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels?is_pager=0`),",
        "  createOrganizationAlertChannel: (organizationID: string, input: AlertChannelInput) => request<AlertChannel>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels`, { method: 'POST', body: JSON.stringify(input) }),",
        "  updateOrganizationAlertChannel: (organizationID: string, id: string, input: AlertChannelInput) => request<AlertChannel>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),",
        "  deleteOrganizationAlertChannel: (organizationID: string, id: string) => request<void>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels/${encodeURIComponent(id)}`, { method: 'DELETE' }),",
        "  testOrganizationAlertChannel: (organizationID: string, id: string) => request<{ status: string }>(`/api/v1/organizations/${encodeURIComponent(organizationID)}/alert-channels/${encodeURIComponent(id)}/test`, { method: 'POST' }),",
        "  listProjectMembers: (id: string) => request<ProjectMemberList>(`/api/v1/projects/${encodeURIComponent(id)}/members?is_pager=0`),",
        "  listProjectMemberCandidates: (id: string) => request<UserList>(`/api/v1/projects/${encodeURIComponent(id)}/member-candidates?is_pager=0`),",
        "  listProjectCredentials: (id: string) => request<ProjectKeyList>(`/api/v1/projects/${encodeURIComponent(id)}/credentials?is_pager=0`),",
        "  createProjectCredential: (id: string, input: CreateProjectKeyInput) => request<{ item: ProjectKey; secret?: string; token?: string; signing_secret?: string }>(`/api/v1/projects/${encodeURIComponent(id)}/credentials`, { method: 'POST', body: JSON.stringify(input) }),",
        "  revokeProjectCredential: (id: string, keyId: string) => request<void>(`/api/v1/projects/${encodeURIComponent(id)}/credentials/${encodeURIComponent(keyId)}`, { method: 'DELETE' }),",
        "  listNotifications: () => request<NotificationList>('/api/v1/notifications?is_pager=0'),",
        "  readNotification: (id: string) => request<{ read: boolean }>(`/api/v1/notifications/${encodeURIComponent(id)}/read`, { method: 'POST' }),",
        "  readAllNotifications: () => request<{ read: boolean }>('/api/v1/notifications/read-all', { method: 'POST' }),",
        "  addProjectMember: (id: string, input: ProjectMemberInput) => request<{ user: ProjectMemberUser; role: string }>(`/api/v1/projects/${encodeURIComponent(id)}/members`, { method: 'POST', body: JSON.stringify(input) }),",
        "  updateProjectMember: (id: string, userId: string, role: 'project_admin' | 'project_member') => request<void>(`/api/v1/projects/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`, { method: 'PUT', body: JSON.stringify({ role }) }),",
        "  removeProjectMember: (id: string, userId: string) => request<void>(`/api/v1/projects/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`, { method: 'DELETE' }),",
        "};",
        "",
    ])
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text("\n".join(lines))
    print(f"Generated {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
