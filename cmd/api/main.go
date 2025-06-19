package main

import (
	"dhis2gw/config"
	"dhis2gw/controllers"
	"dhis2gw/db"
	"dhis2gw/middleware"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"
	"html/template"
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
╺┳┓╻ ╻╻┏━┓┏━┓   ┏━╸┏━┓╺┳╸┏━╸╻ ╻┏━┓╻ ╻   ┏━┓┏━╸┏━┓╻ ╻┏━╸┏━┓   ┏━┓┏━┓╻
 ┃┃┣━┫┃┗━┓┏━┛   ┃╺┓┣━┫ ┃ ┣╸ ┃╻┃┣━┫┗┳┛   ┗━┓┣╸ ┣┳┛┃┏┛┣╸ ┣┳┛   ┣━┫┣━┛┃
╺┻┛╹ ╹╹┗━┛┗━╸   ┗━┛╹ ╹ ╹ ┗━╸┗┻┛╹ ╹ ╹    ┗━┛┗━╸╹┗╸┗┛ ┗━╸╹┗╸   ╹ ╹╹  ╹
`
var client *asynq.Client

func main() {
	fmt.Printf(splash)
	client = asynq.NewClient(asynq.RedisClientOpt{Addr: config.DHIS2GWConf.Server.RedisAddress})
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
		config.DHIS2GWConf.Server.TemplatesDirectory + "/*"))
	router.SetHTMLTemplate(tmpl)

	// Serve static files
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

	port := config.DHIS2GWConf.Server.Port
	if err := router.Run(":" + fmt.Sprintf("%s", port)); err != nil {
		log.Fatalf("Could not start GIN server: %v", err)
	}
}
