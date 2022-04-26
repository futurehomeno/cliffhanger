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
	if c.MQTTServerURI == "" {
		c.MQTTServerURI = "tcp://localhost:11883"
	}

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

	s.mqtt = fimpgo.NewMqttTransport(
		s.Config.MQTTServerURI,
		s.Config.MQTTClientIDPrefix,
		s.Config.MQTTUsername,
		s.Config.MQTTPassword,
		true,
		1,
		1,
	)

	s.mqtt.SetDefaultSource("cliffhanger_test_suite")

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

	err := s.mqtt.Start()
	if err != nil {
		t.Fatalf("failed to stop the MQTT client: %s", err)
	}
}
