package app

import (
	"database/sql"
	"net/http"
	"time"
)

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	var sales, unpaid, purchases, expenses, customers, products int64
	// Only finalized invoices and BL not yet converted to a finalized invoice count toward revenue.
	query := `SELECT COALESCE(SUM(d.total_cents),0),COALESCE(SUM(d.total_cents-d.paid_cents),0) FROM documents d WHERE d.status='FINAL' AND d.doc_type IN ('INVOICE','DELIVERY') AND (d.doc_type<>'DELIVERY' OR NOT EXISTS (SELECT 1 FROM documents i WHERE i.source_document_id=d.id AND i.doc_type='INVOICE' AND i.status='FINAL'))`
	_ = db.QueryRow(query).Scan(&sales, &unpaid)
	_ = db.QueryRow("SELECT COALESCE(SUM(total_cents),0) FROM documents WHERE doc_type='PURCHASE_INVOICE' AND status='FINAL'").Scan(&purchases)
	_ = db.QueryRow("SELECT COALESCE(SUM(amount_cents),0) FROM expenses").Scan(&expenses)
	_ = db.QueryRow("SELECT COUNT(*) FROM parties WHERE kind='CUSTOMER'").Scan(&customers)
	_ = db.QueryRow("SELECT COUNT(*) FROM products").Scan(&products)
	rows, err := db.Query(`SELECT DATE_FORMAT(issued_at,'%Y-%m-%d'),SUM(total_cents) FROM documents d WHERE d.status='FINAL' AND d.doc_type IN ('INVOICE','DELIVERY') AND issued_at>=UTC_TIMESTAMP()-INTERVAL 30 DAY AND (d.doc_type<>'DELIVERY' OR NOT EXISTS(SELECT 1 FROM documents i WHERE i.source_document_id=d.id AND i.doc_type='INVOICE' AND i.status='FINAL')) GROUP BY DATE(issued_at) ORDER BY DATE(issued_at)`)
	trend := []any{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var date string
			var total int64
			if rows.Scan(&date, &total) == nil {
				trend = append(trend, envelope{"date": date, "totalCents": total})
			}
		}
	}
	var alerts int64
	_ = db.QueryRow("SELECT COUNT(*) FROM payments WHERE status='PENDING' AND method IN ('CHEQUE','EFFECT') AND due_date<=CURDATE()+INTERVAL 7 DAY").Scan(&alerts)
	write(w, 200, envelope{"salesCents": sales, "unpaidCents": unpaid, "purchasesCents": purchases, "expensesCents": expenses, "customers": customers, "products": products, "chequeAlerts": alerts, "salesTrend": trend})
}
func (a *App) reports(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	if _, e := time.Parse("2006-01-02", from); e != nil {
		failure(w, 422, "Invalid start date")
		return
	}
	if _, e := time.Parse("2006-01-02", to); e != nil {
		failure(w, 422, "Invalid end date")
		return
	}
	db, _ := a.database()
	switch kind {
	case "stock":
		a.stockList(w, r)
		return
	case "expenses":
		a.expensesList(w, r)
		return
	case "receivables":
		rows, err := db.Query(`SELECT p.id,p.code,p.name,COALESCE(SUM(d.total_cents-d.paid_cents),0) FROM parties p LEFT JOIN documents d ON d.party_id=p.id AND d.status='FINAL' AND d.doc_type IN ('INVOICE','DELIVERY') AND (d.doc_type<>'DELIVERY' OR NOT EXISTS(SELECT 1 FROM documents i WHERE i.source_document_id=d.id AND i.doc_type='INVOICE' AND i.status='FINAL')) WHERE p.kind='CUSTOMER' GROUP BY p.id ORDER BY 4 DESC LIMIT 250`)
		if err != nil {
			failure(w, 500, "Receivables report failed")
			return
		}
		defer rows.Close()
		items := []any{}
		for rows.Next() {
			var id, total int64
			var code, name string
			if rows.Scan(&id, &code, &name, &total) == nil {
				items = append(items, envelope{"id": id, "code": code, "name": name, "balanceCents": total})
			}
		}
		write(w, 200, envelope{"items": items, "kind": kind})
		return
	case "sales", "purchases":
		typ := "('INVOICE','DELIVERY')"
		if kind == "purchases" {
			typ = "('PURCHASE_INVOICE')"
		}
		qry := `SELECT d.id,d.doc_type,d.doc_number,d.issued_at,p.name,d.total_cents,d.paid_cents FROM documents d JOIN parties p ON p.id=d.party_id WHERE d.status='FINAL' AND d.doc_type IN ` + typ + ` AND DATE(d.issued_at) BETWEEN ? AND ?`
		if kind == "sales" {
			qry += ` AND (d.doc_type<>'DELIVERY' OR NOT EXISTS(SELECT 1 FROM documents i WHERE i.source_document_id=d.id AND i.doc_type='INVOICE' AND i.status='FINAL'))`
		}
		qry += ` ORDER BY d.issued_at DESC LIMIT 500`
		rows, err := db.Query(qry, from, to)
		if err != nil {
			failure(w, 500, "Report query failed")
			return
		}
		defer rows.Close()
		items := []any{}
		var grandTotal, paidTotal int64
		for rows.Next() {
			var id, total, paid int64
			var docType, no, date, partner string
			if rows.Scan(&id, &docType, &no, &date, &partner, &total, &paid) == nil {
				items = append(items, envelope{"id": id, "type": docType, "number": no, "date": date, "partyName": partner, "totalCents": total, "paidCents": paid})
				grandTotal += total
				paidTotal += paid
			}
		}
		write(w, 200, envelope{"items": items, "totalCents": grandTotal, "paidCents": paidTotal, "from": from, "to": to, "kind": kind})
		return
	default:
		failure(w, 422, "Report kind must be sales, purchases, stock, expenses or receivables")
	}
}
func (a *App) auditList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query("SELECT a.id,COALESCE(u.username,'system'),a.action,a.entity_type,a.entity_id,a.created_at,COALESCE(CAST(a.details_json AS CHAR),'{}') FROM audit_log a LEFT JOIN users u ON u.id=a.actor_id ORDER BY a.id DESC LIMIT 300")
	if err != nil {
		failure(w, 500, "Audit list failed")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id, eID int64
		var user, action, entity, date, details string
		if rows.Scan(&id, &user, &action, &entity, &eID, &date, &details) == nil {
			items = append(items, envelope{"id": id, "user": user, "action": action, "entity": entity, "entityId": eID, "date": date, "details": details})
		}
	}
	write(w, 200, envelope{"items": items})
}

var _ *sql.DB // kept for report unit test patterns
