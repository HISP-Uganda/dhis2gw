package main

import (
	"context"
	"dhis2gw/cmd"
	"dhis2gw/config"
	"dhis2gw/controllers"
	"dhis2gw/db"
	"dhis2gw/middleware"
	"dhis2gw/tasks"
	"fmt"
	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
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
╺┳┓╻ ╻╻┏━┓┏━┓   ┏━╸┏━┓╺┳╸┏━╸╻ ╻┏━┓╻ ╻
 ┃┃┣━┫┃┗━┓┏━┛   ┃╺┓┣━┫ ┃ ┣╸ ┃╻┃┣━┫┗┳┛
╺┻┛╹ ╹╹┗━┛┗━╸   ┗━┛╹ ╹ ╹ ┗━╸┗┻┛╹ ╹ ╹
`

var client *asynq.Client

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := cmd.RunMigrations(db.GetDB()); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		return
	}
	fmt.Printf(splash)
	client = asynq.NewClient(asynq.RedisClientOpt{Addr: config.DHIS2GWConf.Server.RedisAddress})
	defer func(client *asynq.Client) {
		_ = client.Close()
	}(client)

	dhis2Client := sdk.NewClient(
		config.DHIS2GWConf.API.DHIS2GWDHIS2BaseURL,
		config.DHIS2GWConf.API.DHIS2GWDHIS2User,
		config.DHIS2GWConf.API.DHIS2GWDHIS2Password)
	tasks.SetClient(dhis2Client)

	var wg sync.WaitGroup

	wg.Add(2)
	go startAPIServer(ctx, &wg)
	go startWorker(ctx, &wg)

	wg.Wait()
}

func startAPIServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	router := gin.Default()

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob(
		config.DHIS2GWConf.Server.TemplatesDirectory + "/*"))
	router.SetHTMLTemplate(tmpl)
	router.Static("/static", config.DHIS2GWConf.Server.StaticDirectory)

	v2 := router.Group("/api", middleware.BasicAuth(db.GetDB(), client))
	{
		v2.GET("/test2", func(c *gin.Context) {
			c.String(200, "Authorized")
		})

		userController := &controllers.UserController{}
		v2.POST("/user", userController.CreateUser)
		v2.GET("/users/:uid", userController.GetUserByUID)
		v2.PUT("/users/:uid", userController.UpdateUser)
		v2.POST("/users/getToken", userController.CreateUserToken)
		v2.POST("/users/refreshToken", userController.RefreshUserToken)

		aggregateController := &controllers.AggregateController{}
		v2.POST("/aggregate", aggregateController.CreateRequest)
	}
	router.NoRoute(func(c *gin.Context) {
		c.String(404, "Page Not Found!")
	})

	// Use http.Server for graceful shutdown
	httpServer := &http.Server{
		Addr:    ":" + fmt.Sprintf("%s", config.DHIS2GWConf.Server.Port),
		Handler: router,
	}

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		log.Infof("Starting API server on :%s...", config.DHIS2GWConf.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("Shutdown signal received. Shutting down API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Errorf("Error shutting down API server: %v", err)
		} else {
			log.Info("API server shut down gracefully.")
		}
	case err := <-errCh:
		log.Fatalf("API server error: %v", err)
	}
}

func startWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: config.DHIS2GWConf.Server.RedisAddress},
		asynq.Config{
			Concurrency: config.DHIS2GWConf.Server.MaxConcurrent,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeAggregate, tasks.HandleAggregateTask)

	// Run the worker in a goroutine and listen for shutdown
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Run(mux); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("Shutdown signal received. Shutting down Asynq worker...")
		srv.Shutdown() // No error to check in current Asynq
		log.Info("Asynq worker shut down gracefully.")
	case err := <-errCh:
		log.Fatalf("Asynq worker error: %v", err)
	}
}
