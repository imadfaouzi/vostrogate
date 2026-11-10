package app

import (
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

func (a *App) usersList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query("SELECT u.id,u.full_name,u.username,u.email,u.role_id,ro.code,u.is_active,u.is_owner_admin,u.created_at FROM users u JOIN roles ro ON ro.id=u.role_id ORDER BY u.id")
	if err != nil {
		failure(w, 500, "Users unavailable")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id, roleID int64
		var name, username, email, role, date string
		var active, owner bool
		if rows.Scan(&id, &name, &username, &email, &roleID, &role, &active, &owner, &date) == nil {
			items = append(items, envelope{"id": id, "name": name, "username": username, "email": email, "roleId": roleID, "role": role, "isActive": active, "isOwnerAdmin": owner, "createdAt": date})
		}
	}
	write(w, 200, envelope{"items": items})
}
func (a *App) rolesList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query(`SELECT r.id,r.code,r.label,r.is_system,COALESCE(GROUP_CONCAT(rp.permission_code ORDER BY rp.permission_code SEPARATOR ','),'') FROM roles r LEFT JOIN role_permissions rp ON rp.role_id=r.id GROUP BY r.id ORDER BY r.id`)
	if err != nil {
		failure(w, 500, "Roles unavailable")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id int64
		var code, label, perm string
		var system bool
		if rows.Scan(&id, &code, &label, &system, &perm) == nil {
			perms := []string{}
			if perm != "" {
				perms = strings.Split(perm, ",")
			}
			items = append(items, envelope{"id": id, "code": code, "label": label, "isSystem": system, "permissions": perms})
		}
	}
	write(w, 200, envelope{"items": items, "availablePermissions": permissionCodes})
}
func (a *App) userCreate(w http.ResponseWriter, r *http.Request, p Principal) {
	var x struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		RoleID   int64  `json:"roleId"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if x.Name == "" || len(x.Username) < 3 || len(x.Password) < 12 || len(x.Password) > 128 || x.RoleID <= 0 {
		failure(w, 422, "Name, username (3+), password (12+) and role required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(x.Password), bcrypt.DefaultCost)
	if err != nil {
		failure(w, 500, "Cannot hash password")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO users(full_name,username,email,password_hash,role_id,is_active,is_owner_admin) VALUES(?,?,?,?,?,1,0)`, x.Name, x.Username, x.Email, string(hash), x.RoleID)
	if err != nil {
		failure(w, 409, "Username/email already exists or role unknown")
		return
	}
	id, _ := res.LastInsertId()
	audit(tx, p.ID, "user.create", "user", id, envelope{"roleId": x.RoleID})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "User transaction failed")
		return
	}
	write(w, 201, envelope{"id": id})
}
func (a *App) userUpdate(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	var x struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		RoleID      int64  `json:"roleId"`
		IsActive    bool   `json:"isActive"`
		NewPassword string `json:"newPassword"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if x.Name == "" || x.RoleID <= 0 {
		failure(w, 422, "Name and role required")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	var owner bool
	if err = tx.QueryRow("SELECT is_owner_admin FROM users WHERE id=? FOR UPDATE", id).Scan(&owner); err != nil {
		failure(w, 404, "User not found")
		return
	}
	if owner {
		failure(w, 403, "The original owner administrator cannot be altered by this endpoint")
		return
	}
	if id == p.ID && !x.IsActive {
		failure(w, 403, "You cannot deactivate your current account")
		return
	}
	_, err = tx.Exec("UPDATE users SET full_name=?,email=?,role_id=?,is_active=? WHERE id=?", x.Name, x.Email, x.RoleID, x.IsActive, id)
	if err != nil {
		failure(w, 409, "Email/role invalid")
		return
	}
	if x.NewPassword != "" {
		if len(x.NewPassword) < 12 || len(x.NewPassword) > 128 {
			failure(w, 422, "New password must have 12–128 characters")
			return
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(x.NewPassword), bcrypt.DefaultCost)
		if e != nil {
			failure(w, 500, "Password hash failed")
			return
		}
		if _, e = tx.Exec("UPDATE users SET password_hash=? WHERE id=?", string(hash), id); e != nil {
			failure(w, 500, "Password save failed")
			return
		}
		_, _ = tx.Exec("DELETE FROM sessions WHERE user_id=?", id)
	}
	audit(tx, p.ID, "user.update", "user", id, envelope{"active": x.IsActive, "roleId": x.RoleID})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "User update failed")
		return
	}
	write(w, 200, envelope{"ok": true})
}
func (a *App) roleCreate(w http.ResponseWriter, r *http.Request, p Principal) {
	var x struct {
		Code        string   `json:"code"`
		Label       string   `json:"label"`
		Permissions []string `json:"permissions"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if len(x.Code) < 2 || len(x.Code) > 50 || x.Label == "" {
		failure(w, 422, "Role code and label required")
		return
	}
	allowedPerms := map[string]bool{}
	for _, perm := range permissionCodes {
		allowedPerms[perm] = true
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec("INSERT INTO roles(code,label,is_system) VALUES(?,?,0)", strings.ToUpper(x.Code), x.Label)
	if err != nil {
		failure(w, 409, "Role code exists")
		return
	}
	id, _ := res.LastInsertId()
	for _, perm := range x.Permissions {
		if !allowedPerms[perm] {
			failure(w, 422, "Invalid permission: "+perm)
			return
		}
		if _, err = tx.Exec("INSERT IGNORE INTO role_permissions(role_id,permission_code) VALUES(?,?)", id, perm); err != nil {
			failure(w, 500, "Role permission failed")
			return
		}
	}
	audit(tx, p.ID, "role.create", "role", id, x)
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Role save failed")
		return
	}
	write(w, 201, envelope{"id": id})
}

func (a *App) roleUpdate(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	var x struct {
		Label       string   `json:"label"`
		Permissions []string `json:"permissions"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if strings.TrimSpace(x.Label) == "" {
		failure(w, 422, "Role label required")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	var system bool
	if err = tx.QueryRow("SELECT is_system FROM roles WHERE id=? FOR UPDATE", id).Scan(&system); err != nil {
		failure(w, 404, "Role not found")
		return
	}
	if system {
		failure(w, 403, "System roles are read-only; create a custom role instead")
		return
	}
	valid := map[string]bool{}
	for _, perm := range permissionCodes {
		valid[perm] = true
	}
	if _, err = tx.Exec("UPDATE roles SET label=? WHERE id=?", x.Label, id); err != nil {
		failure(w, 500, "Cannot change role label")
		return
	}
	if _, err = tx.Exec("DELETE FROM role_permissions WHERE role_id=?", id); err != nil {
		failure(w, 500, "Cannot update role permissions")
		return
	}
	for _, permission := range x.Permissions {
		if !valid[permission] {
			failure(w, 422, "Invalid permission: "+permission)
			return
		}
		if _, err = tx.Exec("INSERT IGNORE INTO role_permissions(role_id,permission_code) VALUES(?,?)", id, permission); err != nil {
			failure(w, 500, "Cannot save permissions")
			return
		}
	}
	audit(tx, p.ID, "role.update", "role", id, x)
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Role update failed")
		return
	}
	write(w, 200, envelope{"ok": true})
}
