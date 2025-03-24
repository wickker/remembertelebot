package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"remembertelebot/bot"
	"remembertelebot/config"
	"remembertelebot/riverjobs"
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
	//queries := sqlc.New(pool)

	riverClient := setupRiverClient(envCfg, pool)

	//j, err := riverClient.Insert(context.Background(),
	//	ScheduledArgs{
	//		Message: "hello from the future",
	//	},
	//	&river.InsertOpts{
	//		// Schedule the job to be worked in three hours.
	//		ScheduledAt: time.Now().Add(1 * time.Minute),
	//	})
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(j)

	//a := riverClient.PeriodicJobs().Add(river.NewPeriodicJob(
	//	river.PeriodicInterval(10*time.Second),
	//	func() (river.JobArgs, *river.InsertOpts) {
	//		return PeriodicJobArgs{
	//			Msg: "Every 10 seconds",
	//		}, nil
	//	},
	//	//&river.PeriodicJobOpts{RunOnStart: true},
	//	nil,
	//))
	//fmt.Println("a : ", a)
	//
	//b := riverClient.PeriodicJobs().Add(river.NewPeriodicJob(
	//	river.PeriodicInterval(15*time.Second),
	//	func() (river.JobArgs, *river.InsertOpts) {
	//		return PeriodicJobArgs{
	//			Msg: "Every 15 seconds",
	//		}, nil
	//	},
	//	//&river.PeriodicJobOpts{RunOnStart: true},
	//	nil,
	//))
	//fmt.Println("b : ", b)

	//if _, err := riverClient.JobCancel(context.Background(), 3); err != nil {
	//	fmt.Println(err)
	//}

	botClient, err := bot.NewClient(envCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create bot client.")
	}

	botChannel := botClient.CreateBotChannel()
	botCtx, botCancel := context.WithCancel(context.Background())

	commandsHandler := commands.NewHandler(botClient)
	messagesHandler := messages.NewHandler(botClient)

	// TODO: Figure out how to disable the probe
	// Need this to pass Google Cloud Run's TCP probe 💀
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "Remember bot is up!")
	})
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal().Err(err).Msg("Unable to start server.")
		}
	}()

	go func() {
		log.Info().Msg("Bot is alive and listening.")

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

	gracefulShutdown(botCancel, riverClient)
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

func gracefulShutdown(botCancel context.CancelFunc, riverClient *river.Client[pgx.Tx]) {
	channel := make(chan os.Signal)
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)
	<-channel

	log.Info().Msg("Shutting down Telegram bot.")
	botCancel()

	riverCtx, riverCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer riverCancel()
	if err := riverClient.StopAndCancel(riverCtx); err != nil {
		log.Err(err).Msg("Unable to shutdown river client.")
	}
}

func setupRiverClient(envCfg config.EnvConfig, pool *pgxpool.Pool) *river.Client[pgx.Tx] {
	workers := river.NewWorkers()
	river.AddWorker(workers, &riverjobs.ScheduledJobWorker{})
	river.AddWorker(workers, &riverjobs.PeriodicJobWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: slog.Default(),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		TestOnly: envCfg.IsDev(),
		Workers:  workers,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to initialize new river client.")
	}

	if err := riverClient.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Unable to start river client.")
	}

	return riverClient
}
