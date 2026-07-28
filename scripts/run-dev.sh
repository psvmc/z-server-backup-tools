#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export PATH="${HOME}/go/bin:${PATH}"

echo "[ZServerBackup] 启动开发模式..."
wails3 task dev
