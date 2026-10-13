package main

import (
	"fmt"

	"dhis2gw/internal/bootstrap"
	"dhis2gw/internal/config"
	"dhis2gw/internal/db"
	"dhis2gw/internal/migrate"
	log "github.com/sirupsen/logrus"
)

func main() {
	bootstrap.InitLogging()

	runtimeCfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	config.Set(runtimeCfg)

	if _, err := db.Init(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	if err := migrate.RunMigrations(db.GetDB(), runtimeCfg.Config.Server.MigrationsDirectory); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Println("Migrations applied successfully")
}
