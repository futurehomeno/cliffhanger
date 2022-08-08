package prime_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/prime"
)

func TestDevices(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name    string
		devices prime.Devices
		call    func(devices prime.Devices) interface{}
		want    interface{}
	}{
		{
			name:    "filter by thing id",
			devices: prime.Devices{{ID: 1, ThingID: makeInt(1)}, {ID: 2, ThingID: makeInt(1)}, {ID: 3, ThingID: makeInt(2)}},
			call:    func(devices prime.Devices) interface{} { return devices.FilterByThingID(1) },
			want:    prime.Devices{{ID: 1, ThingID: makeInt(1)}, {ID: 2, ThingID: makeInt(1)}},
		},
		{
			name:    "filter by thing id - no matches",
			devices: prime.Devices{{ID: 1, ThingID: makeInt(1)}, {ID: 2, ThingID: makeInt(1)}, {ID: 3, ThingID: makeInt(2)}},
			call:    func(devices prime.Devices) interface{} { return devices.FilterByThingID(3) },
			want:    (prime.Devices)(nil),
		},
		{
			name:    "filter by thing id - requested 0",
			devices: prime.Devices{{ID: 1, ThingID: makeInt(1)}, {ID: 2, ThingID: makeInt(1)}, {ID: 3, ThingID: makeInt(2)}},
			call:    func(devices prime.Devices) interface{} { return devices.FilterByThingID(0) },
			want:    (prime.Devices)(nil),
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.call(tc.devices)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDevice(t *testing.T) {
	t.Parallel()

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
			name:   "get thing id",
			device: &prime.Device{ID: 1, ThingID: makeInt(1)},
			call:   func(d *prime.Device) interface{} { return d.GetThingID() },
			want:   1,
		},
		{
			name:   "get thing id - nil value",
			device: &prime.Device{ID: 1},
			call:   func(d *prime.Device) interface{} { return d.GetThingID() },
			want:   0,
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

		{
			name:   "get addresses",
			device: &prime.Device{Services: map[string]*prime.Service{"s1": {Addr: "address1"}, "s2": {Addr: "address2"}}},
			call: func(d *prime.Device) interface{} {
				return d.GetAddresses()
			},
			want: []string{"address1", "address2"},
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

func TestStateDevices_FindDevice(t *testing.T) {
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

			got := tc.devices.FindDevice(tc.id)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateDevice_GetAttributeValue(t *testing.T) {
	t.Parallel()

	makeDevice := func(service, attribute string, value interface{}, timestamp string) *prime.StateDevice {
		return &prime.StateDevice{
			Services: []*prime.StateService{
				{
					Name: service,
					Attributes: []*prime.StateAttribute{
						{
							Name: attribute,
							Values: []*prime.StateAttributeValue{
								{
									Value:     value,
									Timestamp: timestamp,
								},
							},
						},
					},
				},
			},
		}
	}

	tt := []struct {
		name      string
		device    *prime.StateDevice
		call      func(d *prime.StateDevice) (interface{}, time.Time)
		wantValue interface{}
		wantTime  time.Time
	}{
		{
			name:   "get attribute string value",
			device: makeDevice("test_service", "test_attribute", "test_value", "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeStringValue("test_service", "test_attribute", nil)
			},
			wantValue: "test_value",
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute string value - missing value",
			device: &prime.StateDevice{},
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeStringValue("test_service", "test_attribute", nil)
			},
			wantValue: "",
			wantTime:  time.Time{},
		},
		{
			name:   "get attribute string value - incorrect value type",
			device: makeDevice("test_service", "test_attribute", 1, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeStringValue("test_service", "test_attribute", nil)
			},
			wantValue: "",
			wantTime:  time.Time{},
		},
		{
			name:   "get attribute string value - incorrect timestamp",
			device: makeDevice("test_service", "test_attribute", "test_value", "wrong timestamp"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeStringValue("test_service", "test_attribute", nil)
			},
			wantValue: "",
			wantTime:  time.Time{},
		},
		{
			name:   "get attribute int value",
			device: makeDevice("test_service", "test_attribute", int64(1), "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeIntValue("test_service", "test_attribute", nil)
			},
			wantValue: int64(1),
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute float value",
			device: makeDevice("test_service", "test_attribute", float64(1), "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeFloatValue("test_service", "test_attribute", nil)
			},
			wantValue: float64(1),
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute bool value",
			device: makeDevice("test_service", "test_attribute", true, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeBoolValue("test_service", "test_attribute", nil)
			},
			wantValue: true,
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute string array value",
			device: makeDevice("test_service", "test_attribute", []string{"test_value"}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeStringArrayValue("test_service", "test_attribute", nil)
			},
			wantValue: []string{"test_value"},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute int array value",
			device: makeDevice("test_service", "test_attribute", []int64{1}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeIntArrayValue("test_service", "test_attribute", nil)
			},
			wantValue: []int64{1},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute float array value",
			device: makeDevice("test_service", "test_attribute", []float64{1}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeFloatArrayValue("test_service", "test_attribute", nil)
			},
			wantValue: []float64{1},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute bool array value",
			device: makeDevice("test_service", "test_attribute", []bool{true}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeBoolArrayValue("test_service", "test_attribute", nil)
			},
			wantValue: []bool{true},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute string map value",
			device: makeDevice("test_service", "test_attribute", map[string]string{"key": "test_value"}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeStringMapValue("test_service", "test_attribute", nil)
			},
			wantValue: map[string]string{"key": "test_value"},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute int map value",
			device: makeDevice("test_service", "test_attribute", map[string]int64{"key": 1}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeIntMapValue("test_service", "test_attribute", nil)
			},
			wantValue: map[string]int64{"key": 1},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute float map value",
			device: makeDevice("test_service", "test_attribute", map[string]float64{"key": 1}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeFloatMapValue("test_service", "test_attribute", nil)
			},
			wantValue: map[string]float64{"key": 1},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
		{
			name:   "get attribute bool map value",
			device: makeDevice("test_service", "test_attribute", map[string]bool{"key": true}, "2022-08-15 12:15:30 +0100"),
			call: func(d *prime.StateDevice) (interface{}, time.Time) {
				return d.GetAttributeBoolMapValue("test_service", "test_attribute", nil)
			},
			wantValue: map[string]bool{"key": true},
			wantTime:  time.Date(2022, 8, 15, 12, 15, 30, 0, time.FixedZone("", 1*60*60)),
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotValue, gotTime := tc.call(tc.device)

			assert.Equal(t, tc.wantValue, gotValue)
			assert.Equal(t, tc.wantTime, gotTime)
		})
	}
}

func TestStateDevice_FindAttributeValue(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name          string
		device        *prime.StateDevice
		serviceName   string
		attributeName string
		properties    map[string]string
		want          *prime.StateAttributeValue
	}{
		{
			name: "success",
			device: &prime.StateDevice{
				Services: []*prime.StateService{
					{
						Name: "meter_elec",
						Attributes: []*prime.StateAttribute{
							{
								Name: "meter",
								Values: []*prime.StateAttributeValue{
									{
										Props: map[string]string{"unit": "W"},
									},
								},
							},
						},
					},
				},
			},
			serviceName:   "meter_elec",
			attributeName: "meter",
			properties:    map[string]string{"unit": "W"},
			want: &prime.StateAttributeValue{
				Props: map[string]string{"unit": "W"},
			},
		},
		{
			name: "missing value",
			device: &prime.StateDevice{
				Services: []*prime.StateService{
					{
						Name: "meter_elec",
						Attributes: []*prime.StateAttribute{
							{
								Name: "meter",
								Values: []*prime.StateAttributeValue{
									{
										Props: map[string]string{"unit": "W"},
									},
								},
							},
						},
					},
				},
			},
			serviceName:   "meter_elec",
			attributeName: "meter",
			properties:    map[string]string{"unit": "kWh"},
			want:          nil,
		},
		{
			name: "missing attribute",
			device: &prime.StateDevice{
				Services: []*prime.StateService{
					{
						Name: "meter_elec",
					},
				},
			},
			serviceName:   "meter_elec",
			attributeName: "meter",
			properties:    map[string]string{"unit": "kWh"},
			want:          nil,
		},
		{
			name:          "missing service",
			device:        &prime.StateDevice{},
			serviceName:   "meter_elec",
			attributeName: "meter",
			properties:    map[string]string{"unit": "kWh"},
			want:          nil,
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.device.FindAttributeValue(tc.serviceName, tc.attributeName, tc.properties)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateDevice_FindService(t *testing.T) {
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

			got := tc.service.FindService(tc.serviceName)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateService_FindAttribute(t *testing.T) {
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

			got := tc.service.FindAttribute(tc.attributeName)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateAttribute_FindValue(t *testing.T) {
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

			got := tc.attribute.FindValue(tc.properties)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStateAttributeValue_Get(t *testing.T) {
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
			name:      "get bool map value - corrupted",
			attribute: &prime.StateAttributeValue{Value: json.RawMessage(`"`)},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetBoolMapValue() },
			want:      (map[string]bool)(nil),
			wantErr:   true,
		},
		{
			name:      "get time",
			attribute: &prime.StateAttributeValue{Timestamp: "2022-08-15 12:30:10 +0100"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetTime() },
			want:      time.Date(2022, 8, 15, 12, 30, 10, 0, time.FixedZone("", 1*60*60)),
			wantErr:   false,
		},
		{
			name:      "get time - error",
			attribute: &prime.StateAttributeValue{Timestamp: "unparsable string"},
			call:      func(a *prime.StateAttributeValue) (interface{}, error) { return a.GetTime() },
			want:      time.Time{},
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

func makeInt(v int) *int {
	return &v
}

func makeString(v string) *string {
	return &v
}
