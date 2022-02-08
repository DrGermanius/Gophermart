package internal

import "errors"

var (
	ErrLoginIsAlreadyTaken = errors.New("login is already taken")
	ErrInvalidCredentials  = errors.New("invalid credentials")

	ErrOrderIsAlreadySent            = errors.New("order is already sent")
	ErrOrderIsAlreadySentByOtherUser = errors.New("order is already sent by other user")
	ErrNoOrders                      = errors.New("no orders")

	ErrLuhnInvalid = errors.New("number invalid by luhn")
)
