package internal

import "github.com/shopspring/decimal"

type User struct {
	ID       int
	Login    string
	Password string
	Balance  decimal.Decimal
}

type LoginInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type BalanceWithdrawn struct {
	Balance   decimal.Decimal `json:"balance"`
	Withdrawn decimal.Decimal `json:"withdrawn"`
}