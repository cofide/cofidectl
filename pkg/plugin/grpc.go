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

// DataSourcePluginServerGRPC is used by plugins to map GRPC calls from the clients to
// methods of the DataSource interface.
type DataSourcePluginServerGRPC struct {
	Impl DataSource
}

func (c *DataSourcePluginServerGRPC) GetTrustZones(ctx context.Context, req *cofidectl_proto.GetTrustZonesRequest) (*cofidectl_proto.GetTrustZonesResponse, error) {
	v, err := c.Impl.GetTrustZones()
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.GetTrustZonesResponse{TrustZones: v}, nil
}

/*
func (c *DataSourcePluginClientGRPC) CreateTrustZone() (*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.CreateTrustZone(context.Background(), &cofidectl_proto.CreateTrustZoneRequest{})
	if err != nil {
		return nil, err
	}

	return resp.TrustZone, nil
}

func (c *DataSourcePluginClientGRPC) CreateAttestationPolicy() (*attestation_policy_proto.AttestationPolicy, error) {
	resp, err := c.client.CreateAttestationPolicy(context.Background(), &cofidectl_proto.CreateAttestationPolicyRequest{})
	if err != nil {
		return nil, err
	}

	return resp.AttestationPolicy, nil
}
*/
