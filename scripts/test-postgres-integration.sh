#!/bin/sh
set -eu

runtime="${CONTAINER_RUNTIME:-podman}"
name="duku-postgres-test-$$"
port="${DUKU_TEST_POSTGRES_PORT:-55432}"

cleanup() {
  "$runtime" rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

"$runtime" run --rm -d --name "$name" \
  -e POSTGRES_DB=duku \
  -e POSTGRES_USER=duku \
  -e POSTGRES_PASSWORD=duku \
  -p "127.0.0.1:${port}:5432" \
  postgres:16-alpine >/dev/null

attempt=0
until "$runtime" exec "$name" pg_isready -U duku -d duku >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 30 ]; then
    echo "PostgreSQL fixture did not become ready" >&2
    exit 1
  fi
  sleep 1
done

DUKU_TEST_DATABASE_URL="postgres://duku:duku@127.0.0.1:${port}/duku?sslmode=disable" \
  scripts/go.sh test ./internal/store -run TestPostgres -count=1
