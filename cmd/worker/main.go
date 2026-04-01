package main

import (
	"dhis2gw/bootstrap"
	"dhis2gw/config"
	"dhis2gw/db"
	"dhis2gw/models"
	"dhis2gw/tasks"
	"fmt"
	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
)

var splash = `
╺┳┓╻ ╻╻┏━┓┏━┓   ┏━╸┏━┓╺┳╸┏━╸╻ ╻┏━┓╻ ╻   ╻ ╻┏━┓┏━┓╻┏ ┏━╸┏━┓
 ┃┃┣━┫┃┗━┓┏━┛   ┃╺┓┣━┫ ┃ ┣╸ ┃╻┃┣━┫┗┳┛   ┃╻┃┃ ┃┣┳┛┣┻┓┣╸ ┣┳┛
╺┻┛╹ ╹╹┗━┛┗━╸   ┗━┛╹ ╹ ╹ ┗━╸┗┻┛╹ ╹ ╹    ┗┻┛┗━┛╹┗╸╹ ╹┗━╸╹┗╸
`

func main() {
	bootstrap.InitLogging()
	fmt.Print(splash)
	runtimeCfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	config.Set(runtimeCfg)
	cfg := runtimeCfg.Config
	if _, err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	if err := models.InitLocation(); err != nil {
		log.Fatalf("Failed to initialize schedules location: %v", err)
	}
	if err := models.InitServers(); err != nil {
		log.Fatalf("Failed to initialize server cache: %v", err)
	}
	if _, err := config.Watch(func(_, _ *config.RuntimeConfig) {
		if _, err := db.Init(); err != nil {
			log.WithError(err).Error("Failed to reload database")
		}
		if err := models.InitLocation(); err != nil {
			log.WithError(err).Error("Failed to reload schedules location")
		}
		if err := models.InitServers(); err != nil {
			log.WithError(err).Error("Failed to reload server cache")
		}
	}); err != nil {
		log.WithError(err).Warn("Failed to start config watcher")
	}

	// Initialize DHIS2 client and inject it into tasks package
	dhis2Client := sdk.NewClient(
		cfg.API.DHIS2BaseURL,
		cfg.API.DHIS2User,
		cfg.API.DHIS2Password)
	tasks.SetClient(dhis2Client)

	// Set up Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Server.RedisAddress},
		asynq.Config{
			Concurrency: cfg.Server.MaxConcurrent,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// Add any additional config here (timeout, logger, etc)
		},
	)

	// Register task handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeAggregate, tasks.HandleAggregateTask)

	// Start the worker (blocking)
	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run asynq worker: %v", err)
	}
}
