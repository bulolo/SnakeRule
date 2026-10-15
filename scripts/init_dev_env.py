#!/usr/bin/env python3
"""Create or upgrade the local dev env without overwriting existing settings."""

from pathlib import Path
import secrets
import shutil

root = Path(__file__).resolve().parents[1]
env_file = root / "backend/.env"
template = root / "backend/.env.example"

if not env_file.exists():
    shutil.copyfile(template, env_file)
    print("Created backend/.env from the example")

lines = env_file.read_text().splitlines()
key_line = next((i for i, line in enumerate(lines) if line.startswith("LICENSE_ENCRYPTION_KEY=")), None)
if key_line is None:
    lines.append(f"LICENSE_ENCRYPTION_KEY={secrets.token_hex(32)}")
    env_file.write_text("\n".join(lines) + "\n")
    print("Generated a local license encryption key in ignored backend/.env")
elif not lines[key_line].partition("=")[2].strip():
    lines[key_line] = f"LICENSE_ENCRYPTION_KEY={secrets.token_hex(32)}"
    env_file.write_text("\n".join(lines) + "\n")
    print("Generated a local license encryption key in ignored backend/.env")
else:
    print("backend/.env already has a license encryption key; kept it unchanged")
