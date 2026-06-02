#!/bin/sh
set -eu

ENV_FILE="${1:-.env}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Missing $ENV_FILE. Run make bootstrap first." >&2
  exit 1
fi
if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required to generate local secrets." >&2
  exit 1
fi

postgres_password="$(openssl rand -hex 24)"
host_token="$(openssl rand -hex 32)"
data_dir="${DUKU_DATA_DIR:-$HOME/.duku-net-lab}"
temp_file="$(mktemp "${ENV_FILE}.tmp.XXXXXX")"
trap 'rm -f "$temp_file"' EXIT INT TERM

awk -v postgres_password="$postgres_password" -v host_token="$host_token" -v data_dir="$data_dir" '
  BEGIN { saw_password = 0; saw_token = 0; saw_data_dir = 0 }
  /^POSTGRES_PASSWORD=/ {
    saw_password = 1
    current = substr($0, length("POSTGRES_PASSWORD=") + 1)
    if (current == "" || current == "replace-with-a-local-password" || current == "duku" || current == "duku-local-only") {
      print "POSTGRES_PASSWORD=" postgres_password
    } else {
      print
    }
    next
  }
  /^DUKU_HOST_TOKEN=/ {
    saw_token = 1
    current = substr($0, length("DUKU_HOST_TOKEN=") + 1)
    if (current == "" || current == "replace-with-a-random-local-token" || current == "change-this-local-token" || current == "local-secret" || current == "token") {
      print "DUKU_HOST_TOKEN=" host_token
    } else {
      print
    }
    next
  }
  /^DUKU_DATA_DIR=/ {
    saw_data_dir = 1
    current = substr($0, length("DUKU_DATA_DIR=") + 1)
    if (current == "" || current == "/absolute/local/path") {
      print "DUKU_DATA_DIR=" data_dir
    } else {
      print
    }
    next
  }
  { print }
  END {
    if (!saw_password) print "POSTGRES_PASSWORD=" postgres_password
    if (!saw_token) print "DUKU_HOST_TOKEN=" host_token
    if (!saw_data_dir) print "DUKU_DATA_DIR=" data_dir
  }
' "$ENV_FILE" > "$temp_file"

chmod 600 "$temp_file"
mv "$temp_file" "$ENV_FILE"
trap - EXIT INT TERM
scripts/check-local-secrets.sh "$ENV_FILE"
echo "Local .env is configured. Secret values were not printed."
