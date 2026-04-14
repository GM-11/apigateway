package routing

import "time"

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

type Route struct {
	Prefix       string    `yaml:"prefix"`
	AuthRequired bool      `yaml:"auth_required"`
	Upstreams    []string  `yaml:"upstreams"`
	RateLimit    RateLimit `yaml:"rate_limit"`
}

type RateLimit struct {
	Rate  int `yaml:"rate"`
	Burst int `yaml:"burst"`
}
