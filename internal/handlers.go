package internal

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

type Handlers struct {
	Service IService
}

func NewHandlers(Service IService) *Handlers {
	return &Handlers{Service: Service}
}

func (h *Handlers) Login(c *fiber.Ctx) error {
	var i LoginInput

	if err := c.BodyParser(&i); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Error on login request", "data": err})
	}

	t, err := h.Service.Login(c.Context(), i.Login, i.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Error on login request", "data": err})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on login request", "data": err})
	}

	return c.JSON(fiber.Map{"token": t})
}

func (h *Handlers) Register(c *fiber.Ctx) error {
	var i LoginInput

	if err := c.BodyParser(&i); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Error on register request", "data": err})
	}

	t, err := h.Service.Register(c.Context(), i.Login, i.Password)
	if err != nil {
		if errors.Is(err, ErrLoginIsAlreadyTaken) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "error", "message": "Error on register request", "data": err})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Error on register request", "data": err})
	}

	return c.JSON(fiber.Map{"token": t})
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

	err = h.Service.SentOrder(c.Context(), orderNumber, uid)
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
		if errors.Is(err, ErrNoOrders) {
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

func getUserIDFromToken(c *fiber.Ctx) (int, error) {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	id := claims["id"].(string)
	return strconv.Atoi(id)
}
