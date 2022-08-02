package prime_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/prime"
)

func TestDevice(t *testing.T) {
	t.Parallel()

	makeString := func(s string) *string {
		return &s
	}

	tt := []struct {
		name   string
		device *prime.Device
		call   func(device *prime.Device) interface{}
		want   interface{}
	}{
		{
			name:   "get name",
			device: &prime.Device{Client: prime.ClientType{Name: makeString("user name")}},
			call:   func(d *prime.Device) interface{} { return d.GetName() },
			want:   "user name",
		},
		{
			name:   "get name - fallback ",
			device: &prime.Device{ModelAlias: "model alias"},
			call:   func(d *prime.Device) interface{} { return d.GetName() },
			want:   "model alias",
		},
		{
			name:   "get name - second fallback",
			device: &prime.Device{Model: "model"},
			call:   func(d *prime.Device) interface{} { return d.GetName() },
			want:   "model",
		},
		{
			name:   "get type",
			device: &prime.Device{Type: map[string]interface{}{"type": "boiler"}},
			call:   func(d *prime.Device) interface{} { return d.GetType() },
			want:   "boiler",
		},
		{
			name:   "get type - invalid type",
			device: &prime.Device{Type: map[string]interface{}{"type": 1}},
			call:   func(d *prime.Device) interface{} { return d.GetType() },
			want:   "",
		},
		{
			name:   "get type - missing type",
			device: &prime.Device{Type: map[string]interface{}{}},
			call:   func(d *prime.Device) interface{} { return d.GetType() },
			want:   "",
		},
		{
			name:   "get sub type",
			device: &prime.Device{Type: map[string]interface{}{"subtype": "boiler"}},
			call:   func(d *prime.Device) interface{} { return d.GetSubType() },
			want:   "boiler",
		},
		{
			name:   "get sub type - invalid sub type",
			device: &prime.Device{Type: map[string]interface{}{"subtype": 1}},
			call:   func(d *prime.Device) interface{} { return d.GetSubType() },
			want:   "",
		},
		{
			name:   "get sub type - missing sub type",
			device: &prime.Device{Type: map[string]interface{}{}},
			call:   func(d *prime.Device) interface{} { return d.GetSubType() },
			want:   "",
		},

		{
			name:   "supports sub type",
			device: &prime.Device{Type: map[string]interface{}{"supported": map[string]interface{}{"meter": []interface{}{"main_elec"}}}},
			call:   func(d *prime.Device) interface{} { return d.SupportsSubType("meter", "main_elec") },
			want:   true,
		},
		{
			name:   "supports sub type - invalid sub type",
			device: &prime.Device{Type: map[string]interface{}{"supported": map[string]interface{}{"meter": []interface{}{1}}}},
			call:   func(d *prime.Device) interface{} { return d.SupportsSubType("meter", "main_elec") },
			want:   false,
		},
		{
			name:   "supports sub type - missing sub type",
			device: &prime.Device{Type: map[string]interface{}{"supported": map[string]interface{}{"meter": []interface{}{}}}},
			call:   func(d *prime.Device) interface{} { return d.SupportsSubType("meter", "main_elec") },
			want:   false,
		},
		{
			name:   "supports sub type - invalid sub types",
			device: &prime.Device{Type: map[string]interface{}{"supported": map[string]interface{}{"meter": []string{"main_elec"}}}},
			call:   func(d *prime.Device) interface{} { return d.SupportsSubType("meter", "main_elec") },
			want:   false,
		},
		{
			name:   "supports sub type - missing type",
			device: &prime.Device{Type: map[string]interface{}{"supported": map[string]interface{}{}}},
			call:   func(d *prime.Device) interface{} { return d.SupportsSubType("meter", "main_elec") },
			want:   false,
		},
		{
			name:   "supports sub type - invalid types",
			device: &prime.Device{Type: map[string]interface{}{"supported": map[string]string{}}},
			call:   func(d *prime.Device) interface{} { return d.SupportsSubType("meter", "main_elec") },
			want:   false,
		},
		{
			name:   "supports sub type - missing supported",
			device: &prime.Device{Type: map[string]interface{}{}},
			call:   func(d *prime.Device) interface{} { return d.SupportsSubType("meter", "main_elec") },
			want:   false,
		},
		{
			name:   "has service",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {}}},
			call:   func(d *prime.Device) interface{} { return d.HasService("meter_elec") },
			want:   true,
		},
		{
			name:   "has service - missing service",
			device: &prime.Device{},
			call:   func(d *prime.Device) interface{} { return d.HasService("meter_elec") },
			want:   false,
		},
		{
			name:   "has interfaces",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Interfaces: []string{"cmd.meter.get_report", "evt.meter.report"}}}},
			call: func(d *prime.Device) interface{} {
				return d.HasInterfaces("meter_elec", "cmd.meter.get_report", "evt.meter.report")
			},
			want: true,
		},
		{
			name:   "has interfaces - missing interface",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Interfaces: []string{"evt.meter.report"}}}},
			call: func(d *prime.Device) interface{} {
				return d.HasInterfaces("meter_elec", "cmd.meter.get_report", "evt.meter.report")
			},
			want: false,
		},
		{
			name:   "has interfaces - missing service",
			device: &prime.Device{},
			call: func(d *prime.Device) interface{} {
				return d.HasInterfaces("meter_elec", "cmd.meter.get_report", "evt.meter.report")
			},
			want: false,
		},
		{
			name:   "get service property strings",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Props: map[string]interface{}{"sup_units": []interface{}{"W", "kWh"}}}}},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyStrings("meter_elec", "sup_units")
			},
			want: []string{"W", "kWh"},
		},
		{
			name:   "get service property strings - invalid property type",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Props: map[string]interface{}{"sup_units": []interface{}{"W", 1}}}}},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyStrings("meter_elec", "sup_units")
			},
			want: ([]string)(nil),
		},
		{
			name:   "get service property strings - invalid property type",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Props: map[string]interface{}{"sup_units": []string{"W", "kWh"}}}}},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyStrings("meter_elec", "sup_units")
			},
			want: ([]string)(nil),
		},
		{
			name:   "get service property strings - missing property",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Props: map[string]interface{}{}}}},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyStrings("meter_elec", "sup_units")
			},
			want: ([]string)(nil),
		},
		{
			name:   "get service property strings - missing service",
			device: &prime.Device{},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyStrings("meter_elec", "sup_units")
			},
			want: ([]string)(nil),
		},
		{
			name:   "get service property string",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Props: map[string]interface{}{"sup_unit": "W"}}}},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyString("meter_elec", "sup_unit")
			},
			want: "W",
		},
		{
			name:   "get service property string - invalid property type",
			device: &prime.Device{Services: map[string]*prime.Service{"meter_elec": {Props: map[string]interface{}{"sup_unit": 1}}}},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyString("meter_elec", "sup_unit")
			},
			want: "",
		},
		{
			name:   "get service property string - missing service",
			device: &prime.Device{},
			call: func(d *prime.Device) interface{} {
				return d.GetServicePropertyString("meter_elec", "sup_unit")
			},
			want: "",
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.call(tc.device)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateDevices_GetDevice(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name    string
		devices prime.StateDevices
		id      int
		want    *prime.StateDevice
	}{
		{
			name:    "get device by id",
			devices: prime.StateDevices{{ID: 1}},
			id:      1,
			want:    &prime.StateDevice{ID: 1},
		},
		{
			name:    "missing device",
			devices: nil,
			id:      1,
			want:    nil,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.devices.GetDevice(tc.id)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateDevice_GetService(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name        string
		service     *prime.StateDevice
		serviceName string
		want        *prime.StateService
	}{
		{
			name: "get service by name",
			service: &prime.StateDevice{
				Services: []*prime.StateService{
					{Name: "meter_elec"},
				},
			},
			serviceName: "meter_elec",
			want:        &prime.StateService{Name: "meter_elec"},
		},
		{
			name:        "missing service",
			service:     &prime.StateDevice{},
			serviceName: "meter_elec",
			want:        nil,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.service.GetService(tc.serviceName)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateService_GetAttribute(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name          string
		service       *prime.StateService
		attributeName string
		want          *prime.StateAttribute
	}{
		{
			name: "get attribute by name",
			service: &prime.StateService{
				Attributes: []*prime.StateAttribute{
					{Name: "meter"},
				},
			},
			attributeName: "meter",
			want:          &prime.StateAttribute{Name: "meter"},
		},
		{
			name: "get attribute by interface name",
			service: &prime.StateService{
				Attributes: []*prime.StateAttribute{
					{Name: "meter"},
				},
			},
			attributeName: "evt.meter.report",
			want:          &prime.StateAttribute{Name: "meter"},
		},
		{
			name:          "missing attribute",
			service:       &prime.StateService{},
			attributeName: "meter",
			want:          nil,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.service.GetAttribute(tc.attributeName)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateAttribute_GetValue(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name       string
		attribute  *prime.StateAttribute
		properties map[string]string
		want       *prime.StateAttributeValue
	}{
		{
			name: "value",
			attribute: &prime.StateAttribute{
				Values: []*prime.StateAttributeValue{
					{
						Props: map[string]string{"unit": "W"},
					},
				},
			},
			properties: map[string]string{"unit": "W"},
			want: &prime.StateAttributeValue{
				Props: map[string]string{"unit": "W"},
			},
		},
		{
			name: "missing value",
			attribute: &prime.StateAttribute{
				Values: []*prime.StateAttributeValue{
					{
						Props: map[string]string{"unit": "W"},
					},
				},
			},
			properties: map[string]string{"unit": "kWh"},
			want:       nil,
		},
		{
			name:       "no values",
			attribute:  &prime.StateAttribute{},
			properties: map[string]string{"unit": "W"},
			want:       nil,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.attribute.GetValue(tc.properties)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateAttributeValue_GetValue(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name      string
		attribute *prime.StateAttributeValue
		call      func(a *prime.StateAttributeValue) (interface{}, error)
		want      interface{}
		wantErr   bool
	}{
		{
			name:      "get string value",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetStringValue() },
			want:      "test",
		},
		{
			name:      "get string value - error",
			attribute: &prime.StateAttributeValue{Value: 1},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetStringValue() },
			want:      "",
			wantErr:   true,
		},
		{
			name:      "get int value",
			attribute: &prime.StateAttributeValue{Value: int64(1)},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetIntValue() },
			want:      int64(1),
		},
		{
			name:      "get int value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetIntValue() },
			want:      int64(0),
			wantErr:   true,
		},
		{
			name:      "get int value",
			attribute: &prime.StateAttributeValue{Value: float64(1)},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetFloatValue() },
			want:      float64(1),
		},
		{
			name:      "get int value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetFloatValue() },
			want:      float64(0),
			wantErr:   true,
		},
		{
			name:      "get bool value",
			attribute: &prime.StateAttributeValue{Value: true},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolValue() },
			want:      true,
		},
		{
			name:      "get bool value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolValue() },
			want:      false,
			wantErr:   true,
		},

		{
			name:      "get string array value",
			attribute: &prime.StateAttributeValue{Value: []string{"test"}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetStringArrayValue() },
			want:      []string{"test"},
		},
		{
			name:      "get string array value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetStringArrayValue() },
			want:      ([]string)(nil),
			wantErr:   true,
		},
		{
			name:      "get int array value",
			attribute: &prime.StateAttributeValue{Value: []int64{1}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetIntArrayValue() },
			want:      []int64{1},
		},
		{
			name:      "get int array value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetIntArrayValue() },
			want:      ([]int64)(nil),
			wantErr:   true,
		},
		{
			name:      "get float array value",
			attribute: &prime.StateAttributeValue{Value: []float64{1}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetFloatArrayValue() },
			want:      []float64{1},
		},
		{
			name:      "get float array value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetFloatArrayValue() },
			want:      ([]float64)(nil),
			wantErr:   true,
		},
		{
			name:      "get bool array value",
			attribute: &prime.StateAttributeValue{Value: []bool{true}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolArrayValue() },
			want:      []bool{true},
		},
		{
			name:      "get bool array value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolArrayValue() },
			want:      ([]bool)(nil),
			wantErr:   true,
		},
		{
			name:      "get string map value",
			attribute: &prime.StateAttributeValue{Value: map[string]string{"key": "test"}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetStringMapValue() },
			want:      map[string]string{"key": "test"},
		},
		{
			name:      "get string map value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetStringMapValue() },
			want:      (map[string]string)(nil),
			wantErr:   true,
		},
		{
			name:      "get int map value",
			attribute: &prime.StateAttributeValue{Value: map[string]int64{"key": 1}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetIntMapValue() },
			want:      map[string]int64{"key": 1},
		},
		{
			name:      "get int map value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetIntMapValue() },
			want:      (map[string]int64)(nil),
			wantErr:   true,
		},
		{
			name:      "get float map value",
			attribute: &prime.StateAttributeValue{Value: map[string]float64{"key": 1}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetFloatMapValue() },
			want:      map[string]float64{"key": 1},
		},
		{
			name:      "get float map value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetFloatMapValue() },
			want:      (map[string]float64)(nil),
			wantErr:   true,
		},
		{
			name:      "get bool map value",
			attribute: &prime.StateAttributeValue{Value: map[string]bool{"key": true}},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolMapValue() },
			want:      map[string]bool{"key": true},
		},
		{
			name:      "get bool map value - error",
			attribute: &prime.StateAttributeValue{Value: "test"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolMapValue() },
			want:      (map[string]bool)(nil),
			wantErr:   true,
		},
		{
			name:      "get bool map value - invalid payload",
			attribute: &prime.StateAttributeValue{Value: json.RawMessage(`"`)},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolMapValue() },
			want:      (map[string]bool)(nil),
			wantErr:   true,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.call(tc.attribute)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}
