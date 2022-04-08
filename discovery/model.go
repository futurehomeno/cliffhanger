package discovery

// Constants defining resource types.
const (
	ResourceTypeApp = "app"
	ResourceTypeAd  = "ad"
)

type Resource struct {
	ResourceName           string            `json:"resource_name"`            // Name of the application or adapter.
	ResourceType           string            `json:"resource_type"`            // Type of the service: "app", "ad"
	ResourceFullName       string            `json:"resource_full_name"`       // Full name as shown in registry.
	Description            string            `json:"description"`              // Description as shown in registry.
	Author                 string            `json:"author"`                   // Author of the application.
	Version                string            `json:"version"`                  // Version of the application.
	PackageName            string            `json:"package_name"`             // Package name may be different from the resource name.
	State                  string            `json:"state"`                    // Current application state.
	AppInfo                AppInfo           `json:"app_info"`                 // Additional information base onf the resource type.
	AdapterInfo            AdapterInfo       `json:"adapter_info"`             // Additional information base onf the resource type.
	ConfigRequired         bool              `json:"config_required"`          // If true, the service should be configured before it can be used.
	Configs                map[string]string `json:"configs"`                  // Configuration parameters.
	Props                  map[string]string `json:"props"`                    // Service properties
	DocURL                 string            `json:"doc_url"`                  // URL containing documentation.
	IsInstanceConfigurable bool              `json:"is_instance_configurable"` // If true, the service should be configured before it can be used.
	InstanceID             string            `json:"instance_id"`              // An instance ID of the service, usually 1.
}

// AppInfo contains specific information about the application. Deprecated.
type AppInfo struct{}

// AdapterInfo contains specific information about the adapter. Deprecated.
type AdapterInfo struct {
	FwVersion             string            `json:"fw_version"`              // Firmware version, preferably in accordance to the semantic versioning.
	Technology            string            `json:"technology"`              // Technology of communication, e.g.: "cloud"
	HwDependency          map[string]string `json:"hw_dependency"`           // Hardware dependencies, e.g.: "serialPort": "/dev/ttyUSB0"
	NetworkManagementType string            `json:"network_management_type"` // Possible values: "inclusion_exclusion", "inclusion_dev_remove", "full_sync"
}
