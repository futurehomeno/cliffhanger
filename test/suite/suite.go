package suite

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
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

	s.Init(t)
	defer s.TearDown(t)

	for _, tc := range s.Cases {
		c := tc

		t.Run(c.Name, func(t *testing.T) {
			t.Helper()
			c.Run(t, s.mqtt)
		})
	}
}

// Init initializes the test suite.
func (s *Suite) Init(t *testing.T) {
	t.Helper()

	s.Config.configure()

	s.mqtt = DefaultMQTT(
		s.Config.MQTTClientIDPrefix,
		s.Config.MQTTServerURI,
		s.Config.MQTTUsername,
		s.Config.MQTTPassword,
	)

	err := s.mqtt.Start()
	if err != nil {
		t.Fatalf("failed to start the MQTT client: %s", err)
	}

	err = s.mqtt.Subscribe("#")
	if err != nil {
		t.Fatalf("failed to subscribe to all topics: %s", err)
	}
}

// TearDown tears down the test suite.
func (s *Suite) TearDown(t *testing.T) {
	t.Helper()

	s.mqtt.Stop()
}

// MQTT returns the MQTT transport.
func (s *Suite) MQTT() *fimpgo.MqttTransport {
	return s.mqtt
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
