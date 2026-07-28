#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
export PATH="${HOME}/go/bin:${PATH}"
echo "[ZServerBackup] Building zipbak-srv.exe (remote Windows server)..."
wails3 task build:zipbak-srv
