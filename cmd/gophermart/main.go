package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	. "github.com/DrGermanius/Gophermart/internal"
)

func main() {
	//decimals at json as string
	//https://github.com/shopspring/decimal/issues/21
	decimal.MarshalJSONWithoutQuotes = true

	cfg := NewConfig()
	z, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	sugaredLogger := z.Sugar()

	repository, err := NewRepository(cfg.DatabaseURI, sugaredLogger)
	if err != nil {
		sugaredLogger.Fatal(err)
	}

	service := NewService(repository, sugaredLogger)
	handlers := NewHandlers(service, sugaredLogger)

	//jwtMiddleware := jwtware.New(jwtware.Config{
	//	ErrorHandler: func(c *fiber.Ctx, err error) error {
	//		return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired JWT")
	//	},
	//	SigningKey: []byte("secret"),
	//})

	app := fiber.New()
	app.Use(logger.New())

	api := app.Group("/api")

	usr := api.Group("/user")
	usr.Post("/login", handlers.Login)
	usr.Post("/register", handlers.Register)

	usr.Get("/orders", handlers.GetOrders)
	usr.Post("/orders", handlers.CreateOrder)

	usr.Get("/balance", handlers.GetBalance)

	usr.Get("/balance/withdraw", handlers.WithdrawHistory)
	usr.Post("/balance/withdraw", handlers.Withdraw)

	go sugaredLogger.Fatal(app.Listen(cfg.RunAddress))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	sugaredLogger.Info("Shutting down service...")
}
