package hub

import (
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

// LoadToken loads the hub token from Cloud Bridge.
func LoadToken(serviceName string) (string, error) {
	loader := NewTokenLoader(&TokenLoaderConfig{
		ServiceName: serviceName,
	})

	return loader.LoadToken()
}

// TokenLoaderConfig is a configuration object for a token loader service.
type TokenLoaderConfig struct {
	ServiceName        string
	MQTTServerURI      string
	MQTTUsername       string
	MQTTPassword       string
	MQTTClientIDPrefix string
	Retry              int
	RetryDelay         time.Duration
}

// setDefaults sets default configuration for a token loader service.
func (cfg *TokenLoaderConfig) setDefaults() {
	if cfg.MQTTServerURI == "" {
		cfg.MQTTServerURI = "tcp://localhost:1883"
	}

	if cfg.Retry == 0 {
		cfg.Retry = 7
	}

	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 30 * time.Second
	}

	if cfg.MQTTClientIDPrefix == "" {
		cfg.MQTTClientIDPrefix = cfg.ServiceName
	}

	cfg.MQTTClientIDPrefix += "_hub_token_loader"
}

// TokenLoader is an interface representing a service responsible for loading the hub token.
type TokenLoader interface {
	// LoadToken loads the hub token from Cloud Bridge service.
	LoadToken() (string, error)
}

// NewTokenLoader creates new instance of a token loader service.
func NewTokenLoader(cfg *TokenLoaderConfig) TokenLoader {
	cfg.setDefaults()

	return &tokenLoader{
		cfg: cfg,
	}
}

// tokenLoader is a private implementation of a token loader.
type tokenLoader struct {
	cfg *TokenLoaderConfig
}

// LoadToken loads the hub token from Cloud Bridge service.
func (g *tokenLoader) LoadToken() (string, error) {
	mqtt := fimpgo.NewMqttTransport(g.cfg.MQTTServerURI, g.cfg.MQTTClientIDPrefix, g.cfg.MQTTUsername, g.cfg.MQTTPassword, true, 1, 1)

	if err := mqtt.Start(); err != nil {
		return "", fmt.Errorf("token loader: failed to start MQTT: %w", err)
	}

	mqtt.SetDefaultSource(g.cfg.ServiceName)

	defer mqtt.Stop()

	syncClient := fimpgo.NewSyncClient(mqtt)

	defer syncClient.Stop()

	return g.requestToken(syncClient)
}

// requestToken requests the hub token from Cloud Bridge service using FIMP protocol.
func (g *tokenLoader) requestToken(client *fimpgo.SyncClient) (string, error) {
	responseTopic := fmt.Sprintf("pt:j1/mt:rsp/rt:app/rn:%s/ad:1", g.cfg.ServiceName)

	client.AddSubscription(responseTopic)

	reqMsg := fimpgo.NewStringMessage("cmd.clbridge.get_auth_token", "clbridge", "", nil, nil, nil)

	reqMsg.ResponseToTopic = responseTopic

	var (
		err      error
		response *fimpgo.FimpMessage
	)

	for i := 0; i < g.cfg.Retry; i++ {
		response, err = client.SendFimp("pt:j1/mt:cmd/rt:app/rn:clbridge/ad:1", reqMsg, 5)
		if err == nil {
			break
		}

		log.Errorf("token loader: CloudBridge is not responding, retrying in %s...", g.cfg.RetryDelay.String())

		time.Sleep(g.cfg.RetryDelay)
	}

	if err != nil {
		return "", fmt.Errorf("token loader: failed to retrieve hub token: %w", err)
	}

	if response.Type != "evt.clbridge.auth_token_report" {
		return "", fmt.Errorf("token loader: wrong message type receiced, expected %s, got %s", "evt.clbridge.auth_token_report", response.Type)
	}

	token, err := response.GetStringValue()
	if err != nil {
		return "", fmt.Errorf("token loader: wrong message format: %w", err)
	}

	return token, nil
}
