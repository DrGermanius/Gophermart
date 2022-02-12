package internal

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Handlers struct {
	Service IService
	logger  *zap.SugaredLogger
}

func NewHandlers(Service IService, logger *zap.SugaredLogger) *Handlers {
	return &Handlers{Service: Service, logger: logger}
}

func (h *Handlers) Login(c *fiber.Ctx) error {
	var i LoginInput

	if err := c.BodyParser(&i); err != nil {
		h.logger.Errorf("Error on login request: %s", err.Error())
		return c.SendStatus(fiber.StatusBadRequest)
	}

	t, err := h.Service.Login(c.Context(), i.Login, i.Password)
	if err != nil {
		h.logger.Errorf("Error on login request: %s", err.Error())
		if errors.Is(err, ErrInvalidCredentials) {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	setAuthCookie(c, t)
	return c.SendStatus(fiber.StatusOK)
}

func (h *Handlers) Register(c *fiber.Ctx) error {
	var i LoginInput

	if err := c.BodyParser(&i); err != nil {
		h.logger.Errorf("Error on register request: %s", err.Error())
		return c.SendStatus(fiber.StatusBadRequest)
	}

	t, err := h.Service.Register(c.Context(), i.Login, i.Password)
	if err != nil {
		h.logger.Errorf("Error on register request: %s", err.Error())
		if errors.Is(err, ErrLoginIsAlreadyTaken) {
			return c.SendStatus(fiber.StatusConflict)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	setAuthCookie(c, t)
	return c.SendStatus(fiber.StatusOK)
}

func (h *Handlers) CreateOrder(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		h.logger.Errorf("Error on CreateOrder request: %s", err.Error())
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	if c.GetReqHeaders()["Content-Type"] != "text/plain" {
		h.logger.Errorf("Error on CreateOrder request: %s", "incorrect Content-Type")
		return c.SendStatus(fiber.StatusBadRequest)
	}

	orderNumber := string(c.Body())
	if err != nil {
		h.logger.Errorf("Error on CreateOrder request: %s", err.Error())
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	err = h.Service.SendOrder(c.Context(), orderNumber, uid)
	if err != nil {
		h.logger.Errorf("Error on CreateOrder request: %s", err.Error())
		if errors.Is(err, ErrLuhnInvalid) {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}
		if errors.Is(err, ErrOrderIsAlreadySent) {
			return c.SendStatus(fiber.StatusOK)
		}
		if errors.Is(err, ErrOrderIsAlreadySentByOtherUser) {
			return c.SendStatus(fiber.StatusConflict)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusAccepted)
}

func (h *Handlers) GetOrders(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	orders, err := h.Service.GetOrders(c.Context(), uid)
	if err != nil {
		h.logger.Errorf("Error on GetOrders request: %s", err.Error())
		if errors.Is(err, ErrNoRecords) {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	h.logger.Infof("ORDERS: %s", orders)
	return c.Status(fiber.StatusOK).JSON(orders)
}

func (h *Handlers) GetBalance(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		h.logger.Errorf("Error on GetBalance request: %s", err.Error())
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	bw, err := h.Service.GetBalanceByUserID(c.Context(), uid)
	if err != nil {
		h.logger.Errorf("Error on GetBalance request: %s", err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	h.logger.Infof("BALANCE: %s", bw)
	return c.Status(fiber.StatusOK).JSON(bw)
}

func (h *Handlers) Withdraw(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		h.logger.Errorf("Error on Withdraw request: %s", err.Error())
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var i WithdrawInput

	if err = c.BodyParser(&i); err != nil || i.OrderNumber == "" || i.Sum.Equal(decimal.NewFromInt(0)) { //todo beautify?
		return c.SendStatus(fiber.StatusBadRequest)
	}

	err = h.Service.Withdraw(c.Context(), i, uid)
	if err != nil {
		h.logger.Errorf("Error on Withdraw request: %s", err.Error())
		if errors.Is(err, ErrLuhnInvalid) {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}
		if errors.Is(err, ErrInsufficientFunds) {
			return c.SendStatus(fiber.StatusPaymentRequired)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handlers) WithdrawHistory(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		h.logger.Errorf("Error on WithdrawHistory request: %s", err.Error())
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	wh, err := h.Service.GetWithdrawHistory(c.Context(), uid)
	if err != nil {
		h.logger.Errorf("Error on WithdrawHistory request: %s", err.Error())
		if errors.Is(err, ErrNoRecords) {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	h.logger.Infof("WITHDRAWS: %s", wh)
	return c.Status(fiber.StatusOK).JSON(wh)
}

func setAuthCookie(c *fiber.Ctx, token string) {
	cookie := &fiber.Cookie{
		Name:    "token",
		Value:   token,
		Path:    "/",
		MaxAge:  100,
		Expires: time.Now().Add(24 * time.Hour),
	}

	c.Cookie(cookie)
}

func getUserIDFromToken(c *fiber.Ctx) (int, error) {
	tokenString := c.Cookies("token")
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		return 0, err
	}

	id := claims["id"].(string)

	return strconv.Atoi(id)
}
