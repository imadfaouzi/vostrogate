let csrfToken = '';
export const setCSRF = (token) => { csrfToken = token || ''; };
export class APIError extends Error { constructor(message, status) { super(message); this.status = status; } }
export async function api(path, { method = 'GET', body, signal, setupToken, ...rest } = {}) {
  const headers = { ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}), ...rest.headers };
  if (method !== 'GET' && method !== 'HEAD' && csrfToken) headers['X-CSRF-Token'] = csrfToken;
  if (setupToken) headers['X-Setup-Token'] = setupToken;
  const response = await fetch('/api/' + path.replace(/^\/+/, ''), {
    method, headers, credentials: 'same-origin', signal,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const value = await response.json().catch(() => ({}));
  if (!response.ok) throw new APIError(value.error || `Erreur HTTP ${response.status}`, response.status);
  return value;
}
export const qs = (params) => new URLSearchParams(Object.entries(params).filter(([,v]) => v !== undefined && v !== null && String(v) !== '')).toString();
export function money(cents = 0) { return new Intl.NumberFormat('fr-MA',{style:'currency', currency:'MAD', minimumFractionDigits:2}).format(Number(cents)/100); }
export function quantity(milli = 0) { return new Intl.NumberFormat('fr-FR',{maximumFractionDigits:3}).format(Number(milli)/1000); }
export function parseMoney(value) { const num = Number(String(value).replace(/\s/g,'').replace(',','.')); return Number.isFinite(num) ? Math.round(num*100) : 0; }
export function parseQty(value) { const num = Number(String(value).replace(/\s/g,'').replace(',','.')); return Number.isFinite(num) ? Math.round(num*1000) : 0; }
export function formatDate(value) { if (!value) return '—'; return String(value).slice(0,10).split('-').reverse().join('/'); }
export function calculateLine(line) {
  const gross = Math.round((Number(line.qtyMilli)||0)*(Number(line.unitPriceCents)||0)/1000);
  const sub = Math.round(gross * (10000-(Number(line.discountBps)||0))/10000);
  const vat = Math.round(sub * (Number(line.vatBps)||0)/10000);
  return { subtotalCents:sub, vatCents:vat, totalCents:sub+vat };
}
export function getError(e) { return e?.message || 'Une erreur est survenue'; }
