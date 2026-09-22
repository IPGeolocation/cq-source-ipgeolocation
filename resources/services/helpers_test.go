package services

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudquery/plugin-sdk/v4/scalar"
	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/stretchr/testify/require"
)

// buildTable applies the table's Transform (as the plugin does at startup) and
// returns the table with its columns populated.
func buildTable(t *testing.T, tbl *schema.Table) *schema.Table {
	t.Helper()
	require.NoError(t, transformers.TransformTables(schema.Tables{tbl}))
	return tbl
}

// columnNames returns the set of column names of a built table.
func columnNames(t *testing.T, tbl *schema.Table) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for _, n := range buildTable(t, tbl).Columns.Names() {
		names[n] = true
	}
	return names
}

// resolveRow runs every column resolver of the (already built) table against
// item, exactly as the SDK does during a sync, and returns column -> scalar.
func resolveRow(t *testing.T, tbl *schema.Table, item any) map[string]scalar.Scalar {
	t.Helper()
	res := schema.NewResourceData(tbl, nil, item)
	row := make(map[string]scalar.Scalar, len(tbl.Columns))
	for _, col := range tbl.Columns {
		require.NoError(t, col.Resolver(context.Background(), nil, res, col), "resolving column %s", col.Name)
		row[col.Name] = res.Get(col.Name)
	}
	return row
}

// jsonTags returns the json field names of a struct type (value or pointer).
func jsonTags(v any) []string {
	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	var tags []string
	for i := 0; i < rt.NumField(); i++ {
		tag := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			tags = append(tags, tag)
		}
	}
	return tags
}
