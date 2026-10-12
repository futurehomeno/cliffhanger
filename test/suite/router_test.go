package suite_test

import (
	"syscall"
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
	suite.mqtt = fimpgo.NewMqttTransport("tcp://127.0.0.1:11883", "router-test-suite", "", "", true, 1, 1, nil)
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

func (suite *RouterTestSuite) TestConcurrentMatchesOfOneExpectation() {
	topic := "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:out_bin_switch/ad:1_0"

	suite.router.Expect(
		cliffSuite.ExpectBool(topic, "cmd.binary.set", "out_bin_switch", true).
			PublishInResponseFn(func() *fimpgo.Message { return nil }).
			ReplyWithFn(func() *fimpgo.FimpMessage { return nil }),
	)

	addr, err := fimpgo.NewAddressFromString(topic)
	suite.Require().NoError(err)

	msg := fimpgo.NewBoolMessage("cmd.binary.set", "out_bin_switch", true, nil, nil, nil)

	for range 20 {
		suite.Require().NoError(suite.mqtt.Publish(addr, msg))
	}

	suite.router.AssertExpectations(100 * time.Millisecond)
}

func (suite *RouterTestSuite) TestAssertExpectationsDoesNotSpin() {
	topic := "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:out_bin_switch/ad:1_0"

	suite.router.Expect(cliffSuite.ExpectBool(topic, "cmd.binary.set", "out_bin_switch", true).Never())

	var before, after syscall.Rusage

	suite.Require().NoError(syscall.Getrusage(syscall.RUSAGE_SELF, &before))
	suite.router.AssertExpectations(300 * time.Millisecond)
	suite.Require().NoError(syscall.Getrusage(syscall.RUSAGE_SELF, &after))

	cpu := time.Duration(after.Utime.Nano() + after.Stime.Nano() - before.Utime.Nano() - before.Stime.Nano())
	suite.Less(cpu, 100*time.Millisecond)
}

func (suite *RouterTestSuite) TestAtMostOnceDoesNotWaitOutTimeout() {
	topic := "pt:j1/mt:cmd/rt:dev/rn:test/ad:1/sv:out_bin_switch/ad:1_0"

	suite.router.Expect(cliffSuite.ExpectBool(topic, "cmd.binary.set", "out_bin_switch", true).AtMostOnce())

	start := time.Now()
	suite.router.AssertExpectations(time.Second)
	suite.Less(time.Since(start), 500*time.Millisecond)
}
