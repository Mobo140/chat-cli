package env

import (
	"errors"
	"net"
	"os"
)

const (
	jaegerHost = "JAEGER_HOST"
	jaegerPort = "JAEGER_PORT"
)

type jaegerConfig struct {
	host string
	port string
}

func NewJaegerConfig() (*jaegerConfig, error) {
	host := os.Getenv(jaegerHost)
	if len(host) == 0 {
		return nil, errors.New("jaeger host not set")
	}

	port := os.Getenv(jaegerPort)
	if len(port) == 0 {
		return nil, errors.New("jaeger port not set")
	}

	return &jaegerConfig{
		host: host,
		port: port,
	}, nil
}

func (c *jaegerConfig) Address() string {
	return net.JoinHostPort(c.host, c.port)
}
