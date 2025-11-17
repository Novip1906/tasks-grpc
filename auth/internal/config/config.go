package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Address      string        `yaml:"address" env-required:"true"`
	JWTSecretKey string        `yaml:"jwt_secret_key" env-required:"true"`
	UserDb       Postgres      `yaml:"postgres" env-required:"true"`
	CodesDb      Redis         `yaml:"redis" env-required:"true"`
	Params       Params        `yaml:"params"`
	CodeExp      time.Duration `yaml:"code_exp" env-default:"5m"`
	Kafka        Kafka         `yaml:"kafka" env-required:"true"`
}

type Postgres struct {
	Host     string `yaml:"host" env-default:"localhost"`
	Port     string `yaml:"port" env-default:"5431"`
	User     string `yaml:"user" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	DBName   string `yaml:"db_name" env-required:"true"`
}

type Redis struct {
	Address  string `yaml:"address" env-default:":6379"`
	Password string `yaml:"password" env-default:""`
	DB       int    `yaml:"db" env-default:"0"`
}

type Params struct {
	Username MinMaxLen `yaml:"username"`
	Password MinMaxLen `yaml:"password"`
}

type MinMaxLen struct {
	Min int `yaml:"min" env-required:"true"`
	Max int `yaml:"max" env-required:"true"`
}

type Kafka struct {
	Brokers           []string `yaml:"brokers" env-required="true"`
	VerificationTopic string   `yaml:"topic" env-required="true"`
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
