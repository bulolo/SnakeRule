#!/usr/bin/env python3
"""Create local production Compose settings with private random credentials."""

from pathlib import Path
import secrets
import shutil

root = Path(__file__).resolve().parents[1]
env_file = root / "deploy/.env.prod"
template = root / "deploy/.env.prod.example"
if not env_file.exists():
    shutil.copyfile(template, env_file)
lines = env_file.read_text().splitlines()
changed = False
for key, value in (("AUTH_PASSWORD", secrets.token_urlsafe(32)), ("LICENSE_ENCRYPTION_KEY", secrets.token_hex(32))):
    prefix = f"{key}="
    index = next((i for i, line in enumerate(lines) if line.startswith(prefix)), None)
    if index is None:
        lines.append(prefix + value)
        changed = True
    elif lines[index].partition("=")[2].strip() in ("", "generated-by-make-prod-init"):
        lines[index] = prefix + value
        changed = True
if changed:
    env_file.write_text("\n".join(lines) + "\n")
    print("Generated private AUTH_PASSWORD and LICENSE_ENCRYPTION_KEY in ignored deploy/.env.prod")
else:
    print("deploy/.env.prod already has production secrets; kept them unchanged")
