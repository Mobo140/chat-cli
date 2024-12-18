package root

import (
	"os"

	"github.com/Mobo140/platform_common/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "chat-cli",
	Short: "Chat CLI",
	Long:  "Chat CLI",
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creating something",
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete something",
}

var createUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Создает нового пользователя",
	Run: func(cmd *cobra.Command, args []string) {
		usernamesStr, err := cmd.Flags().GetString("username")
		if err != nil {
			logger.Error("failed to get usernames", zap.Error(err))

			return
		}

		logger.Info("Creating users", zap.String("usernames", usernamesStr))
	},
}

var createChatCmd = &cobra.Command{
	Use:   "user",
	Short: "Создает новый chat",
	Run: func(cmd *cobra.Command, args []string) {
		usernamesStr, err := cmd.Flags().GetString("username")
		if err != nil {
			logger.Error("failed to get usernames", zap.Error(err))

			return
		}

		logger.Info("Creating users", zap.String("usernames", usernamesStr))
	},
}

var deleteUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Удаляет пользователя",
	Run: func(cmd *cobra.Command, args []string) {
		usernamesStr, err := cmd.Flags().GetString("username")
		if err != nil {
			logger.Error("failed to get usernames", zap.Error(err))

			return
		}

		logger.Info("Deleting user", zap.String("username", usernamesStr))
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)

	createCmd.AddCommand(createUserCmd)
	deleteCmd.AddCommand(deleteUserCmd)

	createUserCmd.Flags().StringP("username", "u", "", "Username")
	err := createUserCmd.MarkFlagRequired("username")
	if err != nil {
		logger.Error("failed to mark username flag as required", zap.Error(err))

		return
	}

	deleteUserCmd.Flags().StringP("username", "u", "", "Username")
	err = deleteUserCmd.MarkFlagRequired("username")
	if err != nil {
		logger.Error("failed to mark username flag as required", zap.Error(err))

		return
	}
}
