package chat

import (
	"context"
	"strconv"

	descChat "github.com/Mobo140/chat/pkg/chat_v1"
	"github.com/Mobo140/platform_common/pkg/logger"
	"go.uber.org/zap"
)

type client struct {
	chatClient descChat.ChatV1Client
}

func NewChatClient(chatClient descChat.ChatV1Client) *client {
	return &client{chatClient: chatClient}
}

func (c *client) Create(ctx context.Context, usernames []string) (string, error) {
	resp, err := c.chatClient.Create(ctx, &descChat.CreateRequest{
		Info: &descChat.ChatInfo{
			Usernames: usernames,
		},
	})
	if err != nil {
		logger.Error("failed to create chat", zap.Error(err))

		return "", err
	}

	return strconv.FormatInt(resp.GetId(), 10), nil
}

func (c *client) Delete(ctx context.Context, chatID string) error {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		logger.Error("failed to parse chat ID", zap.Error(err))

		return err
	}

	_, err = c.chatClient.Delete(ctx, &descChat.DeleteRequest{
		Id: id,
	})
	if err != nil {
		logger.Error("failed to delete chat", zap.Error(err))

		return err
	}

	return nil
}
