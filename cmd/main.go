package main

import (
	"context"
	"log"

	"github.com/Mobo140/chat-cli/cmd/root"
	"github.com/Mobo140/chat-cli/internal/config"
	"github.com/Mobo140/chat-cli/internal/config/env"
	descChat "github.com/Mobo140/chat/pkg/chat_v1"
	"github.com/Mobo140/platform_common/pkg/closer"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var configPath string

type App struct {
	chatClient descChat.ChatV1Client
	configPath string
}

func main() {
	ctx := context.Background()

	app, err := NewApp(ctx, configPath)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	// Передаем клиент в команды
	root.InitCommands(app.chatClient)

	// Выполняем команды
	root.Execute()
}

func NewApp(ctx context.Context, configPath string) (*App, error) {
	a := &App{configPath: configPath}

	chatClient, err := initChatClient(ctx)
	if err != nil {
		return nil, err
	}
	a.chatClient = chatClient

	return a, nil
}

func initChatClient(ctx context.Context) (descChat.ChatV1Client, error) {
	creds, err := credentials.NewClientTLSFromFile("../../chat.pem", "")
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

	// Регистрируем закрытие соединения
	closer.Add(conn.Close)

	return descChat.NewChatV1Client(conn), nil
}

func ChatClientConfig() config.ChatClientConfig {
	cfg, err := env.NewChatClientConfig()
	if err != nil {
		log.Fatalf("failed to load chat client config: %v", err)
	}

	return cfg
}
