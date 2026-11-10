package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type partyInput struct {
	Kind             string `json:"kind"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	ICE              string `json:"ice"`
	TaxID            string `json:"taxId"`
	Address          string `json:"address"`
	City             string `json:"city"`
	CreditLimitCents int64  `json:"creditLimitCents"`
	IsActive         bool   `json:"isActive"`
}

func scanParty(rows *sql.Rows) (envelope, error) {
	var id int64
	var x partyInput
	err := rows.Scan(&id, &x.Kind, &x.Code, &x.Name, &x.Phone, &x.Email, &x.ICE, &x.TaxID, &x.Address, &x.City, &x.CreditLimitCents, &x.IsActive)
	if err != nil {
		return nil, err
	}
	return envelope{"id": id, "kind": x.Kind, "code": x.Code, "name": x.Name, "phone": x.Phone, "email": x.Email, "ice": x.ICE, "taxId": x.TaxID, "address": x.Address, "city": x.City, "creditLimitCents": x.CreditLimitCents, "isActive": x.IsActive}, nil
}

const partyCols = `id,kind,code,name,phone,email,ice,tax_id,address,city,credit_limit_cents,is_active`

func (a *App) partyList(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind != "SUPPLIER" {
		kind = "CUSTOMER"
	}
	limit := queryLimit(r, 50, 200)
	offset := queryOffset(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	pat := "%" + q + "%"
	db, _ := a.database()
	rows, err := db.Query("SELECT "+partyCols+" FROM parties WHERE kind=? AND (name LIKE ? OR code LIKE ? OR phone LIKE ? OR ice LIKE ?) ORDER BY name,id LIMIT ? OFFSET ?", kind, pat, pat, pat, pat, limit, offset)
	if err != nil {
		failure(w, 500, "Unable to load contacts")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		v, e := scanParty(rows)
		if e == nil {
			items = append(items, v)
		}
	}
	var total int
	_ = db.QueryRow("SELECT COUNT(*) FROM parties WHERE kind=? AND (name LIKE ? OR code LIKE ? OR phone LIKE ? OR ice LIKE ?)", kind, pat, pat, pat, pat).Scan(&total)
	write(w, 200, envelope{"items": items, "total": total, "limit": limit, "offset": offset})
}
func (a *App) partyGet(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	db, _ := a.database()
	rows, err := db.Query("SELECT "+partyCols+" FROM parties WHERE id=?", id)
	if err != nil {
		failure(w, 500, "Lookup failed")
		return
	}
	defer rows.Close()
	if !rows.Next() {
		failure(w, 404, "Contact not found")
		return
	}
	item, err := scanParty(rows)
	if err != nil {
		failure(w, 500, "Data error")
		return
	}
	if item["kind"] == "SUPPLIER" {
		if !allowed(w, p, "suppliers.read") {
			return
		}
	} else if !allowed(w, p, "customers.read") {
		return
	}
	write(w, 200, item)
}
func (a *App) partySave(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	var x partyInput
	if !parseJSON(w, r, &x) {
		return
	}
	if x.Kind != "CUSTOMER" && x.Kind != "SUPPLIER" {
		failure(w, 422, "Invalid contact type")
		return
	}
	requiredPerm := "customers.write"
	if x.Kind == "SUPPLIER" {
		requiredPerm = "suppliers.write"
	}
	if !allowed(w, p, requiredPerm) {
		return
	}
	if strings.TrimSpace(x.Name) == "" || strings.TrimSpace(x.Code) == "" || len(x.Name) > 220 || len(x.Code) > 50 {
		failure(w, 422, "Name and code required")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	if id == 0 {
		result, e := tx.Exec(`INSERT INTO parties(kind,code,name,phone,email,ice,tax_id,address,city,credit_limit_cents,is_active) VALUES(?,?,?,?,?,?,?,?,?,?,1)`, x.Kind, x.Code, x.Name, x.Phone, x.Email, x.ICE, x.TaxID, x.Address, x.City, x.CreditLimitCents)
		err = e
		if err == nil {
			id, _ = result.LastInsertId()
		}
	} else {
		var oldKind string
		if tx.QueryRow("SELECT kind FROM parties WHERE id=?", id).Scan(&oldKind) != nil || oldKind != x.Kind {
			failure(w, 404, "Contact not found in this module")
			return
		}
		_, err = tx.Exec(`UPDATE parties SET code=?,name=?,phone=?,email=?,ice=?,tax_id=?,address=?,city=?,credit_limit_cents=?,is_active=? WHERE id=?`, x.Code, x.Name, x.Phone, x.Email, x.ICE, x.TaxID, x.Address, x.City, x.CreditLimitCents, x.IsActive, id)
	}
	if err != nil {
		failure(w, 409, "Contact reference already used or invalid")
		return
	}
	audit(tx, p.ID, "party.save", x.Kind, id, map[string]any{"code": x.Code, "name": x.Name})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Save failed")
		return
	}
	write(w, 200, envelope{"id": id})
}
func (a *App) partyStatement(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	db, _ := a.database()
	var kind, name, code string
	if err := db.QueryRow("SELECT kind,name,code FROM parties WHERE id=?", id).Scan(&kind, &name, &code); err != nil {
		failure(w, 404, "Contact not found")
		return
	}
	if kind == "SUPPLIER" {
		if !allowed(w, p, "suppliers.read") {
			return
		}
	} else if !allowed(w, p, "customers.read") {
		return
	}
	var invoiced, paid int64
	types := "('DELIVERY','INVOICE')"
	if kind == "SUPPLIER" {
		types = "('PURCHASE_INVOICE')"
	}
	qry := fmt.Sprintf(`SELECT COALESCE(SUM(d.total_cents),0),COALESCE(SUM(d.paid_cents),0) FROM documents d
 WHERE d.party_id=? AND d.status='FINAL' AND d.doc_type IN %s
 AND (d.doc_type<>'DELIVERY' OR NOT EXISTS (SELECT 1 FROM documents inv WHERE inv.source_document_id=d.id AND inv.doc_type='INVOICE' AND inv.status='FINAL'))`, types)
	_ = db.QueryRow(qry, id).Scan(&invoiced, &paid)
	rows, err := db.Query(`SELECT id,doc_type,doc_number,status,issued_at,total_cents,paid_cents
 FROM documents WHERE party_id=? ORDER BY id DESC LIMIT 100`, id)
	if err != nil {
		failure(w, 500, "Statement cannot be loaded")
		return
	}
	defer rows.Close()
	docs := []any{}
	for rows.Next() {
		var d int64
		var typ, no, status string
		var when string
		var total, paidDoc int64
		if rows.Scan(&d, &typ, &no, &status, &when, &total, &paidDoc) == nil {
			docs = append(docs, envelope{"id": d, "type": typ, "number": no, "status": status, "date": when, "totalCents": total, "paidCents": paidDoc})
		}
	}
	var credit int64
	if kind == "CUSTOMER" {
		_ = db.QueryRow("SELECT COALESCE(SUM(total_cents),0) FROM documents WHERE party_id=? AND doc_type='CREDIT' AND status='FINAL'", id).Scan(&credit)
	}
	write(w, 200, envelope{"id": id, "kind": kind, "name": name, "code": code, "invoicedCents": invoiced, "paidCents": paid, "creditCents": credit, "balanceCents": invoiced - paid - credit, "documents": docs})
}
func queryLimit(r *http.Request, def, max int) int {
	x, e := strconv.Atoi(r.URL.Query().Get("limit"))
	if e != nil || x < 1 {
		return def
	}
	if x > max {
		return max
	}
	return x
}
func queryOffset(r *http.Request) int {
	x, e := strconv.Atoi(r.URL.Query().Get("offset"))
	if e != nil || x < 0 {
		return 0
	}
	return x
}
