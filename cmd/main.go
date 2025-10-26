package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/odlev/websockets/internal/config"
	"github.com/odlev/websockets/internal/wsserver"
	"github.com/rs/zerolog"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	cfg := config.MustLoad(os.Getenv("CONFIG_PATH"))
	log := SetZerolog()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	wsSrv := wsserver.New(cfg.Websocket.Address, cfg, log)
	log.Info().Str("address", wsSrv.Address()).Msg("started ws server")
	go func() {
		if err := wsSrv.Start(); err != nil {
			log.Fatal().Err(err).Msg("failed to start ws server")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("app shutting down...")
	if err := wsSrv.Stop(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func SetZerolog() zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}
	log := zerolog.New(output).With().Timestamp().Logger()
	return log
}
