package workers

import (
	"dhis2gw/config"
	"dhis2gw/db"
	"dhis2gw/models"
	"dhis2gw/tasks"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
)

func main() {
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

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Server.RedisAddress},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: cfg.Server.MaxConcurrent,
			// Optionally, specify multiple queues with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// See the godoc for other configuration options
		},
	)

	mux := asynq.NewServeMux()

	mux.HandleFunc(tasks.TypeAggregate, tasks.HandleAggregateTask)

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}

}
