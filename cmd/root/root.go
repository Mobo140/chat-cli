package root

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Mobo140/chat-cli/internal/clients"
	"github.com/Mobo140/chat-cli/internal/clients/chat"
	"github.com/Mobo140/platform_common/pkg/logger"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

const (
	refreshTokenCronInterval = 4 * time.Minute
	accessTokenCronInterval  = 59 * time.Minute
	timeout                  = 5 * time.Second
)

var (
	ConfigPath string
	LogLevel   string
)

func init() {
	RootCmd.PersistentFlags().StringVar(&ConfigPath, "config-path", ".env", "Path to config file")
	RootCmd.PersistentFlags().StringVarP(&LogLevel, "log-level", "l", "info", "Log level")
}

var RootCmd = &cobra.Command{
	Use:   "chat-cli",
	Short: "Chat CLI",
	Long:  "Chat CLI for managing chats",
	Run: func(cmd *cobra.Command, args []string) {
		StartREPL(cmd)
	},
}

func StartREPL(cmd *cobra.Command) {
	rl, err := readline.New("> ")
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	// Создаем канал для сигнала завершения
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Welcome to Chat CLI. Type 'exit' to quit or press Ctrl+C.")

	// Запускаем горутину для обработки сигнала завершения
	go func() {
		<-done
		fmt.Println("\nReceived interrupt signal. Exiting...")
		os.Exit(0)
	}()

	for {
		fmt.Println()

		line, err := rl.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" || line == "q" {
			fmt.Println("Goodbye!")
			break
		}

		if line == "clear" {
			fmt.Print("\033[H\033[2J") // Очистка экрана
			continue
		}

		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}

		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		cmd.SetArgs(nil)

		time.Sleep(100 * time.Millisecond)
	}
}

func InitCommands(chatClient clients.ChatServiceClient,
	authClient clients.AuthServiceClient,
	sessionFile string,
	loginDoneCh chan struct{},
) {
	loginCmd := newLoginCmd(authClient, sessionFile, loginDoneCh)
	createChatCmd := newCreateChatCmd(chatClient, sessionFile)
	deleteChatCmd := newDeleteChatCmd(chatClient)
	sendMessageCmd := newSendMessageCmd(chatClient, sessionFile)
	connectChatCmd := newConnectChatCmd(chatClient)

	RootCmd.AddCommand(loginCmd)
	RootCmd.AddCommand(createChatCmd)
	RootCmd.AddCommand(deleteChatCmd)
	RootCmd.AddCommand(sendMessageCmd)
	RootCmd.AddCommand(connectChatCmd)
}

func newCreateChatCmd(chatClient clients.ChatServiceClient, sessionFile string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-chat --username user1 [user2 user3...]",
		Short: "Create a new chat",
		Long: `Create a new chat with specified users.
Examples: 
  create-chat --username john              # Create personal chat
  create-chat --username john alice bob    # Create group chat`,
		Run: func(cmd *cobra.Command, args []string) {
			username, _ := cmd.Flags().GetString("username")
			
			// Создаем список пользователей, начиная с указанного в --username
			usernames := []string{username}
			// Добавляем всех остальных пользователей из аргументов
			usernames = append(usernames, args...)

			logger.Debug("Creating chat with users", zap.Strings("usernames", usernames))

			// Загружаем сессию для получения access token
			session, err := loadSession(sessionFile)
			if err != nil {
				logger.Error("failed to load session", zap.Error(err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			ctx = addAccessTokenToContext(ctx, session.AccessToken)

			chatID, err := chatClient.Create(ctx, usernames)
			if err != nil {
				logger.Error("failed to create chat", zap.Error(err))
				return
			}

			logger.Info("Chat created successfully", 
				zap.String("chat_id", chatID),
				zap.Strings("usernames", usernames))
		},
	}

	cmd.Flags().String("username", "", "First username (required)")
	cmd.MarkFlagRequired("username")

	return cmd
}

func newLoginCmd(authClient clients.AuthServiceClient, sessionFile string, loginDoneCh chan struct{}) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the chat",
		Run: func(cmd *cobra.Command, args []string) {
			username, _ := cmd.Flags().GetString("username")
			password, _ := cmd.Flags().GetString("password")

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			refreshToken, err := authClient.Login(ctx, username, password)
			if err != nil {
				logger.Error("failed to login", zap.Error(err))
				return
			}

			accessToken, err := authClient.GetAccessToken(ctx, refreshToken)
			if err != nil {
				logger.Error("failed to get access token", zap.Error(err))
				return
			}

			session := &Session{
				Username:     username,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			}

			if err := saveSession(session, sessionFile); err != nil {
				logger.Error("failed to save session", zap.Error(err))
				return
			}

			logger.Info("Logged in successfully", zap.String("username", username))

			select {
			case loginDoneCh <- struct{}{}:
				logger.Debug("Sent signal about new login")
			default:

				select {
				case <-loginDoneCh:
				default:
				}
				loginDoneCh <- struct{}{}
			}
		},
	}

	cmd.Flags().String("username", "", "Username for login")
	cmd.Flags().String("password", "", "Password for login")
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")

	return cmd
}

func RefreshTokenCron(authClient clients.AuthServiceClient, sessionFile string, loginDoneCh chan struct{}, refreshTokenDoneCh chan struct{}) {
	for {
		<-loginDoneCh

		select {
		case <-refreshTokenDoneCh:
		default:
		}

		for {
			session, err := loadSession(sessionFile)
			if err != nil {
				logger.Error("failed to load session", zap.Error(err))
				break
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			newRefreshToken, err := authClient.GetRefreshToken(ctx, session.RefreshToken)
			cancel()

			if err != nil {
				logger.Error("failed to refresh token", zap.Error(err))
				break
			}

			session.RefreshToken = newRefreshToken

			if err := saveSession(session, sessionFile); err != nil {
				logger.Error("failed to save session", zap.Error(err))
				break
			}

			logger.Info("Refresh token updated")

			select {
			case refreshTokenDoneCh <- struct{}{}:
				logger.Debug("Sent signal to start access token updates")
			default:

			}

			time.Sleep(refreshTokenCronInterval)
		}
	}
}

func AccessTokenCron(authClient clients.AuthServiceClient, sessionFile string, refreshTokenDoneCh chan struct{}) {
	<-refreshTokenDoneCh

	for {
		session, err := loadSession(sessionFile)
		if err != nil {
			logger.Error("failed to load session", zap.Error(err))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		newAccessToken, err := authClient.GetAccessToken(ctx, session.RefreshToken)
		cancel()

		if err != nil {
			logger.Error("failed to generate access token", zap.Error(err))
			continue
		}

		session.AccessToken = newAccessToken
		if err := saveSession(session, sessionFile); err != nil {
			logger.Error("failed to save session", zap.Error(err))
			continue
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

func newConnectChatCmd(chatClient clients.ChatServiceClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect-chat",
		Short: "Connect to chat",
		Long: `Connect to chat and start receiving messages. 
Use Ctrl+C to disconnect from chat.`,
		Run: func(cmd *cobra.Command, args []string) {
			chatID, _ := cmd.Flags().GetString("chat-id")
			username, _ := cmd.Flags().GetString("username")

			logger.Info("Attempting to connect to chat...",
				zap.String("chat_id", chatID),
				zap.String("username", username))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() // Гарантируем, что cancel будет вызван при выходе из функции

			// Создаем канал для сигнала завершения
			done := make(chan os.Signal, 1)
			signal.Notify(done, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(done) // Очищаем обработчик сигналов

			// Создаем канал для ошибок
			errChan := make(chan error, 1)

			// Запускаем подключение в отдельной горутине
			go func() {
				errChan <- chatClient.ConnectChat(ctx, chatID, username)
			}()

			logger.Info("Successfully connected to chat. Press Ctrl+C to disconnect.",
				zap.String("chat_id", chatID),
				zap.String("username", username))

			// Ждем либо сигнала завершения, либо ошибки
			select {
			case <-done:
				logger.Info("Disconnecting from chat",
					zap.String("chat_id", chatID),
					zap.String("username", username))
				return
			case err := <-errChan:
				if err != nil {
					logger.Error("Error in chat connection",
						zap.Error(err),
						zap.String("chat_id", chatID),
						zap.String("username", username))
				}
				return
			}
		},
	}

	cmd.Flags().String("chat-id", "", "Chat ID to connect to")
	cmd.Flags().String("username", "", "Username to connect to chat")
	cmd.MarkFlagRequired("chat-id")
	cmd.MarkFlagRequired("username")

	return cmd
}

func newSendMessageCmd(chatClient clients.ChatServiceClient, sessionFile string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-message --chat-id=ID MESSAGE",
		Short: "Send message to chat",
		Long: `Send message to chat. The message should be the last argument:
Example: send-message --chat-id=1 Hello, world!`,
		Run: func(cmd *cobra.Command, args []string) {
			chatID, _ := cmd.Flags().GetString("chat-id")

			// Проверяем, что есть аргументы для сообщения
			if len(args) == 0 {
				logger.Error("no message provided")
				return
			}

			// Собираем сообщение из всех оставшихся аргументов
			message := strings.Join(args, " ")

			logger.Debug("Command arguments:",
				zap.Strings("args", args),
				zap.String("chat_id", chatID),
				zap.String("message", message))

			// Загружаем сессию для получения access token
			session, err := loadSession(sessionFile)
			if err != nil {
				logger.Error("failed to load session", zap.Error(err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			ctx = addAccessTokenToContext(ctx, session.AccessToken)

			err = chatClient.SendMessage(ctx, &chat.Message{
				ChatID:   chatID,
				Text:     message,
				Username: session.Username,
			})
			if err != nil {
				logger.Error("failed to send message", zap.Error(err))
				return
			}

			logger.Info("Message sent successfully",
				zap.String("chat_id", chatID),
				zap.String("message", message),
				zap.String("username", session.Username))
		},
	}

	// Добавляем флаг chat-id перед аргументами сообщения
	cmd.Flags().String("chat-id", "", "Chat ID to send message to")
	cmd.MarkFlagRequired("chat-id")

	return cmd
}

func addAccessTokenToContext(ctx context.Context, accessToken string) context.Context {
	authHeader := fmt.Sprintf("Bearer %s", accessToken)
	logger.Debug("Adding auth header to context",
		zap.String("header", authHeader))
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", authHeader))
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
	if err := RootCmd.Execute(); err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}
