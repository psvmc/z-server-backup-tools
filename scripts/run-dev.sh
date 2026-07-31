#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export PATH="${HOME}/go/bin:${PATH}"

echo "[ZServerBackup] Dev mode"
echo "  - Frontend: Vite HMR (edit .vue/.ts/.css, window stays open)"
echo "  - Backend: rebuild+restart only when *.go changes"
wails3 task dev
