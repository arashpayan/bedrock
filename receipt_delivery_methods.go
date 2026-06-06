package bedrock

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// LastSuccessfulDelivery returns the most recent successful email delivery for
// a receipt, or (nil, nil) if the receipt has never been delivered successfully.
func (db *DB) LastSuccessfulDelivery(receiptID ID) (*ReceiptDelivery, error) {
	query, args := db.sq.Select("*").
		From("receipt_deliveries").
		Where("receipt_id = ?", receiptID).
		Where("status = ?", DeliveryStatusSuccess).
		OrderBy("sent_at DESC").
		Limit(1).
		MustSql()

	var d ReceiptDelivery
	err := db.conn.Get(&d, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last successful delivery: %w", err)
	}
	return &d, nil
}

// RecordReceiptDelivery appends a row to the receipt email delivery log. A
// successful row marks the receipt as "sent" for the purposes of UnsentReceipts.
func (db *DB) RecordReceiptDelivery(receiptID ID, sentAt time.Time, method, recipientAddress string, status DeliveryStatus, errorMessage string) (*ReceiptDelivery, error) {
	query, args := db.sq.Insert("receipt_deliveries").
		SetMap(map[string]any{
			"receipt_id":        receiptID,
			"sent_at":           sentAt.Round(0),
			"method":            method,
			"recipient_address": recipientAddress,
			"status":            status,
			"error_message":     errorMessage,
		}).
		Suffix("RETURNING *").
		MustSql()

	var d ReceiptDelivery
	if err := db.conn.Get(&d, query, args...); err != nil {
		return nil, fmt.Errorf("failed to record receipt delivery: %w", err)
	}
	return &d, nil
}

// ReceiptDeliveries returns the full delivery history for a receipt, most
// recent first.
func (db *DB) ReceiptDeliveries(receiptID ID) ([]ReceiptDelivery, error) {
	query, args := db.sq.Select("*").
		From("receipt_deliveries").
		Where("receipt_id = ?", receiptID).
		OrderBy("sent_at DESC").
		MustSql()

	var deliveries []ReceiptDelivery
	if err := db.conn.Select(&deliveries, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get receipt deliveries: %w", err)
	}
	return deliveries, nil
}
