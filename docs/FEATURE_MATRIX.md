# Functional coverage and scope

Vostrogate 1.0.0 implements a **single-company** independent application inspired by the SNSYSTEM 4.0.0 video and the earlier Vostrogate wireframes. The names of reference features are **observational**, not a claim of compatibility with the proprietary SNSYSTEM code or database.

| UX area | Implemented UI | Go/MySQL operation | Validation note |
|---|---|---|---|
| First-run identity | Company + logo + Moroccan identifiers | Company singleton ID 1 + logo BLOB | No extra enterprise accounts |
| Environment step | Path input and Verify button | Backend reads authorized config file and pings MySQL | Installer token is a separate security gate |
| Owner admin | First user + password | bcrypt, root owner flag and RBAC seed | Protected owner cannot be deactivated |
| Dashboard | Financial KPI cards and daily trend | SQL reporting | Credit notes and tax reports need accountant review |
| Customer management | List/form/situation | MySQL CRUD and balances | Account statement PDF via browser print |
| Supplier management | List/form/situation | MySQL CRUD and supplier history | Multi-currency not implemented |
| Product catalog | Article form/list, category/brand/unit | MySQL CRUD and unique SKU/barcode | Variations/pricing groups are reference entries, not complete pricing engines |
| Article suggestion | Searchable combobox with suggestions, price and stock | MySQL ranked query | Validate performance with 1000+ SKUs |
| Sales quotes/orders | List/editor/detail/finalize/convert | Transactional document engine | Multiple partial deliveries require extra design |
| Delivery notes | Editor, print, pay, invoice convert | Stock debit on finalize; conversion trace | Stock not deducted twice |
| Sales invoices | Editor/detail/print/pay | Transactional document engine | Legal numbering/invoicing needs validation |
| Credit notes/returns | Source-based conversion and stock return | Source-bound documents and stock ledger | Financial credit/refund application needs further design/testing |
| Purchases | Orders/receipts/invoices | Inbound stock receipts and supplier docs | Purchase tax/accounting integration is basic |
| Inventory | Stock by warehouse, manual adjustment, movements | Atomic stock balance and ledger | Inventory count approvals not implemented |
| Payments | Register cash, bank, card or pending cheque | Transactional payment and document balance | Bank reconciliation not integrated |
| Cheques/effects | List, due date, mark cleared/rejected | Pending payments and alerts | No bank API integration |
| POS | Search items and collect | Atomic invoice + settled payment + stock | Receipt printer integrations not implemented |
| Expenses | Add/list + categories | Expense table | No expense edit/delete workflow in this release |
| Reporting | Sales, purchases, stock, receivables and expense list | SQL report endpoints | CSV currently exports loaded rows; not all data at once |
| Users/roles | Create, assign role, active status, custom role | Server-side RBAC and audit | Add role editing UX and granular approvals as follow-up |
| Settings | Company identity, logo, document prefixes | Stored in MySQL | Not all SNSYSTEM print format variants are implemented |
| Authentication | Login/logout/change password | Server sessions + CSRF | MFA/password reset/recovery not implemented |
| Disaster recovery | Scripts and documentation | Database dump/restore | Protect env secrets separately |

## Explicitly not in scope / follow-up

- Multi-company switching, cross-tenant shared database, import previous company's setup.
- Fully compliant national fiscal e-invoicing / certified accounting exports or tax compliance attestations.
- Production-grade payment gateway, bank reconciliation, fiscal signing, email delivery, barcode scanner drivers, cash drawer integrations.
- Distributed job processing, multi-store stock reservations, partial deliveries/returns, approval workflows, serial numbers/batches, payroll.
- Complete automated browser/E2E and integration test coverage in this offline build environment.

## Team acceptance decisions to capture

1. Which documents reduce/replenish stock; what happens on cancellation and partial returns?
2. Are BL payments allowed before invoices? (Current: **yes**, converted payments follow invoice.)
3. Should direct invoices reduce stock? (Current: **yes**.)
4. How are credit notes applied to receivables/refunds in the company's accounting policy?
5. What fiscal invoice numbering formats and printable mentions are mandatory?
6. Which staff permissions may create/finalize/cancel documents and clear cheques?
7. Which exports are required for the client's accountant and Moroccan authorities?
