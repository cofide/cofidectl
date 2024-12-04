// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	cofidectl_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
)

// Type check to ensure DataSourcePluginClientGRPC implements DataSource.
var _ DataSource = &DataSourcePluginClientGRPC{}

// DataSourcePluginClientGRPC is used by clients (main application) to translate the
// DataSource interface of plugins to GRPC calls.
type DataSourcePluginClientGRPC struct {
	ctx    context.Context
	client cofidectl_proto.DataSourcePluginServiceClient
}

func (c *DataSourcePluginClientGRPC) Validate() error {
	_, err := c.client.Validate(c.ctx, &cofidectl_proto.ValidateRequest{})
	return err
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

func (c *DataSourcePluginClientGRPC) AddTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.AddTrustZone(c.ctx, &cofidectl_proto.AddTrustZoneRequest{TrustZone: trustZone})
	if err != nil {
		return nil, err
	}

	return resp.TrustZone, nil
}

func (c *DataSourcePluginClientGRPC) UpdateTrustZone(trustZone *trust_zone_proto.TrustZone) error {
	_, err := c.client.UpdateTrustZone(c.ctx, &cofidectl_proto.UpdateTrustZoneRequest{TrustZone: trustZone})
	return err
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
