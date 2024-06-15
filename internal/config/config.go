package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

const (
	secret = "my gophermart secret"
)

var (
	addressApp            string
	databaseURI           string
	acrcuralSystemAddress string
)

type Config struct {
	AddressApp            string `env:"RUN_ADDRESS"`
	DatabaseURI           string `env:"DATABASE_URI"`
	AccruralSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	Secret                string `env:"SECRET"`
}

func LoadConfig() (Config, error) {
	flag.StringVar(&addressApp, "a", "", "run address app")
	flag.StringVar(&databaseURI, "d", "", "database URI")
	flag.StringVar(&acrcuralSystemAddress, "r", "", "accrural system address")

	flag.Parse()

	cfg := Config{}

	err := env.Parse(&cfg)
	if err != nil {
		return Config{}, err
	}
	if cfg.AddressApp == "" {
		cfg.AddressApp = addressApp
	}
	if cfg.DatabaseURI == "" {
		cfg.DatabaseURI = databaseURI
	}
	if cfg.AccruralSystemAddress == "" {
		cfg.AccruralSystemAddress = acrcuralSystemAddress
	}
	if cfg.Secret == "" {
		cfg.Secret = secret
	}
	return cfg, nil
}
