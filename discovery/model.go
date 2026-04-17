package discovery

// Constants defining resource types.
const (
	ResourceTypeApp = "app"
	ResourceTypeAd  = "ad"
)

type Resource struct {
	ResourceName string `json:"resource_name"` // Name of the application or adapter.
	ResourceType string `json:"resource_type"` // Type of the service: "app", "ad"
	InstanceID   string `json:"instance_id"`   // An instance ID of the service, usually 1.
}
