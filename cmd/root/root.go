package root

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/Mobo140/chat/pkg/chat_v1"
	"github.com/Mobo140/platform_common/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "chat-cli",
	Short: "Chat CLI",
	Long:  "Chat CLI for managing chats",
}

func InitCommands(chatClient chat_v1.ChatV1Client) {
	rootCmd.AddCommand(newCreateChatCmd(chatClient))
	rootCmd.AddCommand(newDeleteChatCmd(chatClient))
}

func newCreateChatCmd(chatClient chat_v1.ChatV1Client) *cobra.Command {
	return &cobra.Command{
		Use:   "create-chat",
		Short: "Create a new chat",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			usernames, err := cmd.Flags().GetStringSlice("username")
			if err != nil {
				logger.Error("failed to get usernames", zap.Error(err))
				return
			}

			logger.Info("Creating chat", zap.Any("usernames", usernames))

			resp, err := chatClient.Create(ctx, &chat_v1.CreateRequest{
				Info: &chat_v1.ChatInfo{
					Usernames: usernames,
				},
			})
			if err != nil {
				logger.Error("failed to create chat", zap.Error(err))
				return
			}

			logger.Info("Chat created successfully", zap.String("chat_id", strconv.FormatInt(resp.Id, 10)))
		},
	}
}

func newDeleteChatCmd(chatClient chat_v1.ChatV1Client) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-chat",
		Short: "Delete an existing chat",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			chatID, err := cmd.Flags().GetString("chat_id")
			if err != nil {
				logger.Error("failed to get chat_id", zap.Error(err))
				return
			}

			logger.Info("Deleting chat", zap.String("chat_id", chatID))
			
			id, err := strconv.ParseInt(chatID, 10, 64)
			if err != nil {
				logger.Error("failed to parse chat_id", zap.Error(err))
				return
			}

			_, err = chatClient.Delete(ctx, &chat_v1.DeleteRequest{
				Id: id,
			})
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
