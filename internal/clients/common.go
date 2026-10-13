package clients

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	RestClient    *resty.Client
	BaseURL       string
	MaxRetries    int
	BaseDelay     time.Duration
	RateLimit     time.Duration
	Burst         int
	tokens        chan struct{}
	lastCall      time.Time
	rateInit      sync.Once
	mu            sync.Mutex
	BatchSize     int // Optional: for batching requests
	FailDir       string
	RetentionDays int
}

type Server struct {
	BaseUrl    string `json:"base_url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	AuthToken  string `json:"auth_token"`
	AuthMethod string `json:"auth_method"`
}

func (c *Client) GetResource(resourcePath string, params map[string]string) (*resty.Response, error) {
	request := c.RestClient.R()

	if params != nil {
		request.SetQueryParams(params)
	}

	resp, err := request.Get(resourcePath)
	if err != nil {
		log.WithError(err).Infof("Error when calling `GetResource`: %v", err)
	}
	return resp, err
}
func (c *Client) PostResource(resourcePath string, data interface{}, opts ...RequestOption) (*resty.Response, error) {
	req := c.RestClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data)

	for _, opt := range opts {
		if opt != nil {
			opt(req)
		}
	}

	resp, err := req.Post(resourcePath)
	if err != nil {
		log.Errorf("Error when calling `PostResource: %v`", err)
	}
	// Optional: treat non-2xx as an error (often useful in APIs)
	if resp != nil && resp.IsError() {
		// you can wrap a richer error if you want
		return resp, fmt.Errorf("PostResource failed: status=%d body=%s", resp.StatusCode(), string(resp.Body()))
	}

	return resp, nil
}

type RequestOption func(*resty.Request)

// WithQuery adds query parameters (?a=b&c=d)
func WithQuery(params map[string]string) RequestOption {
	return func(r *resty.Request) {
		if len(params) > 0 {
			r.SetQueryParams(params)
		}
	}
}

// WithHeader adds a single header
func WithHeader(key, value string) RequestOption {
	return func(r *resty.Request) {
		if key != "" {
			r.SetHeader(key, value)
		}
	}
}

// WithHeaders adds multiple headers
func WithHeaders(headers map[string]string) RequestOption {
	return func(r *resty.Request) {
		if len(headers) > 0 {
			r.SetHeaders(headers)
		}
	}
}

func WithContext(ctx context.Context) RequestOption {
	return func(r *resty.Request) {
		r.SetContext(ctx)
	}
}

func (c *Client) PutResource(resourcePath string, data interface{}) (*resty.Response, error) {
	resp, err := c.RestClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Put(resourcePath)
	if err != nil {
		log.Errorf("Error when calling `PutResource`: %v", err)
	}
	return resp, err
}

func (c *Client) DeleteResource(resourcePath string) (*resty.Response, error) {
	resp, err := c.RestClient.R().
		Delete(resourcePath)
	if err != nil {
		log.Errorf("Error when calling `DeleteResource`: %v", err)
	}
	return resp, err
}

func (c *Client) PatchResource(resourcePath string, data interface{}) (*resty.Response, error) {
	resp, err := c.RestClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Patch(resourcePath)
	if err != nil {
		log.Errorf("Error when calling `PatchResource`: %v", err)
	}
	return resp, err
}

func (c *Client) GetResourceValues(path string, queryParams url.Values) (*resty.Response, error) {
	request := c.RestClient.R()

	if queryParams != nil {
		request.SetQueryParamsFromValues(queryParams)
	}

	resp, err := request.Get(path)
	if err != nil {
		log.WithError(err).Infof("Error when calling `GetResource`: %v", err)
	}
	return resp, err
}
