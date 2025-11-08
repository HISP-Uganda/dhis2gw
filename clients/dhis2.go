package clients

import (
	"dhis2gw/config"
	"errors"
	"strings"

	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
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
