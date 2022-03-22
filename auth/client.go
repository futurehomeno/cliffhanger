package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/hub"
)

// ProxyClientConfig is a configuration object for an authentication proxy client.
type ProxyClientConfig struct {
	PartnerCode string
	Token       string
	URL         string
	Retry       int
	RetryDelay  time.Duration
	Timeout     time.Duration
}

// setDefaults sets default configuration for a authentication proxy client.
func (cfg *ProxyClientConfig) setDefaults() {
	if cfg.URL == "" {
		cfg.URL = ProxyURL(hub.EnvProd)
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
}

// ProxyClient is an interface representing a service responsible for utilization of the Partners API.
type ProxyClient interface {
	// ExchangeAuthorizationCode exchanges a one-time authorization code for the access token response.
	ExchangeAuthorizationCode(code string) (*OAuth2TokenResponse, error)
	// ExchangeRefreshToken exchanges a refresh token for the access token response.
	ExchangeRefreshToken(refreshToken string) (*OAuth2TokenResponse, error)
}

// NewProxyClient creates new instance of a proxy proxyClient.
func NewProxyClient(cfg *ProxyClientConfig) ProxyClient {
	cfg.setDefaults()

	return &proxyClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// proxyClient is a private implementation of an authentication proxy client.
type proxyClient struct {
	cfg    *ProxyClientConfig
	client *http.Client
}

// ExchangeAuthorizationCode exchanges a one-time authorization code for the access token response.
func (c *proxyClient) ExchangeAuthorizationCode(code string) (*OAuth2TokenResponse, error) {
	request := &OAuth2AuthCodeProxyRequest{AuthCode: code, PartnerCode: c.cfg.PartnerCode}

	return c.getToken(request, c.cfg.URL+"/api/control/edge/proxy/auth-code")
}

// ExchangeRefreshToken exchanges a refresh token for the access token response.
func (c *proxyClient) ExchangeRefreshToken(refreshToken string) (*OAuth2TokenResponse, error) {
	request := OAuth2RefreshProxyRequest{RefreshToken: refreshToken, PartnerCode: c.cfg.PartnerCode}

	return c.getToken(request, c.cfg.URL+"/api/control/edge/proxy/refresh")
}

// getToken retrieves token from Partners API.
func (c *proxyClient) getToken(request interface{}, url string) (*OAuth2TokenResponse, error) {
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("proxy proxyClient: failed to create request: %w", err)
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.cfg.Token)

	for i := 0; i <= c.cfg.Retry; i++ {
		var response *OAuth2TokenResponse

		response, err = c.requestToken(r)
		if err == nil {
			return response, nil
		}

		if i < c.cfg.Retry {
			log.Errorf("proxy proxyClient: Partner API is not responding with success, retrying in %s...", c.cfg.RetryDelay.String())

			time.Sleep(c.cfg.RetryDelay)
		}
	}

	return nil, err
}

// requestToken requests token from Partners API.
func (c *proxyClient) requestToken(r *http.Request) (*OAuth2TokenResponse, error) {
	response, err := c.client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("proxy proxyClient: failed to retrieve token from partner API due to an error: %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("proxy proxyClient: failed to retrieve token from partner API, received status code: %d", response.StatusCode)
	}

	responseData, err := ioutil.ReadAll(response.Body)
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
