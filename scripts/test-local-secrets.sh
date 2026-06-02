#!/bin/sh
set -eu

TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TEMP_DIR"' EXIT INT TERM
ENV_FILE="$TEMP_DIR/.env"

write_env() {
  cat > "$ENV_FILE" <<EOF
POSTGRES_PASSWORD=$1
DUKU_HOST_TOKEN=$2
DUKU_DATA_DIR=$TEMP_DIR/data
EOF
  chmod "$3" "$ENV_FILE"
}

write_env "replace-with-a-local-password" "replace-with-a-random-local-token" 600
if scripts/check-local-secrets.sh "$ENV_FILE" >/dev/null 2>&1; then
  echo "Placeholder credentials unexpectedly passed validation" >&2
  exit 1
fi

write_env "long-but-unsafe/password" "abcdefghijklmnopqrstuvwxyz123456" 600
if scripts/check-local-secrets.sh "$ENV_FILE" >/dev/null 2>&1; then
  echo "URL-unsafe PostgreSQL password unexpectedly passed validation" >&2
  exit 1
fi

write_env "local-password-123456" "abcdefghijklmnopqrstuvwxyz123456" 644
if scripts/check-local-secrets.sh "$ENV_FILE" >/dev/null 2>&1; then
  echo "Overly broad .env permissions unexpectedly passed validation" >&2
  exit 1
fi

write_env "local-password-123456" "abcdefghijklmnopqrstuvwxyz123456" 600
scripts/check-local-secrets.sh "$ENV_FILE" >/dev/null

write_env "replace-with-a-local-password" "replace-with-a-random-local-token" 600
scripts/configure-local-env.sh "$ENV_FILE" >/dev/null
scripts/check-local-secrets.sh "$ENV_FILE" >/dev/null
if grep -Eq '(replace-with-a-local-password|replace-with-a-random-local-token)' "$ENV_FILE"; then
  echo "Local environment initializer kept placeholder credentials" >&2
  exit 1
fi
echo "Local secret validator verified"
