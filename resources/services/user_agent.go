package services

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// UserAgentFlat is a flattened user-agent parse response for CloudQuery columns.
type UserAgentFlat struct {
	UserAgentString string `json:"user_agent_string"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Version         string `json:"version"`
	VersionMajor    string `json:"version_major"`

	// Device
	DeviceName  string `json:"device_name"`
	DeviceType  string `json:"device_type"`
	DeviceBrand string `json:"device_brand"`
	DeviceCPU   string `json:"device_cpu"`

	// Engine
	EngineName         string `json:"engine_name"`
	EngineType         string `json:"engine_type"`
	EngineVersion      string `json:"engine_version"`
	EngineVersionMajor string `json:"engine_version_major"`

	// OS
	OSName         string `json:"os_name"`
	OSType         string `json:"os_type"`
	OSVersion      string `json:"os_version"`
	OSVersionMajor string `json:"os_version_major"`
	OSBuild        string `json:"os_build"`
}

// UserAgentTable returns the table definition for user-agent parsing results.
func UserAgentTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_user_agent",
		Description: "Parsed user-agent data from IPGeolocation.io /v3/user-agent endpoint. Returns browser, device, engine, and OS details for each configured user-agent string.",
		Resolver:    fetchUserAgent,
		Transform:   transformers.TransformWithStruct(&UserAgentFlat{}, transformers.WithPrimaryKeys("UserAgentString")),
	}
}

func fetchUserAgent(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
	c := meta.(*client.Client)

	if len(c.Spec.UserAgents) == 0 {
		c.Logger.Debug().Msg("no user_agents configured, skipping user-agent table")
		return nil
	}

	for _, ua := range c.Spec.UserAgents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		parsed, err := c.IPGeo.GetUserAgent(ctx, ua)
		if err != nil {
			c.Logger.Warn().Str("user_agent", ua).Err(err).Msg("failed to parse user-agent, skipping")
			continue
		}

		flat := flattenUserAgent(parsed)
		res <- flat
	}

	return nil
}

func flattenUserAgent(ua *ipgeolocation.UserAgentResponse) *UserAgentFlat {
	f := &UserAgentFlat{
		UserAgentString: ua.UserAgentString,
		Name:            ua.Name,
		Type:            ua.Type,
		Version:         ua.Version,
		VersionMajor:    ua.VersionMajor,
	}

	if ua.Device != nil {
		f.DeviceName = ua.Device.Name
		f.DeviceType = ua.Device.Type
		f.DeviceBrand = ua.Device.Brand
		f.DeviceCPU = ua.Device.CPU
	}

	if ua.Engine != nil {
		f.EngineName = ua.Engine.Name
		f.EngineType = ua.Engine.Type
		f.EngineVersion = ua.Engine.Version
		f.EngineVersionMajor = ua.Engine.VersionMajor
	}

	if ua.OperatingSystem != nil {
		f.OSName = ua.OperatingSystem.Name
		f.OSType = ua.OperatingSystem.Type
		f.OSVersion = ua.OperatingSystem.Version
		f.OSVersionMajor = ua.OperatingSystem.VersionMajor
		f.OSBuild = ua.OperatingSystem.Build
	}

	return f
}
