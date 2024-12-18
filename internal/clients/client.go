package clients

import (
	"context"
)

type ChatServiceClient interface {
	Create(ctx context.Context, usernames []string) (string, error)
	Delete(ctx context.Context, chatID string) error
}

type AuthServiceClient interface {
	Login(ctx context.Context, name string, password string) error
}


