package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPServer        HTTPServer
	Environment       string `env:"ENVIRONMENT" env-default:"development"`
	AllowedExtensions string `env:"ALLOWED_EXTENSIONS" env-default:".pdf,.jpeg,.jpg"`
}

type HTTPServer struct {
	Address     string        `env:"HTTP_ADDRESS" env-default:":8080"`
	Timeout     time.Duration `env:"HTTP_TIMEOUT" env-default:"60s"`
	IdleTimeout time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"180s"`
}

func MustLoad() Config {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("Не получается прочитать конфиг: %v", err)
	}

	return cfg
}
