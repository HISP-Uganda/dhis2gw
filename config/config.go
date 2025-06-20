package config

import (
	goflag "flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/lib/pq"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

// DHIS2GWConf is the global conf
var DHIS2GWConf Config
var ForceSync *bool
var SkipSync *bool
var PilotMode *bool
var StartDate *string
var EndDate *string
var DisableHTTPServer *bool
var SkipRequestProcessing *bool // used to ignore the attempt to send request. Don't produce or consume requests
var SkipScheduleProcessing *bool
var SkipFectchingByDate *bool
var DIS2GWDHIS2ServersConfigMap = make(map[string]ServerConf)
var ShowVersion *bool

const VERSION = "1.0.0"

func init() {
	// ./dhis2gw --config-file /etc/dhis2gw/dhis2gw.yml
	var configFilePath, configDir, conf_dDir string
	currentOS := runtime.GOOS
	switch currentOS {
	case "windows":
		configDir = "C:\\ProgramData\\dhis2gw"
		configFilePath = "C:\\ProgramData\\dhis2gw\\dhis2gw.yml"
		conf_dDir = "C:\\ProgramData\\dhis2gw\\conf.d"
	case "darwin", "linux":
		configFilePath = "/etc/dhis2gw/dhis2gw.yml"
		configDir = "/etc/dhis2gw/"
		conf_dDir = "/etc/dhis2gw/conf.d" // for the conf.d directory where to dump server confs
	default:
		fmt.Println("Unsupported operating system")
		return
	}

	configFile := flag.String("config-file", configFilePath,
		"The path to the configuration file of the application")

	startDate := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")
	ForceSync = flag.Bool("force-sync", false, "Whether to forcefully sync organisation unit hierarchy")
	SkipSync = flag.Bool("skip-sync", false, "Whether to skip measurements sync.")
	PilotMode = flag.Bool("pilot-mode", false, "Whether we're running integrator in pilot mode")
	StartDate = flag.String("start-date", startDate, "Date from which to start fetching data (YYYY-MM-DD)")
	EndDate = flag.String("end-date", endDate, "Date until which to fetch data (YYYY-MM-DD)")
	DisableHTTPServer = flag.Bool("disable-http-server", false, "Whether to disable HTTP Server")
	SkipRequestProcessing = flag.Bool("skip-request-processing", false, "Whether to skip requests processing")
	SkipScheduleProcessing = flag.Bool("skip-schedule-processing", false, "Whether to skip schedule processing")
	SkipFectchingByDate = flag.Bool("skip-fetching-by-date", false, "Whether to skip fetching measurements by start and end date")
	ShowVersion = flag.Bool("version", false, "Display version of DIS2GW Integrator")
	// FakeSyncToBaseDHIS2 = flag.Bool("fake-sync-to-base-dhis2", false, "Whether to fake sync to base DHIS2")

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
	if *ShowVersion {
		fmt.Println("OneHealth Gateway: ", VERSION)
		os.Exit(1)
	}

	viper.SetConfigName("dhis2gw")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	if len(*configFile) > 0 {
		viper.SetConfigFile(*configFile)
		// log.Printf("Config File %v", *configFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// log.Fatalf("Configuration File: %v Not Found", *configFile, err)
			panic(fmt.Errorf("Fatal Error %w \n", err))

		} else {
			log.Fatalf("Error Reading Config: %v", err)

		}
	}

	err := viper.Unmarshal(&DHIS2GWConf)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		err = viper.ReadInConfig()
		if err != nil {
			log.Fatalf("unable to reread configuration into global conf: %v", err)
		}
		_ = viper.Unmarshal(&DHIS2GWConf)
	})
	viper.WatchConfig()

	v := viper.New()
	v.SetConfigType("json")

	fileList, err := getFilesInDirectory(conf_dDir)
	if err != nil {
		log.WithError(err).Info("Error reading directory")
	}
	// Loop through the files and read each one
	for _, file := range fileList {
		v.SetConfigFile(file)

		if err := v.ReadInConfig(); err != nil {
			log.WithError(err).WithField("File", file).Error("Error reading config file:")
			continue
		}

		// Unmarshal the config data into your structure
		var config ServerConf
		if err := v.Unmarshal(&config); err != nil {
			log.WithError(err).WithField("File", file).Error("Error unmarshaling config file:")
			continue
		}
		DIS2GWDHIS2ServersConfigMap[config.Name] = config

		// Now you can use the config structure as needed
		// fmt.Printf("Configuration from %s: %+v\n", file, config)
	}
	v.OnConfigChange(func(e fsnotify.Event) {
		if err := v.ReadInConfig(); err != nil {
			log.WithError(err).WithField("File", e.Name).Error("Error reading config file:")
		}

		// Unmarshal the config data into your structure
		var config ServerConf
		if err := v.Unmarshal(&config); err != nil {
			log.WithError(err).WithField("File", e.Name).Fatalf("Error unmarshaling config file:")
		}
		DIS2GWDHIS2ServersConfigMap[config.Name] = config
	})
	v.WatchConfig()
}

// Config is the top level cofiguration object
type Config struct {
	Database struct {
		URI string `mapstructure:"uri" env:"DHIS2GW_DB" env-default:"postgres://postgres:postgres@localhost/dhis2gw?sslmode=disable"`
	} `yaml:"database"`

	Server struct {
		Host                        string `mapstructure:"host" env:"DHIS2GW_HOST" env-default:"localhost"`
		Port                        string `mapstructure:"http_port" env:"DHIS2GW_SERVER_PORT" env-description:"Server port" env-default:"9090"`
		ProxyPort                   string `mapstructure:"proxy_port" env:"DHIS2GW_PROXY_PORT" env-description:"Server port" env-default:"9191"`
		RedisAddress                string `mapstructure:"redis_address" env:"DHIS2GW_REDIS" env-description:"Redis address" env-default:"127.0.0.1:6379"`
		MaxRetries                  int    `mapstructure:"max_retries" env:"DHIS2GW_MAX_RETRIES" env-default:"3"`
		StartOfSubmissionPeriod     string `mapstructure:"start_submission_period" env:"DHIS2GW_START_SUBMISSION_PERIOD" env-default:"18"`
		EndOfSubmissionPeriod       string `mapstructure:"end_submission_period" env:"DHIS2GW_END_SUBMISSION_PERIOD" env-default:"24"`
		MaxConcurrent               int    `mapstructure:"max_concurrent" env:"DHIS2GW_MAX_CONCURRENT" env-default:"5"`
		SkipRequestProcessing       bool   `mapstructure:"skip_request_processing" env:"DHIS2GW_SKIP_REQUEST_PROCESSING" env-default:"false"`
		ForceSync                   bool   `mapstructure:"force_sync" env:"DHIS2GW_FORCE_SYNC" env-default:"false"` // Assume OU hierarchy already there
		SyncOn                      bool   `mapstructure:"sync_on" env:"DHIS2GW_SYNC_ON" env-default:"true"`
		FakeSyncToBaseDHIS2         bool   `mapstructure:"fake_sync_to_base_dhis2" env:"DHIS2GW_FAKE_SYNC_TO_BASE_DHIS2" env-default:"false"`
		RequestProcessInterval      int    `mapstructure:"request_process_interval" env:"DHIS2GW_REQUEST_PROCESS_INTERVAL" env-default:"4"`
		Dhis2JobStatusCheckInterval int    `mapstructure:"dhis2_job_status_check_interval" env:"DHIS2_JOB_STATUS_CHECK_INTERVAL" env-description:"The DHIS2 job status check interval in seconds" env-default:"30"`
		TemplatesDirectory          string `mapstructure:"templates_directory" env:"DHIS2GW_TEMPLATES_DIR" env-default:"./templates"`
		StaticDirectory             string `mapstructure:"static_directory" env:"DHIS2GW_STATIC_DIR" env-default:"./static"`
		LogDirectory                string `mapstructure:"logdir" env:"DHIS2GW_LOGDIR" env-default:"/var/log/dhis2gw"`
		MigrationsDirectory         string `mapstructure:"migrations_dir" env:"DHIS2GW_MIGRATTIONS_DIR" env-default:"file:///usr/share/dhis2gw/db/migrations"`
		UseSSL                      string `mapstructure:"use_ssl" env:"DHIS2GW_USE_SSL" env-default:"true"`
		SSLClientCertKeyFile        string `mapstructure:"ssl_client_certkey_file" env:"SSL_CLIENT_CERTKEY_FILE" env-default:""`
		SSLServerCertKeyFile        string `mapstructure:"ssl_server_certkey_file" env:"SSL_SERVER_CERTKEY_FILE" env-default:""`
		SSLTrustedCAFile            string `mapstructure:"ssl_trusted_cafile" env:"SSL_TRUSTED_CA_FILE" env-default:""`
		TimeZone                    string `mapstructure:"timezone" env:"DISPATCHER2_TIMEZONE" env-default:"Africa/Kampala" env-description:"The time zone used for this dispatcher2 deployment"`
	} `yaml:"server"`

	API struct {
		DHIS2Country              string `mapstructure:"dhis2_country" env:"dhis2_country" env-description:"The DIS2GW base DHIS2 Country"`
		DHIS2BaseURL              string `mapstructure:"dhis2_base_url" env:"dhis2_base_url" env-description:"The DIS2GW base DHIS2 instance base API URL"`
		DHIS2User                 string `mapstructure:"dhis2_user" env:"dhis2_user" env-description:"The DIS2GW base DHIS2 username"`
		DHIS2Password             string `mapstructure:"dhis2_password" env:"dhis2_password" env-description:"The DIS2GW base DHIS2  user password"`
		DHIS2PAT                  string `mapstructure:"dhis2_pat" env:"dhis2_pat" env-description:"The DIS2GW base DHIS2  Personal Access Token"`
		SaveResponse              string `mapstructure:"save_response" env:"save_response" env-description:"Whether to save the response from DHIS2 in the database" env-default:"true"`
		DHIS2DataSet              string `mapstructure:"dhis2_data_set" env:"dhis2_data_set" env-description:"The DIS2GW base DHIS2 DATASET"`
		DHIS2AttributeOptionCombo string `mapstructure:"dhis2_attribute_option_combo" env:"dhis_2_attribute_option_combo" env-description:"The DIS2GW base DHIS2 Attribute Option Combo"`
		DHIS2AuthMethod           string `mapstructure:"dhis2_auth_method" env:"dhis2_auth_method" env-description:"The DIS2GW base DHIS2  Authentication Method"`
		DHIS2TreeIDs              string `mapstructure:"dhis2_tree_i_ds" env:"dhis2_tree_i_ds" env-description:"The DIS2GW base DHIS2  orgunits top level ids"`
		DHIS2FacilityLevel        int    `mapstructure:"dhis2_facility_level" env:"dhis2_facility_level" env-description:"The base DHIS2  Orgunit Level for health facilities" env-default:"5"`
		DHIS2DistrictLevelName    string `mapstructure:"dhis2_district_oulevel_name"  env:"DHIS2GW_DHIS2_DISTRICT_OULEVEL_NAME" env-description:"The DIS2GW base DHIS2 OU Level name for districts" env-default:"District/City"`
		CCDHIS2Servers            string `mapstructure:"cc_dhis2_servers" env:"cc_dhis2_servers" env-description:"The CC DHIS2 instances to receive copy of facilities"`
		CCDHIS2HierarchyServers   string `mapstructure:"cc_dhis2_hierarchy_servers" env:"cc_dhis2_hierarchy_servers" env-description:"The DIS2GW CC DHIS2 instances to receive copy of OU hierarchy"`
		CCDHIS2CreateServers      string `mapstructure:"cc_dhis2_create_servers" env:"cc_dhis2_create_servers" env-description:"The DIS2GW CC DHIS2 instances to receive copy of OU creations"`
		CCDHIS2UpdateServers      string `mapstructure:"cc_dhis2_update_servers" env:"cc_dhis2_update_servers" env-description:"The DIS2GW CC DHIS2 instances to receive copy of OU updates"`
		CCDHIS2OuGroupAddServers  string `mapstructure:"cc_dhis2_ou_group_add_servers" env:"cc_dhis2_ou_group_add_servers" env-description:"The DIS2GW CC DHIS2 instances APIs used to add ous to groups"`
		MetadataBatchSize         int    `mapstructure:"metadata_batch_size" env:"metadata_batch_size" env-description:"The DIS2GW Metadata items to chunk in a metadata request" env-default:"50"`
		SyncCronExpression        string `mapstructure:"sync_cron_expression" env:"sync_cron_expression" env-description:"The DIS2GW Measurements Syncronisation Cron Expression" env-default:"0 0-23/6 * * *"`
		RetryCronExpression       string `mapstructure:"retry_cron_expression" env:"retry_cron_expression" env-description:"The DIS2GW request retry Cron Expression" env-default:"*/5 * * * *"`
	} `yaml:"api"`
}

type ServerConf struct {
	ID                      int64          `mapstructure:"id" json:"-"`
	UID                     string         `mapstructure:"uid" json:"uid,omitempty"`
	Name                    string         `mapstructure:"name" json:"name" validate:"required"`
	Username                string         `mapstructure:"username" json:"username"`
	Password                string         `mapstructure:"password" json:"password,omitempty"`
	IsProxyServer           bool           `mapstructure:"isProxyserver" json:"isProxyServer,omitempty"`
	SystemType              string         `mapstructure:"systemType" json:"systemType,omitempty"`
	EndPointType            string         `mapstructure:"endpointType" json:"endPointType,omitempty"`
	AuthToken               string         `mapstructure:"authToken" db:"auth_token" json:"AuthToken"`
	IPAddress               string         `mapstructure:"IPAddress"  json:"IPAddress"`
	URL                     string         `mapstructure:"URL" json:"URL" validate:"required,url"`
	CCURLS                  pq.StringArray `mapstructure:"CCURLS" json:"CCURLS,omitempty"`
	CallbackURL             string         `mapstructure:"callbackURL" json:"callbackURL,omitempty"`
	HTTPMethod              string         `mapstructure:"HTTPMethod" json:"HTTPMethod" validate:"required"`
	AuthMethod              string         `mapstructure:"AuthMethod" json:"AuthMethod" validate:"required"`
	AllowCallbacks          bool           `mapstructure:"allowCallbacks" json:"allowCallbacks,omitempty"`
	AllowCopies             bool           `mapstructure:"allowCopies" json:"allowCopies,omitempty"`
	UseAsync                bool           `mapstructure:"useAsync" json:"useAsync,omitempty"`
	UseSSL                  bool           `mapstructure:"useSSL" json:"useSSL,omitempty"`
	ParseResponses          bool           `mapstructure:"parseResponses" json:"parseResponses,omitempty"`
	SSLClientCertKeyFile    string         `mapstructure:"sslClientCertkeyFile" json:"sslClientCertkeyFile"`
	StartOfSubmissionPeriod int            `mapstructure:"startSubmissionPeriod" json:"startSubmissionPeriod"`
	EndOfSubmissionPeriod   int            `mapstructure:"endSubmissionPeriod" json:"endSubmissionPeriod"`
	XMLResponseXPATH        string         `mapstructure:"XMLResponseXPATH"  json:"XMLResponseXPATH"`
	JSONResponseXPATH       string         `mapstructure:"JSONResponseXPATH" json:"JSONResponseXPATH"`
	Suspended               bool           `mapstructure:"suspended" json:"suspended,omitempty"`
	URLParams               map[string]any `mapstructure:"URLParams" json:"URLParams,omitempty"`
	Created                 time.Time      `mapstructure:"created" json:"created,omitempty"`
	Updated                 time.Time      `mapstructure:"updated" json:"updated,omitempty"`
	AllowedSources          []string       `mapstructure:"allowedSources" json:"allowedSources,omitempty"`
}

func getFilesInDirectory(directory string) ([]string, error) {
	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
