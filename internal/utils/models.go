package utils

import (
	"time"

	"example.com/m/v2/internal/circuitbreaker"
)

type Route struct {
	Prefix       string      `yaml:"prefix"`
	AuthRequired bool        `yaml:"auth_required"`
	Upstreams    []*Upstream `yaml:"upstreams"`
	RateLimit    *RateLimit  `yaml:"rate_limit"`
}

type Upstream struct {
	Config         UpstreamConfig
	CircuitBreaker *circuitbreaker.CircuitBreaker
}

type UpstreamConfig struct {
	URL              string        `yaml:"url"`
	FailureThreshold int           `yaml:"failure_threshold"`
	RecoveryWindow   time.Duration `yaml:"recovery_window"`
	FailureWindow    time.Duration `yaml:"failure_window"`
}

type RateLimit struct {
	Rate  float64 `yaml:"rate"`
	Burst float64 `yaml:"burst"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Auth   AuthConfig   `yaml:"auth"`
	Routes []Route      `yaml:"routes"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type AuthConfig struct {
	JwksUrl string        `yaml:"jwks_url"`
	Ttl     time.Duration `yaml:"ttl"`
}
