package suite

import (
	"testing"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
)

type Setup interface {
	apply(t *testing.T, mqtt *fimpgo.MqttTransport, c *Case)
}

type BaseSetup func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []Mock)

func (f BaseSetup) apply(t *testing.T, mqtt *fimpgo.MqttTransport, c *Case) {
	routing, tasks, mocks := f(t, mqtt)

	c.Routing = append(c.Routing, routing...)
	c.Tasks = append(c.Tasks, tasks...)
	c.Mocks = append(c.Mocks, mocks...)
}

type ServiceSetup func(t *testing.T) (service Service, mocks []Mock)

func (f ServiceSetup) apply(t *testing.T, _ *fimpgo.MqttTransport, c *Case) {
	service, mocks := f(t)

	c.Service = service
	c.Mocks = append(c.Mocks, mocks...)
}
