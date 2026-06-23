#!/bin/bash
# dds-billing 前端发布脚本：本地构建一次，部署到两台机器（密码各自不同）
# 参照 dds-official/scripts/release.sh

# 部署目标目录（宿主机 nginx 静态根目录）
REMOTE_DIR="/var/www/dds-billing/html"

# 默认两台服务器 IP（按需修改）
DEFAULT_IP_1=""
DEFAULT_IP_2=""

read -p "Enter remote server 1 IP [${DEFAULT_IP_1}]: " REMOTE_IP_1
REMOTE_IP_1=${REMOTE_IP_1:-$DEFAULT_IP_1}
read -s -p "Enter password for ubuntu@${REMOTE_IP_1}: " REMOTE_PWD_1
echo

read -p "Enter remote server 2 IP [${DEFAULT_IP_2}]: " REMOTE_IP_2
REMOTE_IP_2=${REMOTE_IP_2:-$DEFAULT_IP_2}
read -s -p "Enter password for ubuntu@${REMOTE_IP_2}: " REMOTE_PWD_2
echo

# 检查 sshpass 是否安装（密码登录需要）
if ! command -v sshpass >/dev/null 2>&1; then
  echo "Error: sshpass is not installed. Install it first (macOS: brew install hudochenkov/sshpass/sshpass)." >&2
  exit 1
fi

# 构建一次（前端在 web/ 子目录），部署到两台机器
( cd web && npm run build ) || exit 1

# 部署函数：$1=IP, $2=password
deploy() {
  local ip="$1"
  local pwd="$2"

  if [ -z "$ip" ]; then
    echo "Skip: empty IP."
    return 0
  fi

  echo "==> Deploying to ubuntu@${ip}:${REMOTE_DIR} ..."
  # 从 web/ 目录打包 dist，流式传输到远端解压（--strip-components=1 去掉 dist/ 前缀）
  tar --no-xattrs --no-mac-metadata -C web -czf - dist | \
    sshpass -p "$pwd" ssh -o StrictHostKeyChecking=no "ubuntu@${ip}" "
sudo rm -rf ${REMOTE_DIR} &&
sudo mkdir -p ${REMOTE_DIR} &&
sudo tar -xzf - -C ${REMOTE_DIR} --strip-components=1
"

  if [ $? -eq 0 ]; then
    echo "==> Deploy to ${ip} succeeded."
  else
    echo "==> Deploy to ${ip} FAILED." >&2
    return 1
  fi
}

deploy "$REMOTE_IP_1" "$REMOTE_PWD_1"
deploy "$REMOTE_IP_2" "$REMOTE_PWD_2"
