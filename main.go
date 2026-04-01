package main

import (
	"context"
	"dhis2gw/bootstrap"
	"dhis2gw/cmd"
	"dhis2gw/config"
	"dhis2gw/controllers"
	"dhis2gw/db"
	"dhis2gw/docs"
	"dhis2gw/middleware"
	"dhis2gw/models"
	"dhis2gw/tasks"
	"dhis2gw/utils"
	"fmt"
	"html/template"
	"net/http"
	"os/signal"
	"syscall"

	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gomarkdown/markdown"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"os"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
)

var splash = `
╺┳┓╻ ╻╻┏━┓┏━┓   ┏━╸┏━┓╺┳╸┏━╸╻ ╻┏━┓╻ ╻
 ┃┃┣━┫┃┗━┓┏━┛   ┃╺┓┣━┫ ┃ ┣╸ ┃╻┃┣━┫┗┳┛
╺┻┛╹ ╹╹┗━┛┗━╸   ┗━┛╹ ╹ ╹ ┗━╸┗┻┛╹ ╹ ╹
`

var client *asynq.Client

// @title DHIS2 Gateway Service
// @version 1.0.1
// @description This service provides sends aggregate and tracker data to DHIS2 from third-party systems
// @contact.name API Support
// @contact.url http://www.hispuganda.org
// @contact.email ssekiwere@hispuganda.org
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host dhis2gw.hispuganda.org
// @BasePath /api/v2
// @schemes http https
// @securityDefinitions.basic BasicAuth
// @security basicAuth
// @in header
// @name Authorization

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := cmd.RunMigrations(db.GetDB()); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		return
	}
	client = asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Server.RedisAddress})
	defer func(client *asynq.Client) {
		_ = client.Close()
	}(client)

	dhis2Client := sdk.NewClient(
		cfg.API.DHIS2BaseURL,
		cfg.API.DHIS2User,
		cfg.API.DHIS2Password)
	tasks.SetClient(dhis2Client)

	var wg sync.WaitGroup

	wg.Add(2)
	go startAPIServer(ctx, &wg, cfg)
	go startWorker(ctx, &wg, cfg)

	wg.Wait()
}

func startAPIServer(ctx context.Context, wg *sync.WaitGroup, cfg config.Config) {
	defer wg.Done()

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // your frontend
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob(
		cfg.Server.TemplatesDirectory + "/*"))
	router.SetHTMLTemplate(tmpl)
	staticDir := cfg.Server.StaticDirectory
	router.Static("/static", staticDir)

	docs.SwaggerInfo.BasePath = "/api/v2"

	v2 := router.Group("/api/v2", middleware.BasicAuth(db.GetDB(), client))
	{
		v2.GET("/test2", func(c *gin.Context) {
			c.String(200, "Authorized")
		})

		userController := &controllers.UserController{}
		v2.POST("/user", userController.CreateUser)
		v2.GET("/users", userController.GetUsersHandler(db.GetDB()))
		v2.GET("/users/:uid", userController.GetUserByUID)
		v2.PUT("/users/:uid", userController.UpdateUser)
		v2.POST("/users/getToken", userController.CreateUserToken)
		v2.POST("/users/refreshToken", userController.RefreshUserToken)

		aggregateController := &controllers.AggregateController{}
		v2.POST("/aggregate", aggregateController.CreateRequest)
		v2.GET("/aggregate/reenqueue/:task_id", aggregateController.ReEnqueueAggregateTask)
		v2.POST("/aggregate/reenqueue/batch", aggregateController.BatchReEnqueueAggregateTasksByIDs)

		logController := &controllers.LogsController{}
		v2.GET("/logs/:id", logController.GetLogByIdHandler(db.GetDB()))
		v2.GET("/logs", logController.GetLogsHandler(db.GetDB()))
		v2.DELETE("/logs/:id", logController.DeleteSubmissionLogHandler(db.GetDB()))
		v2.DELETE("/logs/purge", logController.PurgeSubmissionLogsByDateHandler(db.GetDB()))
		// reporcess log
		// v2.POST("/logs/reprocess/:id", logController.ReprocessLogHandler(db.GetDB()))

		mappingsController := &controllers.MappingController{}
		v2.GET("/mappings", mappingsController.GetMappingsHandler())
		v2.POST("/mappings/import/csv", mappingsController.ImportCSVHandler)
		v2.POST("/mappings/import/excel", mappingsController.ImportExcelHandler)
		v2.GET("/mappings/export/excel", mappingsController.ExportExcelMappingsHandler)

	}
	mappingsController := &controllers.MappingController{}
	router.GET("/mappings/export/excel-template", mappingsController.ExportExcelTemplateHandler)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	router.StaticFile("/logs-viewer", fmt.Sprintf("%s/logs_viewer.html", staticDir))
	// Documentation Routes
	// Home Route
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "API Documentation",
		})
	})

	router.GET("/docs/:page", func(c *gin.Context) {
		docName := c.Param("page")

		// Construct Markdown file path
		mdFile := fmt.Sprintf("%s/%s.md", cfg.Server.DocsDirectory, docName)

		// Read Markdown file
		mdContent, err := os.ReadFile(mdFile)
		if err != nil {
			c.String(http.StatusNotFound, "Documentation not found")
			return
		}

		// Convert Markdown to HTML
		htmlContent := template.HTML(markdown.ToHTML(mdContent, nil, nil))

		// Render docs.html template
		c.HTML(http.StatusOK, "docs.html", gin.H{
			"title":   docName,
			"content": htmlContent,
		})
	})
	router.NoRoute(func(c *gin.Context) {
		c.String(404, "Page Not Found!")
	})

	// Use http.Server for graceful shutdown
	httpServer := &http.Server{
		Addr:    ":" + fmt.Sprintf("%s", cfg.Server.Port),
		Handler: router,
	}

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		log.Infof("Starting API server on :%s...", cfg.Server.Port)
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

func startWorker(ctx context.Context, wg *sync.WaitGroup, cfg config.Config) {
	defer wg.Done()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr: cfg.Server.RedisAddress, DB: cfg.Server.RedisDB},
		asynq.Config{
			Concurrency: cfg.Server.MaxConcurrent,

			Queues: utils.Queues(cfg.Server.QueuePrefix),
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
