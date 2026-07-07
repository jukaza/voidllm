#!/bin/bash
set -euo pipefail

VPS_IP="${VPS_IP:-160.250.181.95}"
SSH_USER="${SSH_USER:-root}"
REMOTE_PATH="/root/tavo"

if [[ -z "${VPS_PASSWORD:-}" && -z "${SSH_AUTH_SOCK:-}" ]]; then
  echo "Set VPS_PASSWORD or use SSH key auth"
  exit 1
fi

ssh_cmd() {
  if [[ -n "${VPS_PASSWORD:-}" ]]; then
    sshpass -p "$VPS_PASSWORD" ssh -o StrictHostKeyChecking=no "${SSH_USER}@${VPS_IP}" "$@"
  else
    ssh -o StrictHostKeyChecking=no "${SSH_USER}@${VPS_IP}" "$@"
  fi
}

scp_cmd() {
  if [[ -n "${VPS_PASSWORD:-}" ]]; then
    sshpass -p "$VPS_PASSWORD" scp -o StrictHostKeyChecking=no "$@"
  else
    scp -o StrictHostKeyChecking=no "$@"
  fi
}

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "========== BUILD =========="
cd "$ROOT/ui" && npm ci && npm run build
cd "$ROOT"
CGO_ENABLED=0 go build -ldflags="-s -w" -o tavo ./cmd/tavo
echo "✅ Built tavo"

echo "========== UPLOAD =========="
ssh_cmd "mkdir -p ${REMOTE_PATH}/data ${REMOTE_PATH}/logs"
scp_cmd "$ROOT/tavo" "${SSH_USER}@${VPS_IP}:${REMOTE_PATH}/tavo.new"
scp_cmd "$ROOT/deploy/tavo.production.yaml" "${SSH_USER}@${VPS_IP}:${REMOTE_PATH}/tavo.yaml"

ssh_cmd "
  mv ${REMOTE_PATH}/tavo.new ${REMOTE_PATH}/tavo
  chmod +x ${REMOTE_PATH}/tavo
  if pm2 show tavo >/dev/null 2>&1; then
    pm2 restart tavo
  else
    cd ${REMOTE_PATH} && set -a && source .env && set +a && pm2 start ecosystem.config.cjs
  fi
  pm2 save
  pm2 status tavo
  curl -sS http://127.0.0.1:8080/healthz
"

echo ""
echo "✅ Deploy complete — https://api.tavo.io.vn/healthz"