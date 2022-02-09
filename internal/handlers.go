package internal

import (
	"errors"
	"strconv"

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

	return c.JSON(t)
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

	return c.JSON(t)
}

func (h *Handlers) CreateOrder(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	if c.GetReqHeaders()["Content-Type"] != "text/plain" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Error on create order request", "data": "incorrect request format"})
	}

	orderNumber, err := strconv.Atoi(string(c.Body()))
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"status": "error", "message": "Error on create order request", "data": err})
	}

	err = h.Service.SendOrder(c.Context(), orderNumber, uid)
	if err != nil {
		if errors.Is(err, ErrLuhnInvalid) {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}
		if errors.Is(err, ErrOrderIsAlreadySent) {
			return c.SendStatus(fiber.StatusOK)
		}
		if errors.Is(err, ErrOrderIsAlreadySentByOtherUser) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "error", "message": "Error on sending request", "data": err})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	return c.SendStatus(fiber.StatusAccepted)
}

func (h *Handlers) GetOrders(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	orders, err := h.Service.GetOrders(c.Context(), uid)
	if err != nil {
		if errors.Is(err, ErrNoRecords) {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	return c.Status(fiber.StatusOK).JSON(orders)
}

func (h *Handlers) GetBalance(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	bw, err := h.Service.GetBalanceByUserID(c.Context(), uid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	return c.Status(fiber.StatusOK).JSON(bw)
}

func (h *Handlers) Withdraw(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	var i WithdrawInput

	if err = c.BodyParser(&i); err != nil || i.OrderNumber == 0 || i.Sum.Equal(decimal.NewFromInt(0)) { //todo beautify?
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Error on register request", "data": err})
	}

	err = h.Service.Withdraw(c.Context(), i, uid)
	if err != nil {
		if errors.Is(err, ErrLuhnInvalid) {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}
		if errors.Is(err, ErrInsufficientFunds) {
			return c.SendStatus(fiber.StatusPaymentRequired)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on sending order request", "data": err})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handlers) WithdrawHistory(c *fiber.Ctx) error {
	uid, err := getUserIDFromToken(c)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	wh, err := h.Service.GetWithdrawHistory(c.Context(), uid)
	if err != nil {
		if errors.Is(err, ErrNoRecords) {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(wh)
}

func getUserIDFromToken(c *fiber.Ctx) (int, error) {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	id := claims["id"].(string)
	return strconv.Atoi(id)
}
