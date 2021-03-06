package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/DrGermanius/Gophermart/internal/model"
)

//go:generate mockgen -source accrual.go -destination ./mock/accrual.go

type IAccrual interface {
	Run()
	SendToQueue(context.Context, int, string)
	ProcessAccrual(context.Context, int, string)
}

type AccrualService struct {
	repo   IRepository
	url    string
	ch     chan input
	ctx    context.Context
	logger *zap.SugaredLogger
}

func NewAccrualService(repo IRepository, url string, ctx context.Context, logger *zap.SugaredLogger) IAccrual {
	s := &AccrualService{
		repo:   repo,
		url:    url,
		ch:     make(chan input),
		ctx:    ctx,
		logger: logger,
	}

	go s.Run()
	return s
}

type input struct {
	uid         int
	orderNumber string
	ctx         context.Context
}

func (s AccrualService) Run() {
	for {
		select {
		case v := <-s.ch:
			s.ProcessAccrual(v.ctx, v.uid, v.orderNumber)
			time.Sleep(1 * time.Second) // avoid too many requests
		case <-s.ctx.Done():
			s.logger.Info("context is done")
			return
		}
	}
}

func (s AccrualService) SendToQueue(ctx context.Context, uid int, orderNumber string) {
	s.ch <- input{
		uid:         uid,
		orderNumber: orderNumber,
		ctx:         ctx,
	}
}

type accrualResponse struct {
	Order   string          `json:"order"`
	Status  string          `json:"status"`
	Accrual decimal.Decimal `json:"accrual,omitempty"`
}

func (s AccrualService) ProcessAccrual(ctx context.Context, uid int, orderNumber string) {
	body, err := s.makeRequest(orderNumber)
	if err != nil {
		if errors.Is(err, ErrTooManyRequests) {
			go s.SendToQueue(ctx, uid, orderNumber)
		}
		s.logger.Errorf("ProcessAccrual error: %s", err.Error())
		return
	}

	res := accrualResponse{}

	err = json.Unmarshal(body, &res)
	if err != nil {
		s.logger.Errorf("json.Unmarshal ProcessAccrual error: %s", err.Error())
		return
	}

	if res.Status == model.OrderStatusRegistered || res.Status == model.OrderStatusProcessing {
		err = s.repo.UpdateOrderStatus(ctx, orderNumber, res.Status)
		if err != nil {
			s.logger.Errorf("ProcessAccrual error: %s", err.Error())
			return
		}
		go s.SendToQueue(ctx, uid, orderNumber)
		return
	}

	bw, err := s.repo.GetBalanceByUserID(ctx, uid)
	if err != nil {
		s.logger.Errorf("ProcessAccrual error: %s", err.Error())
		return
	}

	newBalance := bw.Balance.Add(res.Accrual)

	err = s.repo.MakeAccrual(ctx, uid, res.Status, orderNumber, res.Accrual, newBalance)
	if err != nil {
		s.logger.Errorf("ProcessAccrual error: %s", err.Error())
		return
	}
}

func (s AccrualService) makeRequest(orderNumber string) ([]byte, error) {
	client := &http.Client{}

	url := s.url + "/api/orders/" + orderNumber
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Length", "0")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ErrTooManyRequests
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, res.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
