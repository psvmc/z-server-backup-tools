#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"
export PATH="${HOME}/go/bin:${PATH}"
echo "[ZServerBackup] Building zipbak-srv (Windows + Linux)..."
wails3 task build:zipbak-srv-all
echo "Output: dist/zipbak-srv.exe  (remote Windows)"
echo "        dist/zipbak-srv      (remote Linux)"
