package main

import (
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
		config.DHIS2GWConf.API.DHIS2GWBaseURL,
		config.DHIS2GWConf.API.DHIS2GWDHIS2User,
		config.DHIS2GWConf.API.DHIS2GWDHIS2Password)
	tasks.SetClient(dhis2Client)

	var wg sync.WaitGroup

	wg.Add(2)
	go startAPIServer(&wg)
	go startWorker(&wg)

	wg.Wait()
}

func startAPIServer(wg *sync.WaitGroup) {
	defer wg.Done()

	router := gin.Default()

	// Define template functions
	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s) // Mark string as safe HTML
		},
	}
	// Load templates with custom functions
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob(
		config.DHIS2GWConf.Server.TemplatesDirectory + "/*"))
	router.SetHTMLTemplate(tmpl)

	// Serve Static Files
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
	// Handle error response when a route is not defined
	router.NoRoute(func(c *gin.Context) {
		c.String(404, "Page Not Found!")
	})

	if err := router.Run(":" + fmt.Sprintf("%s", config.DHIS2GWConf.Server.Port)); err != nil {
		log.Fatalf("Could not start GIN server: %v", err)
	}

}

func startWorker(wg *sync.WaitGroup) {
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

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run asynq worker: %v", err)
	}
}
