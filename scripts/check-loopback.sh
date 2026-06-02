#!/bin/sh
set -eu

if grep -R "ports:.*0\\.0\\.0\\.0" compose.yml dashboard 2>/dev/null; then
  echo "Refusing non-loopback publication" >&2
  exit 1
fi
grep -q '"127.0.0.1:8080:8080"' compose.yml
grep -q '"127.0.0.1:4173:4173"' compose.yml
echo "Loopback publication verified"

