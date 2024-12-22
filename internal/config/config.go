package config

import "github.com/joho/godotenv"

type ChatClientConfig interface {
	Address() string
}

type AuthClientConfig interface {
	Address() string
}

type JaegerConfig interface {
	Address() string
}

func Load(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}

	return nil
}
