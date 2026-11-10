package app

import (
	"net/http"
	"strings"
	"time"
)

func (a *App) expensesList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query("SELECT e.id,e.title,COALESCE(rv.name,''),e.amount_cents,e.spent_on,e.method,e.notes FROM expenses e LEFT JOIN reference_values rv ON rv.id=e.category_id ORDER BY e.id DESC LIMIT 250")
	if err != nil {
		failure(w, 500, "Expenses unavailable")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id, amount int64
		var title, cat, date, method, notes string
		if rows.Scan(&id, &title, &cat, &amount, &date, &method, &notes) == nil {
			items = append(items, envelope{"id": id, "title": title, "category": cat, "amountCents": amount, "date": date, "method": method, "notes": notes})
		}
	}
	write(w, 200, envelope{"items": items})
}
func (a *App) expenseCreate(w http.ResponseWriter, r *http.Request, p Principal) {
	var x struct {
		Title       string `json:"title"`
		CategoryID  int64  `json:"categoryId"`
		AmountCents int64  `json:"amountCents"`
		Date        string `json:"date"`
		Method      string `json:"method"`
		Notes       string `json:"notes"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if strings.TrimSpace(x.Title) == "" || x.AmountCents <= 0 {
		failure(w, 422, "Expense title and positive amount required")
		return
	}
	if x.Date == "" {
		x.Date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", x.Date); err != nil {
		failure(w, 422, "Invalid expense date")
		return
	}
	if x.Method == "" {
		x.Method = "CASH"
	}
	db, _ := a.database()
	res, err := db.Exec("INSERT INTO expenses(title,category_id,amount_cents,spent_on,method,notes,created_by) VALUES(?,?,?,?,?,?,?)", x.Title, nullFK(x.CategoryID), x.AmountCents, x.Date, x.Method, x.Notes, p.ID)
	if err != nil {
		failure(w, 422, "Expense save failed")
		return
	}
	id, _ := res.LastInsertId()
	write(w, 201, envelope{"id": id})
}
