package hub

import (
	"encoding/json"
	"fmt"
	"os"
)

var hubV1FilePath = "/var/lib/futurehome/hub/hub.json"
var hubV2FilePath = "/var/lib/futurehome/hub/hub_v2.json"

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
		path = hubV1FilePath
	}

	if path == hubV1FilePath {
		infoV2, err := os.Stat(hubV2FilePath)
		if err == nil && !infoV2.IsDir() {
			infoV1, infoV1err := os.Stat(hubV2FilePath)
			if infoV1err != nil || infoV2.ModTime().After(infoV1.ModTime()) {
				path = hubV2FilePath
				// prefer v2 if exists and newer then v1
			}
		}
	}

	info := &Info{}

	body, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		path = hubV2FilePath          // always check v2
		body, err = os.ReadFile(path) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("info loader: failed to load info file at path %s: %w", path, err)
		}
	}

	err = json.Unmarshal(body, info)
	if err != nil {
		return nil, fmt.Errorf("info loader: failed to unmarshal info file at path %s: %w", path, err)
	}

	return info, nil
}
