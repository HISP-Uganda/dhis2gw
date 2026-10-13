package bootstrap

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func InitLogging() {
	formatter := new(log.TextFormatter)
	formatter.TimestampFormat = time.RFC3339
	formatter.FullTimestamp = true

	log.SetFormatter(formatter)
	log.SetOutput(os.Stdout)
}
