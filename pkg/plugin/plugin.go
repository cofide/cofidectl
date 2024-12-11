// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"slices"

	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	go_plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

const (
	// DataSourcePluginName is the name that should be used in the plugin map.
	DataSourcePluginName = "data_source"
)

// DataSourcePluginArgs contains the arguments passed to plugins when executing them as a data source.
// TODO: change to plugin serve
var DataSourcePluginArgs []string = []string{"data-source", "serve"}

// IsDataSourceServeCmd returns whether the provided command line arguments indicate that a plugin should serve a data source.
func IsDataSourceServeCmd(args []string) bool {
	return slices.Equal(args, DataSourcePluginArgs)
}

// DataSourcePlugin implements the plugin.Plugin interface to provide the GRPC
// server or client back to the plugin machinery. The server side should
// proved the Impl field with a concrete implementation of the DataSource
// interface.
type DataSourcePlugin struct {
	go_plugin.Plugin
	Impl DataSource
}

func (dsp *DataSourcePlugin) GRPCClient(ctx context.Context, broker *go_plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &DataSourcePluginClientGRPC{ctx: ctx, client: cofidectl_proto.NewDataSourcePluginServiceClient(c)}, nil
}

func (dsp *DataSourcePlugin) GRPCServer(broker *go_plugin.GRPCBroker, s *grpc.Server) error {
	cofidectl_proto.RegisterDataSourcePluginServiceServer(s, &GRPCServer{Impl: dsp.Impl})
	return nil
}

type GRPCServer struct {
	Impl DataSource
}

func (s *GRPCServer) Validate(ctx context.Context, req *cofidectl_proto.ValidateRequest) (*cofidectl_proto.ValidateResponse, error) {
	err := s.Impl.Validate(ctx)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ValidateResponse{}, nil
}

func (s *GRPCServer) GetTrustZone(_ context.Context, req *cofidectl_proto.GetTrustZoneRequest) (*cofidectl_proto.GetTrustZoneResponse, error) {
	trustZone, err := s.Impl.GetTrustZone(req.GetName())
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.GetTrustZoneResponse{TrustZone: trustZone}, nil
}

func (s *GRPCServer) ListTrustZones(_ context.Context, req *cofidectl_proto.ListTrustZonesRequest) (*cofidectl_proto.ListTrustZonesResponse, error) {
	trustZones, err := s.Impl.ListTrustZones()
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ListTrustZonesResponse{TrustZones: trustZones}, nil
}

func (s *GRPCServer) AddTrustZone(_ context.Context, req *cofidectl_proto.AddTrustZoneRequest) (*cofidectl_proto.AddTrustZoneResponse, error) {
	trustZone, err := s.Impl.AddTrustZone(req.TrustZone)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.AddTrustZoneResponse{TrustZone: trustZone}, nil
}

func (s *GRPCServer) UpdateTrustZone(_ context.Context, req *cofidectl_proto.UpdateTrustZoneRequest) (*cofidectl_proto.UpdateTrustZoneResponse, error) {
	err := s.Impl.UpdateTrustZone(req.TrustZone)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.UpdateTrustZoneResponse{}, nil
}

func (s *GRPCServer) AddAttestationPolicy(_ context.Context, req *cofidectl_proto.AddAttestationPolicyRequest) (*cofidectl_proto.AddAttestationPolicyResponse, error) {
	policy, err := s.Impl.AddAttestationPolicy(req.Policy)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.AddAttestationPolicyResponse{Policy: policy}, nil
}

func (s *GRPCServer) GetAttestationPolicy(_ context.Context, req *cofidectl_proto.GetAttestationPolicyRequest) (*cofidectl_proto.GetAttestationPolicyResponse, error) {
	resp, err := s.Impl.GetAttestationPolicy(req.GetName())
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.GetAttestationPolicyResponse{Policy: resp}, nil
}

func (s *GRPCServer) ListAttestationPolicies(_ context.Context, req *cofidectl_proto.ListAttestationPoliciesRequest) (*cofidectl_proto.ListAttestationPoliciesResponse, error) {
	policies, err := s.Impl.ListAttestationPolicies()
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ListAttestationPoliciesResponse{Policies: policies}, nil
}

func (s *GRPCServer) AddAPBinding(_ context.Context, req *cofidectl_proto.AddAPBindingRequest) (*cofidectl_proto.AddAPBindingResponse, error) {
	binding, err := s.Impl.AddAPBinding(req.Binding)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.AddAPBindingResponse{Binding: binding}, nil
}

func (s *GRPCServer) DestroyAPBinding(_ context.Context, req *cofidectl_proto.DestroyAPBindingRequest) (*cofidectl_proto.DestroyAPBindingResponse, error) {
	err := s.Impl.DestroyAPBinding(req.Binding)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.DestroyAPBindingResponse{}, nil
}

func (s *GRPCServer) AddFederation(_ context.Context, req *cofidectl_proto.AddFederationRequest) (*cofidectl_proto.AddFederationResponse, error) {
	federation, err := s.Impl.AddFederation(req.Federation)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.AddFederationResponse{Federation: federation}, nil
}

func (s *GRPCServer) ListFederations(_ context.Context, req *cofidectl_proto.ListFederationsRequest) (*cofidectl_proto.ListFederationsResponse, error) {
	federations, err := s.Impl.ListFederations()
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ListFederationsResponse{Federations: federations}, nil
}

func (s *GRPCServer) ListFederationsByTrustZone(_ context.Context, req *cofidectl_proto.ListFederationsByTrustZoneRequest) (*cofidectl_proto.ListFederationsByTrustZoneResponse, error) {
	federations, err := s.Impl.ListFederationsByTrustZone(req.GetTrustZoneName())
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ListFederationsByTrustZoneResponse{Federations: federations}, nil
}
