package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"

	jwtware "github.com/gofiber/jwt/v3"

	. "github.com/DrGermanius/Gophermart/internal"
)

func main() {
	cfg := NewConfig()
	//todo logger
	repository, err := NewRepository(cfg.DatabaseURI)
	if err != nil {
		log.Fatal(err)
	}

	service := NewService(repository)
	handlers := NewHandlers(service)

	jwtMiddleware := jwtware.New(jwtware.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired JWT")
		},
		SigningKey: []byte("secret"),
	})

	app := fiber.New()
	api := app.Group("/api")

	usr := api.Group("/user")
	usr.Post("/login", handlers.Login)
	usr.Post("/register", handlers.Register)

	usr.Get("/orders", jwtMiddleware, handlers.GetOrders)
	usr.Post("/orders", jwtMiddleware, handlers.CreateOrder)

	usr.Get("/balance", jwtMiddleware, handlers.GetBalance)

	usr.Get("/balance/withdraw", jwtMiddleware, handlers.WithdrawHistory)
	usr.Post("/balance/withdraw", jwtMiddleware, handlers.Withdraw)

	go log.Fatal(app.Listen(cfg.RunAddress))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down service...")
}
