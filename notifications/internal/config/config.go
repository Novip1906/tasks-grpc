package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Address string `yaml:"address" env-required:"true"`
	Env     string `yaml:"env" env-default:"dev"`
	SMTP    SMTP   `yaml:"smtp" env-required:"true"`
	Kafka   Kafka  `yaml:"kafka" env-required:"true"`
}

type SMTP struct {
	Email    string `yaml:"email" env-required:"true"`
	Password string
	Host     string `yaml:"host" env-required:"true"`
	Port     int    `yaml:"port" env-required:"true"`
}

type Kafka struct {
	Brokers                []string `yaml:"brokers" env-required:"true"`
	GroupId                string   `yaml:"group_id" env-required:"true"`
	EmailVerificationTopic string   `yaml:"email_verification_topic" env-default:"email-verification"`
	EventsTopic            string   `yaml:"events_topic" env-default:"events"`
}

func MustLoadConfig() *Config {
	godotenv.Load()

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		panic("no config path in env")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic(err)
	}

	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
	if cfg.SMTP.Password == "" {
		panic("SMTP_PASSWORD in env is empty")
	}
	return &cfg
}
