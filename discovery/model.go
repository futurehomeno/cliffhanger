package discovery

import "github.com/futurehomeno/cliffhanger/lifecycle"

// Constants defining resource types.
const (
	ResourceTypeApp = "app"
	ResourceTypeAd  = "ad"
)

// resourceT is the payload serialized in evt.discovery.report.
type resourceT struct {
	ResourceName string               `json:"resource_name"`
	ResourceType string               `json:"resource_type"`
	PackageName  string               `json:"package_name"`
	InstanceID   string               `json:"instance_id"`
	Version      string               `json:"version"`
	States       *lifecycle.AppStates `json:"states"`
}
