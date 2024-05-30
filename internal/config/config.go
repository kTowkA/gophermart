package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

var (
	addressApp           string
	databaseURI          string
	accuralSystemAddress string
)

type Config struct {
	AddressApp           string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccuralSystemAddress string `env:"ACCURAL_SYSTEM_ADDRESS"`
}

func LoadConfig() (Config, error) {
	flag.StringVar(&addressApp, "a", "", "run address app")
	flag.StringVar(&databaseURI, "d", "", "database URI")
	flag.StringVar(&accuralSystemAddress, "r", "", "accural system address")

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
	if cfg.AccuralSystemAddress == "" {
		cfg.AccuralSystemAddress = accuralSystemAddress
	}
	return Config{}, nil
}
