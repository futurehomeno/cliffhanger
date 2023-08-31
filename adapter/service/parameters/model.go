package parameters

const (
	WidgetTypeInput       = "input"
	WidgetTypeSelect      = "select"
	WidgetTypeMultiSelect = "multiselect"

	ValueTypeInt         = "int"
	ValueTypeString      = "string"
	ValueTypeBool        = "bool"
	ValueTypeIntArray    = "int_array"
	ValueTypeStringArray = "string_array"
)

// TODO: add helper methods
type SupportedParameter struct {
	ID           string         `json:"parameter_id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	ValueType    string         `json:"value_type"`
	WidgetType   string         `json:"widget_type"`
	Options      []SelectOption `json:"options,omitempty"`
	Min          int            `json:"min,omitempty"`
	Max          int            `json:"max,omitempty"`
	DefaultValue int            `json:"default_value"`
	ReadOnly     bool           `json:"read_only"`
}

func (p SupportedParameter) Validate() error {
	return nil // TODO!
}

type SelectOption struct {
	Label string `json:"label"` // TODO: optional means we can have omitempty tag here?
	Value any    `json:"value"`
}

// TODO: add helper methods like IntValue, StringValue, etc.
type Parameter struct {
	ID        string `json:"parameter_id"`
	ValueType string `json:"value_type"`
	Value     any    `json:"value"`
	Size      int    `json:"size"`
}

func (p Parameter) Validate() error {
	return nil // TODO!
}

func allowedWidgetTypes() []string {
	return []string{
		WidgetTypeInput,
		WidgetTypeSelect,
		WidgetTypeMultiSelect,
	}
}

func allowedValueTypes() []string {
	return []string{
		ValueTypeInt,
		ValueTypeString,
		ValueTypeBool,
		ValueTypeIntArray,
		ValueTypeStringArray,
	}
}
