package plugin

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type DataSourcePlugin struct {
	plugin.Plugin
	Impl DataSource
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return nil, nil
}
