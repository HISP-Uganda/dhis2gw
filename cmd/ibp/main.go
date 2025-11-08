package main

import (
	"dhis2gw/clients"
	"dhis2gw/config"
	"dhis2gw/utils"
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
)

var tokens = &TokenStore{}

func main() {
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
	tokens.SetTokens(loginResp.AccessToken, loginResp.RefreshToken)

	dhis2Server := clients.Server{
		BaseUrl:    utils.CoalesceString(cfg.DHIS2URL, config.DHIS2GWConf.API.DHIS2BaseURL, "https://play.im.dhis2.org/stable-2-42-3/api/"),
		Username:   utils.CoalesceString(cfg.DHIS2User, config.DHIS2GWConf.API.DHIS2User, "admin"),
		Password:   utils.CoalesceString(cfg.DHIS2Password, config.DHIS2GWConf.API.DHIS2Password, "district"),
		AuthMethod: "Basic",
	}
	dhis2Client, err2 := dhis2Server.NewDhis2Client()
	if err2 != nil || dhis2Client == nil {
		log.Errorf("Failed to create new dhis2 client: %v", err2)
		return
	}
	r, err3 := dhis2Client.GetResource("me", map[string]string{"level": "1", "fields": "id", "pageSize": "1"})
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
		syncErr := SyncProjects(client, cfg.Server.BaseURL)
		if syncErr != nil {
			log.Fatalf("Failed to sync projects: %v", syncErr)
		}
	})
	if err1 != nil {
		log.WithError(err).Error("Failed to sync cron")
	}
	s.StartAsync()

	//}()

	select {} //keep main alive

	//fmt.Println("✅", loginResp.Message)
	//fmt.Println("Access:", loginResp.AccessToken)
	//fmt.Println("Refresh:", loginResp.RefreshToken)
	//router := gin.Default()
	//v2 := router.Group("/api")
	//{
	//	v2.GET("/test", func(c *gin.Context) {
	//		c.JSON(
	//			http.StatusOK, gin.H{"message": "test"})
	//		return
	//	})
	//}
	//_ = router.Run(":" + fmt.Sprintf("%s", "8484"))
}
