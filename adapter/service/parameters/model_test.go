package parameters_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
)

func TestParameterSpecification_ValidateParameter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		spec    parameters.ParameterSpecification
		param   parameters.Parameter
		wantErr bool
	}{
		{
			name: "spec: input, int - param: int",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
				Min:        0,
				Max:        10,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     5,
			},
		},
		{
			name: "spec: input, int - param: int - lower than allowed",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
				Min:        0,
				Max:        10,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     -1,
			},
			wantErr: true,
		},
		{
			name: "spec: input, int - param: int - higher than allowed",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
				Min:        0,
				Max:        10,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     11,
			},
			wantErr: true,
		},
		{
			name: "spec: input, string - param: string",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeString,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeString,
				Value:     "test",
			},
		},
		{
			name: "spec: input, bool - param: bool",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeBool,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeBool,
				Value:     true,
			},
		},
		{
			name: "widget type select, int - param: int",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeSelect,
				ValueType:  parameters.ValueTypeInt,
				Options: parameters.SelectOptions{
					{
						Value: 1,
					},
					{
						Value: 2,
					},
					{
						Value: 3,
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     2,
			},
		},
		{
			name: "spec: select, int - param: int - not allowed value",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeSelect,
				ValueType:  parameters.ValueTypeInt,
				Options: parameters.SelectOptions{
					{
						Value: 1,
					},
					{
						Value: 2,
					},
					{
						Value: 3,
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     4,
			},
			wantErr: true,
		},
		{
			name: "spec: multiselect, string - param: string_array",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeMultiSelect,
				ValueType:  parameters.ValueTypeStringArray,
				Options: parameters.SelectOptions{
					{
						Value: "1",
					},
					{
						Value: "2",
					},
					{
						Value: "3",
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeStringArray,
				Value:     []string{"1", "2"},
			},
		},
		{
			name: "spec: multiselect, string - param: string_array - not allowed value",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeMultiSelect,
				ValueType:  parameters.ValueTypeStringArray,
				Options: parameters.SelectOptions{
					{
						Value: "1",
					},
					{
						Value: "2",
					},
					{
						Value: "3",
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeStringArray,
				Value:     []string{"2", "test"},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.spec.ValidateParameter(tc.param)
			if tc.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestParameterSpecification_ValidateParameter_MismatchingWidgetsAndTypes_Error(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		spec  parameters.ParameterSpecification
		param parameters.Parameter
	}{
		{
			name: "spec: input, int - param: string_array",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeStringArray,
				Value:     []string{"1", "2"},
			},
		},
		{
			name: "spec: input, int - param: int_array",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeIntArray,
				Value:     []int{1, 2},
			},
		},
		{
			name: "spec: input, int - param: bool",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeBool,
				Value:     true,
			},
		},
		{
			name: "spec: select, string - param: int_array",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeSelect,
				ValueType:  parameters.ValueTypeString,
				Options: parameters.SelectOptions{
					{
						Value: "1",
					},
					{
						Value: "2",
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeIntArray,
				Value:     []int{1, 2},
			},
		},
		{
			name: "spec: select, string - param: string_array",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeSelect,
				ValueType:  parameters.ValueTypeString,
				Options: parameters.SelectOptions{
					{
						Value: "1",
					},
					{
						Value: "2",
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeStringArray,
				Value:     []string{"1", "2"},
			},
		},
		{
			name: "spec: select, string - param: int",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeSelect,
				ValueType:  parameters.ValueTypeString,
				Options: parameters.SelectOptions{
					{
						Value: "1",
					},
					{
						Value: "2",
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     1,
			},
		},
		{
			name: "spec: multiselect, string - param: int_array",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeMultiSelect,
				ValueType:  parameters.ValueTypeStringArray,
				Options: parameters.SelectOptions{
					{
						Value: "1",
					},
					{
						Value: "2",
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeIntArray,
				Value:     []int{1, 2},
			},
		},
		{
			name: "spec: multiselect, string - param: bool",
			spec: parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeMultiSelect,
				ValueType:  parameters.ValueTypeStringArray,
				Options: parameters.SelectOptions{
					{
						Value: "1",
					},
					{
						Value: "2",
					},
				},
			},
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeBool,
				Value:     true,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.spec.ValidateParameter(tc.param)

			assert.Error(t, err)
		})
	}
}

func TestParameter_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		param   parameters.Parameter
		wantErr bool
	}{
		{
			name: "valid parameter",
			param: parameters.Parameter{
				ID:        "1",
				ValueType: parameters.ValueTypeInt,
				Value:     5,
			},
		},
		{
			name: "error - empty ID",
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     5,
			},
			wantErr: true,
		},
		{
			name: "error - value type not allowed",
			param: parameters.Parameter{
				ValueType: parameters.ValueType("test"),
				Value:     5,
			},
			wantErr: true,
		},
		{
			name: "error - value type not matching value type of value",
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     "test",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.param.Validate()
			if tc.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestParameter_GetValue_JSONProcessing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		param      parameters.Parameter
		assertions func(p parameters.Parameter)
	}{
		{
			name: "int value",
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeInt,
				Value:     1,
			},
			assertions: func(p parameters.Parameter) {
				val, err := p.IntValue()

				assert.NoError(t, err)
				assert.Equal(t, 1, val)
			},
		},
		{
			name: "int_array value",
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeIntArray,
				Value:     []int{1, 2},
			},
			assertions: func(p parameters.Parameter) {
				val, err := p.IntArrayValue()

				assert.NoError(t, err)
				assert.Equal(t, []int{1, 2}, val)
			},
		},
		{
			name: "string_array value",
			param: parameters.Parameter{
				ValueType: parameters.ValueTypeStringArray,
				Value:     []string{"1", "2"},
			},
			assertions: func(p parameters.Parameter) {
				val, err := p.StringArrayValue()

				assert.NoError(t, err)
				assert.Equal(t, []string{"1", "2"}, val)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(tc.param)
			require.NoError(t, err)

			var p parameters.Parameter

			err = json.Unmarshal(b, &p)
			require.NoError(t, err)

			tc.assertions(p)
		})
	}
}
