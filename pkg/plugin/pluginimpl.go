package plugin

import (
	"context"
	"log/slog"

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

func (dsp *DataSourcePlugin) ConnectDataSourceGRPCServer(broker *go_plugin.GRPCBroker, s *grpc.Server) error {
	cofidectl_proto.RegisterDataSourcePluginServiceServer(s, &DataSourcePluginServerGRPC{Impl: dsp.Impl})
	return nil
}

func (dsp *DataSourcePlugin) ConnectDataSourceGRPCClient(ctx context.Context, broker *go_plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	slog.Info("HEY")
	return &DataSourcePluginClientGRPC{client: cofidectl_proto.NewDataSourcePluginServiceClient(c)}, nil
}
