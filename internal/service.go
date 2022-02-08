package internal

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/theplant/luhn"
)

type IService interface {
	Register(context.Context, string, string) (string, error)
	Login(context.Context, string, string) (string, error)
	GetJWTToken(string) (string, error)
	SentOrder(context.Context, int, int) error
	GetOrders(context.Context, int) ([]Order, error)
	GetBalanceByUserID(context.Context, int) (BalanceWithdrawn, error)
}

func NewService(Repository IRepository) *Service {
	return &Service{Repository: Repository}
}

type Service struct {
	Repository IRepository
}

func (s Service) SentOrder(ctx context.Context, orderNumber int, uid int) error {
	if !luhn.Valid(orderNumber) {
		return ErrLuhnInvalid
	}

	order, err := s.Repository.GetOrderByID(ctx, orderNumber)
	if err != nil {
		return err
	}

	if order.UserID == uid {
		return ErrOrderIsAlreadySent
	}

	if order.UserID != -1 && order.UserID != uid {
		return ErrOrderIsAlreadySentByOtherUser
	}

	err = s.Repository.SentOrder(ctx, orderNumber, uid)
	if err != nil {
		return err
	}
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

	t, err := token.SignedString([]byte("secret")) //todo secret
	if err != nil {
		return "", err
	}

	return t, nil
}

func (s Service) GetOrders(ctx context.Context, uid int) ([]Order, error) {
	orders, err := s.Repository.GetOrders(ctx, uid)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, ErrNoOrders
	}
	return orders, nil
}

func (s Service) GetBalanceByUserID(ctx context.Context, uid int) (BalanceWithdrawn, error) {
	bw, err := s.Repository.GetBalanceByUserID(ctx, uid)
	if err != nil {
		return bw, err
	}

	return bw, nil
}

func getHash(s string) string {
	h := sha256.New()
	ph := h.Sum([]byte(s))
	return base64.StdEncoding.EncodeToString(ph)
}
