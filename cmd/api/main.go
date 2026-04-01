package main

import (
	"dhis2gw/bootstrap"
	"dhis2gw/config"
	"dhis2gw/controllers"
	"dhis2gw/db"
	"dhis2gw/middleware"
	"dhis2gw/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
	"html/template"
)

var splash = `
╺┳┓╻ ╻╻┏━┓┏━┓   ┏━╸┏━┓╺┳╸┏━╸╻ ╻┏━┓╻ ╻   ┏━┓┏━╸┏━┓╻ ╻┏━╸┏━┓   ┏━┓┏━┓╻
 ┃┃┣━┫┃┗━┓┏━┛   ┃╺┓┣━┫ ┃ ┣╸ ┃╻┃┣━┫┗┳┛   ┗━┓┣╸ ┣┳┛┃┏┛┣╸ ┣┳┛   ┣━┫┣━┛┃
╺┻┛╹ ╹╹┗━┛┗━╸   ┗━┛╹ ╹ ╹ ┗━╸┗┻┛╹ ╹ ╹    ┗━┛┗━╸╹┗╸┗┛ ┗━╸╹┗╸   ╹ ╹╹  ╹
`
var client *asynq.Client

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

	client = asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Server.RedisAddress})
	defer func(client *asynq.Client) {
		_ = client.Close()
	}(client)

	router := gin.Default()

	// Define template functions
	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s) // Mark string as safe HTML
		},
	}
	// Load templates with custom functions
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob(
		cfg.Server.TemplatesDirectory + "/*"))
	router.SetHTMLTemplate(tmpl)

	// Serve static files
	router.Static("/static", cfg.Server.StaticDirectory)

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
	// Handle error response when a route is not defined
	router.NoRoute(func(c *gin.Context) {
		c.String(404, "Page Not Found!")
	})

	port := cfg.Server.Port
	if err := router.Run(":" + fmt.Sprintf("%s", port)); err != nil {
		log.Fatalf("Could not start GIN server: %v", err)
	}
}
