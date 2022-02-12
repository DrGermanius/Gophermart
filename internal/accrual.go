package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type AccrualService struct {
	repo   IRepository
	logger *zap.SugaredLogger
	url    string
}

func NewAccrualService(repo IRepository, logger *zap.SugaredLogger, url string) *AccrualService {
	return &AccrualService{repo: repo, logger: logger, url: url}
}

func (s AccrualService) GetAccrual(ctx context.Context, uid int, orderNumber string) {
	//todo mutex?????
	//todo restart when: 1) too many requests; 2) status order != INVALID || != PROCESSED
	body, err := s.makeRequest(orderNumber)
	if err != nil {
		s.logger.Errorf("GetAccrual error: %s", err.Error())
		return
	}

	res := accrualResponse{}

	err = json.Unmarshal(body, &res)
	if err != nil {
		s.logger.Errorf("GetAccrual error: %s", err.Error())
		return
	}

	if res.Accrual.Equal(decimal.NewFromInt(0)) {
		err = s.repo.UpdateOrderStatus(ctx, orderNumber, res.Status)
		//return
	}

	bw, err := s.repo.GetBalanceByUserID(ctx, uid)
	newBalance := bw.Balance.Add(res.Accrual)

	err = s.repo.MakeAccrual(ctx, uid, res.Status, orderNumber, res.Accrual, newBalance)
	if err != nil {
		s.logger.Errorf("GetAccrual error: %s", err.Error())
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
	req.Header.Set("Content-Length,", "0")

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
