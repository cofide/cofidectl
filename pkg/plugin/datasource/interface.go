// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/pkg/plugin/validator"
)

// DataSource is the interface data source plugins have to implement.
type DataSource interface {
	validator.Validator

	AddTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error)
	DestroyTrustZone(id string) error
	GetTrustZone(id string) (*trust_zone_proto.TrustZone, error)
	GetTrustZoneByName(name string) (*trust_zone_proto.TrustZone, error)
	ListTrustZones() ([]*trust_zone_proto.TrustZone, error)
	UpdateTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error)

	AddCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error)
	DestroyCluster(id string) error
	GetCluster(id string) (*clusterpb.Cluster, error)
	GetClusterByName(name, trustZoneID string) (*clusterpb.Cluster, error)
	ListClusters(filter *datasourcepb.ListClustersRequest_Filter) ([]*clusterpb.Cluster, error)
	UpdateCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error)

	AddAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) (*attestation_policy_proto.AttestationPolicy, error)
	DestroyAttestationPolicy(id string) error
	GetAttestationPolicy(id string) (*attestation_policy_proto.AttestationPolicy, error)
	GetAttestationPolicyByName(name string) (*attestation_policy_proto.AttestationPolicy, error)
	ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error)

	AddAPBinding(binding *ap_binding_proto.APBinding) (*ap_binding_proto.APBinding, error)
	DestroyAPBinding(id string) error
	ListAPBindings(filter *datasourcepb.ListAPBindingsRequest_Filter) ([]*ap_binding_proto.APBinding, error)

	AddFederation(federation *federation_proto.Federation) (*federation_proto.Federation, error)
	DestroyFederation(id string) error
	ListFederations(filter *datasourcepb.ListFederationsRequest_Filter) ([]*federation_proto.Federation, error)
}
