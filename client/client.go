package client

import (
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
	"github.com/cloudquery/plugin-sdk/v4/state"
	"github.com/rs/zerolog"
)

// Client is the resolver-facing client. Every resolver receives it via
// meta.(*client.Client). It implements schema.ClientMeta.
type Client struct {
	Logger  zerolog.Logger
	Spec    Spec
	IPGeo   *ipgeolocation.Client
	Backend state.Client
}

// New creates a new resolver-facing Client.
func New(logger zerolog.Logger, spec Spec, ipgeo *ipgeolocation.Client, backend state.Client) *Client {
	return &Client{
		Logger:  logger,
		Spec:    spec,
		IPGeo:   ipgeo,
		Backend: backend,
	}
}

// ID returns a unique identifier for this client instance.
// Required by the schema.ClientMeta interface.
func (c *Client) ID() string {
	return "ipgeolocation"
}
