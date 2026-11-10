package app

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func (a *App) paymentsList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	onlyCheques := r.URL.Query().Get("cheques") == "true"
	qry := `SELECT pay.id,pay.document_id,d.doc_number,pay.party_id,p.name,pay.amount_cents,pay.method,pay.reference,pay.bank,COALESCE(DATE_FORMAT(pay.due_date,'%Y-%m-%d'),''),pay.status,pay.received_at
 FROM payments pay JOIN parties p ON p.id=pay.party_id JOIN documents d ON d.id=pay.document_id`
	if onlyCheques {
		qry += " WHERE pay.method IN ('CHEQUE','EFFECT')"
	}
	qry += " ORDER BY pay.id DESC LIMIT 250"
	rows, err := db.Query(qry)
	if err != nil {
		failure(w, 500, "Payments unavailable")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id, docID, partyID, amount int64
		var doc, party, method, ref, bank, due, status, date string
		if rows.Scan(&id, &docID, &doc, &partyID, &party, &amount, &method, &ref, &bank, &due, &status, &date) == nil {
			items = append(items, envelope{"id": id, "documentId": docID, "documentNumber": doc, "partyId": partyID, "partyName": party, "amountCents": amount, "method": method, "reference": ref, "bank": bank, "dueDate": due, "status": status, "date": date})
		}
	}
	write(w, 200, envelope{"items": items})
}
func (a *App) paymentCreate(w http.ResponseWriter, r *http.Request, p Principal) {
	var x struct {
		DocumentID  int64  `json:"documentId"`
		AmountCents int64  `json:"amountCents"`
		Method      string `json:"method"`
		Reference   string `json:"reference"`
		Bank        string `json:"bank"`
		DueDate     string `json:"dueDate"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if x.DocumentID <= 0 || x.AmountCents <= 0 {
		failure(w, 422, "Document and positive amount required")
		return
	}
	switch x.Method {
	case "CASH", "CARD", "TRANSFER", "CHEQUE", "EFFECT":
	default:
		failure(w, 422, "Invalid payment method")
		return
	}
	if (x.Method == "CHEQUE" || x.Method == "EFFECT") && (x.Reference == "" || x.DueDate == "") {
		failure(w, 422, "Cheque number and due date required")
		return
	}
	var due any
	if x.DueDate != "" {
		if _, err := time.Parse("2006-01-02", x.DueDate); err != nil {
			failure(w, 422, "Date must be YYYY-MM-DD")
			return
		}
		due = x.DueDate
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	var typ, status string
	var party, amount, paid int64
	err = tx.QueryRow("SELECT doc_type,status,party_id,total_cents,paid_cents FROM documents WHERE id=? FOR UPDATE", x.DocumentID).Scan(&typ, &status, &party, &amount, &paid)
	if err != nil {
		failure(w, 404, "Document not found")
		return
	}
	if status != "FINAL" || !(typ == "DELIVERY" || typ == "INVOICE" || typ == "PURCHASE_INVOICE") {
		failure(w, 409, "Only a validated invoice or uninvoiced delivery note can be paid")
		return
	}
	if typ == "DELIVERY" {
		var cnt int
		_ = tx.QueryRow("SELECT COUNT(*) FROM documents WHERE source_document_id=? AND doc_type='INVOICE' AND status='FINAL'", x.DocumentID).Scan(&cnt)
		if cnt > 0 {
			failure(w, 409, "Please pay the converted invoice")
			return
		}
	}
	var pending int64
	_ = tx.QueryRow("SELECT COALESCE(SUM(amount_cents),0) FROM payments WHERE document_id=? AND status='PENDING'", x.DocumentID).Scan(&pending)
	if x.AmountCents > amount-paid-pending {
		failure(w, 409, "Payment exceeds available balance including pending cheques")
		return
	}
	newStatus := "CLEARED"
	if x.Method == "CHEQUE" || x.Method == "EFFECT" {
		newStatus = "PENDING"
	}
	result, err := tx.Exec(`INSERT INTO payments(document_id,party_id,amount_cents,method,reference,bank,due_date,status,created_by) VALUES(?,?,?,?,?,?,?,?,?)`, x.DocumentID, party, x.AmountCents, x.Method, x.Reference, x.Bank, due, newStatus, p.ID)
	if err != nil {
		failure(w, 500, "Payment insert failed")
		return
	}
	id, _ := result.LastInsertId()
	if newStatus == "CLEARED" {
		_, err = tx.Exec("UPDATE documents SET paid_cents=paid_cents+? WHERE id=?", x.AmountCents, x.DocumentID)
		if err != nil {
			failure(w, 500, "Balance update failed")
			return
		}
	}
	audit(tx, p.ID, "payment.create", "payment", id, envelope{"status": newStatus, "amountCents": x.AmountCents})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Payment not saved")
		return
	}
	write(w, 201, envelope{"id": id, "status": newStatus})
}
func (a *App) paymentStatus(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	var x struct {
		Status string `json:"status"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if x.Status != "CLEARED" && x.Status != "REJECTED" {
		failure(w, 422, "Only CLEARED or REJECTED allowed")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	var docID, amount int64
	var status, method string
	err = tx.QueryRow("SELECT document_id,amount_cents,status,method FROM payments WHERE id=? FOR UPDATE", id).Scan(&docID, &amount, &status, &method)
	if err != nil {
		failure(w, 404, "Payment not found")
		return
	}
	if status != "PENDING" || !(strings.EqualFold(method, "CHEQUE") || strings.EqualFold(method, "EFFECT")) {
		failure(w, 409, "Only pending cheques/effects can be cleared or rejected")
		return
	}
	if x.Status == "CLEARED" {
		var total, paid int64
		if err = tx.QueryRow("SELECT total_cents,paid_cents FROM documents WHERE id=? FOR UPDATE", docID).Scan(&total, &paid); err != nil {
			failure(w, 500, "Document unavailable")
			return
		}
		if paid+amount > total {
			failure(w, 409, "Cheque exceeds unpaid document balance")
			return
		}
		if _, err = tx.Exec("UPDATE documents SET paid_cents=paid_cents+? WHERE id=?", amount, docID); err != nil {
			failure(w, 500, "Cannot update balance")
			return
		}
	}
	_, err = tx.Exec("UPDATE payments SET status=? WHERE id=?", x.Status, id)
	if err != nil {
		failure(w, 500, "Cheque status update failed")
		return
	}
	audit(tx, p.ID, "cheque.status", "payment", id, envelope{"status": x.Status})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Payment status not committed")
		return
	}
	write(w, 200, envelope{"id": id, "status": x.Status})
}
