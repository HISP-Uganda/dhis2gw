package clients

import (
	"dhis2gw/config"
	"errors"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"strings"
)

var Dhis2Client *Client
var Dhis2Server *Server

func init() {
	InitDhis2Server()
	Dhis2Client, _ = Dhis2Server.NewDhis2Client()
}

func GetDHIS2BaseURL(url string) (string, error) {
	if strings.Contains(url, "/api/") {
		pos := strings.Index(url, "/api/")
		return url[:pos], nil
	}
	return url, errors.New("URL doesn't contain /api/ part")
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
	}
	return &Client{
		RestClient: client,
		BaseURL:    baseUrl + "/api",
	}, nil
}
