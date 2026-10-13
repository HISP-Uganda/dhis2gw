package pbs

import (
	"net"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
)

type Client struct {
	gql      graphql.Client
	tokenSrc JWTTokenSource
}

func NewClient(baseURL string, ts JWTTokenSource) *Client {
	baseTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   50 * time.Second, // connect timeout
			KeepAlive: 50 * time.Second,
		}).DialContext,

		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 240 * time.Second, // wait for server to start responding
		ExpectContinueTimeout: 1 * time.Second,

		// Optional but nice:
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		ForceAttemptHTTP2:   true,
	}
	rt := &authRoundTripper{
		// base:     http.DefaultTransport,
		base:     baseTransport,
		tokenSrc: ts,
	}

	httpClient := &http.Client{
		Timeout:   240 * time.Second,
		Transport: rt,
	}
	gqlClient := graphql.NewClient(baseURL, httpClient)
	return &Client{gql: gqlClient, tokenSrc: ts}
}

func (c *Client) Gql() graphql.Client { return c.gql }

// authRoundTripper injects Authorization header dynamically
type authRoundTripper struct {
	base     http.RoundTripper
	tokenSrc JWTTokenSource
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := rt.tokenSrc.Token(req.Context())
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+tok.Value)
	return rt.base.RoundTrip(req)
}

// NewBareClient returns a GraphQL client without auth injection.
// Used internally by PBSTokenSource for login/refresh mutations.
func NewBareClient(baseURL string) graphql.Client {
	httpClient := &http.Client{Timeout: 240 * time.Second}
	return graphql.NewClient(baseURL, httpClient)
}
