package internal

import (
	"time"

	"github.com/shopspring/decimal"
)

type Withdraw struct {
	ID          int             `json:"ID"`
	OrderNumber int64           `json:"orderNumber"`
	UserID      int             `json:"userID"`
	Amount      decimal.Decimal `json:"amount"`
	ProcessedAt time.Time       `json:"processedAt"`
}

type WithdrawInput struct {
	OrderNumber int64           `json:"order"`
	Sum         decimal.Decimal `json:"sum"`
}

type WithdrawOutput struct {
	OrderNumber int64           `json:"order"`
	Sum         decimal.Decimal `json:"sum"`
	ProcessedAt time.Time       `json:"processedAt"`
}
