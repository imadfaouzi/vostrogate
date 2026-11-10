package app

import (
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var loginAttempts = struct {
	sync.Mutex
	m map[string][]time.Time
}{m: map[string][]time.Time{}}

func tooMany(ip string) bool {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	now := time.Now()
	arr := loginAttempts.m[ip]
	var keep []time.Time
	for _, t := range arr {
		if now.Sub(t) < 10*time.Minute {
			keep = append(keep, t)
		}
	}
	if len(keep) >= 12 {
		loginAttempts.m[ip] = keep
		return true
	}
	loginAttempts.m[ip] = append(keep, now)
	return false
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if !a.isInstalled() {
		failure(w, 403, "Finish first-run setup")
		return
	}
	ip := strings.Split(r.RemoteAddr, ":")[0]
	if tooMany(ip) {
		failure(w, 429, "Too many login attempts")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !parseJSON(w, r, &req) {
		return
	}
	db, _ := a.database()
	var id int64
	var hash string
	var active bool
	err := db.QueryRow("SELECT id,password_hash,is_active FROM users WHERE username=? OR email=? LIMIT 1", req.Username, req.Username).Scan(&id, &hash, &active)
	if err != nil || !active || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		failure(w, 401, "Invalid credentials")
		return
	}
	token, err := randomToken()
	if err != nil {
		failure(w, 500, "Session initialization error")
		return
	}
	csrf, err := randomToken()
	if err != nil {
		failure(w, 500, "Security initialization error")
		return
	}
	tokenHash := sha(token)
	csrfHash := sha(csrf)
	_, err = db.Exec("INSERT INTO sessions(token_hash,user_id,csrf_hash,expires_at) VALUES(?,?,?,UTC_TIMESTAMP(6)+INTERVAL 12 HOUR)", tokenHash[:], id, csrfHash[:])
	if err != nil {
		failure(w, 500, "Session save failed")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "vg_session", Value: token, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600})
	write(w, 200, envelope{"authenticated": true, "csrfToken": csrf})
}
func (a *App) me(w http.ResponseWriter, r *http.Request, p Principal) {
	token, err := randomToken()
	if err != nil {
		failure(w, 500, "Could not rotate CSRF token")
		return
	}
	c, _ := r.Cookie("vg_session")
	sessionHash := sha(c.Value)
	csrfHash := sha(token)
	db, _ := a.database()
	if _, err = db.Exec("UPDATE sessions SET csrf_hash=? WHERE token_hash=?", csrfHash[:], sessionHash[:]); err != nil {
		failure(w, 500, "Could not refresh security token")
		return
	}
	p.CSRF = token
	write(w, 200, p)
}
func (a *App) logout(w http.ResponseWriter, r *http.Request, p Principal) {
	c, _ := r.Cookie("vg_session")
	h := sha(c.Value)
	db, _ := a.database()
	_, _ = db.Exec("DELETE FROM sessions WHERE token_hash=?", h[:])
	http.SetCookie(w, &http.Cookie{Name: "vg_session", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.secure})
	write(w, 200, envelope{"ok": true})
}
func queryAdminCount(tx *sql.Tx) (int, error) {
	var n int
	err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE is_owner_admin=1 AND is_active=1").Scan(&n)
	return n, err
}

func (a *App) changePassword(w http.ResponseWriter, r *http.Request, p Principal) {
	var req struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if !parseJSON(w, r, &req) {
		return
	}
	if len(req.Next) < 12 || len(req.Next) > 128 {
		failure(w, 422, "New password must be between 12 and 128 characters")
		return
	}
	db, _ := a.database()
	var hash string
	if err := db.QueryRow("SELECT password_hash FROM users WHERE id=?", p.ID).Scan(&hash); err != nil {
		failure(w, 404, "User not found")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Current)) != nil {
		failure(w, 403, "Current password is incorrect")
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.Next), bcrypt.DefaultCost)
	if err != nil {
		failure(w, 500, "Password hashing failed")
		return
	}
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec("UPDATE users SET password_hash=? WHERE id=?", string(newHash), p.ID); err != nil {
		failure(w, 500, "Cannot update password")
		return
	}
	if _, err = tx.Exec("DELETE FROM sessions WHERE user_id=?", p.ID); err != nil {
		failure(w, 500, "Session revocation failed")
		return
	}
	audit(tx, p.ID, "password.change", "user", p.ID, nil)
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Could not save password change")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "vg_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode})
	write(w, 200, envelope{"ok": true, "relogin": true})
}
