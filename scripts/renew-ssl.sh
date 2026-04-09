#!/bin/bash

# Renew Let's Encrypt SSL certificates
# Run via crontab: 0 3 1 * * /opt/dds-billing/scripts/renew-ssl.sh >> /var/log/dds-billing-ssl-renew.log 2>&1

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "[$(date)] Starting certificate renewal ..."

docker compose run --rm certbot renew

docker compose exec nginx nginx -s reload

echo "[$(date)] Certificate renewal completed."
