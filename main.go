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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"remembertelebot/asynqjobs"
	"remembertelebot/bot"
	"remembertelebot/config"
	"remembertelebot/db/sqlc"
	"remembertelebot/deepseekai"
	"remembertelebot/ristrettocache"
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

	asynqClient := asynqjobs.NewClient(envCfg, botClient, queries)
	go func() {
		log.Info().Msg("Init asynq job server.")
		if err := asynqClient.Server.Run(asynqClient.Mux); err != nil {
			log.Fatal().Err(err).Msg("Unable to init asynq job server.")
		}
	}()
	go func() {
		log.Info().Msg("Init asynq job scheduler.")
		if err := asynqClient.Scheduler.Run(); err != nil {
			log.Fatal().Err(err).Msg("Unable to init asynq job scheduler.")
		}
	}()

	cache, err := ristrettocache.NewCache[[]deepseek.ChatCompletionMessage]()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create ristretto cache.")
	}
	defer cache.Cache.Close()

	deepSeekClient := deepseekai.NewClient(envCfg.DeepSeekAPIKey)

	commandsHandler := commands.NewHandler(botClient, queries, cache, asynqClient)
	messagesHandler := messages.NewHandler(botClient, queries, deepSeekClient, cache)
	callbackQueriesHandler := callbackqueries.NewHandler(botClient, queries, asynqClient)

	webhookServer := &http.Server{
		Addr:    ":9000",
		Handler: nil,
	}
	go func() {
		log.Info().Msg("Init webhook server.")
		if err := webhookServer.ListenAndServe(); err != nil {
			log.Fatal().Err(err).Msg("Unable to start server.")
		}
	}()

	log.Info().Msg("Init telegram bot message processors.")
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

	gracefulShutdown(botCancel, webhookServer, asynqClient)
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

func gracefulShutdown(botCancel context.CancelFunc, webhookServer *http.Server, asynqClient *asynqjobs.Client) {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)
	<-channel

	log.Info().Msg("Shutting down Telegram bot.")
	botCancel()

	log.Info().Msg("Shutting down http server.")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := webhookServer.Shutdown(ctx); err != nil {
		log.Err(err).Msg("Webhook server failed to shutdown.")
	}

	log.Info().Msg("Shutting down asynq job processing.")
	asynqClient.Scheduler.Shutdown()
	asynqClient.Server.Shutdown()
	if err := asynqClient.Client.Close(); err != nil {
		log.Err(err).Msg("Asynq client failed to shutdown.")
	}
}

func isCommand(text string) bool {
	command := strings.TrimSpace(text)
	return len(command) > 0 && fmt.Sprintf("%c", rune(command[0])) == "/"
}
