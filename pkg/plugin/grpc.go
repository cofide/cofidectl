package plugin

import (
	"context"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/cofidectl_plugin/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/trust_zone/v1"
)

// DataSourcePluginClientGRPC is used by clients (main application) to translate the
// DataSource interface of plugins to GRPC calls.
type DataSourcePluginClientGRPC struct {
	client cofidectl_proto.DataSourcePluginServiceClient
}

func (c *DataSourcePluginClientGRPC) GetTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.GetTrustZones(context.Background(), &cofidectl_proto.GetTrustZonesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.TrustZones, nil
}
