#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
name="vostrogate_$(date -u +%Y%m%d_%H%M%S).sql.gz"
# The credentials are supplied from the MySQL container environment, not printed to STDOUT.
docker compose exec -T mysql sh -c 'exec mysqldump -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" --single-transaction --routines --triggers --events "$MYSQL_DATABASE"' | gzip > "$name"
chmod 600 "$name"
echo "Backup written to $name. Also secure the separate .env secrets."
