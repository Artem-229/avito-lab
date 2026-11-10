package config

import (
	"fmt"
	"net"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Configuration struct {
	HTTP        HTTP
	Log         Log
	Postgres    Postgres
	Idempotency Idempotency
}

type Idempotency struct {
	KeyTTL time.Duration `env:"IDEMPOTENCY_KEY_TTL" envDefault:"24h" validate:"gt=0s"`
}

type HTTP struct {
	Addr            string        `env:"HTTP_ADDR,required,notEmpty"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required,notEmpty" validate:"gt=0s"`

	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"10s" validate:"gt=0s"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"5s" validate:"gt=0s"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"10s" validate:"gt=0s"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s" validate:"gt=0s"`
}

type Log struct {
	Level string `env:"LOG_LEVEL,required,notEmpty" validate:"oneof=debug info warn error"`
}

type Postgres struct {
	URL             string        `env:"DATABASE_URL,required,notEmpty" validate:"url"`
	MaxConns        int32         `env:"DATABASE_MAX_CONNS,required,notEmpty" validate:"gte=1"`
	MinConns        int32         `env:"DATABASE_MIN_CONNS,required,notEmpty" validate:"gte=0,ltefield=MaxConns"`
	MaxConnLifetime time.Duration `env:"DATABASE_MAX_CONN_LIFETIME,required,notEmpty" validate:"gt=0s"`
	ConnectTimeout  time.Duration `env:"DATABASE_CONNECT_TIMEOUT,required,notEmpty" validate:"gt=0s"`
	QueryTimeout    time.Duration `env:"DATABASE_QUERY_TIMEOUT,required,notEmpty" validate:"gt=0s"`
}

func ReadConfig() (*Configuration, error) {
	cfg, err := env.ParseAs[Configuration]()
	if err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}

	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	if _, _, err := net.SplitHostPort(cfg.HTTP.Addr); err != nil {
		return nil, fmt.Errorf("HTTP_ADDR %q: %w", cfg.HTTP.Addr, err)
	}

	return &cfg, nil
}
