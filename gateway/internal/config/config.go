package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Address      string `yaml:"address" env-default:":8080"`
	AuthAddress  string `yaml:"auth-address" env-required:"true"`
	TasksAddress string `yaml:"tasks-address" env-required:"true"`
}

func MustLoadConfig() *Config {
	godotenv.Load()

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
