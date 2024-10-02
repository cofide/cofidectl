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

func (p *DataSourcePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	// TODO
	return nil
}

func (p *DataSourcePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// TODO
	return nil, nil
}
