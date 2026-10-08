#!/usr/bin/env bash
# 本地部署脚本（Windows Git Bash 或任意 Linux/macOS 运行）。
# 用法：先改下面 SERVER_IP / SSH_USER，并确保 .env.production 已就绪，然后 bash deploy.sh
set -euo pipefail
cd "$(dirname "$0")"

# ===== 部署配置（按需修改）=====
SERVER_IP="你的服务器IP"
SSH_USER="root"
REMOTE_DIR="/opt/medical-monitor"

# ===== 校验本地文件 =====
[ -f "../backend/server-linux" ] || { echo "错误：找不到 ../backend/server-linux"; exit 1; }
[ -f ".env.production" ] || { echo "错误：找不到 .env.production，请先从 .env.production.example 复制并填写"; exit 1; }

echo "=== 创建远程目录 ==="
ssh "${SSH_USER}@${SERVER_IP}" "mkdir -p ${REMOTE_DIR}"

echo "=== 上传文件 ==="
scp "../backend/server-linux"       "${SSH_USER}@${SERVER_IP}:${REMOTE_DIR}/medical-monitor"
scp "config.yaml"                   "${SSH_USER}@${SERVER_IP}:${REMOTE_DIR}/config.yaml"
scp ".env.production"               "${SSH_USER}@${SERVER_IP}:${REMOTE_DIR}/.env.production"
scp "medical-monitor.service"       "${SSH_USER}@${SERVER_IP}:${REMOTE_DIR}/medical-monitor.service"
scp "install.sh"                    "${SSH_USER}@${SERVER_IP}:${REMOTE_DIR}/install.sh"

echo "=== 执行安装脚本 ==="
ssh "${SSH_USER}@${SERVER_IP}" "bash ${REMOTE_DIR}/install.sh"
