package virtualmeter_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/router"
)

func TestSpecification_DeclaresErrorReport(t *testing.T) {
	t.Parallel()

	s := virtualmeter.Specification("test_adapter", "1", "2", nil, nil, nil)

	assert.Contains(t, s.Interfaces, fimptype.Interface{
		Type: fimptype.TypeOut, MsgType: router.EvtErrorReport, ValueType: fimptype.VTypeString, Version: "1",
	})
}
