package main

import (
	"context"
	"log"

	"github.com/cloudquery/plugin-sdk/v4/serve"

	plugin "github.com/IPGeolocation/cq-source-ipgeolocation/resources/plugin"
)

func main() {
	p := serve.Plugin(plugin.Plugin())
	if err := p.Serve(context.Background()); err != nil {
		log.Fatalf("failed to serve plugin: %v", err)
	}
}
