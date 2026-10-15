#!/usr/bin/env python3
"""Generate TypeScript API schema types from the checked-in OpenAPI document."""

import json
import pathlib
import sys


def type_for(schema: dict) -> str:
    if "$ref" in schema:
        return schema["$ref"].rsplit("/", 1)[-1]
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
        "  constructor(public readonly status: number, message: string) { super(message); }",
        "}",
        "",
        "async function request<T>(path: string, init?: RequestInit): Promise<T> {",
        "  const multipart = typeof FormData !== 'undefined' && init?.body instanceof FormData;",
        "  const response = await fetch(path, {",
        "    ...init,",
        "    headers: { ...(init?.body && !multipart ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },",
        "  });",
        "  if (!response.ok) {",
        "    if (response.status === 401 && !path.startsWith('/api/v1/auth/')) window.dispatchEvent(new Event('snakerule:unauthorized'));",
        "    const body = await response.json().catch(() => ({}));",
        "    throw new SnakeRuleApiError(response.status, body?.error?.message ?? response.statusText);",
        "  }",
        "  return (response.status === 204 ? undefined : response.json()) as Promise<T>;",
        "}",
        "",
        "export const snakeRuleApi = {",
        "  getSession: () => request<SessionResponse>('/api/v1/auth/session'),",
        "  login: (input: LoginRequest) => request<SessionResponse>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(input) }),",
        "  logout: () => request<{ authenticated: boolean }>('/api/v1/auth/logout', { method: 'POST' }),",
        "  listRulesets: () => request<RulesetList>('/api/v1/rulesets'),",
        "  getRuleset: (id: string) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(id)}`),",
        "  saveRuleset: (ruleset: Ruleset) => request<Ruleset>(`/api/v1/rulesets/${encodeURIComponent(ruleset.id)}`, { method: 'PUT', body: JSON.stringify(ruleset) }),",
        "  deleteRuleset: (id: string) => request<void>(`/api/v1/rulesets/${encodeURIComponent(id)}`, { method: 'DELETE' }),",
        "  evaluate: (input: EvaluateRequest) => request<Decision>('/api/v1/engine/evaluate', { method: 'POST', body: JSON.stringify(input) }),",
        "  getLicense: () => request<LicenseStatus>('/api/v1/platform/license'),",
        "  uploadLicense: (file: File) => { const form = new FormData(); form.append('file', file); return request<LicenseStatus>('/api/v1/platform/license', { method: 'POST', body: form }); },",
        "  clearLicense: () => request<LicenseStatus>('/api/v1/platform/license', { method: 'DELETE' }),",
        "  listWorkspaces: () => request<WorkspaceList>('/api/v1/workspaces'),",
        "  createWorkspace: (input: WorkspaceInput) => request<Workspace>('/api/v1/workspaces', { method: 'POST', body: JSON.stringify(input) }),",
        "  updateWorkspace: (id: string, input: WorkspaceInput) => request<Workspace>(`/api/v1/workspaces/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(input) }),",
        "  deleteWorkspace: (id: string) => request<void>(`/api/v1/workspaces/${encodeURIComponent(id)}`, { method: 'DELETE' }),",
        "  setActiveWorkspace: (workspace_id: string) => request<ActiveWorkspaceResponse>('/api/v1/workspaces/active', { method: 'PUT', body: JSON.stringify({ workspace_id }) }),",
        "};",
        "",
    ])
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text("\n".join(lines))
    print(f"Generated {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
