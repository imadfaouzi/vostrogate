# Restore procedure — one company

## Backup set

**Required:** SQL dump of all tables + secure separately preserved deployment `.env` and `config/client.env`. Only MySQL holds company identity, logo, users, passwords, settings, products, stock and document history. Database connection secrets must remain in a protected server file, not MySQL.

## Dry run

1. Create an **isolated** recovery server/stack, with the same or a later compatible Vostrogate schema version.
2. Create matching database/user credentials and application env files; confirm `DB_*` and `MYSQL_*` match. Never put these on GitHub.
3. Start MySQL only, then import the dump (e.g. `docker compose up -d mysql`; use `scripts/restore-db.sh path/to/dump.sql.gz`).
4. Start API/frontend containers with restored settings; migration runner should recognize the existing schema.
5. Visit login directly. **Installation wizard must not reappear** if the restored company row exists and DB is reachable.
6. Verify company logo, customer balances, stock quantities, recent invoices and authorized role access.
7. Check at least one document preview, one financial report and a sample article search.
8. Record the verification and recovery duration. Keep backups encrypted and regularly rehearse this process.

**Never** restore into a live production database without maintenance, a verified pre-restore backup, and understanding of the overwrite. For production, implement point-in-time recovery and tested retention.
