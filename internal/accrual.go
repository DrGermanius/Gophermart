package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type AccrualService struct {
	repo   IRepository
	url    string
	ch     chan input
	ctx    context.Context
	logger *zap.SugaredLogger
}

func NewAccrualService(repo IRepository, url string, ctx context.Context, logger *zap.SugaredLogger) *AccrualService {
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

func (s AccrualService) ProcessAccrual(ctx context.Context, uid int, orderNumber string) {
	//todo mutex?????
	body, err := s.makeRequest(orderNumber)
	if err != nil {
		s.logger.Errorf("ProcessAccrual error: %s", err.Error())
		return
	}

	res := accrualResponse{}

	err = json.Unmarshal(body, &res)
	if err != nil {
		if errors.Is(err, ErrTooManyRequests) {
			go s.SendToQueue(ctx, uid, orderNumber)
		}
		s.logger.Errorf("ProcessAccrual error: %s", err.Error())
		return
	}

	if res.Status == OrderStatusRegistered || res.Status == OrderStatusProcessing {
		err = s.repo.UpdateOrderStatus(ctx, orderNumber, res.Status)
		go s.SendToQueue(ctx, uid, orderNumber)
		return
	}

	bw, err := s.repo.GetBalanceByUserID(ctx, uid)
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

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrTooManyRequests
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, res.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type accrualResponse struct {
	Order   string          `json:"order"`
	Status  string          `json:"status"`
	Accrual decimal.Decimal `json:"accrual,omitempty"`
}
