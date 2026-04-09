#!/bin/bash

# Initialize Let's Encrypt SSL certificates for dds-billing
# Usage: ./scripts/init-ssl.sh your-domain.com your@email.com
# Add --staging flag for test certificates (recommended for first run)

set -e

# Change to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "Working directory: $PROJECT_ROOT"
echo ""

# Parse arguments
DOMAIN=""
EMAIL=""
STAGING=0
RSA_KEY_SIZE=4096
DATA_PATH="./certbot"

while [[ $# -gt 0 ]]; do
  case $1 in
    --staging) STAGING=1; shift ;;
    --email) EMAIL="$2"; shift 2 ;;
    -*) echo "Unknown option: $1"; exit 1 ;;
    *)
      if [ -z "$DOMAIN" ]; then
        DOMAIN="$1"
      elif [ -z "$EMAIL" ]; then
        EMAIL="$1"
      fi
      shift
      ;;
  esac
done

if [ -z "$DOMAIN" ]; then
  echo "Usage: $0 <domain> [email] [--staging]"
  echo ""
  echo "Examples:"
  echo "  $0 pay.example.com admin@example.com --staging  # Test certificate (recommended first)"
  echo "  $0 pay.example.com admin@example.com            # Production certificate"
  exit 1
fi

if [ "$STAGING" = "1" ]; then
  echo ">>> STAGING MODE: Will request TEST certificate (not trusted by browsers)"
  echo ""
fi

# 1. Prepare directories
echo "### Preparing directories ..."
mkdir -p "$DATA_PATH/conf"
mkdir -p "$DATA_PATH/www"

# 2. Check existing certificates
if [ -d "$DATA_PATH/conf/live/$DOMAIN" ]; then
  read -p "Existing certificates found for $DOMAIN. Replace? (y/N) " decision
  if [ "$decision" != "Y" ] && [ "$decision" != "y" ]; then
    exit
  fi
fi

# 3. Download recommended TLS parameters
if [ ! -e "$DATA_PATH/conf/options-ssl-nginx.conf" ] || [ ! -e "$DATA_PATH/conf/ssl-dhparams.pem" ]; then
  echo "### Downloading recommended TLS parameters ..."
  curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf > "$DATA_PATH/conf/options-ssl-nginx.conf"
  curl -s https://raw.githubusercontent.com/certbot/certbot/master/certbot/certbot/ssl-dhparams.pem > "$DATA_PATH/conf/ssl-dhparams.pem"
  echo
fi

# 4. Create dummy certificate so nginx can start
echo "### Creating dummy certificate for $DOMAIN ..."
CERT_PATH="/etc/letsencrypt/live/$DOMAIN"
mkdir -p "$DATA_PATH/conf/live/$DOMAIN"
docker compose run --rm --entrypoint "\
  openssl req -x509 -nodes -newkey rsa:$RSA_KEY_SIZE -days 1 \
    -keyout '$CERT_PATH/privkey.pem' \
    -out '$CERT_PATH/fullchain.pem' \
    -subj '/CN=localhost'" certbot
echo

# 5. Start backend (nginx depends on it)
echo "### Starting mysql and backend ..."
docker compose up -d mysql backend
echo "### Waiting for backend to be ready ..."
sleep 8
echo

# 6. Start nginx with dummy cert
echo "### Starting nginx ..."
docker compose up --force-recreate -d nginx
echo

# 7. Delete dummy certificate
echo "### Deleting dummy certificate ..."
docker compose run --rm --entrypoint "\
  rm -Rf /etc/letsencrypt/live/$DOMAIN && \
  rm -Rf /etc/letsencrypt/archive/$DOMAIN && \
  rm -Rf /etc/letsencrypt/renewal/$DOMAIN.conf" certbot
echo

# 8. Request real certificate
echo "### Requesting Let's Encrypt certificate for $DOMAIN ..."

# Email arg
if [ -z "$EMAIL" ]; then
  EMAIL_ARG="--register-unsafely-without-email"
else
  EMAIL_ARG="--email $EMAIL"
fi

# Staging arg
STAGING_ARG=""
if [ "$STAGING" = "1" ]; then
  STAGING_ARG="--staging"
fi

docker compose run --rm --entrypoint "\
  certbot certonly --webroot -w /var/www/certbot \
    $STAGING_ARG \
    $EMAIL_ARG \
    -d $DOMAIN \
    --rsa-key-size $RSA_KEY_SIZE \
    --agree-tos \
    --force-renewal" certbot
echo

# 9. Reload nginx with real certificate
echo "### Reloading nginx ..."
docker compose exec nginx nginx -s reload

echo ""
echo "============================================"
echo "  SSL certificate obtained successfully!"
if [ "$STAGING" = "1" ]; then
  echo "  (STAGING/TEST certificate - not trusted)"
  echo ""
  echo "  To get a production certificate, run:"
  echo "  $0 $DOMAIN $EMAIL"
else
  echo "  https://$DOMAIN is now ready"
fi
echo "============================================"
