#!/usr/bin/env sh
set -eu
if [ -e .env ] || [ -e config/client.env ]; then
  echo 'Configuration already exists; files left unchanged.' >&2
  exit 1
fi
umask 077
cp .env.example .env
cp config/client.env.example config/client.env
chmod 600 .env config/client.env
# Run the API as the config file owner so mode 600 works on Linux bind mounts.
printf '\nLOCAL_UID=%s\nLOCAL_GID=%s\n' "$(id -u)" "$(id -g)" >> .env
printf '%s\n' 'Created .env and config/client.env. Edit both, use the same MYSQL_PASSWORD and DB_PASSWORD, then run docker compose up --build -d.'
