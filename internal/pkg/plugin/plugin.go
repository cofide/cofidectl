package plugin

import (
	"context"

	"github.com/cofide/cofide-connect/pkg/api"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type DataSourcePlugin struct {
	plugin.Plugin
	Impl DataSource
}

type TrustZonesGRPCClient struct {
	client api.TrustZoneServiceClient
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return nil, nil
	// return &TrustZonesGRPCClient{client: api.NewTrustZoneServiceClient(c)}, nil
}
