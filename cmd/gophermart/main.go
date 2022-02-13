package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	app "github.com/DrGermanius/Gophermart/internal"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	//decimals at json as string
	//https://github.com/shopspring/decimal/issues/21
	decimal.MarshalJSONWithoutQuotes = true

	cfg := app.NewConfig()
	z, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	sugaredLogger := z.Sugar()

	repository, err := app.NewRepository(cfg.DatabaseURI, embedMigrations, sugaredLogger)
	if err != nil {
		sugaredLogger.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	accrualService := app.NewAccrualService(repository, cfg.AccrualSystemAddress, ctx, sugaredLogger)
	service := app.NewService(repository, *accrualService, sugaredLogger) //todo pointer??
	handlers := app.NewHandlers(service, sugaredLogger)

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
