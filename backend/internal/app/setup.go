package app

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
	dbm "vostrogate/backend/internal/db"
)

type setupCompany struct {
	LegalName     string `json:"legalName"`
	TradeName     string `json:"tradeName"`
	ICE           string `json:"ice"`
	TaxID         string `json:"taxId"`
	RC            string `json:"rc"`
	Patente       string `json:"patente"`
	Address       string `json:"address"`
	City          string `json:"city"`
	Phone         string `json:"phone"`
	Email         string `json:"email"`
	Currency      string `json:"currency"`
	Language      string `json:"language"`
	VATDefaultBps int    `json:"vatDefaultBps"`
	LogoDataURI   string `json:"logoDataUri"`
}
type setupPayload struct {
	EnvPath       string       `json:"envPath"`
	Company       setupCompany `json:"company"`
	Administrator struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	} `json:"administrator"`
	Warehouse string `json:"warehouse"`
}

var permissionCodes = []string{
	"dashboard.read", "customers.read", "customers.write", "suppliers.read", "suppliers.write",
	"products.read", "products.write", "catalog.manage", "sales.read", "sales.write", "sales.finalize", "sales.convert", "sales.cancel",
	"purchases.read", "purchases.write", "purchases.finalize", "purchases.convert", "purchases.cancel", "inventory.read", "inventory.adjust",
	"payments.read", "payments.create", "payments.clear", "cheques.read", "cheques.manage",
	"expenses.read", "expenses.write", "reports.read", "reports.export", "pos.use",
	"settings.read", "settings.manage", "users.read", "users.manage", "roles.manage", "audit.read",
}
var seedRoles = map[string][]string{
	"ADMIN":      permissionCodes,
	"SALES":      {"dashboard.read", "customers.read", "customers.write", "products.read", "sales.read", "sales.write", "sales.finalize", "sales.convert", "inventory.read", "payments.read", "reports.read"},
	"CASHIER":    {"dashboard.read", "customers.read", "products.read", "sales.read", "sales.write", "sales.finalize", "inventory.read", "pos.use", "payments.read", "payments.create"},
	"ACCOUNTANT": {"dashboard.read", "customers.read", "suppliers.read", "sales.read", "purchases.read", "payments.read", "payments.create", "payments.clear", "cheques.read", "cheques.manage", "expenses.read", "expenses.write", "reports.read", "reports.export"},
	"STOCK":      {"dashboard.read", "products.read", "products.write", "catalog.manage", "purchases.read", "purchases.write", "purchases.finalize", "purchases.convert", "purchases.cancel", "inventory.read", "inventory.adjust", "reports.read"},
	"MANAGER":    {"dashboard.read", "customers.read", "suppliers.read", "products.read", "sales.read", "sales.write", "sales.finalize", "sales.convert", "purchases.read", "purchases.write", "purchases.finalize", "purchases.convert", "purchases.cancel", "inventory.read", "payments.read", "cheques.read", "expenses.read", "reports.read", "reports.export"},
}

func insertSeed(tx *sql.Tx) error {
	for _, code := range permissionCodes {
		if _, err := tx.Exec("INSERT IGNORE INTO permissions(code,description) VALUES(?,?)", code, code); err != nil {
			return err
		}
	}
	for name, perms := range seedRoles {
		if _, err := tx.Exec("INSERT IGNORE INTO roles(code,label,is_system) VALUES(?,?,1)", name, name); err != nil {
			return err
		}
		var roleID int64
		if err := tx.QueryRow("SELECT id FROM roles WHERE code=?", name).Scan(&roleID); err != nil {
			return err
		}
		for _, perm := range perms {
			if _, err := tx.Exec("INSERT IGNORE INTO role_permissions(role_id,permission_code) VALUES(?,?)", roleID, perm); err != nil {
				return err
			}
		}
	}
	return nil
}
func logoBytes(data string) ([]byte, string, error) {
	if data == "" {
		return nil, "", nil
	}
	if len(data) > 4<<20 {
		return nil, "", errors.New("logo too large (max 2 MB)")
	}
	parts := strings.SplitN(data, ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:image/") {
		return nil, "", errors.New("logo must be a PNG/JPEG data URL")
	}
	img, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", errors.New("invalid logo encoding")
	}
	if len(img) > 2<<20 {
		return nil, "", errors.New("logo exceeds 2 MB")
	}
	mime := http.DetectContentType(img)
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
		return nil, "", errors.New("only PNG, JPEG and WebP logos allowed")
	}
	return img, mime, nil
}
func (a *App) setupStatus(w http.ResponseWriter, r *http.Request) {
	write(w, 200, envelope{"installed": a.isInstalled(), "app": "Vostrogate", "version": "1.0.0"})
}
func (a *App) setupVerify(w http.ResponseWriter, r *http.Request) {
	if a.isInstalled() {
		failure(w, 409, "Already installed")
		return
	}
	if !validateSetupToken(r) {
		failure(w, 403, "Invalid installer token")
		return
	}
	var req struct {
		EnvPath string `json:"envPath"`
	}
	if !parseJSON(w, r, &req) {
		return
	}
	if filepath.Clean(req.EnvPath) != filepath.Clean(a.envPath) {
		failure(w, 422, "Le chemin doit correspondre à VOSTROGATE_ENV_PATH configuré sur le serveur")
		return
	}
	db, err := openConfigured(req.EnvPath)
	if err != nil {
		failure(w, 422, err.Error())
		return
	}
	defer db.Close()
	write(w, 200, envelope{"verified": true, "database": true, "path": req.EnvPath, "message": "Fichier accessible et connexion MySQL vérifiée; les secrets ne sont jamais envoyés au navigateur."})
}
func (a *App) setupComplete(w http.ResponseWriter, r *http.Request) {
	if a.isInstalled() {
		failure(w, 409, "Application already installed")
		return
	}
	if !validateSetupToken(r) {
		failure(w, 403, "Invalid installer token")
		return
	}
	var req setupPayload
	if !parseJSON(w, r, &req) {
		return
	}
	if filepath.Clean(req.EnvPath) != filepath.Clean(a.envPath) {
		failure(w, 422, "Le chemin doit correspondre à VOSTROGATE_ENV_PATH configuré sur le serveur")
		return
	}
	c := req.Company
	admin := req.Administrator
	if strings.TrimSpace(c.LegalName) == "" || strings.TrimSpace(admin.Username) == "" || len(admin.Password) < 12 || len(admin.Password) > 128 {
		failure(w, 422, "Company name, admin login and password (12–128 characters) required")
		return
	}
	if c.VATDefaultBps < 0 || c.VATDefaultBps > 10000 {
		failure(w, 422, "Invalid VAT rate")
		return
	}
	if req.Warehouse == "" {
		req.Warehouse = "Dépôt principal"
	}
	if c.Currency == "" {
		c.Currency = "MAD"
	}
	if c.Language == "" {
		c.Language = "fr"
	}
	logo, mime, err := logoBytes(c.LogoDataURI)
	if err != nil {
		failure(w, 422, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
	if err != nil {
		failure(w, 500, "Unable to secure administrator account")
		return
	}
	conn, err := openConfigured(req.EnvPath)
	if err != nil {
		failure(w, 422, err.Error())
		return
	}
	if err = dbm.Migrate(conn); err != nil {
		conn.Close()
		failure(w, 500, "Database schema initialization failed: "+err.Error())
		return
	}
	tx, err := conn.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		conn.Close()
		failure(w, 500, "Cannot start installation transaction")
		return
	}
	defer tx.Rollback()
	var existing int
	if err := tx.QueryRow("SELECT COUNT(*) FROM company WHERE id=1").Scan(&existing); err != nil || existing > 0 {
		conn.Close()
		failure(w, 409, "Company already configured")
		return
	}
	if err = insertSeed(tx); err != nil {
		conn.Close()
		failure(w, 500, "Could not create permission templates")
		return
	}
	_, err = tx.Exec(`INSERT INTO company(id,legal_name,trade_name,ice,tax_id,rc,patente,address,city,phone,email,currency,language,vat_default_bps,logo,logo_mime,env_path)
 VALUES(1,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.LegalName, c.TradeName, c.ICE, c.TaxID, c.RC, c.Patente, c.Address, c.City, c.Phone, c.Email, c.Currency, c.Language, c.VATDefaultBps, logo, mime, req.EnvPath)
	if err != nil {
		conn.Close()
		failure(w, 500, "Could not register company")
		return
	}
	_, err = tx.Exec("INSERT INTO warehouses(code,name,is_default) VALUES('MAIN',?,1)", req.Warehouse)
	if err != nil {
		conn.Close()
		failure(w, 500, "Cannot create default warehouse")
		return
	}
	_, err = tx.Exec(`INSERT INTO users(full_name,username,email,password_hash,role_id,is_owner_admin) VALUES(?,?,?,?,(SELECT id FROM roles WHERE code='ADMIN'),1)`,
		defaultString(admin.Name, "Administrateur"), admin.Username, admin.Email, string(hash))
	if err != nil {
		conn.Close()
		failure(w, 500, "Cannot create first admin")
		return
	}
	_, err = tx.Exec("INSERT INTO parties(kind,code,name,address,is_active) VALUES('CUSTOMER','COMPTANT','Au comptant','',1)")
	if err != nil {
		conn.Close()
		failure(w, 500, "Cannot create point-of-sale customer")
		return
	}
	for _, entry := range []struct{ key, val string }{{"allow_negative_stock", "false"}, {"document_prefixes", `{"QUOTE":"DEV","ORDER":"CMD","DELIVERY":"BL","INVOICE":"FAC","CREDIT":"AV","RETURN":"RET","PURCHASE_ORDER":"AC","PURCHASE_RECEIPT":"REC","PURCHASE_INVOICE":"FA"}`}} {
		if _, err = tx.Exec("INSERT INTO settings(setting_key,setting_json) VALUES(?,CAST(? AS JSON))", entry.key, entry.val); err != nil {
			conn.Close()
			failure(w, 500, "Settings initialization failed")
			return
		}
	}
	if err = tx.Commit(); err != nil {
		conn.Close()
		failure(w, 500, "Installation transaction did not commit")
		return
	}
	a.mu.Lock()
	if a.db != nil {
		_ = a.db.Close()
	}
	a.db = conn
	a.envPath = req.EnvPath
	a.mu.Unlock()
	write(w, 201, envelope{"installed": true, "message": "Installation complète; seul le compte administrateur initial possède tous les droits."})
}
func defaultString(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
func installNote() string { return fmt.Sprintf("%d permissions", len(permissionCodes)) }
