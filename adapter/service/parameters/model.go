package parameters

import (
	"encoding/json"
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

// AllowedValueTypes returns a slice of allowed value types.
func AllowedValueTypes() []ValueType {
	return []ValueType{
		ValueTypeInt,
		ValueTypeString,
		ValueTypeBool,
		ValueTypeIntArray,
		ValueTypeStringArray,
	}
}

// IsValueTypeAllowed checks if a value type is allowed.
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
func (s *ParameterSpecification) ValidateParameter(p *Parameter) error {
	if s.ValueType != p.ValueType {
		return fmt.Errorf("parameter value type '%s' does not match specification value type '%s'", p.ValueType, s.ValueType)
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

func (s *ParameterSpecification) validateInput(p *Parameter) error {
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

func (s *ParameterSpecification) validateSelect(p *Parameter) error {
	var value any

	switch s.ValueType { //nolint:exhaustive
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

func (s *ParameterSpecification) validateMultiSelect(p *Parameter) error {
	var vals []any

	switch s.ValueType { //nolint:exhaustive
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
	ID        string          `json:"parameter_id"`
	ValueType ValueType       `json:"value_type"`
	Value     json.RawMessage `json:"value"`
}

// NewIntParameter creates a new parameter of a value type: integer.
func NewIntParameter(id string, value int) *Parameter {
	b, _ := json.Marshal(value)

	return &Parameter{
		ID:        id,
		ValueType: ValueTypeInt,
		Value:     b,
	}
}

// NewStringParameter creates a new parameter of a value type: string.
func NewStringParameter(id string, value string) *Parameter {
	b, _ := json.Marshal(value)

	return &Parameter{
		ID:        id,
		ValueType: ValueTypeString,
		Value:     b,
	}
}

// NewBoolParameter creates a new parameter of a value type: boolean.
func NewBoolParameter(id string, value bool) *Parameter {
	b, _ := json.Marshal(value)

	return &Parameter{
		ID:        id,
		ValueType: ValueTypeBool,
		Value:     b,
	}
}

// NewIntArrayParameter creates a new parameter of a value type: integer array.
func NewIntArrayParameter(id string, value []int) *Parameter {
	b, _ := json.Marshal(value)

	return &Parameter{
		ID:        id,
		ValueType: ValueTypeIntArray,
		Value:     b,
	}
}

// NewStringArrayParameter creates a new parameter of a value type: string array.
func NewStringArrayParameter(id string, value []string) *Parameter {
	b, _ := json.Marshal(value)

	return &Parameter{
		ID:        id,
		ValueType: ValueTypeStringArray,
		Value:     b,
	}
}

// IntValue returns an integer value of the parameter.
func (p *Parameter) IntValue() (int, error) {
	if p.ValueType != ValueTypeInt {
		return 0, fmt.Errorf("value type '%s' is not an integer", p.ValueType)
	}

	var v int
	if err := json.Unmarshal(p.Value, &v); err != nil {
		return 0, fmt.Errorf("value is not of type %T", v)
	}

	return v, nil
}

// StringValue returns a string value of the parameter.
func (p *Parameter) StringValue() (string, error) {
	if p.ValueType != ValueTypeString {
		return "", fmt.Errorf("value type '%s' is not a string", p.ValueType)
	}

	var v string
	if err := json.Unmarshal(p.Value, &v); err != nil {
		return "", fmt.Errorf("value is not of type %T", v)
	}

	return v, nil
}

// BoolValue returns a boolean value of the parameter.
func (p *Parameter) BoolValue() (bool, error) {
	if p.ValueType != ValueTypeBool {
		return false, fmt.Errorf("value type '%s' is not a boolean", p.ValueType)
	}

	var v bool
	if err := json.Unmarshal(p.Value, &v); err != nil {
		return false, fmt.Errorf("value is not of type %T", v)
	}

	return v, nil
}

// IntArrayValue returns a value of the parameter as a slice of integers.
//
//nolint:cyclop
func (p *Parameter) IntArrayValue() ([]int, error) {
	if p.ValueType != ValueTypeIntArray {
		return nil, fmt.Errorf("value type '%s' is not an integer array", p.ValueType)
	}

	var v []int
	if err := json.Unmarshal(p.Value, &v); err != nil {
		return nil, fmt.Errorf("value is not of type %T", v)
	}

	return v, nil
}

// StringArrayValue returns a value of the parameter as a slice of strings.
func (p *Parameter) StringArrayValue() ([]string, error) {
	if p.ValueType != ValueTypeStringArray {
		return nil, fmt.Errorf("value type '%s' is not a string array", p.ValueType)
	}

	var v []string
	if err := json.Unmarshal(p.Value, &v); err != nil {
		return nil, fmt.Errorf("value is not of type %T", v)
	}

	return v, nil
}

// Validate validates a parameter.
func (p *Parameter) Validate() error {
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

func (p *Parameter) valueMatchesValueType() bool {
	var err error

	switch p.ValueType {
	case ValueTypeInt:
		_, err = p.IntValue()
	case ValueTypeString:
		_, err = p.StringValue()
	case ValueTypeBool:
		_, err = p.BoolValue()
	case ValueTypeIntArray:
		_, err = p.IntArrayValue()
	case ValueTypeStringArray:
		_, err = p.StringArrayValue()
	}

	return err == nil
}
