package plugin

import (
	"context"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1"
	go_plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// DataSourcePlugin implements the plugin.Plugin interface to provide the GRPC
// server or client back to the plugin machinery. The server side should
// proved the Impl field with a concrete implementation of the DataSource
// interface.
type DataSourcePlugin struct {
	go_plugin.Plugin
	Impl DataSource
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCServer(broker *go_plugin.GRPCBroker, s *grpc.Server) error {
	cofidectl_proto.RegisterDataSourcePluginServiceServer(s, &DataSourcePluginServerGRPC{Impl: p.Impl})
	return nil
}

func (p *DataSourcePlugin) ConnectDataSourceGRPCClient(ctx context.Context, broker *go_plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &DataSourcePluginClientGRPC{client: cofidectl_proto.NewDataSourcePluginServiceClient(c)}, nil
}
