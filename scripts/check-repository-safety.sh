#!/bin/sh
set -eu

publishable="$(git ls-files --cached --others --exclude-standard)"
unsafe_paths="$(printf '%s\n' "$publishable" | grep -E '(^|/)(\.env($|\.)|.*\.(pcap|pcapng|pem|key|crt|p12|mobileprovision|log|sqlite|db|sql\.gz|tar\.gz|zip)$|node_modules/|dist/|bin/|backups/|exports/|volumes/|data/)' | grep -v '^\.env\.example$' || true)"
if [ -n "$unsafe_paths" ]; then
  echo "Unsafe publishable paths detected:" >&2
  printf '%s\n' "$unsafe_paths" >&2
  exit 1
fi

if ! git check-ignore -q .env; then
  echo ".env must remain ignored" >&2
  exit 1
fi

matches="$(git ls-files --cached --others --exclude-standard | while IFS= read -r file; do
  if [ "$file" = "scripts/check-repository-safety.sh" ]; then
    continue
  fi
  grep -nEi \
    '(BEGIN [A-Z ]*PRIVATE KEY|b8:5f:b0:5c:1f:23|b8:5f:b0:5c:1f:24|34:7f:da:80:1a:20|a4:cf:12:4e:90:01|8c:85:90:71:3e:44|4857544301FEDAA9|HWTC01FEDAA9|2150085157EGN2004973)' \
    "$file" 2>/dev/null || true
done)"
if [ -n "$matches" ]; then
  printf '%s\n' "$matches" >&2
  echo "Sensitive router identifiers or private keys detected in publishable files" >&2
  exit 1
fi

if grep -Eq '(duku-local-only|change-this-local-token)' compose.yml; then
  echo "Compose contains weak fallback credentials" >&2
  exit 1
fi

echo "Repository publication safety verified"
