package suite_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/suite"

	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

type RouterTestSuite struct {
	suite.Suite

	mqtt   *fimpgo.MqttTransport
	router *cliffSuite.Router
}

func TestRouterTestSuite(t *testing.T) { //nolint:paralleltest
	suite.Run(t, new(RouterTestSuite))
}

func (suite *RouterTestSuite) SetupTest() {
	suite.mqtt = fimpgo.NewMqttTransport("tcp://localhost:11883", "router-test-suite", "", "", true, 1, 1, nil)
	suite.Require().NoError(suite.mqtt.Start(10 * time.Second))
	suite.Require().NoError(suite.mqtt.Subscribe("#"))

	suite.router = cliffSuite.NewTestRouter(suite.T(), suite.mqtt)
	suite.router.Start()
}

func (suite *RouterTestSuite) TearDownTest() {
	suite.router.Stop()
	suite.mqtt.Stop()
}

func (suite *RouterTestSuite) TestRouter() {
	topic := "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:out_bin_switch/ad:1_0"
	assertionsTimeout := 50 * time.Millisecond

	// first set of expectations
	suite.router.Expect(
		cliffSuite.ExpectBool(topic, "cmd.binary.set", "out_bin_switch", true),
	)

	addr, err := fimpgo.NewAddressFromString(topic)
	suite.Require().NoError(err)

	msg := fimpgo.NewBoolMessage("cmd.binary.set", "out_bin_switch", true, nil, nil, nil)

	err = suite.mqtt.Publish(addr, msg)
	suite.Require().NoError(err)

	suite.router.AssertExpectations(assertionsTimeout)

	// second set of expectations
	suite.router.Expect(
		cliffSuite.ExpectBool(topic, "cmd.binary.set", "out_bin_switch", false),
	)

	msg = fimpgo.NewBoolMessage("cmd.binary.set", "out_bin_switch", false, nil, nil, nil)

	err = suite.mqtt.Publish(addr, msg)
	suite.Require().NoError(err)

	suite.router.AssertExpectations(assertionsTimeout)
}
