package main

import (
	"fmt"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"net"
	"os"
	"tms/internal/config"
	"tms/internal/grpc/documents"
	"tms/internal/grpc/keys"
	"tms/internal/grpc/signature_issuer"
	"tms/internal/grpc/users"
	"tms/internal/lib/logger/handlers/slogpretty"
	"tms/internal/services/crypto"
	"tms/internal/services/document"
	si_service "tms/internal/services/signature_issuer"
	"tms/internal/services/user"
	"tms/internal/storage/mongodb"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "production"
)

func main() {
	// Configuration and logger setup
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log.Info("Starting application", slog.String("env", cfg.Env))

	libPath := os.Getenv("HSM_LIBPATH")
	tokenLabel := os.Getenv("HSM_TOKEN_LABEL")
	pin := os.Getenv("HSM_PIN")

	client, _ := mongodb.New(
		os.Getenv("MONGODB_URI"),
		os.Getenv("MONGODB_DATABASE"),
	)

	operator := crypto.Operator{
		Pkcs11Lib:   libPath,
		TokenLabel:  tokenLabel,
		Pin:         pin,
		MongoClient: client,
	}

	err := operator.Init()
	if err != nil {
		log.Error("Softhsm init error", slog.Any("error", err))
		os.Exit(-1)
	}

	gRPCServer := grpc.NewServer()
	documents.Register(
		gRPCServer,
		log,
		document.New(log, client),
	)
	keys.Register(
		gRPCServer,
		log,
		operator,
	)
	users.Register(
		gRPCServer,
		log,
		user.New(
			log,
			client,
		),
	)
	signature_issuer.Register(
		gRPCServer,
		log,
		si_service.New(
			log,
			operator,
			client,
		),
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", 44047))
	if err != nil {
		log.Error("error listening", slog.Any("err", err))
		return
	}

	if err := gRPCServer.Serve(l); err != nil {
		log.Error("error serving", slog.Any("err", err))
		return
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}

func runningInProduction(cfg *config.Config) bool {
	return cfg.Env == "production"
}
