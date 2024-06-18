package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

// секретный ключ. Если не указали переменную окружения
const (
	secret = "my gophermart secret"
)

// переменные для хранения значений флагов приложения
var (
	addressApp            string
	databaseURI           string
	acrcuralSystemAddress string
)

// Config кастомный конфиг приложения.Чтобы случайно не поменяли значение, делаем их неэкспортируемыми
type Config struct {
	addressApp            string
	databaseURI           string
	accruralSystemAddress string
	secret                string
}

func (c Config) CookieTokenName() string {
	return "app_token"
}
func (c Config) AddressApp() string {
	return c.addressApp
}
func (c Config) DatabaseURI() string {
	return c.databaseURI
}
func (c Config) AccruralSystemAddress() string {
	return c.accruralSystemAddress
}
func (c Config) Secret() string {
	return c.secret
}

// PublicConfig публичный кастомный конфиг приложения
type PublicConfig struct {
	AddressApp            string `env:"RUN_ADDRESS"`
	DatabaseURI           string `env:"DATABASE_URI"`
	AccruralSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	Secret                string `env:"SECRET"`
}

// LoadConfig загрузка конфигурации. В приоритете будут переменные окружения
func LoadConfig() (Config, error) {

	// чтение флагов командной строки
	flag.StringVar(&addressApp, "a", "", "run address app")
	flag.StringVar(&databaseURI, "d", "", "database URI")
	flag.StringVar(&acrcuralSystemAddress, "r", "", "accrural system address")
	flag.Parse()

	pcfg := PublicConfig{}

	// чтение переменных окружения
	err := env.Parse(&pcfg)
	if err != nil {
		return Config{}, err
	}

	// замена значений при отсутсвии переменных окружения
	if pcfg.AddressApp == "" {
		pcfg.AddressApp = addressApp
	}
	if pcfg.DatabaseURI == "" {
		pcfg.DatabaseURI = databaseURI
	}
	if pcfg.AccruralSystemAddress == "" {
		pcfg.AccruralSystemAddress = acrcuralSystemAddress
	}
	if pcfg.Secret == "" {
		pcfg.Secret = secret
	}
	return Config{
		addressApp:            pcfg.AddressApp,
		databaseURI:           pcfg.DatabaseURI,
		accruralSystemAddress: pcfg.AccruralSystemAddress,
		secret:                pcfg.Secret,
	}, nil
}

// NewConfig если хотим задать вручную (для тестов)
func NewConfig(addressApp string, databaseURI string, accruralSystemAddress string, secret string) Config {
	return Config{
		addressApp:            addressApp,
		databaseURI:           databaseURI,
		accruralSystemAddress: accruralSystemAddress,
		secret:                secret,
	}
}
