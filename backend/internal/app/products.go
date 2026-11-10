package app

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

type productInput struct {
	SKU                string `json:"sku"`
	Barcode            string `json:"barcode"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	CategoryID         int64  `json:"categoryId"`
	BrandID            int64  `json:"brandId"`
	UnitID             int64  `json:"unitId"`
	PurchasePriceCents int64  `json:"purchasePriceCents"`
	SalePriceCents     int64  `json:"salePriceCents"`
	VATBps             int    `json:"vatBps"`
	MinStockMilli      int64  `json:"minStockMilli"`
	IsActive           bool   `json:"isActive"`
	InitialStockMilli  int64  `json:"initialStockMilli"`
	WarehouseID        int64  `json:"warehouseId"`
}

const productCols = `p.id,p.sku,COALESCE(p.barcode,''),p.name,p.description,COALESCE(p.category_id,0),COALESCE(p.brand_id,0),COALESCE(p.unit_id,0),p.purchase_price_cents,p.sale_price_cents,p.vat_bps,p.min_stock_milli,p.is_active,COALESCE(SUM(s.quantity_milli),0)`
const productJoin = `FROM products p LEFT JOIN stock_balances s ON s.product_id=p.id`

func scanProduct(rows *sql.Rows) (envelope, error) {
	var id, cat, brand, unit, cost, price, minStock, qty int64
	var sku, barcode, name, description string
	var vat int
	var active bool
	err := rows.Scan(&id, &sku, &barcode, &name, &description, &cat, &brand, &unit, &cost, &price, &vat, &minStock, &active, &qty)
	if err != nil {
		return nil, err
	}
	return envelope{"id": id, "sku": sku, "barcode": barcode, "name": name, "description": description, "categoryId": cat, "brandId": brand, "unitId": unit, "purchasePriceCents": cost, "salePriceCents": price, "vatBps": vat, "minStockMilli": minStock, "isActive": active, "stockMilli": qty}, nil
}
func (a *App) productList(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	pat := "%" + q + "%"
	limit := queryLimit(r, 50, 200)
	offset := queryOffset(r)
	db, _ := a.database()
	rows, err := db.Query("SELECT "+productCols+" "+productJoin+" WHERE p.name LIKE ? OR p.sku LIKE ? OR p.barcode LIKE ? GROUP BY p.id ORDER BY p.name LIMIT ? OFFSET ?", pat, pat, pat, limit, offset)
	if err != nil {
		failure(w, 500, "Products query failed")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		item, e := scanProduct(rows)
		if e == nil {
			items = append(items, item)
		}
	}
	var total int
	_ = db.QueryRow("SELECT COUNT(*) FROM products WHERE name LIKE ? OR sku LIKE ? OR barcode LIKE ?", pat, pat, pat).Scan(&total)
	write(w, 200, envelope{"items": items, "total": total, "limit": limit, "offset": offset})
}
func (a *App) productGet(w http.ResponseWriter, r *http.Request, id int64) {
	db, _ := a.database()
	rows, err := db.Query("SELECT "+productCols+" "+productJoin+" WHERE p.id=? GROUP BY p.id", id)
	if err != nil {
		failure(w, 500, "Lookup failed")
		return
	}
	defer rows.Close()
	if !rows.Next() {
		failure(w, 404, "Article not found")
		return
	}
	item, err := scanProduct(rows)
	if err != nil {
		failure(w, 500, "Bad article")
		return
	}
	write(w, 200, item)
}
func (a *App) productSearch(w http.ResponseWriter, r *http.Request) {
	// Prefer barcode / SKU exact match, then prefix match, then name contains. Debounced in React.
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 1 {
		write(w, 200, envelope{"items": []any{}})
		return
	}
	if len(q) > 150 {
		failure(w, 422, "Search text too long")
		return
	}
	pat := "%" + q + "%"
	prefix := q + "%"
	db, _ := a.database()
	rows, err := db.Query(`SELECT `+productCols+` `+productJoin+`
 WHERE p.is_active=1 AND (p.barcode=? OR p.sku=? OR p.name LIKE ? OR p.sku LIKE ? OR p.barcode LIKE ?)
 GROUP BY p.id ORDER BY CASE WHEN p.barcode=? THEN 0 WHEN p.sku=? THEN 1 WHEN p.sku LIKE ? THEN 2 WHEN p.name LIKE ? THEN 3 ELSE 4 END,p.name LIMIT 20`, q, q, pat, pat, pat, q, q, prefix, prefix)
	if err != nil {
		failure(w, 500, "Search failed")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		v, e := scanProduct(rows)
		if e == nil {
			items = append(items, v)
		}
	}
	write(w, 200, envelope{"items": items})
}
func nullFK(id int64) any {
	if id > 0 {
		return id
	}
	return nil
}
func nullableBarcode(v string) any {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return nil
}
func (a *App) productSave(w http.ResponseWriter, r *http.Request, p Principal, id int64) {
	var x productInput
	if !parseJSON(w, r, &x) {
		return
	}
	if strings.TrimSpace(x.SKU) == "" || strings.TrimSpace(x.Name) == "" || x.PurchasePriceCents < 0 || x.SalePriceCents < 0 || x.VATBps < 0 || x.VATBps > 10000 || x.InitialStockMilli < 0 {
		failure(w, 422, "Invalid article name, reference, price or tax")
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
		result, e := tx.Exec(`INSERT INTO products(sku,barcode,name,description,category_id,brand_id,unit_id,purchase_price_cents,sale_price_cents,vat_bps,min_stock_milli,is_active)
  VALUES(?,?,?,?,?,?,?,?,?,?,?,1)`, x.SKU, nullableBarcode(x.Barcode), x.Name, x.Description, nullFK(x.CategoryID), nullFK(x.BrandID), nullFK(x.UnitID), x.PurchasePriceCents, x.SalePriceCents, x.VATBps, x.MinStockMilli)
		err = e
		if err == nil {
			id, _ = result.LastInsertId()
		}
		if err == nil && x.InitialStockMilli > 0 {
			if x.WarehouseID == 0 {
				err = tx.QueryRow("SELECT id FROM warehouses WHERE is_default=1 LIMIT 1").Scan(&x.WarehouseID)
			}
			if err == nil {
				_, err = tx.Exec("INSERT INTO stock_balances(warehouse_id,product_id,quantity_milli) VALUES(?,?,?)", x.WarehouseID, id, x.InitialStockMilli)
			}
			if err == nil {
				_, err = tx.Exec("INSERT INTO stock_movements(warehouse_id,product_id,movement_key,delta_milli,reason,actor_id) VALUES(?,?,?,?,?,?)", x.WarehouseID, id, "INITIAL:"+itoa(id), x.InitialStockMilli, "Initial stock", p.ID)
			}
		}
	} else {
		_, err = tx.Exec(`UPDATE products SET sku=?,barcode=?,name=?,description=?,category_id=?,brand_id=?,unit_id=?,purchase_price_cents=?,sale_price_cents=?,vat_bps=?,min_stock_milli=?,is_active=? WHERE id=?`, x.SKU, nullableBarcode(x.Barcode), x.Name, x.Description, nullFK(x.CategoryID), nullFK(x.BrandID), nullFK(x.UnitID), x.PurchasePriceCents, x.SalePriceCents, x.VATBps, x.MinStockMilli, x.IsActive, id)
	}
	if err != nil {
		failure(w, 409, "Article reference, barcode or related field invalid")
		return
	}
	audit(tx, p.ID, "product.save", "product", id, envelope{"sku": x.SKU})
	if err = tx.Commit(); err != nil {
		failure(w, 500, "Article save failed")
		return
	}
	write(w, 200, envelope{"id": id})
}
func itoa(i int64) string { return strconv.FormatInt(i, 10) }
func (a *App) catalogList(w http.ResponseWriter, r *http.Request) {
	db, _ := a.database()
	rows, err := db.Query("SELECT id,kind,name,is_active FROM reference_values ORDER BY kind,name")
	if err != nil {
		failure(w, 500, "Catalog unavailable")
		return
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		var id int64
		var kind, name string
		var active bool
		if rows.Scan(&id, &kind, &name, &active) == nil {
			items = append(items, envelope{"id": id, "kind": kind, "name": name, "isActive": active})
		}
	}
	write(w, 200, envelope{"items": items})
}
func (a *App) catalogCreate(w http.ResponseWriter, r *http.Request, p Principal) {
	var x struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	if !parseJSON(w, r, &x) {
		return
	}
	switch x.Kind {
	case "category", "brand", "unit", "price_group", "expense_category":
	default:
		failure(w, 422, "Invalid catalog type")
		return
	}
	if strings.TrimSpace(x.Name) == "" {
		failure(w, 422, "Name required")
		return
	}
	db, _ := a.database()
	res, err := db.Exec("INSERT INTO reference_values(kind,name) VALUES(?,?)", x.Kind, x.Name)
	if err != nil {
		failure(w, 409, "Already exists")
		return
	}
	id, _ := res.LastInsertId()
	write(w, 201, envelope{"id": id})
}
