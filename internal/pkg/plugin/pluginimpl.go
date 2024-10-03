package plugin

import (
	"context"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// DataSourcePlugin implements the plugin.Plugin interface to provide the GRPC
// server or client back to the plugin machinery. The server side should
// proved the Impl field with a concrete implementation of the DataSource
// interface.
type DataSourcePlugin struct {
	plugin.Plugin
	Impl DataSource
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	cofidectl_proto.RegisterDataSourcePluginServiceServer(s, &ConnectDataSourceGRPCServer{Impl: p.Impl})
	return nil
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ConnectDataSourceGRPCClient{client: cofidectl_proto.NewDataSourcePluginServiceClient(c)}, nil
}
