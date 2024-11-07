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

// DataSourcePluginClientGRPC is used by clients (main application) to translate the
// DataSource interface of plugins to GRPC calls.
type DataSourcePluginClientGRPC struct {
	client cofidectl_proto.DataSourcePluginServiceClient
}

func (c *DataSourcePluginClientGRPC) Validate() error {
	// Unimplemented.
	return nil
}

func (c *DataSourcePluginClientGRPC) GetTrustZone(name string) (*trust_zone_proto.TrustZone, error) {
	// Unimplemented.
	return nil, nil
}

func (c *DataSourcePluginClientGRPC) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	resp, err := c.client.ListTrustZones(context.Background(), &cofidectl_proto.ListTrustZonesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.TrustZones, nil
}

func (c *DataSourcePluginClientGRPC) AddTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	// Unimplemented.
	return nil, nil
}

func (c *DataSourcePluginClientGRPC) UpdateTrustZone(*trust_zone_proto.TrustZone) error {
	// Unimplemented.
	return nil
}

func (c *DataSourcePluginClientGRPC) AddAttestationPolicy(*attestation_policy_proto.AttestationPolicy) error {
	// Unimplemented.
	return nil
}

func (c *DataSourcePluginClientGRPC) GetAttestationPolicy(string) (*attestation_policy_proto.AttestationPolicy, error) {
	// Unimplemented.
	return nil, nil
}

func (c *DataSourcePluginClientGRPC) ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error) {
	// Unimplemented.
	return nil, nil
}

func (c *DataSourcePluginClientGRPC) AddAPBinding(*ap_binding_proto.APBinding) error {
	// Unimplemented.
	return nil
}

func (c *DataSourcePluginClientGRPC) AddFederation(*federation_proto.Federation) error {
	// Unimplemented.
	return nil
}

func (c *DataSourcePluginClientGRPC) ListFederations() ([]*federation_proto.Federation, error) {
	// Unimplemented.
	return nil, nil
}

func (c *DataSourcePluginClientGRPC) ListFederationsByTrustZone(string) ([]*federation_proto.Federation, error) {
	// Unimplemented.
	return nil, nil
}
