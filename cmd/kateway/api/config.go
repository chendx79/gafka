package api

import (
	"time"
)

type Config struct {
	AppId  string
	Secret string

	Timeout   time.Duration
	KeepAlive time.Duration

	PubEndpoint string // host:port
	SubEndpoint string // host:port

	Debug bool
}

func DefaultConfig() *Config {
	return &Config{
		Timeout:   time.Second * 120, // FIXME
		KeepAlive: time.Minute,
		Debug:     false,
	}
}
