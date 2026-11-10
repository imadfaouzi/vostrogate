# Security and access controls

1. **Single company**: database invariant `company(id=1)`; all clients/users/documents share only that instance. To serve another customer, deploy an entirely separate stack/database/config.
2. **Installer**: first-run server token `VOSTROGATE_SETUP_TOKEN` at least 24 chars, constant-time checked in backend, installation rejects if company exists. Do not publish the token.
3. **Credentials**: initial user password minimum 12 chars, bcrypt hashes; cookie-only login sessions with hashed token in MySQL; no authentication secrets in localStorage.
4. **CSRF/origin**: authenticated writes require a separately generated CSRF token. Backend applies same-origin checks; production reverse proxy must preserve correct host/protocol.
5. **RBAC**: backend checks all sensitive operations. UI permission-based hiding is usability only, not a security boundary. First owner has immutable full access.
6. **Secrets**: `config/client.env` is server-only and should have restrictive filesystem permissions. Never commit `.env` files, secrets, production dumps or database credentials to Git.
7. **SQL**: placeholders are used for user-supplied values; MySQL is the authoritative store. Audit logs record key actions.
8. **Transport**: local Compose listens only on `127.0.0.1:8088`. For remote/prod use TLS reverse proxy, `COOKIE_SECURE=true`, trusted forwarded protocol/host, HSTS and access firewall. Default setup is **local development**, not Internet-ready.
9. **Backups**: encrypt/safeguard SQL dumps and env secrets; restrict access to payroll/payment/customer information. Test restore and verify permissions remain correct.
10. **Before production**: add automated dependency scanning, real integration tests, request throttling/login lockout, centralized logging, session cleanup, sensitive-data redaction, security review and penetration testing, production monitoring and alerts.

**Known limitation:** this ZIP has not been independently penetration-tested or run end-to-end against a live MySQL instance here. Use a staging environment and the acceptance checklist before processing real financial data.
