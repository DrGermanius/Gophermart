package test_test

import (
	"context"
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Repository", func() {
	var (
		repo internal.IRepository
		mock sqlmock.Sqlmock
	)
	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		db, m, err := sqlmock.New()
		Expect(err).ShouldNot(HaveOccurred())

		mock = m
		logger, err := zap.NewDevelopment()
		Expect(err).ShouldNot(HaveOccurred())

		repo = internal.Repository{
			Conn:   db,
			Logger: logger.Sugar(),
		}

	})
	AfterEach(func() {
		err := mock.ExpectationsWereMet()
		Expect(err).ShouldNot(HaveOccurred())
	})
	Context("Repository tests", func() {
		It("GetOrders without error", func() {
			t := time.Now()
			uid := 1

			expectedOrder := model.Order{
				ID:         1,
				Number:     "100",
				UserID:     1,
				Accrual:    decimal.NewFromInt(5),
				Status:     "NEW",
				UploadedAt: t,
			}

			expectedRows := sqlmock.NewRows([]string{
				"Number",
				"Accrual",
				"Status",
				"UploadedAt",
			}).AddRow(expectedOrder.Number, expectedOrder.Accrual, expectedOrder.Status, expectedOrder.UploadedAt)

			mock.ExpectQuery("SELECT (.+) FROM orders WHERE user_id = \\$1 ORDER BY uploaded_at DESC").
				WithArgs(uid).WillReturnRows(expectedRows).RowsWillBeClosed()

			_, err := repo.GetOrders(context.Background(), uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("GetOrders with error", func() {
			uid := 1

			mock.ExpectQuery("SELECT (.+) FROM orders WHERE user_id = \\$1 ORDER BY uploaded_at DESC").
				WithArgs(uid).WillReturnError(errors.New("some error")).WillReturnRows()

			_, err := repo.GetOrders(context.Background(), uid)
			Expect(err).Should(HaveOccurred())
		})
		It("GetOrdersByID without error", func() {
			t := time.Now()
			number := "100"

			expectedOrder := model.Order{
				ID:         1,
				Number:     number,
				UserID:     1,
				Accrual:    decimal.NewFromInt(5),
				Status:     "NEW",
				UploadedAt: t,
			}

			expectedRows := sqlmock.NewRows([]string{
				"Number",
				"Accrual",
				"Status",
				"UploadedAt",
			}).AddRow(expectedOrder.Number, expectedOrder.Accrual, expectedOrder.Status, expectedOrder.UploadedAt)

			mock.ExpectQuery("SELECT (.+) FROM orders WHERE number = \\$1").
				WithArgs(number).WillReturnRows(expectedRows).RowsWillBeClosed()

			_, err := repo.GetOrderByNumber(context.Background(), number)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("GetOrdersByID with error", func() {
			number := "100"

			mock.ExpectQuery("SELECT (.+) FROM orders WHERE number = \\$1").
				WithArgs(number).WillReturnError(errors.New("some error"))

			_, err := repo.GetOrderByNumber(context.Background(), number)
			Expect(err).Should(HaveOccurred())
		})
		It("GetWithdrawHistory without error", func() {
			t := time.Now()
			uid := 1

			expectedWO := model.WithdrawOutput{
				OrderNumber: "100",
				Sum:         decimal.NewFromInt(1),
				ProcessedAt: t,
			}

			expectedRows := sqlmock.NewRows([]string{
				"OrderNumber",
				"Sum",
				"ProcessedAt",
			}).AddRow(expectedWO.OrderNumber, expectedWO.Sum, expectedWO.ProcessedAt)

			mock.ExpectQuery("SELECT (.+) FROM withdraw_history WHERE user_id = \\$1 ORDER BY processed_at DESC").
				WithArgs(uid).WillReturnRows(expectedRows).RowsWillBeClosed()

			_, err := repo.GetWithdrawHistory(context.Background(), uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("GetWithdrawHistory with error", func() {
			uid := 1

			mock.ExpectQuery("SELECT (.+) FROM withdraw_history WHERE user_id = \\$1 ORDER BY processed_at DESC").
				WithArgs(uid).WillReturnError(errors.New("some error"))

			_, err := repo.GetWithdrawHistory(context.Background(), uid)
			Expect(err).Should(HaveOccurred())
		})
		It("GetBalanceByUserID without error", func() {
			uid := 1

			expectedBW := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(1),
				Withdrawn: decimal.NewFromInt(1),
			}

			expectedRows := sqlmock.NewRows([]string{
				"Balance",
				"Withdrawn",
			}).AddRow(expectedBW.Balance, expectedBW.Withdrawn)

			mock.ExpectQuery("SELECT balance, withdrawn FROM users WHERE id = \\$1").
				WithArgs(uid).WillReturnRows(expectedRows).RowsWillBeClosed()

			_, err := repo.GetBalanceByUserID(context.Background(), uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("GetBalanceByUserID with error", func() {
			uid := 1

			mock.ExpectQuery("SELECT balance, withdrawn FROM users WHERE id = \\$1").
				WithArgs(uid).WillReturnError(errors.New("some error"))

			_, err := repo.GetBalanceByUserID(context.Background(), uid)
			Expect(err).Should(HaveOccurred())
		})
		It("SendOrder without error", func() {
			n := "name"
			p := 1

			mock.ExpectExec("INSERT INTO orders (.+) VALUES (.+)").
				WithArgs().WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.SendOrder(context.Background(), n, p)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("SendOrder with error", func() {
			n := "name"
			p := 1

			mock.ExpectExec("INSERT INTO orders (.+) VALUES (.+)").
				WithArgs().WillReturnError(errors.New("some error"))

			err := repo.SendOrder(context.Background(), n, p)
			Expect(err).Should(HaveOccurred())
		})
		It("SendOrder without error", func() {
			uid := 1
			i := model.WithdrawInput{
				OrderNumber: "1",
				Sum:         decimal.NewFromInt(1),
			}
			bw := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(1),
				Withdrawn: decimal.NewFromInt(1),
			}

			mock.ExpectBegin()

			mock.ExpectExec("INSERT INTO withdraw_history (.+) VALUES (.+)").
				WithArgs(i.OrderNumber, uid, i.Sum, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectExec("UPDATE users SET balance = \\$1, withdrawn = \\$2 WHERE id = \\$3").
				WithArgs(bw.Balance, bw.Withdrawn, uid).WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectCommit()

			err := repo.Withdraw(context.Background(), i, bw, uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("SendOrder with error", func() {
			uid := 1
			i := model.WithdrawInput{
				OrderNumber: "1",
				Sum:         decimal.NewFromInt(1),
			}
			bw := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(1),
				Withdrawn: decimal.NewFromInt(1),
			}

			mock.ExpectBegin()

			mock.ExpectExec("INSERT INTO withdraw_history (.+) VALUES (.+)").
				WithArgs().WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectExec("UPDATE users SET balance = \\$1, withdrawn = \\$2 WHERE id = \\$3").
				WithArgs().WillReturnError(errors.New("some error"))

			err := repo.Withdraw(context.Background(), i, bw, uid)
			Expect(err).Should(HaveOccurred())
		})
		It("SendOrder with other error", func() {
			uid := 1
			i := model.WithdrawInput{
				OrderNumber: "1",
				Sum:         decimal.NewFromInt(1),
			}
			bw := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(1),
				Withdrawn: decimal.NewFromInt(1),
			}

			mock.ExpectBegin()

			mock.ExpectExec("INSERT INTO withdraw_history (.+) VALUES (.+)").
				WithArgs().WillReturnError(errors.New("some error"))

			err := repo.Withdraw(context.Background(), i, bw, uid)
			Expect(err).Should(HaveOccurred())
		})
		It("MakeAccrual without error", func() {
			uid := 1
			status := "NEW"
			orderNumber := "100"
			accrual := decimal.NewFromInt(1)
			balance := decimal.NewFromInt(1)

			mock.ExpectBegin()

			mock.ExpectExec("UPDATE orders SET status = \\$1, accrual = \\$2 WHERE number = \\$3").
				WithArgs(status, accrual, orderNumber).WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectExec("UPDATE users SET balance = \\$1 WHERE id = \\$2").
				WithArgs(balance, uid).WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectCommit()

			err := repo.MakeAccrual(context.Background(), uid, status, orderNumber, accrual, balance)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("MakeAccrual with error", func() {
			uid := 1
			status := "NEW"
			orderNumber := "100"
			accrual := decimal.NewFromInt(1)
			balance := decimal.NewFromInt(1)

			mock.ExpectBegin()

			mock.ExpectExec("UPDATE orders SET status = \\$1, accrual = \\$2 WHERE number = \\$3").
				WithArgs(status, accrual, orderNumber).WillReturnError(errors.New("some error"))

			err := repo.MakeAccrual(context.Background(), uid, status, orderNumber, accrual, balance)
			Expect(err).Should(HaveOccurred())
		})
		It("MakeAccrual with other error", func() {
			uid := 1
			status := "NEW"
			orderNumber := "100"
			accrual := decimal.NewFromInt(1)
			balance := decimal.NewFromInt(1)

			mock.ExpectBegin()

			mock.ExpectExec("UPDATE orders SET status = \\$1, accrual = \\$2 WHERE number = \\$3").
				WithArgs(status, accrual, orderNumber).WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectExec("UPDATE users SET balance = \\$1 WHERE id = \\$2").
				WithArgs(balance, uid).WillReturnError(errors.New("some error"))

			err := repo.MakeAccrual(context.Background(), uid, status, orderNumber, accrual, balance)
			Expect(err).Should(HaveOccurred())
		})
		It("UpdateOrderStatus without error", func() {
			status := "NEW"
			orderNumber := "100"

			mock.ExpectExec("UPDATE orders SET status = \\$1 WHERE number = \\$2").
				WithArgs(status, orderNumber).WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.UpdateOrderStatus(context.Background(), orderNumber, status)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("UpdateOrderStatus with error", func() {
			status := "NEW"
			orderNumber := "100"

			mock.ExpectExec("UPDATE orders SET status = \\$1 WHERE number = \\$2").
				WithArgs(status, orderNumber).WillReturnError(errors.New("some error"))

			err := repo.UpdateOrderStatus(context.Background(), orderNumber, status)
			Expect(err).Should(HaveOccurred())
		})
	})
})
