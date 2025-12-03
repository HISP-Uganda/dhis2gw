package main

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Message         string `json:"message"`
	PasswordExpired bool   `json:"password_expired"`
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Message      string `json:"message,omitempty"`
}

func login(client *resty.Client, baseURL, username, password string) (*LoginResponse, error) {
	var result LoginResponse
	resp, err := client.R().
		SetBody(LoginRequest{Username: username, Password: password}).
		SetResult(&result).
		Post(baseURL + "/auth/login")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("login failed: %s", resp.String())
	}
	return &result, nil
}

func refreshToken(client *resty.Client, baseURL, refreshToken string) (*RefreshResponse, error) {
	var result RefreshResponse
	resp, err := client.R().
		SetBody(RefreshRequest{RefreshToken: refreshToken}).
		SetResult(&result).
		Post(baseURL + "/refresh")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("token refresh failed: %s", resp.String())
	}
	return &result, nil
}

func newRestyClient(cfg *Config) *resty.Client {
	client := resty.New().
		SetTimeout(cfg.Timeout()).
		SetHeader("Accept", "application/json")

	// Inject Authorization header before each request
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		access := tokens.GetAccessToken()
		if access != "" {
			req.SetHeader("Authorization", "Bearer "+access)
		}
		return nil
	})

	// Auto refresh token on 401
	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		if resp.StatusCode() == http.StatusUnauthorized {
			log.Warn("⚠️ Access token expired or invalid — attempting recovery...")

			cfg := GetConfig()

			// If refresh token exists, try refresh first
			if refresh := tokens.GetRefreshToken(); refresh != "" {
				newTokens, err := refreshToken(client, cfg.Server.BaseURL, refresh)
				if err != nil {
					log.Errorf("❌ Token refresh failed: %v", err)
					return err
				}
				tokens.SetTokens(newTokens.AccessToken, newTokens.RefreshToken)
				log.Info("✅ Token refreshed — retrying request")

				resp.Request.SetHeader("Authorization", "Bearer "+tokens.GetAccessToken())
				retryResp, retryErr := resp.Request.Execute(resp.Request.Method, resp.Request.URL)
				if retryErr != nil {
					return retryErr
				}
				*resp = *retryResp
				return nil
			}

			// No refresh token available → must re-login
			log.Warn("⚠️ No refresh token found — re-authenticating...")

			loginResp, err := login(client, cfg.Server.BaseURL, cfg.Server.Username, cfg.Server.Password)
			if err != nil {
				log.Errorf("❌ Re-login failed: %v", err)
				return err
			}
			tokens.SetTokens(loginResp.AccessToken, loginResp.RefreshToken)
			log.Info("✅ Logged in again — retrying request")

			resp.Request.SetHeader("Authorization", "Bearer "+tokens.GetAccessToken())
			retryResp, retryErr := resp.Request.Execute(resp.Request.Method, resp.Request.URL)
			if retryErr != nil {
				return retryErr
			}
			*resp = *retryResp
		}
		return nil
	})

	return client
}
