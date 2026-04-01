package main

import (
	// "dhis2gw/clients"
	"dhis2gw/config"
	"dhis2gw/db"
	"dhis2gw/middleware"
	"dhis2gw/models"
	"dhis2gw/utils"
	"fmt"
	"net/http"
	"time"

	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
)

var tokens = &TokenStore{}
var dhis2Client *sdk.Client

func main() {
	runtimeCfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	config.Set(runtimeCfg)
	baseCfg := runtimeCfg.Config
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

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
	log.Infof("Config: %+v", utils.ToPrettyJSON(cfg))

	client := newRestyClient(cfg).
		SetTimeout(cfg.Timeout())
	// login and get tokens
	loginResp, err := login(client, cfg.Server.BaseURL, cfg.Server.Username, cfg.Server.Password)
	if err != nil {
		log.Fatal("Login failed:", err)
	}
	if loginResp.AccessToken == "" {
		log.Fatal("Login failed: no access token")
	}
	tokens.SetTokens(loginResp.AccessToken, loginResp.RefreshToken)

	dhis2Client = sdk.NewClient(
		utils.CoalesceString(cfg.DHIS2URL, baseCfg.API.DHIS2BaseURL, "https://play.im.dhis2.org/stable-2-42-3/api/"),
		utils.CoalesceString(cfg.DHIS2User, baseCfg.API.DHIS2User, "admin"),
		utils.CoalesceString(cfg.DHIS2Password, baseCfg.API.DHIS2Password, "district"))
	//sdkClient.GetResource()

	//dhis2Server := clients.Server{
	//	BaseUrl:    utils.CoalesceString(cfg.DHIS2URL, config.DHIS2GWConf.API.DHIS2BaseURL, "https://play.im.dhis2.org/stable-2-42-3/api/"),
	//	Username:   utils.CoalesceString(cfg.DHIS2User, config.DHIS2GWConf.API.DHIS2User, "admin"),
	//	Password:   utils.CoalesceString(cfg.DHIS2Password, config.DHIS2GWConf.API.DHIS2Password, "district"),
	//	AuthMethod: "Basic",
	//}
	//dhis2Client, err2 := dhis2Server.NewDhis2Client()
	if dhis2Client.Resty == nil {
		log.Errorf("Failed to create new dhis2 client")
		return
	}
	r, err3 := dhis2Client.GetResource("organisationUnits", map[string]string{"level": "1", "fields": "id", "pageSize": "1"})
	if err3 != nil {
		log.Errorf("Failed to authenticate to DHIS2 instance: %v", err3)
	}
	if r.StatusCode() != http.StatusOK {
		log.Errorf("Failed to authenticate to DHIS2 instance")
		return
	} else {
		log.Infof("Successfully authenticated to DHIS2 instance: ping said %s", string(r.Body()))
	}
	// load program
	LoadProgramConfig(dhis2Client)
	log.Infof("Loaded ProgramConfig: %v", utils.ToPrettyJSON(cfg.ProgramConfig.Name))
	// load this after programConfig
	LoadTrackedEntityTypeConfig(dhis2Client)
	log.Infof("Loaded TrackedEntityTypeConfig: %v", utils.ToPrettyJSON(cfg.TrackedEntityTypeConfig))
	if &cfg.ProgramConfig != nil {
		cfg.ProgramConfig.PrintMandatoryDetails(true, true)
		log.Infof("Mandatory Attr: %v", utils.ToPrettyJSON(cfg.MandatoryTrackedEntityAttributes))
		log.Infof("Mandatory DEx: %v", utils.ToPrettyJSON(cfg.MandatoryProgramStageDataElements))
	}

	//go func() {
	// log.Info("Running scheduler go routine")
	// Fetch Project and Do the data sync
	s := gocron.NewScheduler(time.UTC)
	_, err1 := s.Cron(cfg.Server.DataSyncCronExpression).Do(func() {
		log.Info("Running sync projects schedule")
		syncErr := SyncProjects(client, cfg.Server.BaseURL, dhis2Client)
		if syncErr != nil {
			log.Errorf("Failed to sync projects: %v", syncErr)
		}
	})
	if err1 != nil {
		log.WithError(err).Error("Failed to sync cron")
	}
	s.StartAsync()

	//}()

	// select {} //keep main alive

	//fmt.Println("✅", loginResp.Message)
	//fmt.Println("Access:", loginResp.AccessToken)
	//fmt.Println("Refresh:", loginResp.RefreshToken)
	router := gin.Default()
	v2 := router.Group("/api", middleware.BasicAuth(db.GetDB(), nil))
	{
		v2.GET("/test", func(c *gin.Context) {
			c.JSON(
				http.StatusOK, gin.H{"message": "test"})
			return
		})
		v2.GET("/data", GetIBPData)
	}
	_ = router.Run(":" + fmt.Sprintf("%s", "8484"))
}
