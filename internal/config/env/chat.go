package env

import (
	"fmt"
	"net"
	"os"
)

const (
	chatHostEnv = "CHAT_HOST"
	chatPortEnv = "CHAT_PORT"
)

type chatClientConfig struct {
	host string
	port string
}

func NewChatClientConfig() (*chatClientConfig, error) {
	host := os.Getenv(chatHostEnv)
	if len(host) == 0 {
		return nil, fmt.Errorf("chat host is not set")
	}

	port := os.Getenv(chatPortEnv)
	if len(port) == 0 {
		return nil, fmt.Errorf("chat port is not set")
	}

	return &chatClientConfig{host: host, port: port}, nil
}

func (c *chatClientConfig) Address() string {
	return net.JoinHostPort(c.host, c.port)
}
