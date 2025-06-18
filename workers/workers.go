package workers

import (
	"dhis2gw/config"
	"dhis2gw/tasks"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
)

func main() {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: config.DHIS2GWConf.Server.RedisAddress},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: config.DHIS2GWConf.Server.MaxConcurrent,
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
