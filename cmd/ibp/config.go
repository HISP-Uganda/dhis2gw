package main

import (
	"dhis2gw/clients"
	"dhis2gw/config"
	"dhis2gw/models"
	"dhis2gw/utils"
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/goccy/go-json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//go:embed mapping_option_sets.json
var mappingJSON []byte

type Mapping struct {
	Canonical []string          `json:"canonical"`
	Aliases   map[string]string `json:"aliases"`
}

var allMappings map[string]Mapping

// Initialize mappings at startup
func init() {
	allMappings = make(map[string]Mapping)
	if err := json.Unmarshal(mappingJSON, &allMappings); err != nil {
		panic("failed to load mapping_option_sets.json: " + err.Error())
	}
}

// NormalizeValue maps an input value to its canonical DHIS2 option.
// It tries case-insensitive alias matching, then canonical matching.
// Returns the canonical value if found, otherwise returns input unchanged.
func NormalizeValue(uid, input string) string {
	m, ok := allMappings[uid]
	if !ok {
		// UID not in mapping
		return input
	}

	// normalize input
	val := strings.TrimSpace(input)
	lowerVal := strings.ToLower(val)

	// 1. Check aliases (case-insensitive keys)
	for alias, canonical := range m.Aliases {
		if strings.EqualFold(alias, val) {
			return canonical
		}
	}

	// 2. Check canonical values directly (case-insensitive match)
	for _, c := range m.Canonical {
		if strings.EqualFold(c, val) {
			return c
		}
	}

	// 3. Try loose normalization (remove spaces/hyphens)
	normalized := strings.ReplaceAll(lowerVal, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	for _, c := range m.Canonical {
		cNorm := strings.ReplaceAll(strings.ToLower(c), " ", "")
		cNorm = strings.ReplaceAll(cNorm, "-", "")
		if cNorm == normalized {
			return c
		}
	}

	// Fallback: return original input
	return input
}

type StageConfig struct {
	ProgramStage string `yaml:"programStage"`
	// DataValues   map[string]string `yaml:"dataValues"`
	DataValues map[string]string `yaml:"dataValues"`
}

// Config represents ibp.yml structure
type Config struct {
	Debug         bool   `yaml:"debug"`
	DHIS2URL      string `yaml:"dhis2_url" mapstructure:"dhis2_url"`
	DHIS2User     string `yaml:"dhis2_user" mapstructure:"dhis2_user"`
	DHIS2Password string `yaml:"dhis2_password" mapstructure:"dhis2_password"`
	SourceName    string `yaml:"source_name" mapstructure:"source_name"`
	InstanceName  string `yaml:"instance_name" mapstructure:"instance_name"`

	Server struct {
		BaseURL                string `yaml:"BaseURL"`
		Username               string `yaml:"Username" mapstructure:"username"`
		Password               string `yaml:"Password" mapstructure:"password"`
		JWTSecret              string `yaml:"JWTSecret" mapstructure:"jwt_secret"`
		TimeoutSeconds         int    `yaml:"TimeoutSeconds" mapstructure:"timeout_seconds"`
		DataSyncCronExpression string `yaml:"DataSyncCronExpression" mapstructure:"data_sync_cron_expression"`
	} `yaml:"Server" mapstructure:"server"`

	// Mapping in the configuration file
	Program struct {
		OrganisationUnitPath    string                 `yaml:"OrganisationUnitPath" mapstructure:"organisation_unit_path"`
		SearchAttributes        string                 `yaml:"SearchAttributes" mapstructure:"search_attributes"`
		ProgramID               string                 `yaml:"ProgramID"             mapstructure:"program_id"`
		TrackedEntityType       string                 `yaml:"TrackedEntityType"    mapstructure:"tracked_entity_type"`
		TrackedEntityAttributes map[string]string      `yaml:"TrackedEntityAttributes" mapstructure:"tracked_entity_attributes"`
		Stages                  map[string]StageConfig `yaml:"Stages" mapstructure:"stages"` // key = alias (registration, monitoring…)
	} `yaml:"program" mapstructure:"program"`

	// These are necessary for validation not present in the configuration file
	ProgramConfig                     *models.Program
	MandatoryTrackedEntityAttributes  []string
	MandatoryProgramStageDataElements map[string][]string
}

// global safe config reference
var (
	k       = koanf.New(".")
	cfg     *Config
	cfgLock sync.RWMutex
	v       *viper.Viper
)

// LoadConfig initializes viper, reads the file, and watches for changes.
func LoadConfig() (*Config, error) {
	v = viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("ibp")
	v.AddConfigPath("/etc/dhis2gw/conf.d")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	if c.Server.TimeoutSeconds == 0 {
		c.Server.TimeoutSeconds = 10
	}

	cfg = &c
	cfg.InstanceName = "train.ndpme"
	cfg.SourceName = "pbs"
	cfg.DHIS2URL = utils.CoalesceString("", config.DHIS2GWConf.API.DHIS2BaseURL, "https://play.im.dhis2.org/stable-2-42-3/api/")
	cfg.DHIS2User = utils.CoalesceString("", config.DHIS2GWConf.API.DHIS2User, "admin")
	cfg.DHIS2Password = utils.CoalesceString(config.DHIS2GWConf.API.DHIS2Password, "district")
	// Watch for changes
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("🔄 Config file changed: %s", e.Name)
		reloadConfig()
	})

	return cfg, nil
}

// reloadConfig safely updates the global config
func reloadConfig() {
	cfgLock.Lock()
	defer cfgLock.Unlock()

	var newCfg Config
	if err := v.Unmarshal(&newCfg); err != nil {
		log.Printf("⚠️ Failed to reload config: %v", err)
		return
	}

	if newCfg.Server.TimeoutSeconds == 0 {
		newCfg.Server.TimeoutSeconds = 10
	}

	cfg = &newCfg
	log.Printf("✅ Configuration reloaded successfully (base_url=%s)", cfg.Server.BaseURL)
}

// GetConfig returns a thread-safe copy of current config
func GetConfig() *Config {
	cfgLock.RLock()
	defer cfgLock.RUnlock()
	return cfg
}

func LoadConfig2() (*Config, error) {
	configFile := "/etc/dhis2gw/conf.d/ibp.yaml"

	// ✅ Load from YAML (case preserved)
	if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	// ✅ Merge environment overrides (optional)
	if err := k.Load(env.Provider("", ".", nil), nil); err != nil {
		return nil, fmt.Errorf("error loading environment vars: %w", err)
	}

	// ✅ Unmarshal into struct
	var c Config
	if err := k.Unmarshal("", &c); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// ✅ Apply defaults
	if c.Server.TimeoutSeconds == 0 {
		c.Server.TimeoutSeconds = 10
	}

	// ✅ Custom computed fields
	c.InstanceName = "train.ndpme"
	c.SourceName = "pbs"
	c.DHIS2URL = utils.CoalesceString(
		"", config.DHIS2GWConf.API.DHIS2BaseURL, "https://play.im.dhis2.org/stable-2-42-3/api/",
	)
	c.DHIS2User = utils.CoalesceString(
		"", config.DHIS2GWConf.API.DHIS2User, "admin",
	)
	c.DHIS2Password = utils.CoalesceString(
		config.DHIS2GWConf.API.DHIS2Password, "district",
	)

	// ✅ Set global config safely
	cfgLock.Lock()
	cfg = &c
	cfgLock.Unlock()

	// ✅ Watch for file changes
	fileProvider := file.Provider(configFile)
	_ = fileProvider.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("⚠️ Config watch error: %v", err)
			return
		}

		// The event type depends on the underlying watcher.
		// For fsnotify it will be an fsnotify.Event, so we can assert:
		if e, ok := event.(fsnotify.Event); ok {
			log.Printf("🔄 Config file changed: %s", e.Name)
		} else {
			log.Printf("🔄 Config file changed (unknown type): %#v", event)
		}

		reloadConfig2(configFile)
	})

	return cfg, nil
}

// reloadConfig reloads YAML when changed (preserving case)
func reloadConfig2(path string) {
	cfgLock.Lock()
	defer cfgLock.Unlock()

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		log.Printf("⚠️ Failed to reload config: %v", err)
		return
	}

	var newCfg Config
	if err := k.Unmarshal("", &newCfg); err != nil {
		log.Printf("⚠️ Failed to unmarshal reloaded config: %v", err)
		return
	}

	if newCfg.Server.TimeoutSeconds == 0 {
		newCfg.Server.TimeoutSeconds = 10
	}

	cfg = &newCfg
	log.Printf("✅ Configuration reloaded successfully (base_url=%s)", cfg.Server.BaseURL)
}

// Timeout helper
func (c *Config) Timeout() time.Duration {
	return time.Duration(c.Server.TimeoutSeconds) * time.Second
}

func LoadProgramConfig(client *clients.Client) {
	if cfg.Program.ProgramID == "" || !utils.ValidUID(cfg.Program.ProgramID) {
		return
	}
	//var fields = "programStages[id,name,programStageDataElements[compulsary,dataElement[id,name,valueType,optionSetValue,optionSet]]]," +
	//	"programTrackedEntityAttributes[mandatory,valueType,trackedEntityAttribute[id,name,optionSetValue,optionSet]]"

	var fields = "id,name,programStages[id,name,programStageDataElements[compulsary,dataElement[id,name,valueType,optionSetValue,optionSet[id,options[id,code,name]]]]]," +
		"programTrackedEntityAttributes[mandatory,valueType,trackedEntityAttribute[id,name,optionSetValue,optionSet[id,options[id,code,name]]]]"
	resp, err := client.GetResource(fmt.Sprintf("programs/%s", cfg.Program.ProgramID),
		map[string]string{
			"fields": fields,
		})
	if err != nil || resp == nil {
		log.Infof("Failed to load program configuration from DHIS2!!!")
		return
	}
	if resp.StatusCode() == http.StatusOK {
		er := json.Unmarshal(resp.Body(), &cfg.ProgramConfig)
		if er != nil {
			log.Errorf("Failed to load program configuration from DHIS2: %v", er)
		}
		// Immediately get the mandatory attributes and data elements
		if cfg.ProgramConfig != nil {
			cfg.MandatoryTrackedEntityAttributes = []string{}
			for _, a := range cfg.ProgramConfig.ProgramTrackedEntityAttributes {
				if a.Mandatory {
					cfg.MandatoryTrackedEntityAttributes = append(
						cfg.MandatoryTrackedEntityAttributes, a.TrackedEntityAttribute.ID)
				}
			}
			cfg.MandatoryProgramStageDataElements = make(map[string][]string)
			for _, a := range cfg.ProgramConfig.ProgramStages {
				for _, de := range a.ProgramStageDataElements {
					if de.Compulsory && de.DataElement.ID != "" {
						cfg.MandatoryProgramStageDataElements[a.ID] = append(
							cfg.MandatoryProgramStageDataElements[a.ID], de.DataElement.ID)
					}
				}
			}
		}

	}
}
