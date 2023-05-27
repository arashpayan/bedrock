package persistence

import (
	"context"
	"fmt"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
)

func (db *Database) CreateReceipt(ctx context.Context, in model.CreateReceiptInput) (*model.Receipt, error) {
	if len(in.Items) == 0 {
		return nil, ErrReceiptRequiresLineItems
	}

	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	var total model.Money
	for _, ri := range in.Items {
		if ri.Price < 0 {
			return nil, fmt.Errorf("receipt item has an amount less than or equal to 0: %d", ri.Price)
		}
		total += ri.Price
	}
	if total == 0 {
		return nil, ErrTotalIsZero
	}

	now := datetime.Now()
	humanID := now.Time().In(&db.Assembly.Timezone).Format("20060102150405.999")

	query, args := sq.Insert(tableReceipts).SetMap(map[string]any{
		"created_at":  now,
		"modified_at": now,

		"human_id":    humanID,
		"customer_id": in.CustomerID,
		"sold_at":     in.SoldAt,
		"total":       total,
	}).Suffix("RETURNING *").MustSql()

	var rcpt model.Receipt
	if err := tx.Get(&rcpt, query, args...); err != nil {
		return nil, err
	}

	for _, ri := range in.Items {
		_, err := sq.Insert(tableReceiptItems).SetMap(map[string]any{
			"created_at":  now,
			"modified_at": now,

			"receipt_id":  rcpt.ID,
			"item_id":     ri.ItemID,
			"description": ri.Description,
			"price":       ri.Price,
		}).RunWith(tx.Tx).Exec()
		if err != nil {
			return nil, fmt.Errorf("inserting receipt item: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &rcpt, nil
}
