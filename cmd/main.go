package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	descAuth "github.com/Mobo140/auth/pkg/auth_v1"
	"github.com/Mobo140/chat-cli/cmd/root"
	"github.com/Mobo140/chat-cli/internal/clients"
	authClient "github.com/Mobo140/chat-cli/internal/clients/auth"
	chatClient "github.com/Mobo140/chat-cli/internal/clients/chat"
	"github.com/Mobo140/chat-cli/internal/config"
	"github.com/Mobo140/chat-cli/internal/config/env"
	descChat "github.com/Mobo140/chat/pkg/chat_v1"
	"github.com/Mobo140/platform_common/pkg/closer"
	"github.com/Mobo140/platform_common/pkg/logger"
	"github.com/Mobo140/platform_common/pkg/tracing"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logsMaxSize        = 10
	logsMaxBackups     = 3
	logsMaxAge         = 7
	chatCliServiceName = "chat-cli"
)

// App структура для хранения конфигурации и клиентов
type App struct {
	configPath  string
	sessionFile string
	loggerLevel string
	chatClient  clients.ChatServiceClient
	authClient  clients.AuthServiceClient
}

func main() {
	// Парсим флаги перед использованием
	if err := root.RootCmd.ParseFlags(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	ctx := context.Background()
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current directory: %v", err)
	}

	sessionFile := filepath.Join(currentDir, ".chat-cli-session")

	// Используем root.ConfigPath вместо configPath
	app, err := NewApp(ctx, root.ConfigPath, sessionFile)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	// Создаем каналы для синхронизации
	loginDoneChan := make(chan struct{})
	refreshTokenDoneChan := make(chan struct{})

	// Инициализация команд
	root.InitCommands(app.chatClient, app.authClient, app.sessionFile, loginDoneChan)

	// Запускаем горутины для обновления токенов в отдельной группе
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		root.RefreshTokenCron(app.authClient, app.sessionFile, loginDoneChan, refreshTokenDoneChan)
	}()

	go func() {
		defer wg.Done()
		root.AccessTokenCron(app.authClient, app.sessionFile, refreshTokenDoneChan)
	}()

	// Запускаем REPL в основной горутине
	if err := root.RootCmd.Execute(); err != nil {
		log.Fatalf("failed to execute root command: %v", err)
	}

	// Ждем завершения горутин при выходе
	wg.Wait()
}

// NewApp создает новый экземпляр приложения
func NewApp(ctx context.Context, configPath string, sessionFile string) (*App, error) {
	app := &App{
		configPath:  configPath,
		sessionFile: sessionFile,
		loggerLevel: root.LogLevel,
	}

	err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	err = app.initLogger(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %v", err)
	}

	err = initTracer()
	if err != nil {
		return nil, fmt.Errorf("failed to init tracer: %v", err)
	}

	chatClient, err := initChatClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init chat client: %v", err)
	}
	app.chatClient = chatClient

	authClient, err := initAuthClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init auth client: %v", err)
	}
	app.authClient = authClient

	return app, nil
}

// initLogger инициализирует логгер
func (a *App) initLogger(_ context.Context) error {
	logger.Init(getCore(getAtomicLevel(a.loggerLevel)))
	return nil
}

func getCore(level zap.AtomicLevel) zapcore.Core {
	stdout := zapcore.AddSync(os.Stdout)

	file := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    logsMaxSize, // megabytes
		MaxBackups: logsMaxBackups,
		MaxAge:     logsMaxAge, // days
	})

	productionCfg := zap.NewProductionEncoderConfig()
	productionCfg.TimeKey = "timestamp"
	productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	developmentCfg := zap.NewDevelopmentEncoderConfig()
	developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)
	fileEncoder := zapcore.NewJSONEncoder(productionCfg)

	return zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, level),
		zapcore.NewCore(fileEncoder, file, level),
	)
}

func getAtomicLevel(logLevel string) zap.AtomicLevel {
	var level zapcore.Level
	if err := level.Set(logLevel); err != nil {
		log.Fatalf("failed to set log level: %v", err)
	}

	return zap.NewAtomicLevelAt(level)
}

func initChatClient(_ context.Context) (clients.ChatServiceClient, error) {
	creds, err := credentials.NewClientTLSFromFile("../chat.pem", "")
	if err != nil {
		log.Fatalf("failed to load TLS keys for chat client: %v", err)
		return nil, err
	}

	conn, err := grpc.NewClient(
		ChatClientConfig().Address(),
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer()),
		),
	)
	if err != nil {
		log.Fatalf("failed to dial gRPC client: %v", err)
		return nil, err
	}

	closer.Add(conn.Close)

	return chatClient.NewChatClient(descChat.NewChatV1Client(conn)), nil
}

func ChatClientConfig() config.ChatClientConfig {
	cfg, err := env.NewChatClientConfig()
	if err != nil {
		log.Fatalf("failed to load chat client config: %v", err)
	}

	return cfg
}

func initAuthClient(_ context.Context) (clients.AuthServiceClient, error) {
	creds, err := credentials.NewClientTLSFromFile("../auth.pem", "")
	if err != nil {
		log.Fatalf("failed to load TLS keys for auth client: %v", err)
		return nil, err
	}

	conn, err := grpc.NewClient(
		AuthClientConfig().Address(),
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer()),
		),
	)
	if err != nil {
		log.Fatalf("failed to dial gRPC client: %v", err)
		return nil, err
	}

	closer.Add(conn.Close)

	return authClient.NewAuthClient(descAuth.NewAuthV1Client(conn)), nil
}

func AuthClientConfig() config.AuthClientConfig {
	cfg, err := env.NewAuthClientConfig()
	if err != nil {
		log.Fatalf("failed to load auth client config: %v", err)
	}

	return cfg
}

func initTracer() error {
	tracing.Init(logger.Logger(), chatCliServiceName, JaegerConfig().Address())

	return nil
}

func JaegerConfig() config.JaegerConfig {
	cfg, err := env.NewJaegerConfig()
	if err != nil {
		log.Fatalf("failed to load jaeger config: %v", err)
	}

	return cfg
}
