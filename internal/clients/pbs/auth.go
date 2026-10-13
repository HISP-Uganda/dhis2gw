package pbs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/golang-jwt/jwt/v5"
)

// JWT holds the access token and optional refresh token.
type JWT struct {
	Value        string
	Expiry       time.Time
	RefreshToken string
}

// JWTTokenSource provides JWT tokens, refreshing as needed.
type JWTTokenSource interface {
	Token(ctx context.Context) (JWT, error)
	Refresh(ctx context.Context) error
	CanRefresh() bool
}

// --------------------------------------------------------------------
// PBSTokenSource: login + refresh flow
// --------------------------------------------------------------------

type PBSTokenSource struct {
	mu        sync.Mutex
	cur       JWT
	user      string
	pass      string
	ip        string
	proactive time.Duration
	client    graphql.Client // should be a *bare* client
}

func NewPBSTokenSource(baseURL, user, pass, ip string) *PBSTokenSource {
	return &PBSTokenSource{
		user:      user,
		pass:      pass,
		ip:        ip,
		proactive: 60 * time.Second,
		client:    NewBareClient(baseURL), // << here
	}
}

func (p *PBSTokenSource) Token(ctx context.Context) (JWT, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cur.Value == "" || needsRefreshLocked(p.cur, p.proactive) {
		if err := p.refreshLocked(ctx); err != nil {
			return JWT{}, err
		}
	}
	return p.cur, nil
}

func (p *PBSTokenSource) Refresh(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refreshLocked(ctx)
}

func (p *PBSTokenSource) CanRefresh() bool { return true }

func (p *PBSTokenSource) refreshLocked(ctx context.Context) error {
	var (
		access  string
		refresh string
	)

	if p.cur.Value == "" || p.cur.RefreshToken == "" {
		// First-time login
		resp, err := Login(ctx, p.client, p.user, p.pass, p.ip)
		if err != nil {
			return err
		}
		access = resp.Login.GetAccess_token()
		refresh = resp.Login.GetRefresh_token()
	} else {
		// Refresh token flow
		resp, err := Refresh(ctx, p.client, p.cur.Value, p.cur.RefreshToken)
		if err != nil {
			return err
		}
		access = resp.RefreshToken.Access_token
		refresh = resp.RefreshToken.Refresh_token
	}

	p.cur = JWT{
		Value:        access,
		RefreshToken: refresh,
		Expiry:       ParseExpiryFromJWT(access),
	}
	return nil
}

// --------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------

func needsRefreshLocked(tok JWT, proactive time.Duration) bool {
	if tok.Expiry.IsZero() {
		return false
	}
	return time.Now().After(tok.Expiry.Add(-proactive))
}

// ParseExpiryFromJWT extracts "exp" claim as time.Time.
func ParseExpiryFromJWT(token string) time.Time {
	if token == "" {
		return time.Time{}
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var claims jwt.MapClaims
	_, _, err := parser.ParseUnverified(token, &claims)
	if err != nil {
		return time.Time{}
	}
	if v, ok := claims["exp"]; ok {
		switch tv := v.(type) {
		case float64:
			return time.Unix(int64(tv), 0)
		case int64:
			return time.Unix(tv, 0)
		case json.Number:
			if i, err := tv.Int64(); err == nil {
				return time.Unix(i, 0)
			}
		}
	}
	return time.Time{}
}

// --------------------------------------------------------------------
// StaticJWTSource: long-lived tokens
// --------------------------------------------------------------------

type StaticJWTSource struct {
	token string
}

func NewStaticJWTSource(token string) *StaticJWTSource {
	return &StaticJWTSource{token: token}
}

func (s *StaticJWTSource) Token(ctx context.Context) (JWT, error) {
	if s.token == "" {
		return JWT{}, errors.New("pbs: empty static JWT")
	}
	return JWT{Value: s.token, Expiry: ParseExpiryFromJWT(s.token)}, nil
}

func (s *StaticJWTSource) Refresh(ctx context.Context) error { return nil }
func (s *StaticJWTSource) CanRefresh() bool                  { return false }
