package auth

import (
	"context"

	descAuth "github.com/Mobo140/auth/pkg/auth_v1"
	"github.com/Mobo140/chat-cli/internal/clients"
	"github.com/Mobo140/platform_common/pkg/logger"
	"go.uber.org/zap"
)

var _ clients.AuthServiceClient = (*client)(nil)

type client struct {
	authClient descAuth.AuthV1Client
}

func NewAuthClient(authClient descAuth.AuthV1Client) *client {
	return &client{authClient: authClient}
}

func (c *client) Login(ctx context.Context, name string, password string) (string, error) {
	resp, err := c.authClient.Login(ctx, &descAuth.LoginRequest{
		Name:     name,
		Password: password,
	})
	if err != nil {
		logger.Error("failed to login", zap.Error(err))

		return "", err
	}

	return resp.GetRefreshToken(), nil
}

func (c *client) GetAccessToken(ctx context.Context, refreshToken string) (string, error) {
	resp, err := c.authClient.GetAccessToken(ctx, &descAuth.GetAccessTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		logger.Error("failed to get access token", zap.Error(err))

		return "", err
	}

	return resp.GetAccessToken(), nil
}

func (c *client) GetRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	resp, err := c.authClient.GetRefreshToken(ctx, &descAuth.GetRefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		logger.Error("failed to get refresh token", zap.Error(err))
		return "", err
	}

	return resp.GetRefreshToken(), nil
}
