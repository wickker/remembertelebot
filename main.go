package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/config"
	"remembertelebot/services/commands"
	"remembertelebot/services/messages"
)

func main() {
	setupLogger()

	envCfg := loadEnv()

	botClient, err := bot.NewClient(envCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create bot client.")
	}

	botChannel := botClient.CreateBotChannel()
	botCtx, botCancel := context.WithCancel(context.Background())

	commandsHandler := commands.NewHandler(botClient)
	messagesHandler := messages.NewHandler(botClient)

	go func() {
		for {
			select {
			case <-botCtx.Done():
				return
			case update := <-botChannel:
				if update.Message.IsCommand() {
					commandsHandler.ProcessCommand(update)
				} else if update.Message != nil {
					messagesHandler.ProcessMessage(update.Message)
				}
			}
		}
	}()

	log.Info().Msg("Bot is alive and listening!")

	gracefulShutdown(botCancel)
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

func gracefulShutdown(botCancel context.CancelFunc) {
	channel := make(chan os.Signal)                         // create a channel to listen for OS signals
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM) // listen for termination signals
	<-channel                                               // wait for the signal to arrive (blocking call)
	log.Info().Msg("Shutting down Telegram bot.")

	botCancel()
}
