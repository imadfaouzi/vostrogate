package app

import (
	"database/sql"
	"errors"
	"net/http"
)

func moveStock(tx *sql.Tx, warehouseID, productID, deltaMilli, actorID int64, key, reason string, lineID any) error {
	if deltaMilli == 0 {
		return nil
	}
	_, err := tx.Exec(`INSERT IGNORE INTO stock_balances(warehouse_id,product_id,quantity_milli) VALUES(?,?,0)`, warehouseID, productID)
	if err != nil {
		return err
	}
	var existing int
	err = tx.QueryRow("SELECT COUNT(*) FROM stock_movements WHERE movement_key=?", key).Scan(&existing)
	if err != nil {
		return err
	}
	if existing > 0 {
		return errors.New("stock movement already recorded")
	}
	var qty int64
	err = tx.QueryRow(`SELECT quantity_milli FROM stock_balances WHERE warehouse_id=? AND product_id=? FOR UPDATE`, warehouseID, productID).Scan(&qty)
	if err != nil {
		return err
	}
	if qty+deltaMilli < 0 {
		return errors.New("not enough available stock")
	}
	_, err = tx.Exec("UPDATE stock_balances SET quantity_milli=? WHERE warehouse_id=? AND product_id=?", qty+deltaMilli, warehouseID, productID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO stock_movements(warehouse_id,product_id,document_line_id,movement_key,delta_milli,reason,actor_id) VALUES(?,?,?,?,?,?,?)`, warehouseID, productID, lineID, key, deltaMilli, reason, actorID)
	return err
}
func (a *App) stockList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	q := r.URL.Query().Get("q")
	wh := idFrom(r.URL.Query().Get("warehouseId"))
	pat := "%" + q + "%"
	rows, err := db.Query(`SELECT p.id,p.sku,COALESCE(p.barcode,''),p.name,p.min_stock_milli,w.id,w.name,COALESCE(sb.quantity_milli,0)
 FROM products p CROSS JOIN warehouses w LEFT JOIN stock_balances sb ON sb.product_id=p.id AND sb.warehouse_id=w.id
 WHERE (p.name LIKE ? OR p.sku LIKE ? OR p.barcode LIKE ?) AND (?=0 OR w.id=?) ORDER BY p.name,w.name LIMIT 500`, pat, pat, pat, wh, wh)
	if err != nil {
		failure(w, 500, "Stock query failed")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var pid, wid, min, qty int64
		var sku, barcode, name, warehouse string
		if rows.Scan(&pid, &sku, &barcode, &name, &min, &wid, &warehouse, &qty) == nil {
			items = append(items, envelope{"productId": pid, "sku": sku, "barcode": barcode, "name": name, "warehouseId": wid, "warehouse": warehouse, "quantityMilli": qty, "minStockMilli": min, "lowStock": qty < min})
		}
	}
	write(w, 200, envelope{"items": items})
}
func (a *App) stockAdjust(w http.ResponseWriter, r *http.Request, p Principal) {
	var x struct {
		ProductID   int64  `json:"productId"`
		WarehouseID int64  `json:"warehouseId"`
		DeltaMilli  int64  `json:"deltaMilli"`
		Reason      string `json:"reason"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	if x.ProductID <= 0 || x.WarehouseID <= 0 || x.DeltaMilli == 0 || len(x.Reason) < 5 {
		failure(w, 422, "Article, dépôt, écart et motif détaillé requis")
		return
	}
	db, _ := a.database()
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		failure(w, 500, "Transaction error")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec("INSERT INTO stock_adjustments(warehouse_id,product_id,delta_milli,reason,actor_id) VALUES(?,?,?,?,?)", x.WarehouseID, x.ProductID, x.DeltaMilli, x.Reason, p.ID)
	if err != nil {
		failure(w, 422, "Unknown article or warehouse")
		return
	}
	id, _ := res.LastInsertId()
	if err = moveStock(tx, x.WarehouseID, x.ProductID, x.DeltaMilli, p.ID, "ADJUST:"+itoa(id), x.Reason, nil); err != nil {
		failure(w, 409, err.Error())
		return
	}
	audit(tx, p.ID, "stock.adjust", "stock_adjustment", id, x)
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Stock adjustment did not commit")
		return
	}
	write(w, 201, envelope{"id": id})
}
func (a *App) stockHistory(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query(`SELECT sm.id,sm.product_id,p.name,sm.warehouse_id,sm.delta_milli,sm.reason,sm.created_at,u.username FROM stock_movements sm JOIN products p ON p.id=sm.product_id JOIN users u ON u.id=sm.actor_id ORDER BY sm.id DESC LIMIT 200`)
	if err != nil {
		failure(w, 500, "Stock movements unavailable")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id, pid, wid, delta int64
		var name, reason, user string
		var at string
		if rows.Scan(&id, &pid, &name, &wid, &delta, &reason, &at, &user) == nil {
			items = append(items, envelope{"id": id, "productId": pid, "product": name, "warehouseId": wid, "deltaMilli": delta, "reason": reason, "date": at, "user": user})
		}
	}
	write(w, 200, envelope{"items": items})
}
