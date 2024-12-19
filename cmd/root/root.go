package root

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Mobo140/chat-cli/internal/clients"
	"github.com/Mobo140/chat-cli/internal/clients/chat"
	"github.com/Mobo140/platform_common/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

const (
	refreshTokenCronInterval = 4 * time.Minute
	accessTokenCronInterval  = 59 * time.Minute
	timeout                  = 5 * time.Second
)

var rootCmd = &cobra.Command{
	Use:   "chat-cli",
	Short: "Chat CLI",
	Long:  "Chat CLI for managing chats",
}

func InitCommands(chatClient clients.ChatServiceClient, authClient clients.AuthServiceClient, sessionFile string) {
	rootCmd.AddCommand(newCreateChatCmd(chatClient))
	rootCmd.AddCommand(newDeleteChatCmd(chatClient))
	rootCmd.AddCommand(newSendMessageCmd(chatClient, sessionFile))
	rootCmd.AddCommand(newLoginCmd(authClient, sessionFile))
}

func newCreateChatCmd(chatClient clients.ChatServiceClient) *cobra.Command {
	return &cobra.Command{
		Use:   "create-chat",
		Short: "Create a new chat",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			usernames, err := cmd.Flags().GetStringSlice("username")
			if err != nil {
				logger.Error("failed to get usernames", zap.Error(err))
				return
			}

			logger.Info("Creating chat", zap.Any("usernames", usernames))

			chatID, err := chatClient.Create(ctx, usernames)
			if err != nil {
				logger.Error("failed to create chat", zap.Error(err))
				return
			}

			logger.Info("Chat created successfully", zap.String("chat_id", chatID))
		},
	}
}

func newLoginCmd(authClient clients.AuthServiceClient, sessionFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login to the chat",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			username, err := cmd.Flags().GetString("username")
			if err != nil {
				logger.Error("failed to get username", zap.Error(err))

				return
			}

			password, err := cmd.Flags().GetString("password")
			if err != nil {
				logger.Error("failed to get password", zap.Error(err))

				return
			}

			refreshToken, err := authClient.Login(ctx, username, password)
			if err != nil {
				logger.Error("failed to login", zap.Error(err))

				return
			}

			session := &Session{
				RefreshToken: refreshToken,
				Username:     username,
			}

			if err := saveSession(session, sessionFile); err != nil {
				logger.Error("failed to save session", zap.Error(err))

				return
			}

			logger.Info("Logged in successfully", zap.String("username", username))

		},
	}
}

// Cron: get refresh token every 4 minutes and update session
func RefreshTokenCron(authClient clients.AuthServiceClient, sessionFile string) {
	for {
		session, err := loadSession(sessionFile)
		if err != nil {
			logger.Error("failed to load session", zap.Error(err))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		newRefreshToken, err := authClient.GetRefreshToken(ctx, session.RefreshToken)
		if err != nil {
			logger.Error("failed to refresh token", zap.Error(err))
			return
		}

		session.RefreshToken = newRefreshToken

		if err := saveSession(session, sessionFile); err != nil {
			logger.Error("failed to save session", zap.Error(err))
			return
		}

		logger.Info("Refresh token updated")
		time.Sleep(refreshTokenCronInterval)
	}
}

// Cron: Get new access token every 59 minutes
func AccessTokenCron(authClient clients.AuthServiceClient, sessionFile string) {
	for {
		session, err := loadSession(sessionFile)
		if err != nil {
			logger.Error("failed to load session", zap.Error(err))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		newAccessToken, err := authClient.GetAccessToken(ctx, session.RefreshToken)
		if err != nil {
			logger.Error("failed to generate access token", zap.Error(err))
			return
		}

		session.AccessToken = newAccessToken
		if err := saveSession(session, sessionFile); err != nil {
			logger.Error("failed to save session", zap.Error(err))
			return
		}

		logger.Info("Access token updated")
		time.Sleep(accessTokenCronInterval)
	}
}

func loadSession(sessionFile string) (*Session, error) {
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func saveSession(session *Session, sessionFile string) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFile, data, 0600)
}

func newSendMessageCmd(chatClient clients.ChatServiceClient, sessionFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "send-message",
		Short: "Send message to chat",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			session, err := loadSession(sessionFile)
			if err != nil {
				logger.Error("failed to load session", zap.Error(err))
				return
			}

			ctx = addAccessTokenToContext(ctx, session.AccessToken)

			chatID, err := cmd.Flags().GetString("chat_id")
			if err != nil {
				logger.Error("failed to get chat_id", zap.Error(err))
				return
			}

			message, err := cmd.Flags().GetString("message")
			if err != nil {
				logger.Error("failed to get message", zap.Error(err))
				return
			}

			err = chatClient.SendMessage(ctx, &chat.Message{
				ChatID:   chatID,
				Text:     message,
				Username: session.Username,
			})
			if err != nil {
				logger.Error("failed to send message", zap.Error(err))
				return
			}

			logger.Info("Message sent successfully", zap.String("chat_id", chatID))
		},
	}
}

func addAccessTokenToContext(ctx context.Context, accessToken string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("Authorization", fmt.Sprintf("Bearer %s", accessToken)))
}

func newDeleteChatCmd(chatClient clients.ChatServiceClient) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-chat",
		Short: "Delete an existing chat",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			chatID, err := cmd.Flags().GetString("chat_id")
			if err != nil {
				logger.Error("failed to get chat_id", zap.Error(err))
				return
			}

			logger.Info("Deleting chat", zap.String("chat_id", chatID))

			err = chatClient.Delete(ctx, chatID)
			if err != nil {
				logger.Error("failed to delete chat", zap.Error(err))
				return
			}

			logger.Info("Chat deleted successfully", zap.String("chat_id", chatID))
		},
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}
