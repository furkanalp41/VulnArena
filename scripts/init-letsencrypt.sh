#!/usr/bin/env bash
# ============================================================
# One-time Let's Encrypt cert bootstrap for vulnarena.com.
#
# Solves the chicken-and-egg problem: nginx.conf references TLS certs in
# /etc/letsencrypt/live/vulnarena.com/ that don't exist on a fresh host,
# so nginx won't start, so certbot can't reach the ACME challenge endpoint.
#
# This script:
#   1. Generates a placeholder self-signed cert in the letsencrypt volume,
#      so nginx can boot.
#   2. Starts nginx.
#   3. Removes the placeholder.
#   4. Requests the real cert via webroot HTTP-01.
#   5. Reloads nginx to pick up the real cert.
#
# After this runs once, the certbot service in docker-compose.prod.yml
# handles renewal automatically.
#
# Prereqs:
#   - DNS for vulnarena.com and www.vulnarena.com pointing at this host
#   - Ports 80 and 443 reachable from the public internet
#   - .env file with DOMAIN and CERTBOT_EMAIL set
# ============================================================
set -euo pipefail

# Load .env if present
if [ -f .env ]; then
  # shellcheck disable=SC1091
  set -o allexport
  . ./.env
  set +o allexport
fi

DOMAIN="${DOMAIN:-vulnarena.com}"
WWW_DOMAIN="www.${DOMAIN}"
EMAIL="${CERTBOT_EMAIL:?CERTBOT_EMAIL must be set in .env}"
STAGING="${CERTBOT_STAGING:-0}"  # set to 1 to test against Let's Encrypt staging

COMPOSE="docker compose -f docker-compose.prod.yml"

echo "[+] Bootstrapping Let's Encrypt for ${DOMAIN}, ${WWW_DOMAIN}"

# ── Step 1: generate a placeholder cert so nginx can start ──
echo "[+] Creating placeholder self-signed cert..."
$COMPOSE run --rm --entrypoint "\
  sh -c 'mkdir -p /etc/letsencrypt/live/${DOMAIN} && \
         openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
           -keyout /etc/letsencrypt/live/${DOMAIN}/privkey.pem \
           -out    /etc/letsencrypt/live/${DOMAIN}/fullchain.pem \
           -subj  \"/CN=localhost\"'" certbot

# ── Step 2: start nginx so the ACME HTTP-01 challenge can reach it ──
echo "[+] Starting nginx with placeholder cert..."
$COMPOSE up --force-recreate -d nginx

# ── Step 3: delete the placeholder so certbot can write the real cert ──
echo "[+] Deleting placeholder cert..."
$COMPOSE run --rm --entrypoint "\
  sh -c 'rm -rf /etc/letsencrypt/live/${DOMAIN} \
                /etc/letsencrypt/archive/${DOMAIN} \
                /etc/letsencrypt/renewal/${DOMAIN}.conf'" certbot

# ── Step 4: request the real cert ──
echo "[+] Requesting real cert from Let's Encrypt..."
STAGING_ARG=""
if [ "$STAGING" != "0" ]; then
  echo "    (using staging API)"
  STAGING_ARG="--staging"
fi

$COMPOSE run --rm --entrypoint "\
  certbot certonly --webroot --webroot-path=/var/www/certbot \
    ${STAGING_ARG} \
    --email ${EMAIL} \
    --agree-tos --no-eff-email \
    --rsa-key-size 4096 \
    --force-renewal \
    -d ${DOMAIN} -d ${WWW_DOMAIN}" certbot

# ── Step 5: reload nginx to pick up the real cert ──
echo "[+] Reloading nginx..."
$COMPOSE exec nginx nginx -s reload

echo "[+] Done. ${DOMAIN} is now serving HTTPS."
echo "[+] Bring up the rest of the stack with: $COMPOSE up -d"
