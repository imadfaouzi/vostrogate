#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
printf '\n[1/3] Go source and unit tests\n'
(cd backend && go test ./...)
printf '\n[2/3] React production build\n'
(cd frontend && npm ci --no-audit --no-fund && npm run build)
printf '\n[3/3] Docker configuration validation (if Docker present)\n'
if command -v docker >/dev/null 2>&1; then
  docker compose config --quiet
else
  echo 'Docker not installed: skip compose validation'
fi
printf '\nSource verification finished. Run live integration checklist in docs/QA_CHECKLIST.md next.\n'
