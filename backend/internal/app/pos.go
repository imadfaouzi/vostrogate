package app

import (
	"database/sql"
	"net/http"
)

func (a *App) posSale(w http.ResponseWriter, r *http.Request, p Principal) {
	var x struct {
		PartyID     int64       `json:"partyId"`
		WarehouseID int64       `json:"warehouseId"`
		Method      string      `json:"method"`
		Lines       []lineInput `json:"lines"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if x.Method != "CASH" && x.Method != "CARD" && x.Method != "TRANSFER" {
		failure(w, 422, "POS payment must be cash, card or bank transfer")
		return
	}
	if len(x.Lines) == 0 || len(x.Lines) > 150 {
		failure(w, 422, "Add 1–150 articles")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	if x.PartyID == 0 {
		if err = tx.QueryRow("SELECT id FROM parties WHERE kind='CUSTOMER' AND code='COMPTANT'").Scan(&x.PartyID); err != nil {
			failure(w, 422, "Cash customer not found")
			return
		}
	}
	input := documentInput{Type: "INVOICE", PartyID: x.PartyID, WarehouseID: x.WarehouseID, Notes: "Vente comptoir — Vostrogate POS", Lines: x.Lines}
	id, err := createDocument(tx, p.ID, input)
	if err != nil {
		failure(w, 422, err.Error())
		return
	}
	rows, err := tx.Query("SELECT id,product_id,qty_milli FROM document_lines WHERE document_id=?", id)
	if err != nil {
		failure(w, 500, "Cannot check POS lines")
		return
	}
	type stockLine struct{ id, productID, qty int64 }
	var lines []stockLine
	for rows.Next() {
		var l stockLine
		if rows.Scan(&l.id, &l.productID, &l.qty) == nil {
			lines = append(lines, l)
		}
	}
	rows.Close()
	for _, line := range lines {
		if err = moveStock(tx, x.WarehouseID, line.productID, -line.qty, p.ID, "POS:"+itoa(id)+":LINE:"+itoa(line.id), "Vente comptoir", line.id); err != nil {
			failure(w, 409, "Stock insuffisant: "+err.Error())
			return
		}
	}
	var total, party int64
	if err = tx.QueryRow("SELECT total_cents,party_id FROM documents WHERE id=?", id).Scan(&total, &party); err != nil {
		failure(w, 500, "Invalid POS invoice")
		return
	}
	if _, err = tx.Exec("UPDATE documents SET status='FINAL',paid_cents=total_cents,finalized_at=UTC_TIMESTAMP(6) WHERE id=?", id); err != nil {
		failure(w, 500, "POS invoice finalization failed")
		return
	}
	if _, err = tx.Exec(`INSERT INTO payments(document_id,party_id,amount_cents,method,status,created_by) VALUES(?,?,?,?,'CLEARED',?)`, id, party, total, x.Method, p.ID); err != nil {
		failure(w, 500, "POS payment failed")
		return
	}
	audit(tx, p.ID, "pos.sale", "INVOICE", id, envelope{"amountCents": total, "method": x.Method})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "POS transaction did not commit")
		return
	}
	write(w, 201, envelope{"id": id, "totalCents": total, "status": "FINAL", "paid": true})
}
