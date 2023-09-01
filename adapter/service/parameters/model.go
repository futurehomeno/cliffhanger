package parameters

import (
	"fmt"
)

// ValueType represents a value type.
type ValueType string

// Constants below represent allowed value types.
const (
	ValueTypeInt         ValueType = "int"
	ValueTypeString      ValueType = "string"
	ValueTypeBool        ValueType = "bool"
	ValueTypeIntArray    ValueType = "int_array"
	ValueTypeStringArray ValueType = "string_array"
)

func AllowedValueTypes() []ValueType {
	return []ValueType{
		ValueTypeInt,
		ValueTypeString,
		ValueTypeBool,
		ValueTypeIntArray,
		ValueTypeStringArray,
	}
}

func IsValueTypeAllowed(t ValueType) bool {
	for _, allowed := range AllowedValueTypes() {
		if t == allowed {
			return true
		}
	}

	return false
}

// WidgetType represents a widget type.
type WidgetType string

// Constants below represent allowed widget types.
const (
	WidgetTypeInput       WidgetType = "input"
	WidgetTypeSelect      WidgetType = "select"
	WidgetTypeMultiSelect WidgetType = "multiselect"
)

var (
	// widgetTypeToValueTypeMapping maps widget types to allowed value types.
	widgetTypeToValueTypeMapping = map[WidgetType][]ValueType{
		WidgetTypeInput:       {ValueTypeInt, ValueTypeString, ValueTypeBool},
		WidgetTypeSelect:      {ValueTypeInt, ValueTypeString},
		WidgetTypeMultiSelect: {ValueTypeIntArray, ValueTypeStringArray},
	}
)

// ParameterSpecification represents a parameter specification that must be provided by the Controller.
type ParameterSpecification struct {
	ID           string        `json:"parameter_id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	ValueType    ValueType     `json:"value_type"`
	WidgetType   WidgetType    `json:"widget_type"`
	Options      SelectOptions `json:"options,omitempty"`
	Min          int           `json:"min,omitempty"`
	Max          int           `json:"max,omitempty"`
	DefaultValue any           `json:"default_value"`
	ReadOnly     bool          `json:"read_only"`
}

// ValidateParameter validates a parameter against the specification.
func (s ParameterSpecification) ValidateParameter(p Parameter) error {
	if err := s.validateTypeMatching(p); err != nil {
		return err
	}

	switch s.WidgetType {
	case WidgetTypeInput:
		return s.validateInput(p)
	case WidgetTypeSelect:
		return s.validateSelect(p)
	case WidgetTypeMultiSelect:
		return s.validateMultiSelect(p)
	default:
		return fmt.Errorf("widget type '%s' is not supported", s.WidgetType)
	}
}

func (s ParameterSpecification) validateTypeMatching(p Parameter) error {
	allowedTypes := widgetTypeToValueTypeMapping[s.WidgetType]
	for _, allowedType := range allowedTypes {
		if p.ValueType == allowedType {
			return nil
		}
	}

	return fmt.Errorf("parameter value type '%s' is not allowed for widget type '%s'", p.ValueType, s.WidgetType)
}

func (s ParameterSpecification) validateInput(p Parameter) error {
	if s.ValueType != ValueTypeInt {
		return nil
	}

	v, err := p.IntValue()
	if err != nil {
		return err
	}

	if v < s.Min || v > s.Max {
		return fmt.Errorf("parameter value '%d' is out of range [%d, %d]", v, s.Min, s.Max)
	}

	return nil
}

func (s ParameterSpecification) validateSelect(p Parameter) error {
	var value any

	switch s.ValueType {
	case ValueTypeInt:
		v, err := p.IntValue()
		if err != nil {
			return err
		}

		value = v
	case ValueTypeString:
		v, err := p.StringValue()
		if err != nil {
			return err
		}

		value = v
	default:
		return nil
	}

	if !s.Options.HaveValue(value) {
		return fmt.Errorf("parameter value '%v' is not allowed", value)
	}

	return nil
}

func (s ParameterSpecification) validateMultiSelect(p Parameter) error {
	var vals []any

	switch s.ValueType {
	case ValueTypeIntArray:
		v, err := p.IntArrayValue()
		if err != nil {
			return err
		}

		for _, val := range v {
			vals = append(vals, val)
		}
	case ValueTypeStringArray:
		v, err := p.StringArrayValue()
		if err != nil {
			return err
		}

		for _, val := range v {
			vals = append(vals, val)
		}
	default:
		return nil
	}

	if !s.Options.ContainValues(vals) {
		return fmt.Errorf("parameter value '%v' is not allowed", vals)
	}

	return nil
}

// SelectOption represents a select option.
type SelectOption struct {
	Label string `json:"label,omitempty"`
	Value any    `json:"value"`
}

// SelectOptions represents a slice of select options.
type SelectOptions []SelectOption

// HaveValue checks if the slice of select options contains a value.
func (o SelectOptions) HaveValue(v any) bool {
	for _, option := range o {
		if option.Value == v {
			return true
		}
	}

	return false
}

// ContainValues checks if the slice of select options contains provided values.
func (o SelectOptions) ContainValues(v []any) bool {
	for _, value := range v {
		if !o.HaveValue(value) {
			return false
		}
	}

	return true
}

// Parameter represents a parameter.
type Parameter struct {
	ID        string    `json:"parameter_id"`
	ValueType ValueType `json:"value_type"`
	Value     any       `json:"value"`
}

// IntValue returns an integer value of the parameter.
func (p Parameter) IntValue() (int, error) {
	if p.ValueType != ValueTypeInt {
		return 0, fmt.Errorf("value type '%s' is not an integer", p.ValueType)
	}

	switch p.Value.(type) {
	case int, int32, int64:
		return p.Value.(int), nil
	case float64:
		return int(p.Value.(float64)), nil
	default:
		return 0, fmt.Errorf("value of type %T is not an integer", p.Value)
	}
}

// StringValue returns a string value of the parameter.
func (p Parameter) StringValue() (string, error) {
	if p.ValueType != ValueTypeString {
		return "", fmt.Errorf("value type '%s' is not a string", p.ValueType)
	}

	v, ok := p.Value.(string)
	if !ok {
		return "", fmt.Errorf("value of type %T is not a string", p.Value)
	}

	return v, nil
}

// BoolValue returns a boolean value of the parameter.
func (p Parameter) BoolValue() (bool, error) {
	if p.ValueType != ValueTypeBool {
		return false, fmt.Errorf("value type '%s' is not a boolean", p.ValueType)
	}

	v, ok := p.Value.(bool)
	if !ok {
		return false, fmt.Errorf("value of type %T is not a boolean", p.Value)
	}

	return v, nil
}

// IntArrayValue returns a value of the parameter as a slice of integers.
func (p Parameter) IntArrayValue() ([]int, error) {
	if p.ValueType != ValueTypeIntArray {
		return nil, fmt.Errorf("value type '%s' is not an integer array", p.ValueType)
	}

	switch p.Value.(type) {
	case []int:
		return p.Value.([]int), nil
	case []int32:
		var result []int

		for _, v := range p.Value.([]int32) {
			result = append(result, int(v))
		}

		return result, nil
	case []int64:
		var result []int

		for _, v := range p.Value.([]int64) {
			result = append(result, int(v))
		}

		return result, nil
	case []interface{}:
		var result []int

		for _, v := range p.Value.([]interface{}) {
			switch v.(type) {
			case int:
				result = append(result, v.(int))
			case int32:
				result = append(result, int(v.(int32)))
			case int64:
				result = append(result, int(v.(int64)))
			case float64:
				result = append(result, int(v.(float64)))
			default:
				return nil, fmt.Errorf("value of type %T is not an integer or is unsupported", p.Value)
			}
		}

		return result, nil
	default:
		return nil, fmt.Errorf("value of type %T is not an integer array or is unsupported", p.Value)
	}
}

// StringArrayValue returns a value of the parameter as a slice of strings.
func (p Parameter) StringArrayValue() ([]string, error) {
	if p.ValueType != ValueTypeStringArray {
		return nil, fmt.Errorf("value type '%s' is not a string array", p.ValueType)
	}

	switch p.Value.(type) {
	case []string:
		return p.Value.([]string), nil
	case []interface{}:
		var result []string

		for _, v := range p.Value.([]interface{}) {
			switch v.(type) {
			case string:
				result = append(result, v.(string))
			default:
				return nil, fmt.Errorf("value of type %T is not a string or is unsupported", p.Value)
			}
		}

		return result, nil
	default:
		return nil, fmt.Errorf("value of type %T is not a string array or is unsupported", p.Value)
	}
}

// Validate validates a parameter.
func (p Parameter) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("parameter id cannot be empty")
	}

	if !IsValueTypeAllowed(p.ValueType) {
		return fmt.Errorf("value type '%s' is not allowed", p.ValueType)
	}

	if !p.valueMatchesValueType() {
		return fmt.Errorf("value of type %T is not allowed for type %s", p.Value, p.ValueType)
	}

	return nil
}

func (p Parameter) valueMatchesValueType() bool {
	switch p.Value.(type) {
	case int, int32, int64, float64:
		return p.ValueType == ValueTypeInt
	case string:
		return p.ValueType == ValueTypeString
	case bool:
		return p.ValueType == ValueTypeBool
	case []int, []int32, []int64:
		return p.ValueType == ValueTypeIntArray
	case []string:
		return p.ValueType == ValueTypeStringArray
	}

	return false
}
