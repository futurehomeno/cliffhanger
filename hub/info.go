package hub

import (
	"encoding/json"
	"fmt"
	"os"
)

// Environment is a type representing environment within which the hub is registered.
type Environment string

// Constants defining possible environments.
const (
	EnvBeta Environment = "beta"
	EnvProd Environment = "prod"
)

// Info is an object representing basic information about the hub environment.
type Info struct {
	HubID           string      `json:"hub_id"`
	SiteID          string      `json:"site_id"`
	SiteName        string      `json:"site_name"`
	SiteType        string      `json:"site_type"`
	Environment     Environment `json:"environment"`
	CloudAPIRootURL string      `json:"cloud_api_root_url"`
}

// LoadInfo loads info from a well known path on the hub.
func LoadInfo(path string) (*Info, error) {
	if path == "" {
		path = "/var/lib/futurehome/hub/hub.json"
	}

	info := &Info{}

	body, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("info loader: failed to load info file at path %s: %w", path, err)
	}

	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("info loader: failed to unmarshal info file at path %s: %w", path, err)
	}

	return info, nil
}
