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
  grep -nEi '(BEGIN [A-Z ]*PRIVATE KEY)' "$file" 2>/dev/null || true
done)"
if [ -n "$matches" ]; then
  printf '%s\n' "$matches" >&2
  echo "Private keys detected in publishable files" >&2
  exit 1
fi

local_pattern_file="${DUKU_SAFETY_PATTERN_FILE:-.duku-sensitive-patterns}"
if [ -f "$local_pattern_file" ]; then
  local_matches="$(git ls-files --cached --others --exclude-standard | while IFS= read -r file; do
    grep -nEif "$local_pattern_file" "$file" 2>/dev/null || true
  done)"
  if [ -n "$local_matches" ]; then
    printf '%s\n' "$local_matches" >&2
    echo "Local sensitive identifiers detected in publishable files" >&2
    exit 1
  fi
fi

if grep -Eq '(duku-local-only|change-this-local-token)' compose.yml; then
  echo "Compose contains weak fallback credentials" >&2
  exit 1
fi

echo "Repository publication safety verified"
