package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudquery/plugin-sdk/v4/message"
	"github.com/cloudquery/plugin-sdk/v4/plugin"
	"github.com/cloudquery/plugin-sdk/v4/scheduler"
	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/state"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/rs/zerolog"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
	"github.com/IPGeolocation/cq-source-ipgeolocation/resources/services"
)

// Client implements plugin.Client for the ipgeolocation source plugin.
type Client struct {
	logger    zerolog.Logger
	config    client.Spec
	tables    schema.Tables
	scheduler *scheduler.Scheduler
	ipgeo     *ipgeolocation.Client

	plugin.UnimplementedDestination
}

// Configure is called once by the SDK at sync start. It parses the spec,
// creates the API client, and returns a ready-to-use plugin.Client.
func Configure(_ context.Context, logger zerolog.Logger, specBytes []byte, opts plugin.NewClientOptions) (plugin.Client, error) {
	if opts.NoConnection {
		return &Client{
			logger: logger.With().Str("module", "ipgeolocation").Logger(),
			tables: getTables(),
		}, nil
	}

	var spec client.Spec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ipgeolocation spec: %w", err)
	}
	spec.SetDefaults()
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ipgeolocation spec: %w", err)
	}

	ipgeoClient, err := ipgeolocation.NewClient(
		spec.APIKey,
		ipgeolocation.WithEndpoint(spec.Endpoint),
		ipgeolocation.WithTimeout(spec.TimeoutDuration()),
		ipgeolocation.WithRetries(spec.RetryAttempts),
		ipgeolocation.WithUserAgent(spec.UserAgent),
		ipgeolocation.WithRateLimit(spec.RateLimit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipgeolocation client: %w", err)
	}

	return &Client{
		logger: logger.With().Str("module", "ipgeolocation").Logger(),
		config: spec,
		tables: getTables(),
		scheduler: scheduler.NewScheduler(
			scheduler.WithLogger(logger),
			scheduler.WithConcurrency(spec.Concurrency),
		),
		ipgeo: ipgeoClient,
	}, nil
}

// Sync runs the data sync: filters tables, creates the resolver client,
// and lets the scheduler orchestrate the fetching.
func (c *Client) Sync(ctx context.Context, options plugin.SyncOptions, res chan<- message.SyncMessage) error {
	tt, err := c.tables.FilterDfs(options.Tables, options.SkipTables, options.SkipDependentTables)
	if err != nil {
		return fmt.Errorf("filtering tables: %w", err)
	}

	stateClient, err := state.NewConnectedClient(ctx, options.BackendOptions)
	if err != nil {
		return fmt.Errorf("creating state client: %w", err)
	}
	defer stateClient.Close()

	schedulerClient := client.New(c.logger, c.config, c.ipgeo, stateClient)

	if err := c.scheduler.Sync(ctx, schedulerClient, tt, res,
		scheduler.WithSyncDeterministicCQID(options.DeterministicCQID)); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}
	return stateClient.Flush(ctx)
}

// Tables returns the filtered list of tables this plugin exposes.
func (c *Client) Tables(_ context.Context, options plugin.TableOptions) (schema.Tables, error) {
	return c.tables.FilterDfs(options.Tables, options.SkipTables, options.SkipDependentTables)
}

// Close is a no-op for this plugin.
func (*Client) Close(_ context.Context) error {
	return nil
}

// getTables builds and transforms the full list of tables.
func getTables() schema.Tables {
	tables := []*schema.Table{
		services.IPGeolocationTable(),
		services.IPSecurityTable(),
		services.AbuseContactTable(),
		services.ASNDetailTable(),
		services.UserAgentTable(),
	}
	if err := transformers.TransformTables(tables); err != nil {
		panic(fmt.Sprintf("transforming tables: %v", err))
	}
	for _, t := range tables {
		schema.AddCqIDs(t)
	}
	return tables
}
