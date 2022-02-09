package internal

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

const (
	orderFields    = "number, accrual, status, uploaded_at"
	withdrawFields = "order_number, amount, processed_at"
)

type IRepository interface {
	Register(context.Context, string, string) (int, error)
	IsUserExist(context.Context, string) (bool, error)
	CheckCredentials(context.Context, string, string) (int, error)
	GetOrderByID(context.Context, string) (Order, error)
	SendOrder(context.Context, string, int) error
	GetOrders(context.Context, int) ([]OrderOutput, error)
	GetBalanceByUserID(context.Context, int) (BalanceWithdrawn, error)
	Withdraw(context.Context, WithdrawInput, BalanceWithdrawn, int) error
	GetWithdrawHistory(context.Context, int) ([]WithdrawOutput, error)
}

type Repository struct {
	conn   *pgxpool.Pool
	logger *zap.SugaredLogger
}

func NewRepository(connString string, logger *zap.SugaredLogger) (*Repository, error) {
	conn, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}

	err = createDatabaseAndTable(conn) //todo migrations?
	if err != nil {
		return nil, err
	}

	return &Repository{conn: conn, logger: logger}, nil
}

func createDatabaseAndTable(c *pgxpool.Pool) error {
	_, err := c.Exec(context.Background(), "CREATE DATABASE mart2")
	if err != nil {
		return err
	}
	_, err = c.Exec(context.Background(), `CREATE TABLE users
	(
		id       SERIAL PRIMARY KEY,
		login    VARCHAR(255)    NOT NULL,
		password VARCHAR(255)    NOT NULL,
		balance  DECIMAL(36, 18) NOT NULL DEFAULT 0.0
	);
	
	CREATE TABLE orders
	(
		id          SERIAL PRIMARY KEY,
		number      VARCHAR(255) UNIQUE   NOT NULL,
		user_id     INT             NOT NULL REFERENCES users,
		accrual     DECIMAL(36, 18) NOT NULL DEFAULT 0.0,
		status      VARCHAR(255)    NOT NULL,
		uploaded_at TIMESTAMP       NOT NULL
	);
	
	CREATE TABLE withdraw_history
	(
		id           SERIAL PRIMARY KEY,
		order_number VARCHAR(255)          NOT NULL REFERENCES orders (number),
		user_id      INT             NOT NULL REFERENCES users,
		amount       DECIMAL(36, 18) NOT NULL DEFAULT 0.0,
		withdrawn    DECIMAL(36, 18) NOT NULL DEFAULT 0.0
	);`)
	if err != nil {
		return err
	}
	return nil
}

func (r Repository) Register(ctx context.Context, login, password string) (int, error) {
	var id int
	row := r.conn.QueryRow(ctx, "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id", login, password)

	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r Repository) IsUserExist(ctx context.Context, login string) (bool, error) {
	exist := false

	row := r.conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE  login=$1)", login)
	err := row.Scan(&exist)
	if err != nil {
		return false, err
	}

	return exist, nil
}

func (r Repository) CheckCredentials(ctx context.Context, login string, password string) (int, error) {
	var id int
	row := r.conn.QueryRow(ctx, "SELECT id FROM users WHERE login = $1 AND password = $2", login, password)

	err := row.Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvalidCredentials
	}
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r Repository) GetOrderByID(ctx context.Context, orderNumber string) (Order, error) {
	var o Order
	row := r.conn.QueryRow(ctx, "SELECT "+orderFields+" FROM orders WHERE number = $1", orderNumber) //todo sqlx
	err := row.Scan(&o.Number, &o.Accrual, &o.Status, &o.UploadedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{UserID: -1}, nil //todo magic number
		}
		return Order{}, err
	}

	return o, nil
}

func (r Repository) SendOrder(ctx context.Context, orderNumber string, userID int) error {
	_, err := r.conn.Exec(ctx, "INSERT INTO orders (number, user_id, status, uploaded_at) VALUES ($1, $2, $3, $4)", orderNumber, userID, OrderStatusRegistered, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}
	return nil
}

func (r Repository) GetOrders(ctx context.Context, uid int) ([]OrderOutput, error) {
	rows, err := r.conn.Query(ctx, "SELECT "+orderFields+" FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC", uid)
	if err != nil {
		return nil, err
	}

	var orders []OrderOutput
	for rows.Next() {
		var o OrderOutput
		err = rows.Scan(&o.Number, &o.Accrual, &o.Status, &o.UploadedAt)
		if err != nil {
			return nil, err
		}

		orders = append(orders, o)
	}

	return orders, nil
}

func (r Repository) GetBalanceByUserID(ctx context.Context, uid int) (BalanceWithdrawn, error) {
	var bw BalanceWithdrawn

	err := r.conn.QueryRow(ctx, "SELECT balance, withdrawn FROM users WHERE id = $1", uid).Scan(&bw.Balance, &bw.Withdrawn)
	if err != nil {
		return BalanceWithdrawn{}, err
	}

	return bw, nil
}

func (r Repository) Withdraw(ctx context.Context, i WithdrawInput, bw BalanceWithdrawn, uid int) error {
	tx, err := r.conn.Begin(ctx)
	defer tx.Commit(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "INSERT INTO withdraw_history (order_number, user_id, amount, processed_at) VALUES ($1, $2, $3, $4)", i.OrderNumber, uid, i.Sum, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE users SET balance = $1, withdrawn = $2 WHERE id = $3", bw.Balance, bw.Withdrawn, uid)
	if err != nil {
		return err
	}

	return nil
}

func (r Repository) GetWithdrawHistory(ctx context.Context, uid int) ([]WithdrawOutput, error) {
	rows, err := r.conn.Query(ctx, "SELECT "+withdrawFields+" FROM withdraw_history WHERE user_id = $1 ORDER BY processed_at DESC", uid)
	if err != nil {
		return nil, err
	}

	var wh []WithdrawOutput
	for rows.Next() {
		var w WithdrawOutput
		err = rows.Scan(&w.OrderNumber, &w.Sum, &w.ProcessedAt)
		if err != nil {
			return nil, err
		}

		wh = append(wh, w)
	}

	return wh, nil
}
