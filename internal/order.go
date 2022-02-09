package internal

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	OrderStatusRegistered = "REGISTERED"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusInvalid    = "INVALID"
	OrderStatusProcessed  = "PROCESSED"
)

type Order struct {
	ID         int             `json:"ID"`
	Number     int64           `json:"number"`
	UserID     int             `json:"userID"`
	Accrual    decimal.Decimal `json:"accrual"`
	Status     string          `json:"status"`
	UploadedAt time.Time       `json:"uploadedAt"`
}

type OrderOutput struct {
	Number     int64           `json:"number"`
	Status     string          `json:"status"`
	Accrual    decimal.Decimal `json:"accrual"`
	UploadedAt time.Time       `json:"uploadedAt"`
}
