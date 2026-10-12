//go:build unix

package suite_test

import (
	"syscall"
	"time"

	cliffSuite "github.com/futurehomeno/cliffhanger/test/suite"
)

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
