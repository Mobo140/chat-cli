package chat

import (
	"context"
	"fmt"
	"io"
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

func (c *client) SendMessage(ctx context.Context, message *Message) error {
	id, err := strconv.ParseInt(message.ChatID, 10, 64)
	if err != nil {
		logger.Error("failed to parse chat ID", zap.Error(err))
		return err
	}

	_, err = c.chatClient.SendMessage(ctx, &descChat.SendMessageRequest{
		ChatId: id,
		Message: &descChat.Message{
			From: message.Username,
			Text: message.Text,
		},
	})
	if err != nil {
		logger.Error("failed to send message", zap.Error(err))
		return err
	}

	return nil
}

func (c *client) ConnectChat(ctx context.Context, chatID string, username string) error {
	stream, err := c.chatClient.ConnectChat(ctx, &descChat.ConnectChatRequest{
		ChatId:   chatID,
		Username: username,
	})
	if err != nil {
		logger.Error("failed to connect to chat", zap.Error(err))
		return err
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("error receiving message", zap.Error(err))
			return err
		}

		fmt.Printf("\n[%s]: %s\n", msg.GetFrom(), msg.GetText())
	}

	return nil
}

