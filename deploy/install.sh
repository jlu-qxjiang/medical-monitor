#!/usr/bin/env bash
# 服务器端安装脚本。前提：所有部署文件已 scp 到 /opt/medical-monitor/。
set -euo pipefail

APP_DIR="/opt/medical-monitor"
APP_USER="medical"
SERVICE_SRC="${APP_DIR}/medical-monitor.service"
SERVICE_DST="/etc/systemd/system/medical-monitor.service"

echo "=== [1/5] 创建系统用户 ${APP_USER} ==="
if id -u "${APP_USER}" >/dev/null 2>&1; then
  echo "用户 ${APP_USER} 已存在，跳过"
else
  useradd --system --no-create-home "${APP_USER}"
  echo "已创建用户 ${APP_USER}"
fi

echo "=== [2/5] 校验部署文件 ==="
mkdir -p "${APP_DIR}/logs"
for f in medical-monitor config.yaml .env.production medical-monitor.service; do
  [ -f "${APP_DIR}/${f}" ] || { echo "错误：缺少 ${APP_DIR}/${f}，请先上传"; exit 1; }
done

chmod +x "${APP_DIR}/medical-monitor"
chmod 600 "${APP_DIR}/.env.production"
chown -R "${APP_USER}:${APP_USER}" "${APP_DIR}"

echo "=== [3/5] 安装 systemd 服务 ==="
cp "${SERVICE_SRC}" "${SERVICE_DST}"
systemctl daemon-reload

echo "=== [4/5] 启动服务 ==="
systemctl enable --now medical-monitor

echo "=== [5/5] 检查状态 ==="
sleep 2
systemctl status medical-monitor --no-pager || true

echo ""
echo "部署完成。验证：curl http://127.0.0.1:8080/health"
