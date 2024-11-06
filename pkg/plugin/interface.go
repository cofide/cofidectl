// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/proto/ap_binding/v1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

// DataSource is the interface plugins have to implement.
type DataSource interface {
	Validate() error
	GetTrustZone(string) (*trust_zone_proto.TrustZone, error)
	ListTrustZones() ([]*trust_zone_proto.TrustZone, error)
	CreateTrustZone(*trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error)
	UpdateTrustZone(*trust_zone_proto.TrustZone) error

	AddAttestationPolicy(*attestation_policy_proto.AttestationPolicy) error
	GetAttestationPolicy(string) (*attestation_policy_proto.AttestationPolicy, error)
	ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error)

	AddAPBinding(*ap_binding_proto.APBinding) error

	AddFederation(*federation_proto.Federation) error
	ListFederations() ([]*federation_proto.Federation, error)
	ListFederationsByTrustZone(string) ([]*federation_proto.Federation, error)
}
