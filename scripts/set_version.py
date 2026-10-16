#!/usr/bin/env python3
"""Set the single SnakeRule project version in VERSION and Go build metadata."""

import pathlib
import re
import sys
import json

ROOT = pathlib.Path(__file__).resolve().parents[1]
VERSION_FILE = ROOT / "VERSION"
GO_VERSION_FILE = ROOT / "backend/server/internal/version/version.go"
OPENAPI_FILE = ROOT / "backend/server/docs/openapi.json"
ADMIN_PACKAGE_FILE = ROOT / "frontend/admin/package.json"
ADMIN_LOCK_FILE = ROOT / "frontend/admin/package-lock.json"
DOCS_PACKAGE_FILE = ROOT / "frontend/docs/package.json"
DOCS_LOCK_FILE = ROOT / "frontend/docs/package-lock.json"
HELM_CHART_FILE = ROOT / "deploy/helm/Chart.yaml"
SEMVER = re.compile(r"^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$")


def main() -> int:
    if len(sys.argv) != 2 or not SEMVER.fullmatch(sys.argv[1]):
        current = VERSION_FILE.read_text().strip() if VERSION_FILE.exists() else "unknown"
        print(f"Current version: {current}")
        print("Usage: make set-version v=0.1.1")
        return 2 if len(sys.argv) == 2 else 0

    version = sys.argv[1].removeprefix("v")
    go_source = GO_VERSION_FILE.read_text()
    updated, count = re.subn(r'(const Version = ")[^"]+("\n)', rf"\g<1>{version}\g<2>", go_source, count=1)
    if count != 1:
        print(f"Could not update {GO_VERSION_FILE.relative_to(ROOT)}", file=sys.stderr)
        return 1

    openapi = json.loads(OPENAPI_FILE.read_text())
    openapi["info"]["version"] = version
    admin_package = json.loads(ADMIN_PACKAGE_FILE.read_text())
    admin_package["version"] = version
    admin_lock = json.loads(ADMIN_LOCK_FILE.read_text()) if ADMIN_LOCK_FILE.exists() else None
    if admin_lock:
        admin_lock["version"] = version
        admin_lock.get("packages", {}).get("", {})["version"] = version
    docs_package = json.loads(DOCS_PACKAGE_FILE.read_text())
    docs_package["version"] = version
    docs_lock = json.loads(DOCS_LOCK_FILE.read_text()) if DOCS_LOCK_FILE.exists() else None
    if docs_lock:
        docs_lock["version"] = version
        docs_lock.get("packages", {}).get("", {})["version"] = version
    helm_chart = HELM_CHART_FILE.read_text()
    helm_chart, chart_version_count = re.subn(r"(?m)^version: [^\n]+$", f"version: {version}", helm_chart, count=1)
    helm_chart, app_version_count = re.subn(r'(?m)^appVersion: "[^"\n]+"$', f'appVersion: "{version}"', helm_chart, count=1)
    if chart_version_count != 1 or app_version_count != 1:
        print(f"Could not update Helm chart version in {HELM_CHART_FILE.relative_to(ROOT)}", file=sys.stderr)
        return 1
    readme = ROOT / "README.md"
    readme_text = readme.read_text()
    readme_text, readme_count = re.subn(r"(?m)^### v[^\s]+ ·", f"### v{version} ·", readme_text, count=1)
    roadmap = ROOT / "docs/todo.md"
    roadmap_text = roadmap.read_text()
    roadmap_text, roadmap_count = re.subn(r"当前版本：`v[^`]+`", f"当前版本：`v{version}`", roadmap_text, count=1)
    workbench = ROOT / "frontend/admin/src/workbench.tsx"
    workbench_text = workbench.read_text()
    workbench_text, workbench_count = re.subn(r'(useState\(")[^\"]+("\);\s*\n\s*const \[platformBrand)', rf"\g<1>{version}\g<2>", workbench_text, count=1)
    workbench_text, workbench_fallback_count = re.subn(r'(meta\.version \?\? ")[^\"]+(")', rf"\g<1>{version}\g<2>", workbench_text, count=1)
    web_index = ROOT / "backend/server/internal/transport/http/web/index.html"
    web_text = web_index.read_text()
    web_text, web_count = re.subn(r'(<span id="app-version">v)[^<]+(</span>)', rf"\g<1>{version}\g<2>", web_text, count=1)
    if any(count != 1 for count in (readme_count, roadmap_count, workbench_count, workbench_fallback_count, web_count)):
        print("Could not update all current-version labels", file=sys.stderr)
        return 1
    VERSION_FILE.write_text(version + "\n")
    GO_VERSION_FILE.write_text(updated)
    OPENAPI_FILE.write_text(json.dumps(openapi, ensure_ascii=False, indent=2) + "\n")
    ADMIN_PACKAGE_FILE.write_text(json.dumps(admin_package, ensure_ascii=False, indent=2) + "\n")
    if admin_lock:
        ADMIN_LOCK_FILE.write_text(json.dumps(admin_lock, ensure_ascii=False, indent=2) + "\n")
    DOCS_PACKAGE_FILE.write_text(json.dumps(docs_package, ensure_ascii=False, indent=2) + "\n")
    if docs_lock:
        DOCS_LOCK_FILE.write_text(json.dumps(docs_lock, ensure_ascii=False, indent=2) + "\n")
    HELM_CHART_FILE.write_text(helm_chart)
    readme.write_text(readme_text)
    roadmap.write_text(roadmap_text)
    workbench.write_text(workbench_text)
    web_index.write_text(web_text)
    print(f"SnakeRule version set to {version}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
