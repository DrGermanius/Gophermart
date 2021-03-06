package internal

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/DrGermanius/Gophermart/internal/model"
)

//go:generate mockgen -source repository.go -destination ./mock/repository.go

type IRepository interface {
	Register(context.Context, string, string) (int, error)
	IsUserExist(context.Context, string) (bool, error)
	CheckCredentials(context.Context, string, string) (int, error)
	GetOrderByNumber(context.Context, string) (model.Order, error)
	SendOrder(context.Context, string, int) error
	GetOrders(context.Context, int) ([]model.OrderOutput, error)
	GetBalanceByUserID(context.Context, int) (model.BalanceWithdrawn, error)
	Withdraw(context.Context, model.WithdrawInput, model.BalanceWithdrawn, int) error
	GetWithdrawHistory(context.Context, int) ([]model.WithdrawOutput, error)
	UpdateOrderStatus(context.Context, string, string) error
	MakeAccrual(context.Context, int, string, string, decimal.Decimal, decimal.Decimal) error
}

type Repository struct {
	Conn   *sql.DB
	Logger *zap.SugaredLogger
}

func NewRepository(connString string, embedMigrations embed.FS, logger *zap.SugaredLogger) (*Repository, error) {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, err
	}

	goose.SetBaseFS(embedMigrations)

	err = goose.Up(db, "migrations")
	if err != nil {
		return nil, err
	}

	return &Repository{Conn: db, Logger: logger}, nil
}

func (r Repository) Register(ctx context.Context, login, password string) (int, error) {
	var id int
	row := r.Conn.QueryRowContext(ctx, "INSERT INTO users (login, password) VALUES ($1,$2) RETURNING id", login, password)

	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r Repository) IsUserExist(ctx context.Context, login string) (bool, error) {
	exist := false

	row := r.Conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE login=$1)", login)
	err := row.Scan(&exist)
	if err != nil {
		return false, err
	}

	return exist, nil
}

func (r Repository) CheckCredentials(ctx context.Context, login string, password string) (int, error) {
	var id int
	row := r.Conn.QueryRowContext(ctx, "SELECT id FROM users WHERE login = $1 AND password = $2", login, password)

	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidCredentials
	}
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r Repository) GetOrderByNumber(ctx context.Context, orderNumber string) (model.Order, error) {
	var o model.Order
	row := r.Conn.QueryRowContext(ctx, "SELECT number, user_id, status, uploaded_at FROM orders WHERE number = $1", orderNumber)
	err := row.Scan(&o.Number, &o.UserID, &o.Status, &o.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Order{UserID: -1}, nil
		}
		return model.Order{}, err
	}

	return o, nil
}

func (r Repository) SendOrder(ctx context.Context, orderNumber string, userID int) error {
	_, err := r.Conn.ExecContext(ctx, "INSERT INTO orders (number, user_id, status, uploaded_at) VALUES ($1, $2, $3, $4)", orderNumber, userID, model.OrderStatusNew, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}
	return nil
}

func (r Repository) GetOrders(ctx context.Context, uid int) ([]model.OrderOutput, error) {
	rows, err := r.Conn.QueryContext(ctx, "SELECT number, accrual, status, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC", uid)
	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, err
	}

	var orders []model.OrderOutput
	for rows.Next() {
		var o model.OrderOutput
		err = rows.Scan(&o.Number, &o.Accrual, &o.Status, &o.UploadedAt)
		if err != nil {
			return nil, err
		}

		orders = append(orders, o)
	}

	return orders, nil
}

func (r Repository) GetBalanceByUserID(ctx context.Context, uid int) (model.BalanceWithdrawn, error) {
	var bw model.BalanceWithdrawn

	err := r.Conn.QueryRowContext(ctx, "SELECT balance, withdrawn FROM users WHERE id = $1", uid).Scan(&bw.Balance, &bw.Withdrawn)
	if err != nil {
		return model.BalanceWithdrawn{}, err
	}

	return bw, nil
}

func (r Repository) Withdraw(ctx context.Context, i model.WithdrawInput, bw model.BalanceWithdrawn, uid int) error {
	tx, err := r.Conn.Begin()
	defer tx.Commit()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO withdraw_history (order_number, user_id, amount, processed_at) VALUES ($1, $2, $3, $4)", i.OrderNumber, uid, i.Sum, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE users SET balance = $1, withdrawn = $2 WHERE id = $3", bw.Balance, bw.Withdrawn, uid)
	if err != nil {
		return err
	}

	return nil
}

func (r Repository) GetWithdrawHistory(ctx context.Context, uid int) ([]model.WithdrawOutput, error) {
	rows, err := r.Conn.QueryContext(ctx, "SELECT order_number, amount, processed_at FROM withdraw_history WHERE user_id = $1 ORDER BY processed_at DESC", uid)
	if err != nil {
		return nil, err
	}

	if rows.Err() != nil {
		return nil, err
	}

	var wh []model.WithdrawOutput
	for rows.Next() {
		var w model.WithdrawOutput
		err = rows.Scan(&w.OrderNumber, &w.Sum, &w.ProcessedAt)
		if err != nil {
			return nil, err
		}

		wh = append(wh, w)
	}

	return wh, nil
}

func (r Repository) UpdateOrderStatus(ctx context.Context, orderNumber string, status string) error {
	_, err := r.Conn.ExecContext(ctx, "UPDATE orders SET status = $1 WHERE number = $2", status, orderNumber)
	if err != nil {
		return err
	}

	return nil
}

func (r Repository) MakeAccrual(ctx context.Context, uid int, status string, orderNumber string, accrual decimal.Decimal, balance decimal.Decimal) error {
	tx, err := r.Conn.Begin()
	defer tx.Commit()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE orders SET status = $1, accrual = $2 WHERE number = $3", status, accrual, orderNumber)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE users SET balance = $1 WHERE id = $2", balance, uid)
	if err != nil {
		return err
	}
	return nil
}
