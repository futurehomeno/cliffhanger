package config_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestHandleCmdLogGetLevel(t *testing.T) { //nolint:paralleltest
	makeCommand := func(valueType string, value interface{}) *fimpgo.Message {
		return &fimpgo.Message{
			Payload: &fimpgo.FimpMessage{
				Type:      config.CmdLogSetLevel,
				ValueType: valueType,
				Value:     value,
			},
			Addr: &fimpgo.Address{},
		}
	}

	tests := []struct {
		name       string
		logSetter  func(string) error
		msg        *fimpgo.Message
		want       *fimpgo.Message
		wantErr    bool
		wantLogLvl log.Level
	}{
		{
			name:       "happy path",
			logSetter:  func(s string) error { return nil },
			msg:        makeCommand("string", "error"),
			wantLogLvl: log.ErrorLevel,
		},
		{
			name:      "error when checking payload value",
			logSetter: func(s string) error { return nil },
			msg:       makeCommand("bool", true),
			wantErr:   true,
		},
		{
			name:      "error when parsing log level",
			logSetter: func(s string) error { return nil },
			msg:       makeCommand("string", "dummy"),
			wantErr:   true,
		},
		{
			name:      "error when saving log level",
			logSetter: func(s string) error { return errors.New("test error") },
			msg:       makeCommand("string", "error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := config.HandleCmdLogSetLevel("test", tt.logSetter)

			got := f.Handle(tt.msg)

			if tt.wantErr {
				assert.NotNil(t, got)
				assert.Equal(t, "evt.error.report", got.Payload.Type)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, config.EvtLogLevelReport, got.Payload.Type)
				assert.Equal(t, tt.wantLogLvl.String(), got.Payload.Value)
				assert.Equal(t, tt.wantLogLvl, log.GetLevel())
			}
		})
	}
}

func TestRouteConfig(t *testing.T) { //nolint:paralleltest
	type TestObject struct {
		A string
	}

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Successful getter and setter",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
					t.Helper()

					mDuration := newConfigMock[time.Duration]().mockGetter(time.Second).mockSetter(time.Minute, nil)
					mString := newConfigMock[string]().mockGetter("abc").mockSetter("def", nil)
					mBool := newConfigMock[bool]().mockGetter(true).mockSetter(false, nil)
					mInt := newConfigMock[int]().mockGetter(1).mockSetter(2, nil)
					mFloat := newConfigMock[float32]().mockGetter(1).mockSetter(2, nil)
					mMapString := newConfigMock[map[string]string]().mockGetter(map[string]string{"a": "b"}).mockSetter(map[string]string{"c": "d"}, nil)
					mMapBool := newConfigMock[map[string]bool]().mockGetter(map[string]bool{"a": true}).mockSetter(map[string]bool{"c": false}, nil)
					mMapInt := newConfigMock[map[string]int]().mockGetter(map[string]int{"a": 1}).mockSetter(map[string]int{"c": 2}, nil)
					mMapFloat := newConfigMock[map[string]float32]().mockGetter(map[string]float32{"a": 1}).mockSetter(map[string]float32{"c": 2}, nil)
					mArrayString := newConfigMock[[]string]().mockGetter([]string{"b"}).mockSetter([]string{"d"}, nil)
					mArrayBool := newConfigMock[[]bool]().mockGetter([]bool{true}).mockSetter([]bool{false}, nil)
					mArrayInt := newConfigMock[[]int]().mockGetter([]int{1}).mockSetter([]int{2}, nil)
					mArrayFloat := newConfigMock[[]float32]().mockGetter([]float32{1}).mockSetter([]float32{2}, nil)
					mObject := newConfigMock[*TestObject]().mockGetter(&TestObject{"a"}).mockSetter(&TestObject{"b"}, nil)
					mConfig := newConfigMock[*TestObject]().mockGetter(&TestObject{"a"})
					mLog := newConfigMock[string]().mockGetter("debug").mockSetter("info", nil)

					mocks = append(mocks,
						mString, mBool, mInt, mFloat,
						mMapString, mMapBool, mMapInt, mMapFloat,
						mArrayString, mArrayBool, mArrayInt, mArrayFloat,
						mDuration, mObject, mConfig, mLog,
					)

					routing = append(routing,
						config.RouteCmdConfigGetDuration("test_service", "test_setting_duration", mDuration.getter),
						config.RouteCmdConfigSetDuration("test_service", "test_setting_duration", mDuration.setter),
						config.RouteCmdConfigGetString("test_service", "test_setting_string", mString.getter),
						config.RouteCmdConfigSetString("test_service", "test_setting_string", mString.setter),
						config.RouteCmdConfigGetBool("test_service", "test_setting_bool", mBool.getter),
						config.RouteCmdConfigSetBool("test_service", "test_setting_bool", mBool.setter),
						config.RouteCmdConfigGetInt("test_service", "test_setting_int", mInt.getter),
						config.RouteCmdConfigSetInt("test_service", "test_setting_int", mInt.setter),
						config.RouteCmdConfigGetFloat("test_service", "test_setting_float", mFloat.getter),
						config.RouteCmdConfigSetFloat("test_service", "test_setting_float", mFloat.setter),
						config.RouteCmdConfigGetStringMap("test_service", "test_setting_string_map", mMapString.getter),
						config.RouteCmdConfigSetStringMap("test_service", "test_setting_string_map", mMapString.setter),
						config.RouteCmdConfigGetBoolMap("test_service", "test_setting_bool_map", mMapBool.getter),
						config.RouteCmdConfigSetBoolMap("test_service", "test_setting_bool_map", mMapBool.setter),
						config.RouteCmdConfigGetIntMap("test_service", "test_setting_int_map", mMapInt.getter),
						config.RouteCmdConfigSetIntMap("test_service", "test_setting_int_map", mMapInt.setter),
						config.RouteCmdConfigGetFloatMap("test_service", "test_setting_float_map", mMapFloat.getter),
						config.RouteCmdConfigSetFloatMap("test_service", "test_setting_float_map", mMapFloat.setter),
						config.RouteCmdConfigGetStringArray("test_service", "test_setting_string_array", mArrayString.getter),
						config.RouteCmdConfigSetStringArray("test_service", "test_setting_string_array", mArrayString.setter),
						config.RouteCmdConfigGetBoolArray("test_service", "test_setting_bool_array", mArrayBool.getter),
						config.RouteCmdConfigSetBoolArray("test_service", "test_setting_bool_array", mArrayBool.setter),
						config.RouteCmdConfigGetIntArray("test_service", "test_setting_int_array", mArrayInt.getter),
						config.RouteCmdConfigSetIntArray("test_service", "test_setting_int_array", mArrayInt.setter),
						config.RouteCmdConfigGetFloatArray("test_service", "test_setting_float_array", mArrayFloat.getter),
						config.RouteCmdConfigSetFloatArray("test_service", "test_setting_float_array", mArrayFloat.setter),
						config.RouteCmdConfigGetObject("test_service", "test_setting_object", mObject.getter),
						config.RouteCmdConfigSetObject("test_service", "test_setting_object", mObject.setter),
						config.RouteCmdConfigGetReport("test_service", mConfig.getter),
						config.RouteCmdLogGetLevel("test_service", mLog.getter),
						config.RouteCmdLogSetLevel("test_service", mLog.setter),
					)

					return
				}),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_duration", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_duration_report", "test_service", "1s"),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_duration", "test_service", "1m"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_duration_report", "test_service", "1m"),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_string", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_string_report", "test_service", "abc"),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_string", "test_service", "def"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_string_report", "test_service", "def"),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_bool", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_bool_report", "test_service", true),
						},
					},
					{
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_bool", "test_service", false),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_bool_report", "test_service", false),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_int", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_int_report", "test_service", 1),
						},
					},
					{
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_int", "test_service", 2),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_int_report", "test_service", 2),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_float", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_float_report", "test_service", 1),
						},
					},
					{
						Command: suite.FloatMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_float", "test_service", 2),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_float_report", "test_service", 2),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_string_map", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_string_map_report", "test_service", map[string]string{"a": "b"}),
						},
					},
					{
						Command: suite.StringMapMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_string_map", "test_service", map[string]string{"c": "d"}),
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_string_map_report", "test_service", map[string]string{"c": "d"}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_bool_map", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectBoolMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_bool_map_report", "test_service", map[string]bool{"a": true}),
						},
					},
					{
						Command: suite.BoolMapMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_bool_map", "test_service", map[string]bool{"c": false}),
						Expectations: []*suite.Expectation{
							suite.ExpectBoolMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_bool_map_report", "test_service", map[string]bool{"c": false}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_int_map", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_int_map_report", "test_service", map[string]int64{"a": 1}),
						},
					},
					{
						Command: suite.IntMapMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_int_map", "test_service", map[string]int64{"c": 2}),
						Expectations: []*suite.Expectation{
							suite.ExpectIntMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_int_map_report", "test_service", map[string]int64{"c": 2}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_float_map", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_float_map_report", "test_service", map[string]float64{"a": 1}),
						},
					},
					{
						Command: suite.FloatMapMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_float_map", "test_service", map[string]float64{"c": 2}),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_float_map_report", "test_service", map[string]float64{"c": 2}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_string_array", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectStringArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_string_array_report", "test_service", []string{"b"}),
						},
					},
					{
						Command: suite.StringArrayMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_string_array", "test_service", []string{"d"}),
						Expectations: []*suite.Expectation{
							suite.ExpectStringArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_string_array_report", "test_service", []string{"d"}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_bool_array", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectBoolArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_bool_array_report", "test_service", []bool{true}),
						},
					},
					{
						Command: suite.BoolArrayMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_bool_array", "test_service", []bool{false}),
						Expectations: []*suite.Expectation{
							suite.ExpectBoolArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_bool_array_report", "test_service", []bool{false}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_int_array", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectIntArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_int_array_report", "test_service", []int64{1}),
						},
					},
					{
						Command: suite.IntArrayMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_int_array", "test_service", []int64{2}),
						Expectations: []*suite.Expectation{
							suite.ExpectIntArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_int_array_report", "test_service", []int64{2}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_float_array", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_float_array_report", "test_service", []float64{1}),
						},
					},
					{
						Command: suite.FloatArrayMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_float_array", "test_service", []float64{2}),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_float_array_report", "test_service", []float64{2}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_test_setting_object", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_object_report", "test_service", &TestObject{A: "a"}),
						},
					},
					{
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_object", "test_service", &TestObject{A: "b"}),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_object_report", "test_service", &TestObject{A: "b"}),
						},
					},
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.get_report", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.report", "test_service", &TestObject{A: "a"}),
						},
					},

					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.get_level", "test_service"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.level_report", "test_service", "debug"),
						},
					},
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.log.set_level", "test_service", "info"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.log.level_report", "test_service", "info"),
						},
					},
				},
			},
			{
				Name: "Errors and edge cases",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
					t.Helper()

					mDuration := newConfigMock[time.Duration]().mockSetter(time.Second, errors.New("test"))
					mObject := newConfigMock[*TestObject]()
					mIntArray := newConfigMock[[]int]().mockSetter([]int{}, nil)

					mocks = append(mocks, mDuration, mObject)

					routing = append(routing,
						config.RouteCmdConfigSetDuration("test_service", "test_setting_duration", mDuration.setter),
						config.RouteCmdConfigSetObject("test_service", "test_setting_object", mObject.setter),
						config.RouteCmdConfigSetIntArray("test_service", "test_setting_int_array", mIntArray.setter),
					)

					return
				}),
				Nodes: []*suite.Node{
					{
						Name:    "Invalid duration format",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_duration", "test_service", "invalid"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service"),
						},
					},
					{
						Name:    "Invalid value type",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_duration", "test_service", int64(time.Second)),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service"),
						},
					},
					{
						Name:    "Setter error",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_duration", "test_service", "1s"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service"),
						},
					},
					{
						Name:    "Unmarshalling error",
						Command: suite.ObjectMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_object", "test_service", json.RawMessage(`{"a": 1}`)),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:app/rn:test/ad:1", "test_service"),
						},
					},
					{
						Name:    "Properly handle an empty slice",
						Command: suite.IntArrayMessage("pt:j1/mt:cmd/rt:app/rn:test/ad:1", "cmd.config.set_test_setting_int_array", "test_service", []int64{}),
						Expectations: []*suite.Expectation{
							suite.ExpectIntArray("pt:j1/mt:evt/rt:app/rn:test/ad:1", "evt.config.test_setting_int_array_report", "test_service", []int64{}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func newConfigMock[T any]() *configMock[T] {
	return &configMock[T]{}
}

type configMock[T any] struct {
	mock.Mock
}

func (m *configMock[T]) getter() T {
	arg := m.Called()

	return arg.Get(0).(T) //nolint:forcetypeassert
}

func (m *configMock[T]) setter(input T) error {
	arg := m.Called(input)

	return arg.Error(0)
}

func (m *configMock[T]) mockGetter(value T) *configMock[T] {
	m.On("getter").Return(value)

	return m
}

func (m *configMock[T]) mockSetter(value T, err error) *configMock[T] {
	m.On("setter", value).Return(err)

	return m
}
