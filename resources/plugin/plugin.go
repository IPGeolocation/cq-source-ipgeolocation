package plugin

import (
	internalPlugin "github.com/anthropic/cq-source-ipgeolocation/plugin"
	"github.com/cloudquery/plugin-sdk/v4/plugin"
)

// Plugin returns the configured CloudQuery plugin instance.
func Plugin() *plugin.Plugin {
	return plugin.NewPlugin(
		internalPlugin.Name,
		internalPlugin.Version,
		Configure,
		plugin.WithKind(internalPlugin.Kind),
		plugin.WithTeam(internalPlugin.Team),
	)
}
