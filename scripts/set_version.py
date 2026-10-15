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
    VERSION_FILE.write_text(version + "\n")
    GO_VERSION_FILE.write_text(updated)
    OPENAPI_FILE.write_text(json.dumps(openapi, ensure_ascii=False, indent=2) + "\n")
    ADMIN_PACKAGE_FILE.write_text(json.dumps(admin_package, ensure_ascii=False, indent=2) + "\n")
    if admin_lock:
        ADMIN_LOCK_FILE.write_text(json.dumps(admin_lock, ensure_ascii=False, indent=2) + "\n")
    print(f"SnakeRule version set to {version}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
