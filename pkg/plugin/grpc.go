package plugin

import (
	"context"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
)

// DataSourcePluginClientGRPC is used by clients (main application) to translate the
// DataSource interface of plugins to GRPC calls.
type DataSourcePluginClientGRPC struct {
	client cofidectl_proto.DataSourcePluginServiceClient
}

func (c *DataSourcePluginClientGRPC) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.ListTrustZones(context.Background(), &cofidectl_proto.ListTrustZonesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.TrustZones, nil
}
