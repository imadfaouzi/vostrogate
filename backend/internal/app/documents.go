package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type lineInput struct {
	ProductID      int64 `json:"productId"`
	QtyMilli       int64 `json:"qtyMilli"`
	UnitPriceCents int64 `json:"unitPriceCents"`
	DiscountBps    int   `json:"discountBps"`
	VATBps         int   `json:"vatBps"`
}
type documentInput struct {
	Type             string      `json:"type"`
	PartyID          int64       `json:"partyId"`
	WarehouseID      int64       `json:"warehouseId"`
	SourceDocumentID int64       `json:"sourceDocumentId"`
	Date             string      `json:"date"`
	Notes            string      `json:"notes"`
	Lines            []lineInput `json:"lines"`
}

var docTypePrefixes = map[string]string{"QUOTE": "DEV", "ORDER": "CMD", "DELIVERY": "BL", "INVOICE": "FAC", "CREDIT": "AV", "RETURN": "RET", "PURCHASE_ORDER": "AC", "PURCHASE_RECEIPT": "REC", "PURCHASE_INVOICE": "FA"}

func isPurchase(t string) bool { return strings.HasPrefix(t, "PURCHASE_") }
func docPermission(typ, action string) string {
	if isPurchase(typ) {
		return "purchases." + action
	}
	return "sales." + action
}
func docPartyKind(typ string) string {
	if isPurchase(typ) {
		return "SUPPLIER"
	}
	return "CUSTOMER"
}
func nextDocNumber(tx *sql.Tx, docType string) (string, error) {
	prefix, ok := docTypePrefixes[docType]
	if !ok {
		return "", errors.New("unsupported document type")
	}
	var raw []byte
	if err := tx.QueryRow("SELECT setting_json FROM settings WHERE setting_key='document_prefixes'").Scan(&raw); err == nil {
		var stored map[string]string
		if json.Unmarshal(raw, &stored) == nil && regexp.MustCompile(`^[A-Za-z0-9_-]{1,15}$`).MatchString(stored[docType]) {
			prefix = stored[docType]
		}
	}
	if _, err := tx.Exec("INSERT IGNORE INTO document_sequences(doc_type,next_number) VALUES(?,1)", docType); err != nil {
		return "", err
	}
	var n int64
	if err := tx.QueryRow("SELECT next_number FROM document_sequences WHERE doc_type=? FOR UPDATE", docType).Scan(&n); err != nil {
		return "", err
	}
	if _, err := tx.Exec("UPDATE document_sequences SET next_number=? WHERE doc_type=?", n+1, docType); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d-%06d", prefix, time.Now().UTC().Year(), n), nil
}
func (a *App) documentCreate(w http.ResponseWriter, r *http.Request, p Principal) {
	var x documentInput
	if !parseJSON(w, r, &x) {
		return
	}
	if (x.Type == "CREDIT" || x.Type == "RETURN") && x.SourceDocumentID == 0 {
		failure(w, 422, "Create returns/credit notes from a validated source document")
		return
	}
	if docTypePrefixes[x.Type] == "" || x.PartyID <= 0 || x.WarehouseID <= 0 || len(x.Lines) == 0 || len(x.Lines) > 300 {
		failure(w, 422, "Document type, partner, warehouse and 1–300 articles required")
		return
	}
	if !allowed(w, p, docPermission(x.Type, "write")) {
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	id, err := createDocument(tx, p.ID, x)
	if err != nil {
		failure(w, 422, err.Error())
		return
	}
	audit(tx, p.ID, "document.create", x.Type, id, envelope{"source": x.SourceDocumentID})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Document save failed")
		return
	}
	write(w, 201, envelope{"id": id})
}
func createDocument(tx *sql.Tx, actorID int64, x documentInput) (int64, error) {
	var kind string
	err := tx.QueryRow("SELECT kind FROM parties WHERE id=? AND is_active=1", x.PartyID).Scan(&kind)
	if err != nil || kind != docPartyKind(x.Type) {
		return 0, errors.New("customer/supplier incompatible with document type")
	}
	var warehouse int64
	if tx.QueryRow("SELECT id FROM warehouses WHERE id=? AND is_active=1", x.WarehouseID).Scan(&warehouse) != nil {
		return 0, errors.New("unknown warehouse")
	}
	if x.SourceDocumentID > 0 {
		var sourceType, sourceStatus string
		var sourceParty int64
		if tx.QueryRow("SELECT doc_type,status,party_id FROM documents WHERE id=?", x.SourceDocumentID).Scan(&sourceType, &sourceStatus, &sourceParty) != nil || sourceStatus != "FINAL" || sourceParty != x.PartyID || !allowedConversion(sourceType, x.Type) {
			return 0, errors.New("invalid source document or conversion sequence")
		}
		var cnt int
		if tx.QueryRow("SELECT COUNT(*) FROM documents WHERE source_document_id=? AND doc_type=?", x.SourceDocumentID, x.Type).Scan(&cnt) != nil || cnt > 0 {
			return 0, errors.New("this source has already been converted to that document type")
		}
	}
	no, err := nextDocNumber(tx, x.Type)
	if err != nil {
		return 0, err
	}
	issued := time.Now().UTC()
	if x.Date != "" {
		parsed, e := time.Parse("2006-01-02", x.Date)
		if e != nil {
			return 0, errors.New("date must be YYYY-MM-DD")
		}
		issued = parsed
	}
	var source any
	if x.SourceDocumentID > 0 {
		source = x.SourceDocumentID
	}
	res, err := tx.Exec(`INSERT INTO documents(doc_type,doc_number,party_id,warehouse_id,source_document_id,issued_at,notes,created_by) VALUES(?,?,?,?,?,?,?,?)`, x.Type, no, x.PartyID, x.WarehouseID, source, issued, x.Notes, actorID)
	if err != nil {
		return 0, err
	}
	docID, _ := res.LastInsertId()
	var subtotal, vat, total int64
	for _, line := range x.Lines {
		var sku, name string
		var active bool
		err = tx.QueryRow("SELECT sku,name,is_active FROM products WHERE id=?", line.ProductID).Scan(&sku, &name, &active)
		if err != nil || !active {
			return 0, errors.New("inactive or missing product")
		}
		amounts, e := CalculateLine(line.QtyMilli, line.UnitPriceCents, line.DiscountBps, line.VATBps)
		if e != nil {
			return 0, e
		}
		subtotal += amounts.SubtotalCents
		vat += amounts.VatCents
		total += amounts.TotalCents
		if subtotal > 9000000000000000 || total > 9000000000000000 {
			return 0, errors.New("document exceeds amount limit")
		}
		_, err = tx.Exec(`INSERT INTO document_lines(document_id,product_id,sku_snapshot,name_snapshot,qty_milli,unit_price_cents,discount_bps,vat_bps,subtotal_cents,vat_cents,total_cents)
  VALUES(?,?,?,?,?,?,?,?,?,?,?)`, docID, line.ProductID, sku, name, line.QtyMilli, line.UnitPriceCents, line.DiscountBps, line.VATBps, amounts.SubtotalCents, amounts.VatCents, amounts.TotalCents)
		if err != nil {
			return 0, err
		}
	}
	_, err = tx.Exec("UPDATE documents SET subtotal_cents=?,vat_cents=?,total_cents=? WHERE id=?", subtotal, vat, total, docID)
	return docID, err
}
func (a *App) documentsList(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	if typ == "" {
		typ = "INVOICE"
	}
	if docTypePrefixes[typ] == "" {
		failure(w, 422, "Invalid document type")
		return
	}
	q := r.URL.Query().Get("q")
	pat := "%" + q + "%"
	limit := queryLimit(r, 50, 150)
	offset := queryOffset(r)
	db, _ := a.database()
	rows, err := db.Query(`SELECT d.id,d.doc_type,d.doc_number,d.status,d.party_id,p.name,d.warehouse_id,d.source_document_id,
 d.issued_at,d.subtotal_cents,d.vat_cents,d.total_cents,d.paid_cents,d.notes
 FROM documents d JOIN parties p ON p.id=d.party_id WHERE d.doc_type=? AND (d.doc_number LIKE ? OR p.name LIKE ?)
 ORDER BY d.id DESC LIMIT ? OFFSET ?`, typ, pat, pat, limit, offset)
	if err != nil {
		failure(w, 500, "Document list failed")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		v, err := scanDocument(rows)
		if err == nil {
			items = append(items, v)
		}
	}
	var total int
	_ = db.QueryRow("SELECT COUNT(*) FROM documents d JOIN parties p ON p.id=d.party_id WHERE d.doc_type=? AND (d.doc_number LIKE ? OR p.name LIKE ?)", typ, pat, pat).Scan(&total)
	write(w, 200, envelope{"items": items, "total": total, "limit": limit, "offset": offset})
}
func scanDocument(rows *sql.Rows) (envelope, error) {
	var id, partyID, warehouseID, sub, vat, total, paid int64
	var typ, no, status, name, issued, notes string
	var source sql.NullInt64
	err := rows.Scan(&id, &typ, &no, &status, &partyID, &name, &warehouseID, &source, &issued, &sub, &vat, &total, &paid, &notes)
	if err != nil {
		return nil, err
	}
	return envelope{"id": id, "type": typ, "number": no, "status": status, "partyId": partyID, "partyName": name, "warehouseId": warehouseID, "sourceDocumentId": source.Int64, "date": issued, "subtotalCents": sub, "vatCents": vat, "totalCents": total, "paidCents": paid, "remainingCents": total - paid, "notes": notes}, nil
}
func (a *App) documentGet(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	db, _ := a.database()
	rows, err := db.Query(`SELECT d.id,d.doc_type,d.doc_number,d.status,d.party_id,p.name,d.warehouse_id,d.source_document_id,d.issued_at,d.subtotal_cents,d.vat_cents,d.total_cents,d.paid_cents,d.notes FROM documents d JOIN parties p ON p.id=d.party_id WHERE d.id=?`, id)
	if err != nil {
		failure(w, 500, "Document lookup error")
		return
	}
	defer rows.Close()
	if !rows.Next() {
		failure(w, 404, "Document not found")
		return
	}
	v, err := scanDocument(rows)
	if err == nil {
		if !allowed(w, p, docPermission(v["type"].(string), "read")) {
			return
		}
	}
	if err != nil {
		failure(w, 500, "Document invalid")
		return
	}
	lines, err := db.Query(`SELECT id,product_id,sku_snapshot,name_snapshot,qty_milli,unit_price_cents,discount_bps,vat_bps,subtotal_cents,vat_cents,total_cents FROM document_lines WHERE document_id=? ORDER BY id`, id)
	if err != nil {
		failure(w, 500, "Document lines unavailable")
		return
	}
	defer lines.Close()
	items := []any{}
	for lines.Next() {
		var lid, pid, qty, price, sub, vat, total int64
		var sku, name string
		var discount, tax int
		if lines.Scan(&lid, &pid, &sku, &name, &qty, &price, &discount, &tax, &sub, &vat, &total) == nil {
			items = append(items, envelope{"id": lid, "productId": pid, "sku": sku, "name": name, "qtyMilli": qty, "unitPriceCents": price, "discountBps": discount, "vatBps": tax, "subtotalCents": sub, "vatCents": vat, "totalCents": total})
		}
	}
	v["lines"] = items
	write(w, 200, v)
}
func (a *App) documentFinalize(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	var typ, status string
	var warehouse, source, party, amount int64
	err = tx.QueryRow("SELECT doc_type,status,warehouse_id,COALESCE(source_document_id,0),party_id,total_cents FROM documents WHERE id=? FOR UPDATE", id).Scan(&typ, &status, &warehouse, &source, &party, &amount)
	if err != nil {
		failure(w, 404, "Document not found")
		return
	}
	if !allowed(w, p, docPermission(typ, "finalize")) {
		return
	}
	if status != "DRAFT" {
		failure(w, 409, "Only draft documents can be finalized")
		return
	}
	fromDelivery := false
	if typ == "INVOICE" && source > 0 {
		var srcType string
		_ = tx.QueryRow("SELECT doc_type FROM documents WHERE id=?", source).Scan(&srcType)
		fromDelivery = srcType == "DELIVERY"
	}
	lines, err := tx.Query("SELECT id,product_id,qty_milli FROM document_lines WHERE document_id=? ORDER BY id", id)
	if err != nil {
		failure(w, 500, "Cannot load lines")
		return
	}
	type dl struct{ id, pid, qty int64 }
	var itemLines []dl
	for lines.Next() {
		var l dl
		if lines.Scan(&l.id, &l.pid, &l.qty) == nil {
			itemLines = append(itemLines, l)
		}
	}
	lines.Close()
	for _, line := range itemLines {
		delta := stockDelta(typ, fromDelivery, line.qty)
		if err = moveStock(tx, warehouse, line.pid, delta, p.ID, "DOC:"+itoa(id)+":LINE:"+itoa(line.id), "Validation "+typ, line.id); err != nil {
			failure(w, 409, "Stock validation: "+err.Error())
			return
		}
	}
	if typ == "INVOICE" && fromDelivery {
		var srcPaid int64
		err = tx.QueryRow("SELECT paid_cents FROM documents WHERE id=? FOR UPDATE", source).Scan(&srcPaid)
		if err != nil {
			failure(w, 500, "Source document unavailable")
			return
		}
		if srcPaid > amount {
			failure(w, 409, "Source payments exceed new invoice; reconcile manually")
			return
		}
		if srcPaid > 0 {
			if _, err = tx.Exec("UPDATE payments SET document_id=? WHERE document_id=?", id, source); err != nil {
				failure(w, 500, "Payment transfer failed")
				return
			}
			if _, err = tx.Exec("UPDATE documents SET paid_cents=0 WHERE id=?", source); err != nil {
				failure(w, 500, "Source balance transfer failed")
				return
			}
			if _, err = tx.Exec("UPDATE documents SET paid_cents=? WHERE id=?", srcPaid, id); err != nil {
				failure(w, 500, "Invoice balance transfer failed")
				return
			}
		}
	}
	_, err = tx.Exec("UPDATE documents SET status='FINAL',finalized_at=UTC_TIMESTAMP(6) WHERE id=?", id)
	if err != nil {
		failure(w, 500, "Could not finalize document")
		return
	}
	audit(tx, p.ID, "document.finalize", typ, id, envelope{"source": source, "stockApplied": len(itemLines)})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Finalization failed")
		return
	}
	write(w, 200, envelope{"id": id, "status": "FINAL"})
}
func (a *App) documentConvert(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	var req struct {
		TargetType string `json:"targetType"`
	}
	if !parseJSON(w, r, &req) {
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	var typ, status string
	var party, wh int64
	var notes string
	if err = tx.QueryRow("SELECT doc_type,status,party_id,warehouse_id,notes FROM documents WHERE id=? FOR UPDATE", id).Scan(&typ, &status, &party, &wh, &notes); err != nil {
		failure(w, 404, "Source document unavailable")
		return
	}
	if !allowed(w, p, docPermission(typ, "convert")) {
		return
	}
	if !allowedConversion(typ, req.TargetType) || status != "FINAL" {
		failure(w, 409, "Conversion not allowed from this document state")
		return
	}
	if !allowed(w, p, docPermission(req.TargetType, "write")) {
		return
	}
	rows, err := tx.Query("SELECT product_id,qty_milli,unit_price_cents,discount_bps,vat_bps FROM document_lines WHERE document_id=? ORDER BY id", id)
	if err != nil {
		failure(w, 500, "Cannot load source lines")
		return
	}
	input := documentInput{Type: req.TargetType, PartyID: party, WarehouseID: wh, SourceDocumentID: id, Notes: notes}
	for rows.Next() {
		var x lineInput
		if rows.Scan(&x.ProductID, &x.QtyMilli, &x.UnitPriceCents, &x.DiscountBps, &x.VATBps) == nil {
			input.Lines = append(input.Lines, x)
		}
	}
	rows.Close()
	newID, err := createDocument(tx, p.ID, input)
	if err != nil {
		failure(w, 409, err.Error())
		return
	}
	audit(tx, p.ID, "document.convert", req.TargetType, newID, envelope{"sourceId": id})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Conversion transaction failed")
		return
	}
	write(w, 201, envelope{"id": newID, "status": "DRAFT", "sourceId": id})
}
func (a *App) documentCancel(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	var typ, status string
	var wh, source, paid int64
	err = tx.QueryRow("SELECT doc_type,status,warehouse_id,COALESCE(source_document_id,0),paid_cents FROM documents WHERE id=? FOR UPDATE", id).Scan(&typ, &status, &wh, &source, &paid)
	if err != nil {
		failure(w, 404, "Document not found")
		return
	}
	if status == "CANCELLED" {
		failure(w, 409, "Already cancelled")
		return
	}
	if !allowed(w, p, docPermission(typ, "cancel")) {
		return
	}
	if paid > 0 {
		failure(w, 409, "Cannot cancel a paid document; reconcile payments first")
		return
	}
	var children int
	_ = tx.QueryRow("SELECT COUNT(*) FROM documents WHERE source_document_id=? AND status<>'CANCELLED'", id).Scan(&children)
	if children > 0 {
		failure(w, 409, "This document has active converted documents")
		return
	}
	if status == "FINAL" {
		fromBL := false
		if typ == "INVOICE" && source > 0 {
			var x string
			_ = tx.QueryRow("SELECT doc_type FROM documents WHERE id=?", source).Scan(&x)
			fromBL = x == "DELIVERY"
		}
		rows, e := tx.Query("SELECT id,product_id,qty_milli FROM document_lines WHERE document_id=?", id)
		if e != nil {
			failure(w, 500, "Document lines unavailable")
			return
		}
		type line struct{ id, pid, qty int64 }
		var lines []line
		for rows.Next() {
			var x line
			if rows.Scan(&x.id, &x.pid, &x.qty) == nil {
				lines = append(lines, x)
			}
		}
		rows.Close()
		for _, line := range lines {
			rev := -stockDelta(typ, fromBL, line.qty)
			if e = moveStock(tx, wh, line.pid, rev, p.ID, "CANCEL:"+itoa(id)+":LINE:"+itoa(line.id), "Annulation "+typ, line.id); e != nil {
				failure(w, 409, "Cannot reverse stock: "+e.Error())
				return
			}
		}
	}
	_, err = tx.Exec("UPDATE documents SET status='CANCELLED' WHERE id=?", id)
	if err != nil {
		failure(w, 500, "Cancellation failed")
		return
	}
	audit(tx, p.ID, "document.cancel", typ, id, nil)
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Cancellation did not commit")
		return
	}
	write(w, 200, envelope{"id": id, "status": "CANCELLED"})
}
