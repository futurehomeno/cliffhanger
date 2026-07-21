package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/httpclient"
	"github.com/futurehomeno/cliffhanger/hub"
)

type ProxyClientConfig struct {
	PartnerCode string
	Token       string
	URL         string
	// Endpoint, when set, resolves the base URL and bearer token per exchange instead of using
	// the static URL/Token. It lets adapters whose proxy URL or hub token is only known at call
	// time (e.g. resolved from the hub environment and a rotating hub token) use ProxyClient
	// without a bespoke exchanger. A returned error fails the exchange. URL/Token are ignored
	// when Endpoint is set.
	Endpoint   func() (baseURL, token string, err error)
	Retry      int
	RetryDelay time.Duration
	Timeout    time.Duration
	Headers    map[string]string
}

func (cfg *ProxyClientConfig) setDefaults() {
	if cfg.URL == "" {
		cfg.URL = ProxyURL(hub.EnvProd)
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
}

type ProxyClient interface {
	// ExchangeAuthorizationCode exchanges a one-time authorization code for the access token response.
	ExchangeAuthorizationCode(code string) (*OAuth2TokenResponse, error)
	// ExchangeRefreshToken exchanges a refresh token for the access token response.
	ExchangeRefreshToken(refreshToken string) (*OAuth2TokenResponse, error)
}

func NewProxyClient(cfg *ProxyClientConfig) ProxyClient {
	cfg.setDefaults()

	headers := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		headers[k] = v
	}

	cfgCopy := *cfg
	cfgCopy.Headers = headers

	return &proxyClient{
		cfg: &cfgCopy,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

type proxyClient struct {
	cfg    *ProxyClientConfig
	client *http.Client
}

func (c *proxyClient) ExchangeAuthorizationCode(code string) (*OAuth2TokenResponse, error) {
	request := &OAuth2AuthCodeProxyRequest{AuthCode: code, PartnerCode: c.cfg.PartnerCode}

	return c.getToken(request, "/api/control/edge/proxy/auth-code")
}

func (c *proxyClient) ExchangeRefreshToken(refreshToken string) (*OAuth2TokenResponse, error) {
	request := OAuth2RefreshProxyRequest{RefreshToken: refreshToken, PartnerCode: c.cfg.PartnerCode}

	return c.getToken(request, "/api/control/edge/proxy/refresh")
}

// endpoint resolves the base URL and bearer token for one exchange, lazily via cfg.Endpoint
// when set, otherwise from the static cfg.URL/cfg.Token.
func (c *proxyClient) endpoint() (baseURL, token string, err error) {
	if c.cfg.Endpoint != nil {
		return c.cfg.Endpoint()
	}

	return c.cfg.URL, c.cfg.Token, nil
}

func (c *proxyClient) getToken(request any, path string) (*OAuth2TokenResponse, error) {
	baseURL, token, err := c.endpoint()
	if err != nil {
		return nil, fmt.Errorf("proxy proxyClient: failed to resolve endpoint: %w", err)
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+path, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("proxy proxyClient: failed to create request: %w", err)
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+token)

	for k, v := range c.cfg.Headers {
		if http.CanonicalHeaderKey(k) == "Authorization" || http.CanonicalHeaderKey(k) == "Content-Type" {
			continue
		}

		r.Header.Add(k, v)
	}

	for i := 0; i <= c.cfg.Retry; i++ {
		var response *OAuth2TokenResponse

		// Each attempt drains the body, so it must be replaced before sending.
		r.Body = io.NopCloser(bytes.NewReader(requestData))

		response, err = c.requestToken(r)
		if err == nil {
			return response, nil
		}

		// An authorization rejection is definitive, retrying would not change the outcome.
		if errors.Is(err, httpclient.ErrUnauthorized) {
			return nil, err
		}

		// A rate-limited endpoint must not be hammered on a local delay; the caller's
		// backoff paces the next attempt.
		if errors.Is(err, httpclient.ErrTooManyRequests) {
			return nil, err
		}

		if i < c.cfg.Retry {
			log.Errorf("proxy proxyClient: Partner API is not responding with success, retrying in %s...", c.cfg.RetryDelay.String())

			time.Sleep(c.cfg.RetryDelay)
		}
	}

	return nil, err
}

func (c *proxyClient) requestToken(r *http.Request) (*OAuth2TokenResponse, error) {
	response, err := c.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("proxy proxyClient: failed to retrieve token from partner API due to an error: %w", err)
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Errorf("close err: %v", err)
		}
	}()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("proxy proxyClient: failed to retrieve token from partner API: %w", httpclient.ErrorFromResponse(response))
	}

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("proxy proxyClient: failed to read response: %w", err)
	}

	tokenResponse := &OAuth2TokenResponse{}

	err = json.Unmarshal(responseData, tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("proxy proxyClient: failed to read response: %w", err)
	}

	return tokenResponse, nil
}
