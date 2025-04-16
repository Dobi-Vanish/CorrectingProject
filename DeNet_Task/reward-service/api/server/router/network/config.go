package network

import (
	"os"
	"reward-service/pkg/errormsg"
)

type Config struct {
	DB struct {
		DSN string
	}
	Server struct {
		Port string
	}
	JWT struct {
		Secret string
	}
}

func Load() (*Config, error) {
	cfg := &Config{}

	cfg.DB.DSN = os.Getenv("DSN")
	cfg.Server.Port = os.Getenv("PORT")

	if cfg.DB.DSN == "" {
		return nil, errormsg.ErrDSNRequired
	}

	if cfg.Server.Port == "" {
		return nil, errormsg.ErrServerPortRequired
	}

	return cfg, nil
}
