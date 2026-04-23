package discovery

import (
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// Constants defining resource types.
const (
	ResourceTypeApp = "app"
	ResourceTypeAd  = "ad"
)

// resourceT is the payload serialized in evt.discovery.report.
type resourceT struct {
	ResourceName fimptype.ResourceNameT `json:"resource_name"`
	ResourceType fimptype.ResourceTypeT `json:"resource_type"`
	PackageName  string                 `json:"package_name"`
	InstanceID   string                 `json:"instance_id"`
	Version      string                 `json:"version"`
	States       *lifecycle.AppStateT   `json:"app_state,omitempty"`
}
