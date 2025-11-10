package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Address      string `yaml:"address" env-required:"true"`
	JWTSecretKey string `yaml:"jwt_secret_key" env-required:"true"`
	Params       Params `yaml:"params"`
}

type Params struct {
	Username MinMaxLen `yaml:"username"`
	Password MinMaxLen `yaml:"password"`
}

type MinMaxLen struct {
	Min int `yaml:"min" env-required:"true"`
	Max int `yaml:"max" env-required:"true"`
}

func MustLoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		panic("No config path in env")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic(err)
	}
	return &cfg
}
