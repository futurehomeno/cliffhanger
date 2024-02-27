package suite

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	MQTTServerURI      string
	MQTTUsername       string
	MQTTPassword       string
	MQTTClientIDPrefix string
}

func (c *Config) configure() {
	if c.MQTTClientIDPrefix == "" {
		c.MQTTClientIDPrefix = "cliffhanger_test_suite"
	}
}

func NewSuite() *Suite {
	return &Suite{}
}

type Suite struct {
	Cases  []*Case
	Config Config

	mqtt *fimpgo.MqttTransport
}

func (s *Suite) WithCases(cases ...*Case) *Suite {
	s.Cases = append(s.Cases, cases...)

	return s
}

func (s *Suite) Run(t *testing.T) {
	t.Helper()

	s.init(t)
	defer s.tearDown(t)

	for _, tc := range s.Cases {
		c := tc

		t.Run(c.Name, func(t *testing.T) {
			t.Helper()
			c.Run(t, s.mqtt)
		})
	}
}

func (s *Suite) init(t *testing.T) {
	t.Helper()

	s.Config.configure()

	s.mqtt = DefaultMQTT(
		s.Config.MQTTClientIDPrefix,
		s.Config.MQTTServerURI,
		s.Config.MQTTUsername,
		s.Config.MQTTPassword,
	)

	opts := s.mqtt.Options()
	opts.SetConnectTimeout(time.Second)
	opts.SetPingTimeout(time.Second)
	opts.SetAutoReconnect(false)

	s.mqtt.SetOptions(opts)

	log.Debugf("Starting the MQTT client with the following options: %s", spew.Sdump(opts))

	err := s.mqtt.Start()
	if err != nil {
		t.Fatalf("failed to start the MQTT client: %s", err)
	}

	err = s.mqtt.Subscribe("#")
	if err != nil {
		t.Fatalf("failed to subscribe to all topics: %s", err)
	}
}

func (s *Suite) tearDown(t *testing.T) {
	t.Helper()

	s.mqtt.Stop()
}

func DefaultMQTT(clientID, url, user, pass string) *fimpgo.MqttTransport {
	if url == "" {
		url = "tcp://localhost:11883"
	}

	mqtt := fimpgo.NewMqttTransport(
		url,
		clientID,
		user,
		pass,
		true,
		1,
		1,
	)

	mqtt.SetDefaultSource(clientID)

	return mqtt
}
