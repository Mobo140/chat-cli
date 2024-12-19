package clients

import (
	"context"

	chat "github.com/Mobo140/chat-cli/internal/clients/chat"
)

type ChatServiceClient interface {
	Create(ctx context.Context, usernames []string) (string, error)
	Delete(ctx context.Context, chatID string) error
	SendMessage(ctx context.Context, message *chat.Message) error
}

type AuthServiceClient interface {
	Login(ctx context.Context, name string, password string) (string, error)
	GetAccessToken(ctx context.Context, refreshToken string) (string, error)
	GetRefreshToken(ctx context.Context, accessToken string) (string, error)
}
