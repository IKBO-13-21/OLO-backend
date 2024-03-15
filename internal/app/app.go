package app

import (
	grpcapp "auth/internal/app/grpc"
	"auth/internal/config"
	"auth/internal/service/auth"
	"auth/internal/storage"
	"auth/internal/utils/jwt"
	"log/slog"
)

type App struct {
	GRPCSrv *grpcapp.App
	HTTPSrv *grpcapp.App
}

func New(log *slog.Logger, cfg *config.Config) *App {
	if cfg.DataProvider != "mysql" { // todo сделать развилку, чтобы не зависить от одного
		panic("Not not found provider " + cfg.DataProvider)
	}

	mysqlStorage := storage.NewInAuthMysqlStorage(log, cfg.MySQLSettings.Address, cfg.MySQLSettings.Username,
		cfg.MySQLSettings.Password, cfg.MySQLSettings.Database, cfg.MySQLSettings.Port)

	issuer, err := jwt.NewIssuer("static/private.pem")
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, mysqlStorage, issuer, cfg.TokenTTL)
	grpcApp := grpcapp.New(log, cfg.GRPC.Port, cfg.HTTP.Port, authService)

	httpApp := grpcapp.New(log, cfg.GRPC.Port, cfg.HTTP.Port, authService) // Предполагается, что HTTP сервер будет использовать тот же порт, что и gRPC сервер
	return &App{
		GRPCSrv: grpcApp,
		HTTPSrv: httpApp,
	}
}
