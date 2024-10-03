package plugin

import (
	"context"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1"
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

// DataSourcePluginServerGRPC is used by plugins to map GRPC calls from the clients to
// methods of the DataSource interface.
type DataSourcePluginServerGRPC struct {
	Impl DataSource
}

func (c *DataSourcePluginServerGRPC) GetTrustZones(context.Context, *cofidectl_proto.GetTrustZonesRequest) (*cofidectl_proto.GetTrustZonesResponse, error) {
	v, err := c.Impl.GetTrustZones()
	return &cofidectl_proto.GetTrustZonesResponse{TrustZones: v}, err
}
