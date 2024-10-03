package plugin

import (
	"context"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1"
)

type ConnectDataSourceGRPCClient struct {
	client cofidectl_proto.DataSourcePluginServiceClient
}

func (c *ConnectDataSourceGRPCClient) GetTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.GetTrustZones(context.Background(), &cofidectl_proto.GetTrustZonesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.TrustZones, nil
}

type ConnectDataSourceGRPCServer struct {
	Impl DataSource
}

func (c *ConnectDataSourceGRPCServer) GetTrustZones(context.Context, *cofidectl_proto.GetTrustZonesRequest) (*cofidectl_proto.GetTrustZonesResponse, error) {
	v, err := c.Impl.GetTrustZones()
	return &cofidectl_proto.GetTrustZonesResponse{TrustZones: v}, err
}
