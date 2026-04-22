package discovery

import "github.com/futurehomeno/cliffhanger/lifecycle"

// Constants defining resource types.
const (
	ResourceTypeApp = "app"
	ResourceTypeAd  = "ad"
)

type ResourceT struct {
	ResourceName string              `json:"resource_name"`        // Name of the application or adapter.
	ResourceType string              `json:"resource_type"`        // Type of the service: "app", "ad"
	PackageName  string              `json:"package_name"`         // Package name may be different from the resource name.
	InstanceID   string              `json:"instance_id"`          // An instance ID of the service, usually 1.
	Version      string              `json:"version"`              // Version of the application.
	States       lifecycle.AppStates `json:"app_states,omitempty"` // Current states of the application.
}
