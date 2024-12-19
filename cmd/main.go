package main

import (
	"context"
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
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var configPath string

type App struct {
	chatClient  clients.ChatServiceClient
	authClient  clients.AuthServiceClient
	configPath  string
	sessionFile string
}

func main() {
	ctx := context.Background()

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current directory: %v", err)
	}

	sessionFile := filepath.Join(currentDir, ".chat-cli-session")

	app, err := NewApp(ctx, configPath, sessionFile)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	root.InitCommands(app.chatClient, app.authClient, app.sessionFile)

	root.Execute()

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		root.AccessTokenCron(app.authClient, app.sessionFile)
	}()

	go func() {
		defer wg.Done()
		root.RefreshTokenCron(app.authClient, app.sessionFile)
	}()

	wg.Wait()
}

func NewApp(ctx context.Context, configPath string, sessionFile string) (*App, error) {
	a := &App{configPath: configPath, sessionFile: sessionFile}

	chatClient, err := initChatClient(ctx)
	if err != nil {
		return nil, err
	}
	a.chatClient = chatClient

	authClient, err := initAuthClient(ctx)
	if err != nil {
		return nil, err
	}
	a.authClient = authClient

	return a, nil
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
