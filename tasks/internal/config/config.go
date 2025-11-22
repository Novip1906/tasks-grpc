package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	TasksAddress string `yaml:"tasks_address" env-required:"true"`
	AuthAddress  string `yaml:"auth_address" env-required:"true"`
	Params       Params `yaml:"params" env-required:"true"`
	DB           DB     `yaml:"db" env-required:"true"`
	Kafka        Kafka  `yaml:"kafka" env-required:"true"`
}

type Params struct {
	Text MinMaxLen `yaml:"text"`
}

type MinMaxLen struct {
	Min int `yaml:"min" env-required:"true"`
	Max int `yaml:"max" env-required:"true"`
}

type DB struct {
	Host     string `yaml:"host" env-default:"localhost"`
	Port     string `yaml:"port" env-default:"5431"`
	User     string `yaml:"user" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	DBName   string `yaml:"db_name" env-required:"true"`
}

type Kafka struct {
	Brokers     []string `yaml:"brokers" env-required:"true"`
	EventsTopic string   `yaml:"topic" env-required:"true"`
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
