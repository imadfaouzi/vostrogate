package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

type App struct {
	mu      sync.RWMutex
	db      *sql.DB
	envPath string
	secure  bool
}
type Principal struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Username    string   `json:"username"`
	Role        string   `json:"role"`
	Owner       bool     `json:"owner"`
	Permissions []string `json:"permissions"`
	CSRF        string   `json:"csrf,omitempty"`
}
type envelope map[string]any

func New() (*App, error) {
	path := os.Getenv("VOSTROGATE_ENV_PATH")
	if path == "" {
		path = "/config/client.env"
	}
	a := &App{envPath: path, secure: strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")}
	if db, err := openConfigured(path); err == nil {
		a.db = db
	} else {
		log.Printf("setup mode; database not loaded: %v", err)
	}
	return a, nil
}
func readEnv(path string) (map[string]string, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("env path must be absolute on the server")
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, errors.New("environment file not accessible")
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, errors.New("cannot inspect environment file")
	}
	if stat.Size() > 65536 {
		return nil, errors.New("env file too large")
	}
	if stat.Mode().Perm()&0002 != 0 {
		return nil, errors.New("env file must not be world-writable")
	}
	b, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return nil, errors.New("cannot read environment file")
	}
	vars := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		pair := strings.SplitN(line, "=", 2)
		if len(pair) != 2 {
			continue
		}
		k := strings.TrimSpace(pair[0])
		v := strings.TrimSpace(pair[1])
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		vars[k] = v
	}
	return vars, nil
}
func openConfigured(path string) (*sql.DB, error) {
	vars, err := readEnv(path)
	if err != nil {
		return nil, err
	}
	for _, k := range []string{"DB_HOST", "DB_NAME", "DB_USER", "DB_PASSWORD"} {
		if vars[k] == "" {
			return nil, fmt.Errorf("required environment variable missing: %s", k)
		}
	}
	port := vars["DB_PORT"]
	if port == "" {
		port = "3306"
	}
	cfg := mysql.NewConfig()
	cfg.User = vars["DB_USER"]
	cfg.Passwd = vars["DB_PASSWORD"]
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(vars["DB_HOST"], port)
	cfg.DBName = vars["DB_NAME"]
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Collation = "utf8mb4_unicode_ci"
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 8 * time.Second
	cfg.WriteTimeout = 8 * time.Second
	if strings.EqualFold(vars["DB_TLS"], "true") {
		cfg.TLSConfig = "true"
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, errors.New("database connection setup failed")
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(8)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.New("database unreachable; check env credentials and network")
	}
	return db, nil
}
func (a *App) database() (*sql.DB, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.db == nil {
		return nil, errors.New("database not configured")
	}
	return a.db, nil
}
func (a *App) isInstalled() bool {
	db, err := a.database()
	if err != nil {
		return false
	}
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM company WHERE id=1").Scan(&count)
	return err == nil && count == 1
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, status int, s string) { write(w, status, envelope{"error": s}) }
func parseJSON(w http.ResponseWriter, r *http.Request, d any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(d); err != nil {
		failure(w, 400, "Invalid JSON request: "+err.Error())
		return false
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		failure(w, 400, "Only one JSON object allowed")
		return false
	}
	return true
}
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func sha(s string) [32]byte { return sha256.Sum256([]byte(s)) }
func validateSetupToken(r *http.Request) bool {
	expected := os.Getenv("VOSTROGATE_SETUP_TOKEN")
	supplied := r.Header.Get("X-Setup-Token")
	return len(expected) >= 24 && subtle.ConstantTimeCompare([]byte(expected), []byte(supplied)) == 1
}
func idFrom(s string) int64 { n, _ := strconv.ParseInt(s, 10, 64); return n }
func audit(tx *sql.Tx, actor int64, action, entity string, id int64, meta any) {
	b, _ := json.Marshal(meta)
	_, err := tx.Exec("INSERT INTO audit_log(actor_id,action,entity_type,entity_id,details_json) VALUES(?,?,?,?,?)", actor, action, entity, id, string(b))
	if err != nil {
		log.Printf("audit write failed: %v", err)
	}
}
func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", a.api)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { write(w, 200, envelope{"status": "ok"}) })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			// Defense in depth against cross-origin writes, even for public setup/login.
			if o := r.Header.Get("Origin"); o != "" {
				host := r.Host
				proto := "http"
				if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
					proto = "https"
				}
				allowed := proto + "://" + host
				if o != allowed && !(strings.EqualFold(os.Getenv("APP_ENV"), "development") && (o == "http://localhost:5173" || o == "http://127.0.0.1:5173")) {
					failure(w, 403, "Cross-origin request refused")
					return
				}
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func (a *App) requireAuth(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	db, err := a.database()
	if err != nil {
		failure(w, 503, "Database unavailable")
		return Principal{}, false
	}
	c, err := r.Cookie("vg_session")
	if err != nil {
		failure(w, 401, "Login required")
		return Principal{}, false
	}
	hash := sha(c.Value)
	var p Principal
	var storedCsrf []byte
	var isActive bool
	err = db.QueryRow(`SELECT u.id,u.full_name,u.username,ro.code,u.is_owner_admin,u.is_active,s.csrf_hash
 FROM sessions s JOIN users u ON u.id=s.user_id JOIN roles ro ON ro.id=u.role_id
 WHERE s.token_hash=? AND s.expires_at>UTC_TIMESTAMP(6)`, hash[:]).Scan(&p.ID, &p.Name, &p.Username, &p.Role, &p.Owner, &isActive, &storedCsrf)
	if err != nil || !isActive {
		failure(w, 401, "Session expired")
		return Principal{}, false
	}
	if p.Owner {
		p.Permissions = []string{"*"}
	} else {
		rows, err := db.Query(`SELECT permission_code FROM role_permissions rp JOIN users u ON u.role_id=rp.role_id WHERE u.id=?`, p.ID)
		if err != nil {
			failure(w, 500, "Could not read permissions")
			return Principal{}, false
		}
		defer rows.Close()
		p.Permissions = []string{}
		for rows.Next() {
			var code string
			if rows.Scan(&code) == nil {
				p.Permissions = append(p.Permissions, code)
			}
		}
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		token := r.Header.Get("X-CSRF-Token")
		tokenHash := sha(token)
		if token == "" || subtle.ConstantTimeCompare(storedCsrf, tokenHash[:]) != 1 {
			failure(w, 403, "Invalid CSRF token")
			return Principal{}, false
		}
	}
	return p, true
}
func can(p Principal, permission string) bool {
	if p.Owner {
		return true
	}
	for _, v := range p.Permissions {
		if v == permission {
			return true
		}
	}
	return false
}
func allowed(w http.ResponseWriter, p Principal, permission string) bool {
	if can(p, permission) {
		return true
	}
	failure(w, 403, "Permission denied: "+permission)
	return false
}
