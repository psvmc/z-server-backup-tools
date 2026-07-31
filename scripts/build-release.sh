#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export PATH="${HOME}/go/bin:${PATH}"

echo "[ZServerBackup] 构建发布版本..."
wails3 task build

echo "[ZServerBackup] 打包安装程序..."
export INSTALL_SCOPE=user
wails3 task package

echo "[ZServerBackup] 构建 zipbak-srv（Windows + Linux 源机）..."
wails3 task build:zipbak-srv-all

echo "[ZServerBackup] 构建完成，输出目录: dist/"
echo "  客户端: ZServerBackup + 安装包"
echo "  源机:   zipbak-srv.exe  (Windows)"
echo "          zipbak-srv      (Linux, chmod +x)"
echo "  GitHub Release Linux 资产名: zipbak-srv-linux-amd64（放到服务器后改名为 zipbak-srv）"
