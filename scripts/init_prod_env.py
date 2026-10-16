#!/usr/bin/env python3
"""Create production Compose settings from the shared environment example."""

from pathlib import Path
import secrets
import shutil

root = Path(__file__).resolve().parents[1]
env_file = root / "deploy/.env.prod"
template = root / "backend/.env.example"
if not env_file.exists():
    shutil.copyfile(template, env_file)
lines = env_file.read_text().splitlines()
changed = False
defaults = {
    "SERVER_ENV": "production",
    "LICENSE_STORE_PATH": "/data/license.json",
    "LOG_LEVEL": "info",
}
for key, value in defaults.items():
    prefix = f"{key}="
    index = next((i for i, line in enumerate(lines) if line.startswith(prefix)), None)
    if index is None:
        lines.append(prefix + value)
        changed = True
    elif lines[index].partition("=")[2].strip() in ("", "development", "./data/license.json", "debug"):
        lines[index] = prefix + value
        changed = True

for key, value in (("AUTH_PASSWORD", secrets.token_urlsafe(32)), ("LICENSE_ENCRYPTION_KEY", secrets.token_hex(32))):
    prefix = f"{key}="
    index = next((i for i, line in enumerate(lines) if line.startswith(prefix)), None)
    if index is None:
        lines.append(prefix + value)
        changed = True
    elif lines[index].partition("=")[2].strip() in ("", "generated-by-make-prod-init", "change-this-before-run"):
        lines[index] = prefix + value
        changed = True
if changed:
    env_file.write_text("\n".join(lines) + "\n")
    print("Generated production settings in ignored deploy/.env.prod from backend/.env.example")
else:
    print("deploy/.env.prod already has production secrets; kept them unchanged")
