package app

import (
	"net/http"
	"strings"
)

func (a *App) api(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	method := r.Method
	switch path {
	case "setup/status":
		if method == "GET" {
			a.setupStatus(w, r)
			return
		}
	case "setup/verify":
		if method == "POST" {
			a.setupVerify(w, r)
			return
		}
	case "setup/complete":
		if method == "POST" {
			a.setupComplete(w, r)
			return
		}
	case "auth/login":
		if method == "POST" {
			a.login(w, r)
			return
		}
	case "company/logo":
		if method == "GET" {
			a.serveLogo(w, r)
			return
		}
	}
	if !a.isInstalled() {
		failure(w, 503, "Application not installed; open the initial setup wizard")
		return
	}
	p, ok := a.requireAuth(w, r)
	if !ok {
		return
	}
	parts := strings.Split(path, "/")
	switch {
	case path == "auth/me" && method == "GET":
		a.me(w, r, p)
	case path == "auth/logout" && method == "POST":
		a.logout(w, r, p)
	case path == "auth/change-password" && method == "POST":
		a.changePassword(w, r, p)
	case path == "dashboard" && method == "GET":
		if allowed(w, p, "dashboard.read") {
			a.dashboard(w, r)
		}
	case path == "company" && method == "GET":
		a.companyGet(w, r, p)
	case path == "company" && method == "PUT":
		if allowed(w, p, "settings.manage") {
			a.companyUpdate(w, r, p)
		}
	case path == "company/logo" && method == "PUT":
		if allowed(w, p, "settings.manage") {
			a.logoUpdate(w, r, p)
		}
	case path == "settings" && method == "GET":
		if allowed(w, p, "settings.read") {
			a.settingsGet(w, r)
		}
	case path == "settings" && method == "PUT":
		if allowed(w, p, "settings.manage") {
			a.settingsUpdate(w, r, p)
		}
	case path == "warehouses" && method == "GET":
		if allowed(w, p, "inventory.read") {
			a.warehouseList(w, r)
		}
	case path == "warehouses" && method == "POST":
		if allowed(w, p, "settings.manage") {
			a.warehouseCreate(w, r, p)
		}
	case path == "parties" && method == "GET":
		if allowed(w, p, partyPermission(r, "read")) {
			a.partyList(w, r)
		}
	case path == "parties" && method == "POST":
		if allowed(w, p, partyPermission(r, "write")) {
			a.partySave(w, r, p, 0)
		}
	case len(parts) == 2 && parts[0] == "parties" && method == "GET":
		a.partyGet(w, r, p, idFrom(parts[1]))
	case len(parts) == 2 && parts[0] == "parties" && method == "PUT":
		a.partySave(w, r, p, idFrom(parts[1]))
	case len(parts) == 3 && parts[0] == "parties" && parts[2] == "statement" && method == "GET":
		a.partyStatement(w, r, p, idFrom(parts[1]))
	case path == "products" && method == "GET":
		if allowed(w, p, "products.read") {
			a.productList(w, r)
		}
	case path == "products/search" && method == "GET":
		if allowed(w, p, "products.read") {
			a.productSearch(w, r)
		}
	case path == "products" && method == "POST":
		if allowed(w, p, "products.write") {
			a.productSave(w, r, p, 0)
		}
	case len(parts) == 2 && parts[0] == "products" && method == "GET":
		if allowed(w, p, "products.read") {
			a.productGet(w, r, idFrom(parts[1]))
		}
	case len(parts) == 2 && parts[0] == "products" && method == "PUT":
		if allowed(w, p, "products.write") {
			a.productSave(w, r, p, idFrom(parts[1]))
		}
	case path == "catalog" && method == "GET":
		if allowed(w, p, "products.read") {
			a.catalogList(w, r)
		}
	case path == "catalog" && method == "POST":
		if allowed(w, p, "catalog.manage") {
			a.catalogCreate(w, r, p)
		}
	case path == "stock" && method == "GET":
		if allowed(w, p, "inventory.read") {
			a.stockList(w, r)
		}
	case path == "stock/adjust" && method == "POST":
		if allowed(w, p, "inventory.adjust") {
			a.stockAdjust(w, r, p)
		}
	case path == "stock/movements" && method == "GET":
		if allowed(w, p, "inventory.read") {
			a.stockHistory(w, r)
		}
	case path == "documents" && method == "GET":
		if allowed(w, p, documentPermission(r, "read")) {
			a.documentsList(w, r)
		}
	case path == "documents" && method == "POST":
		if allowed(w, p, documentPermission(r, "write")) {
			a.documentCreate(w, r, p)
		}
	case len(parts) == 2 && parts[0] == "documents" && method == "GET":
		a.documentGet(w, r, p, idFrom(parts[1]))
	case len(parts) == 3 && parts[0] == "documents" && parts[2] == "finalize" && method == "POST":
		a.documentFinalize(w, r, p, idFrom(parts[1]))
	case len(parts) == 3 && parts[0] == "documents" && parts[2] == "convert" && method == "POST":
		a.documentConvert(w, r, p, idFrom(parts[1]))
	case len(parts) == 3 && parts[0] == "documents" && parts[2] == "cancel" && method == "POST":
		a.documentCancel(w, r, p, idFrom(parts[1]))
	case path == "pos/sale" && method == "POST":
		if allowed(w, p, "pos.use") {
			a.posSale(w, r, p)
		}
	case path == "payments" && method == "GET":
		if allowed(w, p, "payments.read") {
			a.paymentsList(w, r)
		}
	case path == "payments" && method == "POST":
		if allowed(w, p, "payments.create") {
			a.paymentCreate(w, r, p)
		}
	case len(parts) == 3 && parts[0] == "payments" && parts[2] == "status" && method == "POST":
		if allowed(w, p, "payments.clear") {
			a.paymentStatus(w, r, p, idFrom(parts[1]))
		}
	case path == "expenses" && method == "GET":
		if allowed(w, p, "expenses.read") {
			a.expensesList(w, r)
		}
	case path == "expenses" && method == "POST":
		if allowed(w, p, "expenses.write") {
			a.expenseCreate(w, r, p)
		}
	case path == "reports" && method == "GET":
		if allowed(w, p, "reports.read") {
			a.reports(w, r)
		}
	case path == "users" && method == "GET":
		if allowed(w, p, "users.read") {
			a.usersList(w, r)
		}
	case path == "users" && method == "POST":
		if allowed(w, p, "users.manage") {
			a.userCreate(w, r, p)
		}
	case len(parts) == 2 && parts[0] == "users" && method == "PUT":
		if allowed(w, p, "users.manage") {
			a.userUpdate(w, r, p, idFrom(parts[1]))
		}
	case path == "roles" && method == "GET":
		if allowed(w, p, "users.read") {
			a.rolesList(w, r)
		}
	case path == "roles" && method == "POST":
		if allowed(w, p, "roles.manage") {
			a.roleCreate(w, r, p)
		}
	case len(parts) == 2 && parts[0] == "roles" && method == "PUT":
		if allowed(w, p, "roles.manage") {
			a.roleUpdate(w, r, p, idFrom(parts[1]))
		}
	case path == "audit" && method == "GET":
		if allowed(w, p, "audit.read") {
			a.auditList(w, r)
		}
	default:
		failure(w, 404, "API route not found")
	}
}
func partyPermission(r *http.Request, action string) string {
	if r.URL.Query().Get("kind") == "SUPPLIER" {
		return "suppliers." + action
	}
	return "customers." + action
}
func documentPermission(r *http.Request, action string) string {
	if strings.HasPrefix(r.URL.Query().Get("type"), "PURCHASE_") {
		return "purchases." + action
	}
	return "sales." + action
}
