package internal

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/theplant/luhn"
	"go.uber.org/zap"

	"github.com/DrGermanius/Gophermart/internal/model"
)

//go:generate mockgen -source service.go -destination ./mock/service.go

type IService interface {
	Register(context.Context, string, string) (string, error)
	Login(context.Context, string, string) (string, error)
	GetJWTToken(string) (string, error)
	SendOrder(context.Context, string, int) error
	GetOrders(context.Context, int) ([]model.OrderOutput, error)
	GetBalanceByUserID(context.Context, int) (model.BalanceWithdrawn, error)
	Withdraw(context.Context, model.WithdrawInput, int) error
	GetWithdrawHistory(context.Context, int) ([]model.WithdrawOutput, error)
}

func NewService(Repository IRepository, AccrualService IAccrual, secret string, logger *zap.SugaredLogger) *Service {
	return &Service{Repository: Repository, AccrualService: AccrualService, secret: secret, logger: logger}
}

type Service struct {
	Repository     IRepository
	AccrualService IAccrual
	secret         string
	logger         *zap.SugaredLogger
}

func (s Service) SendOrder(ctx context.Context, orderNumber string, uid int) error {
	o, err := strconv.Atoi(orderNumber)
	if err != nil {
		return err
	}

	if !luhn.Valid(o) {
		return ErrLuhnInvalid
	}

	order, err := s.Repository.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		return err
	}

	if order.UserID == uid {
		return ErrOrderIsAlreadySent
	}

	if order.UserID != -1 && order.UserID != uid {
		return ErrOrderIsAlreadySentByOtherUser
	}

	err = s.Repository.SendOrder(ctx, orderNumber, uid)
	if err != nil {
		return err
	}

	go s.AccrualService.SendToQueue(ctx, uid, orderNumber)
	return nil
}

func (s Service) Register(ctx context.Context, login, password string) (string, error) {
	exist, err := s.Repository.IsUserExist(ctx, login)
	if err != nil {
		return "", err
	}

	if exist {
		return "", ErrLoginIsAlreadyTaken
	}

	h := getHash(password)
	id, err := s.Repository.Register(ctx, login, h)
	if err != nil {
		return "", err
	}

	token, err := s.GetJWTToken(strconv.Itoa(id))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s Service) Login(ctx context.Context, login, password string) (string, error) {
	h := getHash(password)
	id, err := s.Repository.CheckCredentials(ctx, login, h)
	if err != nil {
		return "", err
	}

	token, err := s.GetJWTToken(strconv.Itoa(id))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s Service) GetJWTToken(uid string) (string, error) {
	claims := jwt.MapClaims{
		"id":  uid,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString([]byte(s.secret)) //todo secret
	if err != nil {
		return "", err
	}

	return t, nil
}

func (s Service) GetOrders(ctx context.Context, uid int) ([]model.OrderOutput, error) {
	orders, err := s.Repository.GetOrders(ctx, uid)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, ErrNoRecords
	}
	return orders, nil
}

func (s Service) GetBalanceByUserID(ctx context.Context, uid int) (model.BalanceWithdrawn, error) {
	bw, err := s.Repository.GetBalanceByUserID(ctx, uid)
	if err != nil {
		return bw, err
	}

	return bw, nil
}

func (s Service) Withdraw(ctx context.Context, i model.WithdrawInput, uid int) error {
	//todo is we need mutex here?

	o, err := strconv.Atoi(i.OrderNumber)
	if err != nil {
		return err
	}

	if !luhn.Valid(o) {
		return ErrLuhnInvalid
	}

	bw, err := s.Repository.GetBalanceByUserID(ctx, uid) //todo can be called by 2 goroutines at same time?
	if err != nil {
		return err
	}

	if bw.Balance.LessThan(i.Sum) {
		return ErrInsufficientFunds
	}

	newBw := model.BalanceWithdrawn{
		Balance:   bw.Balance.Sub(i.Sum),
		Withdrawn: bw.Withdrawn.Add(i.Sum),
	}

	err = s.Repository.Withdraw(ctx, i, newBw, uid)
	if err != nil {
		return err
	}

	return nil
}

func (s Service) GetWithdrawHistory(ctx context.Context, uid int) ([]model.WithdrawOutput, error) {
	wh, err := s.Repository.GetWithdrawHistory(ctx, uid)
	if err != nil {
		return nil, err
	}

	if len(wh) == 0 {
		return nil, ErrNoRecords
	}
	return wh, nil
}

func getHash(s string) string {
	h := sha256.New()
	ph := h.Sum([]byte(s))
	return base64.StdEncoding.EncodeToString(ph)
}
