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
	cgf := NewConfig()
	repository, err := NewRepository(cgf.DatabaseURI)
	if err != nil {
		log.Fatal(err)
	}

	service := NewService(repository)
	handlers := NewHandlers(service)

	jwtMiddleware := jwtware.New(jwtware.Config{ //todo when jwt == nil return 400 not 401
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

	go log.Fatal(app.Listen(":3000"))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down service...")
}
