// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"context"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	go_plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// DataSourcePluginName is the name that should be used in the plugin map.
const DataSourcePluginName = "data_source"

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

// Type check to ensure DataSourcePluginClientGRPC implements DataSource.
var _ DataSource = &DataSourcePluginClientGRPC{}

// DataSourcePluginClientGRPC is used by clients (main application) to translate the
// DataSource interface of plugins to GRPC calls.
type DataSourcePluginClientGRPC struct {
	ctx    context.Context
	client cofidectl_proto.DataSourcePluginServiceClient
}

func NewDataSourcePluginClientGRPC(ctx context.Context, client cofidectl_proto.DataSourcePluginServiceClient) *DataSourcePluginClientGRPC {
	return &DataSourcePluginClientGRPC{ctx: ctx, client: client}
}

func (c *DataSourcePluginClientGRPC) Validate(ctx context.Context) error {
	_, err := c.client.Validate(ctx, &cofidectl_proto.ValidateRequest{})
	return err
}

func (c *DataSourcePluginClientGRPC) AddTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.AddTrustZone(c.ctx, &cofidectl_proto.AddTrustZoneRequest{TrustZone: trustZone})
	if err != nil {
		return nil, err
	}

	return resp.TrustZone, nil
}

func (c *DataSourcePluginClientGRPC) GetTrustZone(name string) (*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.GetTrustZone(c.ctx, &cofidectl_proto.GetTrustZoneRequest{Name: &name})
	if err != nil {
		return nil, err
	}

	return resp.TrustZone, nil
}

func (c *DataSourcePluginClientGRPC) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.ListTrustZones(c.ctx, &cofidectl_proto.ListTrustZonesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.TrustZones, nil
}

func (c *DataSourcePluginClientGRPC) UpdateTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.UpdateTrustZone(c.ctx, &cofidectl_proto.UpdateTrustZoneRequest{TrustZone: trustZone})
	if err != nil {
		return nil, err
	}

	return resp.TrustZone, nil
}

func (c *DataSourcePluginClientGRPC) AddCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	resp, err := c.client.AddCluster(c.ctx, &cofidectl_proto.AddClusterRequest{Cluster: cluster})
	if err != nil {
		return nil, err
	}

	return resp.Cluster, nil
}

func (c *DataSourcePluginClientGRPC) GetCluster(name, trustZone string) (*clusterpb.Cluster, error) {
	resp, err := c.client.GetCluster(c.ctx, &cofidectl_proto.GetClusterRequest{Name: &name, TrustZone: &trustZone})
	if err != nil {
		return nil, err
	}

	return resp.Cluster, nil
}

func (c *DataSourcePluginClientGRPC) ListClusters(trustZone string) ([]*clusterpb.Cluster, error) {
	resp, err := c.client.ListClusters(c.ctx, &cofidectl_proto.ListClustersRequest{TrustZone: &trustZone})
	if err != nil {
		return nil, err
	}

	return resp.Clusters, nil
}

func (c *DataSourcePluginClientGRPC) UpdateCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	resp, err := c.client.UpdateCluster(c.ctx, &cofidectl_proto.UpdateClusterRequest{Cluster: cluster})
	if err != nil {
		return nil, err
	}

	return resp.Cluster, nil
}

func (c *DataSourcePluginClientGRPC) AddAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) (*attestation_policy_proto.AttestationPolicy, error) {
	resp, err := c.client.AddAttestationPolicy(c.ctx, &cofidectl_proto.AddAttestationPolicyRequest{Policy: policy})
	if err != nil {
		return nil, err
	}

	return resp.Policy, nil
}

func (c *DataSourcePluginClientGRPC) GetAttestationPolicy(name string) (*attestation_policy_proto.AttestationPolicy, error) {
	resp, err := c.client.GetAttestationPolicy(c.ctx, &cofidectl_proto.GetAttestationPolicyRequest{Name: &name})
	if err != nil {
		return nil, err
	}

	return resp.Policy, nil
}

func (c *DataSourcePluginClientGRPC) ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error) {
	resp, err := c.client.ListAttestationPolicies(c.ctx, &cofidectl_proto.ListAttestationPoliciesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Policies, nil
}

func (c *DataSourcePluginClientGRPC) AddAPBinding(binding *ap_binding_proto.APBinding) (*ap_binding_proto.APBinding, error) {
	resp, err := c.client.AddAPBinding(c.ctx, &cofidectl_proto.AddAPBindingRequest{Binding: binding})
	if err != nil {
		return nil, err
	}

	return resp.Binding, nil
}

func (c *DataSourcePluginClientGRPC) DestroyAPBinding(binding *ap_binding_proto.APBinding) error {
	_, err := c.client.DestroyAPBinding(c.ctx, &cofidectl_proto.DestroyAPBindingRequest{Binding: binding})
	return err
}

func (c *DataSourcePluginClientGRPC) AddFederation(federation *federation_proto.Federation) (*federation_proto.Federation, error) {
	resp, err := c.client.AddFederation(c.ctx, &cofidectl_proto.AddFederationRequest{Federation: federation})
	if err != nil {
		return nil, err
	}

	return resp.Federation, nil
}

func (c *DataSourcePluginClientGRPC) ListFederations() ([]*federation_proto.Federation, error) {
	resp, err := c.client.ListFederations(c.ctx, &cofidectl_proto.ListFederationsRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Federations, nil
}

func (c *DataSourcePluginClientGRPC) ListFederationsByTrustZone(string) ([]*federation_proto.Federation, error) {
	resp, err := c.client.ListFederationsByTrustZone(c.ctx, &cofidectl_proto.ListFederationsByTrustZoneRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Federations, nil
}

type GRPCServer struct {
	cofidectl_proto.UnimplementedDataSourcePluginServiceServer
	Impl DataSource
}

func (s *GRPCServer) Validate(ctx context.Context, req *cofidectl_proto.ValidateRequest) (*cofidectl_proto.ValidateResponse, error) {
	err := s.Impl.Validate(ctx)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ValidateResponse{}, nil
}

func (s *GRPCServer) AddTrustZone(_ context.Context, req *cofidectl_proto.AddTrustZoneRequest) (*cofidectl_proto.AddTrustZoneResponse, error) {
	trustZone, err := s.Impl.AddTrustZone(req.TrustZone)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.AddTrustZoneResponse{TrustZone: trustZone}, nil
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

func (s *GRPCServer) UpdateTrustZone(_ context.Context, req *cofidectl_proto.UpdateTrustZoneRequest) (*cofidectl_proto.UpdateTrustZoneResponse, error) {
	trustZone, err := s.Impl.UpdateTrustZone(req.TrustZone)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.UpdateTrustZoneResponse{TrustZone: trustZone}, nil
}

func (s *GRPCServer) AddCluster(_ context.Context, req *cofidectl_proto.AddClusterRequest) (*cofidectl_proto.AddClusterResponse, error) {
	cluster, err := s.Impl.AddCluster(req.Cluster)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.AddClusterResponse{Cluster: cluster}, nil
}

func (s *GRPCServer) GetCluster(_ context.Context, req *cofidectl_proto.GetClusterRequest) (*cofidectl_proto.GetClusterResponse, error) {
	cluster, err := s.Impl.GetCluster(req.GetName(), req.GetTrustZone())
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.GetClusterResponse{Cluster: cluster}, nil
}

func (s *GRPCServer) ListClusters(_ context.Context, req *cofidectl_proto.ListClustersRequest) (*cofidectl_proto.ListClustersResponse, error) {
	clusters, err := s.Impl.ListClusters(req.GetTrustZone())
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.ListClustersResponse{Clusters: clusters}, nil
}

func (s *GRPCServer) UpdateCluster(_ context.Context, req *cofidectl_proto.UpdateClusterRequest) (*cofidectl_proto.UpdateClusterResponse, error) {
	cluster, err := s.Impl.UpdateCluster(req.Cluster)
	if err != nil {
		return nil, err
	}
	return &cofidectl_proto.UpdateClusterResponse{Cluster: cluster}, nil
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
