package main

import (
	"dhis2gw/config"
	"dhis2gw/tasks"
	"fmt"
	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func init() {
	formatter := new(log.TextFormatter)
	formatter.TimestampFormat = time.RFC3339
	formatter.FullTimestamp = true

	log.SetFormatter(formatter)
	log.SetOutput(os.Stdout)
}

var splash = `
╺┳┓╻ ╻╻┏━┓┏━┓   ┏━╸┏━┓╺┳╸┏━╸╻ ╻┏━┓╻ ╻   ╻ ╻┏━┓┏━┓╻┏ ┏━╸┏━┓
 ┃┃┣━┫┃┗━┓┏━┛   ┃╺┓┣━┫ ┃ ┣╸ ┃╻┃┣━┫┗┳┛   ┃╻┃┃ ┃┣┳┛┣┻┓┣╸ ┣┳┛
╺┻┛╹ ╹╹┗━┛┗━╸   ┗━┛╹ ╹ ╹ ┗━╸┗┻┛╹ ╹ ╹    ┗┻┛┗━┛╹┗╸╹ ╹┗━╸╹┗╸
`

func main() {
	fmt.Printf(splash)

	// Initialize DHIS2 client and inject it into tasks package
	dhis2Client := sdk.NewClient(
		config.DHIS2GWConf.API.DHIS2BaseURL,
		config.DHIS2GWConf.API.DHIS2User,
		config.DHIS2GWConf.API.DHIS2Password)
	tasks.SetClient(dhis2Client)

	// Set up Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: config.DHIS2GWConf.Server.RedisAddress},
		asynq.Config{
			Concurrency: config.DHIS2GWConf.Server.MaxConcurrent,
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
