package test

import (
	"context"
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/DrGermanius/Gophermart/internal"
	mock_internal "github.com/DrGermanius/Gophermart/internal/mock"
	"github.com/DrGermanius/Gophermart/internal/model"
)

var _ = Describe("Repository", func() {
	var (
		srv internal.IService
		acc *mock_internal.MockIAccrual
		rep *mock_internal.MockIRepository
	)
	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		logger, err := zap.NewDevelopment()
		Expect(err).ShouldNot(HaveOccurred())

		rep = mock_internal.NewMockIRepository(ctrl)
		acc = mock_internal.NewMockIAccrual(ctrl)

		srv = internal.NewService(rep, acc, "secret", logger.Sugar())
	})
	Context("Service tests", func() {
		It("Login without error", func() {
			ctx := context.Background()
			l, p := "login", "pass"
			h := internal.GetHash(p)

			rep.EXPECT().CheckCredentials(ctx, l, h).Return(1, nil)

			_, err := srv.Login(ctx, l, p)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Login with error", func() {
			ctx := context.Background()
			l, p := "login", "pass"
			h := internal.GetHash(p)

			rep.EXPECT().CheckCredentials(ctx, l, h).Return(0, errors.New("some error"))

			_, err := srv.Login(ctx, l, p)
			Expect(err).Should(HaveOccurred())
		})
		It("Register without error", func() {
			ctx := context.Background()
			l, p := "login", "pass"
			h := internal.GetHash(p)

			rep.EXPECT().IsUserExist(ctx, l).Return(false, nil)
			rep.EXPECT().Register(ctx, l, h)

			_, err := srv.Register(ctx, l, p)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Register with error already registered", func() {
			ctx := context.Background()
			l, p := "login", "pass"

			rep.EXPECT().IsUserExist(ctx, l).Return(true, nil)

			_, err := srv.Register(ctx, l, p)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrLoginIsAlreadyTaken))
		})
		It("SendOrder without error", func() {
			ctx := context.Background()
			uid := 1
			t := time.Now()

			order := model.Order{
				ID:         1,
				Number:     "79927398713",
				UserID:     -1,
				Accrual:    decimal.NewFromInt(1),
				Status:     "NEW",
				UploadedAt: t,
			}

			rep.EXPECT().GetOrderByNumber(ctx, order.Number).Return(order, nil)
			rep.EXPECT().SendOrder(ctx, order.Number, uid)
			acc.EXPECT().SendToQueue(ctx, uid, order.Number)

			err := srv.SendOrder(ctx, order.Number, uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("SendOrder with error luhn", func() {
			ctx := context.Background()
			uid := 1
			orderNumber := "1"

			err := srv.SendOrder(ctx, orderNumber, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrLuhnInvalid))
		})
		It("SendOrder with error already sent", func() {
			ctx := context.Background()
			uid := 1
			t := time.Now()

			order := model.Order{
				ID:         1,
				Number:     "79927398713",
				UserID:     1,
				Accrual:    decimal.NewFromInt(1),
				Status:     "NEW",
				UploadedAt: t,
			}

			rep.EXPECT().GetOrderByNumber(ctx, order.Number).Return(order, nil)

			err := srv.SendOrder(ctx, order.Number, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrOrderIsAlreadySent))
		})
		It("SendOrder with error already sent", func() {
			ctx := context.Background()
			uid := 1
			t := time.Now()

			order := model.Order{
				ID:         1,
				Number:     "79927398713",
				UserID:     2,
				Accrual:    decimal.NewFromInt(1),
				Status:     "NEW",
				UploadedAt: t,
			}

			rep.EXPECT().GetOrderByNumber(ctx, order.Number).Return(order, nil)

			err := srv.SendOrder(ctx, order.Number, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrOrderIsAlreadySentByOtherUser))
		})
		It("GetOrders without error", func() {
			ctx := context.Background()
			uid := 1
			o := make([]model.OrderOutput, 1, 1)

			rep.EXPECT().GetOrders(ctx, uid).Return(o, nil)

			_, err := srv.GetOrders(ctx, uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("GetOrders with error", func() {
			ctx := context.Background()
			uid := 1
			o := make([]model.OrderOutput, 0, 0)

			rep.EXPECT().GetOrders(ctx, uid).Return(o, nil)

			_, err := srv.GetOrders(ctx, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrNoRecords))
		})
		It("GetBalanceByUserID without error", func() {
			ctx := context.Background()
			uid := 1
			bw := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(1),
				Withdrawn: decimal.NewFromInt(1),
			}

			rep.EXPECT().GetBalanceByUserID(ctx, uid).Return(bw, nil)

			_, err := srv.GetBalanceByUserID(ctx, uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("GetBalanceByUserID with error", func() {
			ctx := context.Background()
			uid := 1
			bw := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(1),
				Withdrawn: decimal.NewFromInt(1),
			}
			e := errors.New("some error")

			rep.EXPECT().GetBalanceByUserID(ctx, uid).Return(bw, e)

			_, err := srv.GetBalanceByUserID(ctx, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(e))
		})
		It("Withdraw without error", func() {
			ctx := context.Background()
			uid := 1
			i := model.WithdrawInput{
				OrderNumber: "79927398713",
				Sum:         decimal.NewFromInt(10),
			}

			bw := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(10),
				Withdrawn: decimal.NewFromInt(10),
			}

			newBw := model.BalanceWithdrawn{
				Balance:   bw.Balance.Sub(i.Sum),
				Withdrawn: bw.Withdrawn.Add(i.Sum),
			}

			rep.EXPECT().GetBalanceByUserID(ctx, uid).Return(bw, nil)
			rep.EXPECT().Withdraw(ctx, i, newBw, uid).Return(nil)

			err := srv.Withdraw(ctx, i, uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Withdraw with error luhn", func() {
			ctx := context.Background()
			uid := 1
			i := model.WithdrawInput{
				OrderNumber: "1",
				Sum:         decimal.NewFromInt(10),
			}

			err := srv.Withdraw(ctx, i, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrLuhnInvalid))
		})
		It("Withdraw with error insufficient funds", func() {
			ctx := context.Background()
			uid := 1
			i := model.WithdrawInput{
				OrderNumber: "79927398713",
				Sum:         decimal.NewFromInt(100),
			}

			bw := model.BalanceWithdrawn{
				Balance:   decimal.NewFromInt(10),
				Withdrawn: decimal.NewFromInt(10),
			}

			rep.EXPECT().GetBalanceByUserID(ctx, uid).Return(bw, nil)

			err := srv.Withdraw(ctx, i, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrInsufficientFunds))
		})
		It("GetWithdrawHistory without error", func() {
			ctx := context.Background()
			uid := 1
			i := make([]model.WithdrawOutput, 1, 1)

			rep.EXPECT().GetWithdrawHistory(ctx, uid).Return(i, nil)

			_, err := srv.GetWithdrawHistory(ctx, uid)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("GetWithdrawHistory with error", func() {
			ctx := context.Background()
			uid := 1
			i := make([]model.WithdrawOutput, 0, 0)

			rep.EXPECT().GetWithdrawHistory(ctx, uid).Return(i, nil)

			_, err := srv.GetWithdrawHistory(ctx, uid)
			Expect(err).Should(HaveOccurred())
			Expect(err).Should(Equal(internal.ErrNoRecords))
		})
	})
})
