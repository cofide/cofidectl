// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"fmt"
	"io"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	provisionpb "github.com/cofide/cofide-api-sdk/gen/go/proto/provision_plugin/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	go_plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// ProvisionPluginName is the name that should be used in the plugin map.
const ProvisionPluginName = "provision"

// ProvisionPlugin implements the plugin.Plugin interface to provide the GRPC
// server or client back to the plugin machinery. The server side should
// provide the Impl field with a concrete implementation of the ProvisionPlugin
// interface.
type ProvisionPlugin struct {
	go_plugin.Plugin
	Impl Provision
}

func (pp *ProvisionPlugin) GRPCClient(ctx context.Context, broker *go_plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ProvisionPluginClientGRPC{client: provisionpb.NewProvisionPluginServiceClient(c), broker: broker}, nil
}

func (pp *ProvisionPlugin) GRPCServer(broker *go_plugin.GRPCBroker, s *grpc.Server) error {
	provisionpb.RegisterProvisionPluginServiceServer(s, &GRPCServer{impl: pp.Impl, broker: broker})
	return nil
}

// Type check to ensure that ProvisionPluginClientGRPC implements the Provision interface.
var _ Provision = &ProvisionPluginClientGRPC{}

// ProvisionPluginClientGRPC is used by clients (main application) to translate the
// Provision interface of plugins to GRPC calls.
type ProvisionPluginClientGRPC struct {
	broker *go_plugin.GRPCBroker
	client provisionpb.ProvisionPluginServiceClient
}

func (c *ProvisionPluginClientGRPC) Validate(ctx context.Context) error {
	_, err := c.client.Validate(ctx, &provisionpb.ValidateRequest{})
	return wrapError(err)
}

func (c *ProvisionPluginClientGRPC) Deploy(ctx context.Context, source datasource.DataSource, kubeCfgFile string) (<-chan *provisionpb.Status, error) {
	server, brokerID := c.startDataSourceServer(source)

	req := provisionpb.DeployRequest{DataSource: &brokerID, KubeCfgFile: &kubeCfgFile}
	stream, err := c.client.Deploy(ctx, &req)
	if err != nil {
		err := wrapError(err)
		return nil, err
	}

	statusCh := make(chan *provisionpb.Status)
	go func() {
		defer close(statusCh)
		defer server.Stop()
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					err := wrapError(err)
					statusCh <- StatusError("Deploying", "Error", err)
				}
				return
			}
			statusCh <- resp.GetStatus()
		}
	}()

	return statusCh, nil
}

func (c *ProvisionPluginClientGRPC) TearDown(ctx context.Context, source datasource.DataSource) (<-chan *provisionpb.Status, error) {
	server, brokerID := c.startDataSourceServer(source)

	req := provisionpb.TearDownRequest{DataSource: &brokerID}
	stream, err := c.client.TearDown(ctx, &req)
	if err != nil {
		err := wrapError(err)
		return nil, err
	}

	statusCh := make(chan *provisionpb.Status)
	go func() {
		defer close(statusCh)
		defer server.Stop()
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					err := wrapError(err)
					statusCh <- StatusError("Tearing down", "Error", err)
				}
				return
			}
			statusCh <- resp.GetStatus()
		}
	}()

	return statusCh, nil
}

// startDataSourceServer returns a grpc.Server and associated broker ID, allowing for bidirectional
// plugin communication.
// The provided DataSource is used as the server's data source implementation.
// The returned server should be shut down when no longer required.
// This uses the bidirectional communication feature of go-plugin. See
// https://pkg.go.dev/github.com/hashicorp/go-plugin/examples/bidirectional for an example.
func (c *ProvisionPluginClientGRPC) startDataSourceServer(source datasource.DataSource) (*grpc.Server, uint32) {
	dsServer := &datasource.GRPCServer{Impl: source}

	serverCh := make(chan *grpc.Server)
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		server := grpc.NewServer(opts...)
		cofidectl_proto.RegisterDataSourcePluginServiceServer(server, dsServer)
		serverCh <- server
		return server
	}

	brokerID := c.broker.NextId()
	go c.broker.AcceptAndServe(brokerID, serverFunc)

	// Wait for the accept goroutine to create and register the server.
	server := <-serverCh
	return server, brokerID
}

// clientError wraps a gRPC Status, reformatting the error message.
type clientError struct {
	status *status.Status
}

// wrapError returns a clientError if the provided error is a gRPC status, or the original error otherwise.
func wrapError(err error) error {
	if err == nil {
		return nil
	}
	if status, ok := status.FromError(err); ok {
		return &clientError{status: status}
	}
	return err
}

func (ce *clientError) Error() string {
	return fmt.Sprintf("provision plugin error: %s: %s", ce.status.Code(), ce.status.Message())
}

// GRPCServer implements provisionpb.ProvisionPluginServiceServer, translating gRPC calls to
// impl, the Provision implementation.
type GRPCServer struct {
	impl   Provision
	broker *go_plugin.GRPCBroker
}

func (s *GRPCServer) Validate(ctx context.Context, req *provisionpb.ValidateRequest) (*provisionpb.ValidateResponse, error) {
	err := s.impl.Validate(ctx)
	if err != nil {
		return nil, err
	}
	return &provisionpb.ValidateResponse{}, nil
}

func (s *GRPCServer) Deploy(req *provisionpb.DeployRequest, stream grpc.ServerStreamingServer[provisionpb.DeployResponse]) error {
	client, conn, err := s.getDataSourceClient(stream.Context(), req.GetDataSource())
	if err != nil {
		return err
	}
	defer conn.Close()

	statusCh, err := s.impl.Deploy(stream.Context(), client, req.GetKubeCfgFile())
	if err != nil {
		return err
	}

	// Read Status messages from the channel and stream back to the client.
	for status := range statusCh {
		resp := provisionpb.DeployResponse{Status: status}
		if err := stream.Send(&resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *GRPCServer) TearDown(req *provisionpb.TearDownRequest, stream grpc.ServerStreamingServer[provisionpb.TearDownResponse]) error {
	client, conn, err := s.getDataSourceClient(stream.Context(), req.GetDataSource())
	if err != nil {
		return err
	}
	defer conn.Close()

	statusCh, err := s.impl.TearDown(stream.Context(), client)
	if err != nil {
		return err
	}

	// Read Status messages from the channel and stream back to the client.
	for status := range statusCh {
		resp := provisionpb.TearDownResponse{Status: status}
		if err := stream.Send(&resp); err != nil {
			return err
		}
	}
	return nil
}

// getDataSourceClient returns a DataSource and associated gRPC connection, allowing for
// bidirectional plugin communication.
// The returned DataSource can be passed to the server's Provision implementation methods.
// The returned client should be closed when no longer required.
// This uses the bidirectional communication feature of go-plugin. See
// https://pkg.go.dev/github.com/hashicorp/go-plugin/examples/bidirectional for an example.
func (s *GRPCServer) getDataSourceClient(ctx context.Context, dataSourceID uint32) (datasource.DataSource, *grpc.ClientConn, error) {
	conn, err := s.broker.Dial(dataSourceID)
	if err != nil {
		return nil, nil, err
	}

	client := datasource.NewDataSourcePluginClientGRPC(ctx, cofidectl_proto.NewDataSourcePluginServiceClient(conn))
	return client, conn, nil
}
