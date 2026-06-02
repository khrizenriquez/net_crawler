#!/bin/sh
set -eu

ENV_FILE="${1:-.env}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Missing $ENV_FILE. Run make bootstrap and configure local secrets." >&2
  exit 1
fi

mode="$(stat -f '%Lp' "$ENV_FILE" 2>/dev/null || stat -c '%a' "$ENV_FILE")"
if [ "$mode" != "600" ]; then
  echo "$ENV_FILE must have mode 600, found $mode" >&2
  exit 1
fi

value() {
  sed -n "s/^$1=//p" "$ENV_FILE" | tail -n 1
}

postgres_password="$(value POSTGRES_PASSWORD)"
host_token="$(value DUKU_HOST_TOKEN)"
data_dir="$(value DUKU_DATA_DIR)"

case "$postgres_password" in
  ""|replace-with-a-local-password|duku|duku-local-only)
    echo "Replace POSTGRES_PASSWORD in $ENV_FILE with a local password of at least 16 characters." >&2
    exit 1
    ;;
esac
if [ "${#postgres_password}" -lt 16 ]; then
  echo "POSTGRES_PASSWORD in $ENV_FILE must contain at least 16 characters." >&2
  exit 1
fi
case "$postgres_password" in
  *[!A-Za-z0-9._~-]*)
    echo "POSTGRES_PASSWORD in $ENV_FILE must use URL-safe characters only: A-Z, a-z, 0-9, '.', '_', '~' or '-'." >&2
    exit 1
    ;;
esac

case "$host_token" in
  ""|replace-with-a-random-local-token|change-this-local-token|local-secret|token)
    echo "Replace DUKU_HOST_TOKEN in $ENV_FILE with a random local token of at least 32 characters." >&2
    exit 1
    ;;
esac
if [ "${#host_token}" -lt 32 ]; then
  echo "DUKU_HOST_TOKEN in $ENV_FILE must contain at least 32 characters." >&2
  exit 1
fi
case "$host_token" in
  *[!A-Za-z0-9._~-]*)
    echo "DUKU_HOST_TOKEN in $ENV_FILE must use URL-safe characters only: A-Z, a-z, 0-9, '.', '_', '~' or '-'." >&2
    exit 1
    ;;
esac

case "$data_dir" in
  ""|"/"|*".."*)
    echo "DUKU_DATA_DIR in $ENV_FILE must be a safe absolute local directory." >&2
    exit 1
    ;;
  /*) ;;
  *)
    echo "DUKU_DATA_DIR in $ENV_FILE must be an absolute local directory." >&2
    exit 1
    ;;
esac

echo "Local secret configuration verified"
