package clients

import (
	"dhis2gw/config"
	"errors"
	"fmt"
	"net/url"
	"strings"

	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
	"github.com/buger/jsonparser"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	log "github.com/sirupsen/logrus"
)

var Dhis2Client *Client
var Dhis2Server *Server

func init() {
	InitDhis2Server()
	c, err := Dhis2Server.NewDhis2Client()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize DHIS2 client")
	}
	Dhis2Client = c
}

func GetDHIS2BaseURL(url string) (string, error) {
	// Accept URLs with or without /api segment
	if strings.Contains(url, "/api/") {
		pos := strings.Index(url, "/api/")
		return url[:pos], nil
	}
	if strings.HasSuffix(url, "/api") {
		return strings.TrimSuffix(url, "/api"), nil
	}
	if url == "" {
		return "", errors.New("empty DHIS2 base URL")
	}
	return url, nil
}

func InitDhis2Server() {
	Dhis2Server = &Server{
		BaseUrl:    config.DHIS2GWConf.API.DHIS2BaseURL,
		Username:   config.DHIS2GWConf.API.DHIS2User,
		Password:   config.DHIS2GWConf.API.DHIS2Password,
		AuthToken:  config.DHIS2GWConf.API.DHIS2PAT,
		AuthMethod: config.DHIS2GWConf.API.DHIS2AuthMethod,
	}
}

func (s *Server) NewDhis2Client() (*Client, error) {
	client := resty.New()
	baseUrl, err := GetDHIS2BaseURL(s.BaseUrl)
	if err != nil {
		log.WithFields(log.Fields{
			"URL": s.BaseUrl, "Error": err}).Error("Failed to get base URL from URL")
		return nil, err
	}
	client.SetBaseURL(baseUrl + "/api")
	client.SetHeaders(map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"User-Agent":   "HIPS-Uganda DHIS2 CLI",
	})
	client.SetDisableWarn(true)
	switch s.AuthMethod {
	case "Basic":
		client.SetBasicAuth(s.Username, s.Password)
	case "Token":
		client.SetAuthScheme("Token")
		client.SetAuthToken(s.AuthToken)
	default:
		log.WithField("AuthMethod", s.AuthMethod).Warn("Unknown DHIS2 auth method; proceeding without auth")
	}
	return &Client{
		RestClient: client,
		BaseURL:    baseUrl + "/api",
	}, nil
}

// PushDataValues pushes data values to DHIS2
func PushDataValues(dataValues []schema.DataValue) error {
	if len(dataValues) == 0 {
		return nil
	}
	client := GetDhis2Client()
	if client == nil || client.RestClient == nil {
		return errors.New("DHIS2 client is not initialized")
	}
	payload := map[string][]schema.DataValue{
		"dataValues": dataValues,
	}
	resp, err := client.RestClient.R().
		SetBody(payload).
		Post("/dataValueSets")
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Failed to push data values to DHIS2")
		return err
	}
	if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
		log.WithFields(log.Fields{
			"Status":     resp.Status(),
			"StatusCode": resp.StatusCode(),
			"Body":       resp.String(),
		}).Error("Failed to push data values to DHIS2")
		return errors.New("Failed to push data values to DHIS2: " + resp.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Body":  string(resp.Body()),
		}).Error("Failed to unmarshal response from DHIS2")
		return err
	}
	if status, ok := result["status"].(string); ok && status == "ERROR" {
		log.WithFields(log.Fields{
			"Response": result,
		}).Error("DHIS2 returned an error status")
		return errors.New("DHIS2 returned an error status: " + resp.String())
	}
	log.WithFields(log.Fields{
		"Response": result,
	}).Info("Successfully pushed data values to DHIS2")
	return nil

}

func GetDhis2Client() *Client {
	return Dhis2Client
}

type MyTrackedEntity struct {
	TrackedEntity     string        `json:"trackedEntity"`
	OrgUnit           string        `json:"orgUnit"`
	TrackedEntityType string        `json:"trackedEntityType"`
	Attributes        []MyAttribute `json:"attributes"`
	Enrollments       []Enrollment  `json:"enrollments,omitempty"`
}
type MyAttribute struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
	ValueType string `json:"valueType,omitempty"`
}

type Enrollment struct {
	Enrollment string  `json:"enrollment,omitempty"`
	Program    string  `json:"program,omitempty"`
	OrgUnit    string  `json:"orgUnit,omitempty"`
	Events     []Event `json:"events,omitempty"`
}

type Event struct {
	Event        string             `json:"event"`
	Program      string             `json:"program,omitempty"`
	ProgramStage string             `json:"programStage,omitempty"`
	orgUnit      string             `json:"orgUnit,omitempty"`
	OccurredAt   string             `json:"occurredAt,omitempty"`
	ScheduledAt  string             `json:"scheduledAt,omitempty"`
	DataValues   []schema.DataValue `json:"dataValues,omitempty"`
}

func SearchTrackedEntity(
	dhis2client *sdk.Client,
	orgUnit string,
	program string,
	attrs map[string]string, // attributeUID → value
	operator string,
) (bool, []MyTrackedEntity) {
	// Must use url.Values to support multiple "filter" params
	params := url.Values{}
	params.Set("orgUnit", orgUnit)
	params.Set("program", program)
	params.Set("ouMode", "SELECTED")
	params.Set("orgUnitMode", "SELECTED")

	// Default operator = EQ
	if operator == "" {
		operator = "EQ"
	}

	// Add all filters as separate entries
	for attrUID, val := range attrs {
		filterExpr := fmt.Sprintf("%s:%s:%s", attrUID, operator, val)
		params.Add("filter", filterExpr)
	}

	// DHIS2 endpoint
	endpoint := "/tracker/trackedEntities"

	// Use the client's method, but adapt it to accept url.Values
	resp, err := dhis2client.GetResourceValues(endpoint, params)
	if err != nil {
		log.Errorf("SearchTE error calling GetResource: %v", err)
		return false, nil
	}

	// Extract "instances" array
	v, _, _, err := jsonparser.Get(resp.Body(), "instances")
	if err != nil {
		log.Errorf("SearchTE error getting instances: %v", err)
		return false, nil
	}

	var instances []MyTrackedEntity
	if err := json.Unmarshal(v, &instances); err != nil {
		log.WithFields(log.Fields{"Instances": string(v)}).Errorf("SearchTE unmarshal error: %v", err)
		return false, nil
	}

	return len(instances) > 0, instances
}
