package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/cohesion-org/deepseek-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/config"
	"remembertelebot/db/sqlc"
	"remembertelebot/deepseekai"
	"remembertelebot/ristrettocache"
	"remembertelebot/riverjobs"
	"remembertelebot/services/callbackqueries"
	"remembertelebot/services/commands"
	"remembertelebot/services/messages"
)

func main() {
	setupLogger()

	envCfg := loadEnv()

	pool, err := pgxpool.New(context.Background(), envCfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to connect to database.")
	}
	defer pool.Close()
	queries := sqlc.New(pool)

	botClient, err := bot.NewClient(envCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create telegram bot client.")
	}
	_, botCancel := context.WithCancel(context.Background())

	riverClient := riverjobs.NewClient(envCfg, pool, botClient, queries)

	cache, err := ristrettocache.NewCache[[]deepseek.ChatCompletionMessage]()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create ristretto cache.")
	}
	defer cache.Cache.Close()

	deepSeekClient := deepseekai.NewClient(envCfg.DeepSeekAPIKey)

	commandsHandler := commands.NewHandler(botClient, queries, riverClient, cache)
	messagesHandler := messages.NewHandler(botClient, queries, deepSeekClient, cache)
	callbackQueriesHandler := callbackqueries.NewHandler(botClient, queries, riverClient, pool)

	server := &http.Server{
		Addr:    ":9000",
		Handler: nil,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("Unable to start server.")
		}
	}()

	for update := range botClient.UpdatesChannel {
		if update.Message != nil {
			if isCommand(update.Message.Text) {
				commandsHandler.ProcessCommand(update)
			} else {
				messagesHandler.ProcessMessage(update.Message)
			}
		} else if update.CallbackQuery != nil {
			callbackQueriesHandler.ProcessCallbackQuery(update.CallbackQuery)
		}
	}

	gracefulShutdown(botCancel, riverClient.Client, riverClient.CancelCompletedChannel, server)
}

func setupLogger() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	zerolog.ErrorStackMarshaler = func(err error) interface{} {
		return string(debug.Stack())
	}
}

func loadEnv() config.EnvConfig {
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("Unable to read from .env file.")
	}

	var envCfg config.EnvConfig
	if err := env.Parse(&envCfg); err != nil {
		log.Fatal().Err(err).Msg("Unable to parse environment variables to struct.")
	}

	return envCfg
}

func gracefulShutdown(botCancel context.CancelFunc, riverClient *river.Client[pgx.Tx],
	cancelRiverCompletedEventSubscription func(), server *http.Server) {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)
	<-channel

	log.Info().Msg("Shutting down Telegram bot.")
	botCancel()

	log.Info().Msg("Shutting down River client.")
	defer cancelRiverCompletedEventSubscription()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := riverClient.StopAndCancel(ctx); err != nil {
		log.Err(err).Msg("Unable to shutdown river client.")
	}

	log.Info().Msg("Shutting down http server.")
	if err := server.Shutdown(ctx); err != nil {
		log.Err(err).Msg("Server forced to shutdown.")
	}
}

func isCommand(text string) bool {
	command := strings.TrimSpace(text)
	return len(command) > 0 && fmt.Sprintf("%c", rune(command[0])) == "/"
}
