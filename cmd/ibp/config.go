package main

import (
	"dhis2gw/config"
	"dhis2gw/models"
	"dhis2gw/utils"
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/fsnotify/fsnotify"
	"github.com/goccy/go-json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	log "github.com/sirupsen/logrus"
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
	Debug         bool   `yaml:"Debug"`
	DHIS2URL      string `yaml:"DHIS2URL" mapstructure:"dhis2_url"`
	DHIS2User     string `yaml:"DHIS2User" mapstructure:"dhis2_user"`
	DHIS2Password string `yaml:"DHIS2Password" mapstructure:"dhis2_password"`
	SourceName    string `yaml:"SourceName" mapstructure:"source_name"`
	InstanceName  string `yaml:"InstanceName" mapstructure:"instance_name"`

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
		SearchAttribute         string                 `yaml:"SearchAttribute" mapstructure:"search_attribute"`
		ProgramID               string                 `yaml:"ProgramID"             mapstructure:"program_id"`
		TrackedEntityType       string                 `yaml:"TrackedEntityType"    mapstructure:"tracked_entity_type"`
		TrackedEntityAttributes map[string]string      `yaml:"TrackedEntityAttributes" mapstructure:"tracked_entity_attributes"`
		Stages                  map[string]StageConfig `yaml:"Stages" mapstructure:"stages"` // key = alias (registration, monitoring…)
	} `yaml:"program" mapstructure:"program"`
	Defaults map[string]interface{} `yaml:"Defaults"`

	// These are necessary for validation not present in the configuration file
	ProgramConfig                     *models.Program
	TrackedEntityTypeConfig           *models.TrackedEntityType
	MandatoryTrackedEntityAttributes  map[string]struct{}
	MandatoryProgramStageDataElements map[string][]string
}

// global safe config reference
var (
	k       = koanf.New(".")
	cfg     *Config
	cfgLock sync.RWMutex
)

// GetConfig returns a thread-safe copy of current config
func GetConfig() *Config {
	cfgLock.RLock()
	defer cfgLock.RUnlock()
	return cfg
}

func LoadConfig() (*Config, error) {
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
	c.Defaults = map[string]any{
		"xR2SRSZxDSl": "true",
	}

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
		switch e := event.(type) {
		case fsnotify.Event:
			log.Printf("🔄 Config file changed (fsnotify): %s", e.Name)
		case string:
			log.Printf("🔄 Config file changed (string): %s", e)
		case nil:
			log.Printf("🔄 Config file changed (nil event)")
		default:
			log.Printf("🔄 Config file changed (unknown): %#v", e)
		}

		reloadConfig(configFile)
	})

	return cfg, nil
}

// reloadConfig reloads YAML when changed (preserving case)
func reloadConfig(path string) {
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

func LoadProgramConfig(client *sdk.Client) {
	if cfg.Program.ProgramID == "" || !utils.ValidUID(cfg.Program.ProgramID) {
		return
	}
	//var fields = "programStages[id,name,programStageDataElements[compulsary,dataElement[id,name,valueType,optionSetValue,optionSet]]]," +
	//	"programTrackedEntityAttributes[mandatory,valueType,trackedEntityAttribute[id,name,optionSetValue,optionSet]]"

	var fields = "id,name,programStages[id,name,programStageDataElements[compulsory,dataElement[id,name,valueType,optionSetValue,optionSet[id,options[id,code,name]]]]]," +
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
			cfg.MandatoryTrackedEntityAttributes = map[string]struct{}{}
			for _, a := range cfg.ProgramConfig.ProgramTrackedEntityAttributes {
				if a.Mandatory {
					_, exists := cfg.MandatoryTrackedEntityAttributes[a.TrackedEntityAttribute.ID]
					if !exists {
						cfg.MandatoryTrackedEntityAttributes[a.TrackedEntityAttribute.ID] = struct{}{}
					}
					//cfg.MandatoryTrackedEntityAttributes = append(
					//	cfg.MandatoryTrackedEntityAttributes, a.TrackedEntityAttribute.ID)
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

func LoadTrackedEntityTypeConfig(client *sdk.Client) {
	if cfg.Program.TrackedEntityType == "" || !utils.ValidUID(cfg.Program.TrackedEntityType) {
		return
	}
	var fields = "id,name,trackedEntityTypeAttributes[mandatory,valueType,trackedEntityAttribute[id,name]]"
	resp, err := client.GetResource(fmt.Sprintf("trackedEntityTypes/%s", cfg.Program.TrackedEntityType),
		map[string]string{
			"fields": fields,
		})
	if err != nil || resp == nil {
		log.Infof("Failed to load tracked entity type from DHIS2!!!")
		return
	}

	if resp.StatusCode() == http.StatusOK {
		er := json.Unmarshal(resp.Body(), &cfg.TrackedEntityTypeConfig)
		if er != nil {
			log.Errorf("Failed to load tracked entity type configuration from DHIS2: %v", er)
		}
	}

	if cfg.TrackedEntityTypeConfig != nil {
		for _, a := range cfg.TrackedEntityTypeConfig.TrackedEntityTypeAttributes {
			if a.Mandatory {
				// add a to cfg.MandatoryTrackedEntityAttributes if not existing
				_, exists := cfg.MandatoryTrackedEntityAttributes[a.TrackedEntityAttribute.ID]
				if !exists {
					cfg.MandatoryTrackedEntityAttributes[a.TrackedEntityAttribute.ID] = struct{}{}
				}

			}
		}
	}

}
