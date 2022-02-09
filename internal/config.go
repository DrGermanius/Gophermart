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
)

const (
	defaultRunAddress           = "localhost:3000"
	defaultAccrualSystemAddress = "" //todo
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
}

func NewConfig() *config {
	c = new(config)

	defaultConn := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s sslmode=disable database=mart", // database=mart
		host, port, user, password)

	flag.StringVar(&c.RunAddress, "a", setEnvOrDefault(RunAddress, defaultRunAddress), "host to listen on")
	flag.StringVar(&c.AccrualSystemAddress, "r", setEnvOrDefault(DatabaseURI, defaultAccrualSystemAddress), "baseURl for short link")
	flag.StringVar(&c.DatabaseURI, "d", setEnvOrDefault(AccrualSystemAddress, defaultConn), "postgres connection path")

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
