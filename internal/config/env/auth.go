package env

import (
	"fmt"
	"net"
	"os"
)

const (
	authHostEnv = "AUTH_HOST"
	authPortEnv = "AUTH_PORT"
)

type authClientConfig struct {
	host string
	port string
}

func NewAuthClientConfig() (*authClientConfig, error) {
	host := os.Getenv(authHostEnv)
	if len(host) == 0 {
		return nil, fmt.Errorf("auth host is not set")
	}

	port := os.Getenv(authPortEnv)
	if len(port) == 0 {
		return nil, fmt.Errorf("auth port is not set")
	}

	return &authClientConfig{host: host, port: port}, nil
}

func (c *authClientConfig) Address() string {
	return net.JoinHostPort(c.host, c.port)
}
