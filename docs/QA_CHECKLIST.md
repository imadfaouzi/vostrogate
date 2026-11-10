# Release acceptance — Vostrogate 1.0.0

## Current verification status

- [x] Go source parsed/formatted (`gofmt`).
- [x] Go source offline compile/unit harness with **temporary stubs** for unavailable mysql/bcrypt modules; not a real build.
- [x] React JSX static parse/transpile pass.
- [ ] Resolve/download actual Go/npm dependencies and build production binaries/assets.
- [ ] Boot Docker Compose with live MySQL 8.4 and complete setup in a real browser.
- [ ] Run integration tests and security review on deployment configuration.

## Install / one-company rules

- [ ] Opening a new installation shows token gate and company setup; **no import/multi-company selector**.
- [ ] Wrong setup token fails; correct token allows file path verification.
- [ ] Only `.env` server path + verify are displayed in environment step.
- [ ] Invalid file path / mismatched database password displays useful error without leaking secrets.
- [ ] Finish writes one company row, binary logo, initial warehouse, roles and owner account to MySQL.
- [ ] Refresh/restart retains installation and goes to login; cannot rerun wizard or create a second company.

## Security / RBAC

- [ ] Admin sees all modules and may create roles and users.
- [ ] Sales, cashier, accountant and stockkeeper can access only authorized endpoints even if API URL is typed manually.
- [ ] API denies mutation without valid CSRF, and cross-origin requests are rejected.
- [ ] Cannot deactivate/replace the initial owner admin through users API.
- [ ] Session expiry, logout and password change invalidate/renew sessions as intended.
- [ ] Real HTTPS reverse proxy, secure cookies and CSP work.

## Articles and search

- [ ] Create 2,000+ sample products, some with EAN/UPC barcodes.
- [ ] Search by exact barcode, partial reference and product-name substring.
- [ ] Arrow keys/Enter/Escape, scanner input, no-results, loading and disabled-stock cases work.
- [ ] Search is consistent across sales editor, POS, inventory; price and stock match MySQL.
- [ ] Category, brand and unit CRUD works; barcodes/SKUs reject duplicates.

## Commercial documents / finance

- [ ] New quote → order → delivery note → invoice preserves customer/lines/amounts.
- [ ] Direct invoice reduces stock once, and delivery → invoice does not reduce twice.
- [ ] Purchase order → goods receipt → supplier invoice receives stock once.
- [ ] Draft/Final/Cancel stock reversals and conversions are concurrent-safe.
- [ ] Discounts/TVA work for 0, fractional quantity and rounding boundaries.
- [ ] Partial cash payment updates document and customer outstanding balance.
- [ ] Cheque pending does not count as paid; clearing updates once; rejection frees remaining capacity.
- [ ] Refund, cancellation and credit-note financial treatment meets accountant-approved policy (not fully validated).
- [ ] Concurrent payments/stock adjustments cannot overspend or oversell.
- [ ] POS sale rolls back invoice + payment + stock together on insufficient stock.
- [ ] Print/PDF preview shows restored company branding and legally required fields.

## Durability / restore

- [ ] Saved records survive browser refresh, logout, app restart and different browsers.
- [ ] Unsaved forms recover from user-scoped local drafts; another user doesn't see previous user's draft.
- [ ] Backup then restore MySQL to a clean stack and check logo, roles, customer balances, stock, invoices and cheque state.
- [ ] Scheduled, encrypted backups with retention and disaster recovery are configured.

## Production blockers

- Real integration test pass and bug fixes.
- Localization/fiscal invoice rules validated for the client's Moroccan accounting workflow.
- Central monitoring, rate limiting, logging, backup scheduling, secure TLS deployment and key management.
- Document-level permissions/approvals, financial credit/return policies and error paths tested with real users.
