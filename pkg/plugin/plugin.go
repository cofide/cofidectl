// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
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

func (dsp *DataSourcePlugin) GRPCClient(ctx context.Context, broker *go_plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &DataSourcePluginClientGRPC{client: cofidectl_proto.NewDataSourcePluginServiceClient(c)}, nil
}

func (dsp *DataSourcePlugin) GRPCServer(broker *go_plugin.GRPCBroker, s *grpc.Server) error {
	cofidectl_proto.RegisterDataSourcePluginServiceServer(s, &GRPCServer{Impl: dsp.Impl})
	return nil
}

type GRPCServer struct {
	Impl DataSource
}

func (s *GRPCServer) ListTrustZones(ctx context.Context, req *cofidectl_proto.ListTrustZonesRequest) (*cofidectl_proto.ListTrustZonesResponse, error) {
	resp, err := s.Impl.ListTrustZones()
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ListTrustZonesResponse{TrustZones: resp}, nil
}

/*
func (s *GRPCServer) CreateTrustZone(ctx context.Context, req *cofidectl_proto.CreateTrustZoneRequest) (*cofidectl_proto.CreateTrustZoneResponse, error) {
	// TODO
	return &cofidectl_proto.CreateTrustZoneResponse{}, nil

}

func (s *GRPCServer) CreateAttestationPolicy(ctx context.Context, req *cofidectl_proto.CreateAttestationPolicyRequest) (*cofidectl_proto.CreateAttestationPolicyResponse, error) {
	// TODO
	return &cofidectl_proto.CreateAttestationPolicyResponse{}, nil
}
*/
