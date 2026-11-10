# Build verification report

Date: 2026-11-10

## Checks executed

- PASS: `go test ./...` with real Go dependencies.
- PASS: clean `npm ci --no-audit --no-fund` and `npm run build`.
- PASS: `docker compose config --quiet`.
- PASS: `docker compose build` for API and web images (existing backend build cache reused).
- PASS: configuration bootstrap in a fresh temporary directory, mode 600 on both secret files, host UID/GID generation, and refusal to overwrite existing configuration.
- PASS: staged files exclude local secrets, overrides and generated dependencies; scanned against local password/token values and private-key markers.

The repository includes `backend/go.sum` and `frontend/package-lock.json`; the frontend Docker build uses `npm ci`.

Full live MySQL migration, installation, browser end-to-end workflows, backup/restore and business acceptance tests were not run in this verification. Complete `docs/QA_CHECKLIST.md` before production use.
