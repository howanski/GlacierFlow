#!/bin/sh
set -e

# Hash the plaintext password with bcrypt
export GF_HTTP_PASSWORD_HASHED=$(caddy hash-password --algorithm bcrypt --plaintext "$GF_HTTP_PASSWORD")

# Start Caddy
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
