package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

func (a *App) companyGet(w http.ResponseWriter, r *http.Request, p Principal) {
	db, _ := a.database()
	var c setupCompany
	var envPath string
	err := db.QueryRow(`SELECT legal_name,trade_name,ice,tax_id,rc,patente,address,city,phone,email,currency,language,vat_default_bps,env_path FROM company WHERE id=1`).Scan(
		&c.LegalName, &c.TradeName, &c.ICE, &c.TaxID, &c.RC, &c.Patente, &c.Address, &c.City, &c.Phone, &c.Email, &c.Currency, &c.Language, &c.VATDefaultBps, &envPath)
	if err != nil {
		failure(w, 500, "Company not found")
		return
	}
	result := envelope{"company": c, "logoUrl": "/api/company/logo"}
	if can(p, "settings.read") {
		result["envPath"] = envPath
	}
	write(w, 200, result)
}
func (a *App) companyUpdate(w http.ResponseWriter, r *http.Request, p Principal) {
	var c setupCompany
	if !parseJSON(w, r, &c) {
		return
	}
	if strings.TrimSpace(c.LegalName) == "" || c.VATDefaultBps < 0 || c.VATDefaultBps > 10000 {
		failure(w, 422, "Invalid company identity or VAT")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(`UPDATE company SET legal_name=?,trade_name=?,ice=?,tax_id=?,rc=?,patente=?,address=?,city=?,phone=?,email=?,currency=?,language=?,vat_default_bps=? WHERE id=1`, c.LegalName, c.TradeName, c.ICE, c.TaxID, c.RC, c.Patente, c.Address, c.City, c.Phone, c.Email, c.Currency, c.Language, c.VATDefaultBps)
	if err != nil {
		failure(w, 500, "Unable to update company")
		return
	}
	audit(tx, p.ID, "company.update", "company", 1, map[string]any{"company": c.LegalName})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Could not commit company settings")
		return
	}
	write(w, 200, envelope{"ok": true})
}
func (a *App) logoUpdate(w http.ResponseWriter, r *http.Request, p Principal) {
	var req struct {
		LogoDataURI string `json:"logoDataUri"`
	}
	if !parseJSON(w, r, &req) {
		return
	}
	data, mime, err := logoBytes(req.LogoDataURI)
	if err != nil {
		failure(w, 422, err.Error())
		return
	}
	db, _ := a.database()
	_, err = db.Exec(`UPDATE company SET logo=?,logo_mime=? WHERE id=1`, data, mime)
	if err != nil {
		failure(w, 500, "Logo save failed")
		return
	}
	write(w, 200, envelope{"ok": true})
}
func (a *App) serveLogo(w http.ResponseWriter, r *http.Request) {
	db, err := a.database()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var b []byte
	var mime sql.NullString
	if err = db.QueryRow("SELECT logo,logo_mime FROM company WHERE id=1").Scan(&b, &mime); err != nil || len(b) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime.String)
	w.Header().Set("Cache-Control", "private,max-age=300")
	_, _ = w.Write(b)
}
func (a *App) settingsGet(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query("SELECT setting_key,setting_json FROM settings ORDER BY setting_key")
	if err != nil {
		failure(w, 500, "Settings query failed")
		return
	}
	defer rows.Close()
	data := map[string]any{}
	for rows.Next() {
		var key string
		var raw []byte
		if rows.Scan(&key, &raw) == nil {
			var v any
			if json.Unmarshal(raw, &v) == nil {
				data[key] = v
			}
		}
	}
	write(w, 200, data)
}
func (a *App) settingsUpdate(w http.ResponseWriter, r *http.Request, p Principal) {
	var req map[string]json.RawMessage
	if !parseJSON(w, r, &req) {
		return
	}
	allowedSettings := map[string]bool{"allow_negative_stock": true, "document_prefixes": true, "company_timezone": true, "default_print_format": true, "notification_days": true}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	for key, val := range req {
		if key == "document_prefixes" {
			var prefixes map[string]string
			if json.Unmarshal(val, &prefixes) != nil {
				failure(w, 422, "Invalid document prefix map")
				return
			}
			pattern := regexp.MustCompile(`^[A-Za-z0-9_-]{1,15}$`)
			for typ, prefix := range prefixes {
				if docTypePrefixes[typ] == "" || !pattern.MatchString(prefix) {
					failure(w, 422, "Invalid document prefix for "+typ)
					return
				}
			}
		}
		if !allowedSettings[key] || !json.Valid(val) {
			failure(w, 422, "Unknown setting: "+key)
			return
		}
		_, err = tx.Exec("INSERT INTO settings(setting_key,setting_json) VALUES(?,CAST(? AS JSON)) ON DUPLICATE KEY UPDATE setting_json=VALUES(setting_json)", key, string(val))
		if err != nil {
			failure(w, 500, "Save setting failed")
			return
		}
	}
	audit(tx, p.ID, "settings.update", "settings", 0, req)
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Settings transaction failed")
		return
	}
	write(w, 200, envelope{"ok": true})
}
func (a *App) warehouseList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query("SELECT id,code,name,is_default,is_active FROM warehouses ORDER BY is_default DESC,name")
	if err != nil {
		failure(w, 500, "Warehouses unavailable")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id int64
		var code, name string
		var def, active bool
		if rows.Scan(&id, &code, &name, &def, &active) == nil {
			items = append(items, envelope{"id": id, "code": code, "name": name, "isDefault": def, "isActive": active})
		}
	}
	write(w, 200, envelope{"items": items})
}
func (a *App) warehouseCreate(w http.ResponseWriter, r *http.Request, p Principal) {
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if !parseJSON(w, r, &req) {
		return
	}
	if req.Code == "" || req.Name == "" {
		failure(w, 422, "Code and name required")
		return
	}
	db, _ := a.database()
	res, err := db.Exec("INSERT INTO warehouses(code,name) VALUES(?,?)", req.Code, req.Name)
	if err != nil {
		failure(w, 409, "Warehouse code already used")
		return
	}
	id, _ := res.LastInsertId()
	write(w, 201, envelope{"id": id})
}
