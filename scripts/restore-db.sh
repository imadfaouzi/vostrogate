#!/usr/bin/env bash
set -euo pipefail
if [ $# -ne 1 ] || [ ! -f "$1" ]; then echo 'Usage: scripts/restore-db.sh backup.sql.gz'; exit 1; fi
cd "$(dirname "$0")/.."
echo 'WARNING: restoration overwrites current business data. Use only on a fresh/recovery database.'
read -r -p 'Type RESTORE to continue: ' confirm
[ "$confirm" = RESTORE ] || exit 1
gzip -dc "$1" | docker compose exec -T mysql sh -c 'exec mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"'
echo 'Database restored. Restart api and web containers.'
