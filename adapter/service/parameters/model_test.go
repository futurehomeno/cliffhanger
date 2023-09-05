package parameters_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
)

func TestParameterSpecification_ValidateParameter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		spec    *parameters.ParameterSpecification
		param   *parameters.Parameter
		wantErr bool
	}{
		{
			name: "spec: input, int - param: int",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
				Min:        0,
				Max:        10,
			},
			param: parameters.NewIntParameter("1", 5),
		},
		{
			name: "spec: input, int - param: int - lower than allowed",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
				Min:        0,
				Max:        10,
			},
			param:   parameters.NewIntParameter("1", -1),
			wantErr: true,
		},
		{
			name: "spec: input, int - param: int - higher than allowed",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
				Min:        0,
				Max:        10,
			},
			param:   parameters.NewIntParameter("1", 11),
			wantErr: true,
		},
		{
			name: "spec: input, string - param: string",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeString,
			},
			param: parameters.NewStringParameter("1", "test"),
		},
		{
			name: "spec: input, bool - param: bool",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeBool,
			},
			param: parameters.NewBoolParameter("1", true),
		},
		{
			name: "widget type select, int - param: int",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
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
			param: parameters.NewIntParameter("1", 2),
		},
		{
			name: "spec: select, int - param: int - not allowed value",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
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
			param:   parameters.NewIntParameter("1", 4),
			wantErr: true,
		},
		{
			name: "spec: multiselect, string - param: string_array",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
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
			param: parameters.NewStringArrayParameter("1", []string{"1", "2"}),
		},
		{
			name: "spec: multiselect, string - param: string_array - not allowed value",
			spec: &parameters.ParameterSpecification{
				ID:         "1",
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
			param:   parameters.NewStringArrayParameter("1", []string{"1", "4"}),
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
		spec  *parameters.ParameterSpecification
		param *parameters.Parameter
	}{
		{
			name: "spec: input, int - param: string_array",
			spec: &parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
			},
			param: parameters.NewStringArrayParameter("1", []string{"1", "2"}),
		},
		{
			name: "spec: input, int - param: int_array",
			spec: &parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
			},
			param: parameters.NewIntArrayParameter("1", []int{1, 2}),
		},
		{
			name: "spec: input, int - param: bool",
			spec: &parameters.ParameterSpecification{
				WidgetType: parameters.WidgetTypeInput,
				ValueType:  parameters.ValueTypeInt,
			},
			param: parameters.NewBoolParameter("1", true),
		},
		{
			name: "spec: select, string - param: int_array",
			spec: &parameters.ParameterSpecification{
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
			param: parameters.NewIntArrayParameter("1", []int{1, 2}),
		},
		{
			name: "spec: select, string - param: string_array",
			spec: &parameters.ParameterSpecification{
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
			param: parameters.NewStringArrayParameter("1", []string{"1", "2"}),
		},
		{
			name: "spec: select, string - param: int",
			spec: &parameters.ParameterSpecification{
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
			param: parameters.NewIntParameter("1", 1),
		},
		{
			name: "spec: multiselect, string - param: int_array",
			spec: &parameters.ParameterSpecification{
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
			param: parameters.NewIntArrayParameter("1", []int{1, 2}),
		},
		{
			name: "spec: multiselect, string - param: bool",
			spec: &parameters.ParameterSpecification{
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
			param: parameters.NewBoolParameter("1", true),
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
		name         string
		paramFactory func() *parameters.Parameter
		wantErr      bool
	}{
		{
			name: "valid parameter",
			paramFactory: func() *parameters.Parameter {
				return parameters.NewIntParameter("1", 5)
			},
		},
		{
			name: "error - empty ID",
			paramFactory: func() *parameters.Parameter {
				return parameters.NewIntParameter("", 5)
			},
			wantErr: true,
		},
		{
			name: "error - value type not allowed",
			paramFactory: func() *parameters.Parameter {
				return &parameters.Parameter{
					ID:        "1",
					ValueType: "test",
					Value:     []byte("5"),
				}
			},
			wantErr: true,
		},
		{
			name: "error - value type not matching value type of value",
			paramFactory: func() *parameters.Parameter {
				return &parameters.Parameter{
					ID:        "1",
					ValueType: parameters.ValueTypeInt,
					Value:     []byte("test"),
				}
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.paramFactory().Validate()
			if tc.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}
