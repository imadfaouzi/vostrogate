# Vostrogate 1.0.0 — Single-company web ERP

**Stack:** Go REST API, MySQL 8.4/InnoDB, React 19 + Vite, responsive CSS, Nginx, Docker Compose.

This is a **working source-code implementation of the principal commercial-management workflows**, built from the Vostrogate UX prototype and publicly demonstrated SNSYSTEM 4.0.0 concepts. It is **not a certified drop-in copy** of SNSYSTEM and **has not yet passed full live integration/user-acceptance testing**. Before using real accounts or issuing legally binding invoices, complete the acceptance and security checklist in `docs/QA_CHECKLIST.md`.

## 1. Product principles

- **Exactly one company per installation.** No tenant selector, company switcher, or company import. There is one `company` database row with fixed ID 1.
- **First-launch wizard** collects business identity and logo, validates the backend-accessible `.env` file path/MySQL connection, creates the owner administrator, and locks installation afterwards.
- **One first admin with full rights.** The owner cannot be reassigned/deactivated via normal user-management endpoints; other accounts use seeded or custom RBAC roles.
- **MySQL is the source of truth.** Company identity, binary logo, settings, accounts, documents, stock, movements, payments, and audit trail persist in MySQL. Browser `localStorage` contains *unsaved form drafts only*, scoped by user ID.
- **The `.env` file is a server-side secret.** It is not uploaded or exposed to the browser. Back up it separately alongside database backups; a database alone cannot recreate a lost encryption key or database password.
- **Prices are stored in integer centimes, quantities in integer thousandths** to avoid floating-point accounting errors in the backend. Currency defaults to MAD.
- **Transaction boundaries** protect document creation/finalization, stock movement, payment registration, and POS sale. Unique movement keys prevent duplicate stock postings.

## 2. Prerequisites

- Docker Engine/Compose plugin compatible with Compose v2; enough RAM for MySQL and a build (recommend 4 GB+ available).
- A terminal on Linux, macOS or WSL, and a modern browser.
- For source-only local development: Go 1.27+, Node.js 22.12+ or 24+, npm, and MySQL 8.4.

## 3. Fast start — Docker Compose (recommended)

```sh
git clone https://github.com/imadfaouzi/vostrogate.git
cd vostrogate
sh scripts/init-config.sh
```

The script records your host UID/GID so the API can read the protected configuration file. Local secrets and Docker overrides are ignored by Git. Never commit them.

Edit **both** files created by the command:

1. `.env` contains `MYSQL_ROOT_PASSWORD`, `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD`, `VOSTROGATE_SETUP_TOKEN`, and cookie/environment flags.
2. `config/client.env` contains `DB_HOST=mysql`, `DB_PORT=3306`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, and optional `DB_TLS`. Match all database details with `.env` exactly, especially `MYSQL_PASSWORD` = `DB_PASSWORD`.

Generate unique high-entropy secrets, for example:

```sh
openssl rand -hex 32
```

Use a different random value for every password/token; do **not** use the example placeholder values. Then:

```sh
docker compose up --build -d
docker compose ps
```

In your browser on the **same computer**, open `http://localhost:8088`.

1. Enter the `VOSTROGATE_SETUP_TOKEN` from the server's deployment `.env` in the initial installer authorization gate (not a company-setting field).
2. Fill company name, logo, Moroccan company details and main warehouse.
3. **Configuration d'environnement**: only enter **`/config/client.env`** and click **Vérifier le chemin et la connexion MySQL**. Do not paste secrets into the page.
4. Create the first administrator (password 12–128 characters).
5. Confirm and finish; the app switches to the login screen. **There is no import-existing-company path or second company.**

`localhost:8088` is bound to loopback for safety. For access from other computers, configure a trusted HTTPS reverse proxy and update your deployment security. Do not simply expose the development HTTP port to the Internet.

### Recovery when installation is interrupted

Schema migrations are safe to run again, but a partial first installation can leave an initialized schema. If company ID 1 exists, setup is intentionally blocked. Recover using the existing administrator or restore the intended MySQL backup; don't delete company data to bypass setup.

## 4. Start / stop / updates

```sh
docker compose ps
docker compose logs -f api
docker compose logs -f web
docker compose down             # leaves the named mysql_data volume intact
docker compose up -d --build     # rebuild from sources
```

**Never** use `docker compose down -v` on a system containing data unless deliberately destroying the database.

## 5. Backups and recovery

```sh
bash scripts/backup-db.sh
# output: vostrogate_YYYYMMDD_HHMMSS.sql.gz

# Restore only to a separate/recovery installation or after deliberate backup and maintenance:
bash scripts/restore-db.sh vostrogate_YYYYMMDD_HHMMSS.sql.gz
```

Securely back up **deployment `.env`** and **`config/client.env`** separately (encrypted, access-restricted). Business data and company logo are in MySQL; environment credentials are not and must not be. Test actual restore procedures before trusting them. See `docs/RESTORE.md`.

## 6. Functionality delivered

| Module | User-visible functions |
|---|---|
| Installation | One-time company/logo/setup token/path validation/owner admin |
| Auth | Login, logout, password change, HttpOnly session cookies, CSRF |
| Dashboard | Turnover, outstanding amounts, purchases, expenses, recent trend, cheque alerts |
| Clients/Fournisseurs | Search, list, create/edit, situation/relevé and document history |
| Articles | Code/ref/barcode, prices, TVA, category/brand/unit, initial stock, pagination |
| Catalog | Categories, brands, units, price groups and expense categories |
| Sales | Quotes, orders, delivery notes, invoices, credits/returns, finalization/conversion/cancellation |
| Purchases | Purchase orders, goods receipts, supplier invoices |
| Payments/Cheques | Partial/full payments, cheque/effect pending, clearing/rejection, due-date alerts |
| Inventory | Multi-location **warehouses** within the one company, stock ledger, adjustments |
| POS | Single atomic sale: invoice + cash payment + stock movements |
| Expenses | Create/list expense items with optional categories |
| Reports | Sales/purchases, receivables, expenses and stock extracts, CSV and print views |
| Users/roles | Owner-admin; seeded roles; custom roles and permission assignment; audit trail |
| Company/settings | Company identity/logo, numbering prefixes, basic business settings |

**Search experience:** the reusable article autocomplete requests up to 20 matches from the server as the user types, ranks exact barcode and SKU before name prefix/substring matches, supports Arrow Up/Down + Enter/Escape, and displays price and available stock. Barcode scanners that type into a focused input can use it too. This is present in sales editor, POS, and inventory operations.

## 7. Project layout

```text
backend/
  cmd/server/                  Entrypoint
  internal/app/                HTTP endpoints, auth, RBAC, documents, ledger, payments, POS
  internal/db/                 Migration runner and MySQL schema
  Dockerfile
  go.mod
frontend/
  src/main.jsx                React router, setup/login gates
  src/components/             Navigation, fields and autocomplete
  src/pages/                  Business screens
  src/hooks/                  Debounced fetch and user-scoped drafts
  src/styles/                 Responsive design system
  public/brand/               Original Vostrogate SVG brand/icons
  Dockerfile, nginx.conf
config/client.env.example      Server file-based database credentials
scripts/                       Config bootstrap, backup, restore
docs/                          Architecture, API, security and QA
```

`docs/original-wireframes.drawio` contains the editable earlier page mockups.

## 8. Backend/frontend development (without Docker)

1. Start a MySQL 8.4 database and create a database and user with DDL permissions for the initial migration.
2. Create and protect the service environment file, e.g. `/srv/vostrogate/client.env` containing `DB_*` values.
3. From `backend/`, run `go mod download` and `VOSTROGATE_ENV_PATH=/srv/vostrogate/client.env VOSTROGATE_SETUP_TOKEN=... APP_ENV=development go run ./cmd/server`.
4. From `frontend/`, run `npm ci` followed by `npm run dev` and browse `http://localhost:5173`. Vite proxies `/api` to the Go backend (see `vite.config.js`).
5. Do first-install setup against that same environment path. API listens on `:8080` by default (override with `HTTP_ADDR`).

Don't create the first company by direct SQL if you want the default RBAC roles and owner constraints initialized consistently.

## 9. Implementation status / verification

Verified on 2026-11-10 with real dependencies: `go test ./...`, a clean `npm ci` and `npm run build`, and fresh-clone configuration initialization/permissions and Compose validation. Dependency checksums are committed in `backend/go.sum` and `frontend/package-lock.json`. See `docs/BUILD_VERIFICATION.md` for details. Full live MySQL/browser integration and business-workflow acceptance testing remain required before production; see `docs/QA_CHECKLIST.md`.

## 10. Release priorities

Before rollout: execute real integration tests; implement reliable upgrade/versioned DB migrations with snapshot+rollback; enforce TLS and secure cookies behind reverse proxy; configure health monitoring, logging and rate limiting; test real MySQL restore; test role-level access with different users; establish Moroccan fiscal document numbering and retention policies with an accountant; conduct real concurrent stock/payment acceptance tests. Full list: `docs/QA_CHECKLIST.md`.

**Ownership:** your organization may edit and extend this source. Any SNSYSTEM trademarks/screenshots in prior reference materials are not part of Vostrogate product branding.
