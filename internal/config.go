package internal

import (
	"flag"
	"fmt"
	"os"
)

var c *config

const (
	RunAddress           = "RUN_ADDRESS"
	DatabaseURI          = "DATABASE_URI"
	AccrualSystemAddress = "ACCRUAL_SYSTEM_ADDRESS"
	JWTSecret            = "JWT_Secret"
)

const (
	defaultRunAddress           = "localhost:8081"
	defaultAccrualSystemAddress = "http://localhost:8080"
	defaultJWTSecret            = "secret"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "12345"
)

type config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecret            string
}

func NewConfig() *config {
	c = new(config)

	defaultConn := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s sslmode=disable", //database=mart
		host, port, user, password)

	flag.StringVar(&c.RunAddress, "a", setEnvOrDefault(RunAddress, defaultRunAddress), "host to listen on")
	flag.StringVar(&c.DatabaseURI, "d", setEnvOrDefault(DatabaseURI, defaultConn), "postgres connection path")
	flag.StringVar(&c.AccrualSystemAddress, "r", setEnvOrDefault(AccrualSystemAddress, defaultAccrualSystemAddress), "Accrual system address")
	flag.StringVar(&c.JWTSecret, "s", setEnvOrDefault(JWTSecret, defaultJWTSecret), "JWT secret")

	flag.Parse()
	return c
}

func setEnvOrDefault(env, def string) string {
	res, e := os.LookupEnv(env)
	if !e {
		res = def
	}
	return res
}
