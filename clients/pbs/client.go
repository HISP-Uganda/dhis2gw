package pbs

import (
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
)

type Client struct {
	gql      graphql.Client
	tokenSrc JWTTokenSource
}

func NewClient(baseURL string, ts JWTTokenSource) *Client {
	// Wrap default transport with our auth injector
	rt := &authRoundTripper{
		base:     http.DefaultTransport,
		tokenSrc: ts,
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
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
	httpClient := &http.Client{Timeout: 30 * time.Second}
	return graphql.NewClient(baseURL, httpClient)
}
