#!/bin/sh
set -eu
umask 077

BACKUP_DIR="${DUKU_BACKUP_DIR:-$HOME/Library/Application Support/DukuNetLab/backups}"
mkdir -p -m 700 "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
STAMP="$(date +%Y%m%d-%H%M%S)"
podman-compose -f compose.yml exec -T postgres pg_dump -U "${POSTGRES_USER:-duku}" "${POSTGRES_DB:-duku_net_lab}" | gzip > "$BACKUP_DIR/duku-$STAMP.sql.gz"
ls -1t "$BACKUP_DIR"/duku-*.sql.gz | awk 'NR > 8' | xargs -r rm -f
