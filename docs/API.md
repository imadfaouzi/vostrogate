# REST API — route reference

**Base path**: `/api`. JSON request/response. Browser session: HttpOnly `vg_session` cookie; authenticated writes require `X-CSRF-Token` returned by login/`auth/me`.

| Method | Route | Notes / permission |
|---|---|---|
| GET | `/setup/status` | Public, installed flag |
| POST | `/setup/verify` | Setup token header, `{envPath}`; checks file + DB |
| POST | `/setup/complete` | Setup token, `{envPath,company,warehouse,administrator}`; single-use |
| POST | `/auth/login` | username,password |
| GET | `/auth/me` | Principal permissions/CSRF |
| POST | `/auth/logout` | Session destruction |
| POST | `/auth/change-password` | Authenticated user password update |
| GET/PUT | `/company` | Company identity (settings.*) |
| GET/PUT | `/company/logo` | Binary logo on GET; base64 image on PUT |
| GET/PUT | `/settings` | JSON document numbering and options |
| GET/POST | `/warehouses` | Warehouse list/create |
| GET/POST | `/parties?kind=CUSTOMER|SUPPLIER` | Search/list/create customer/supplier |
| GET/PUT | `/parties/{id}` | Partner detail/edit |
| GET | `/parties/{id}/statement` | Customer/supplier account statement |
| GET/POST | `/products` | List/create article |
| GET | `/products/search?q=...` | Debounced barcode/name/reference autocomplete |
| GET/PUT | `/products/{id}` | Detail/edit article |
| GET/POST | `/catalog` | Categories, brands, units, pricing groups, expense categories |
| GET | `/stock` | Stock balances |
| POST | `/stock/adjust` | Atomic count/adjustment |
| GET | `/stock/movements` | Ledger history |
| GET/POST | `/documents?type=...` | List/create commercial document |
| GET | `/documents/{id}` | Read including lines |
| POST | `/documents/{id}/finalize` | Draft → Final; post stock if applicable |
| POST | `/documents/{id}/convert` | `{targetType}` from accepted final source |
| POST | `/documents/{id}/cancel` | Apply cancellation rules |
| POST | `/pos/sale` | Atomic POS invoice + payment + stock |
| GET/POST | `/payments` | List/create payment |
| POST | `/payments/{id}/status` | Clear/reject pending cheque |
| GET/POST | `/expenses` | List/create expenses |
| GET | `/reports?kind=sales|purchases|stock|expenses|receivables` | Financial extracts |
| GET/POST | `/users` | List/create employees |
| PUT | `/users/{id}` | Change account/role/status |
| GET/POST | `/roles` | Role list + permission catalog / create custom |
| PUT | `/roles/{id}` | Update custom role |
| GET | `/audit` | Audit trail |

Examples:

```http
GET /api/products/search?q=EAN-123456
Cookie: vg_session=<http-only cookie managed by browser>
```

```json
{
  "items": [
    {"id": 42, "sku": "ART0042", "barcode": "6111234567890", "name": "Article exemple", "salePriceCents": 19900, "stockMilli": 125000}
  ]
}
```

The JSON sample is **illustrative**, not a fixture. See Go types and React `api(...)` requests for exact response formats. Common errors are HTTP 400 malformed JSON, 401 unauthenticated, 403 permission/CSRF, 409 conflict and 422 domain validation. Untrusted client-provided sums are not used as authoritative totals.
