// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"fmt"
	"slices"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl_plugin/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/proto"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"

	"github.com/google/uuid"
)

func generateId() (*string, error) {
	uid, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}
	id := uid.String()
	return &id, nil
}

var _ datasource.DataSource = (*LocalDataSource)(nil)

type LocalDataSource struct {
	loader config.Loader
	config *config.Config
}

func NewLocalDataSource(loader config.Loader) (*LocalDataSource, error) {
	cfg := config.NewConfig()
	lds := &LocalDataSource{
		loader: loader,
		config: cfg,
	}

	err := lds.loadState()
	if err != nil {
		return nil, err
	}

	return lds, nil
}

func (lds *LocalDataSource) Validate(_ context.Context) error {
	return nil
}

func (lds *LocalDataSource) loadState() error {
	dataFileExists, err := lds.loader.Exists()
	if err != nil {
		return err
	}

	if !dataFileExists {
		return fmt.Errorf("the config file doesn't exist. Please run cofidectl init")
	}

	config, err := lds.loader.Read()
	if err != nil {
		return err
	}

	lds.config = config
	return nil
}

func (lds *LocalDataSource) updateDataFile() error {
	return lds.loader.Write(lds.config)
}

func (lds *LocalDataSource) AddTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	if trustZone.GetId() != "" {
		return nil, fmt.Errorf("trust zone %s should not have an ID set, this will be auto generated", trustZone.GetId())
	}

	id, err := generateId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID for trust zone: %w", err)
	}
	trustZone.Id = id

	if _, ok := lds.config.GetTrustZoneByName(trustZone.Name); ok {
		return nil, fmt.Errorf("trust zone %s already exists in local config", trustZone.Name)
	}
	trustZone, err = proto.CloneTrustZone(trustZone)
	if err != nil {
		return nil, err
	}

	lds.config.TrustZones = append(lds.config.TrustZones, trustZone)
	if err := lds.updateDataFile(); err != nil {
		return nil, fmt.Errorf("failed to add trust zone %s to local config: %s", trustZone.Name, err)
	}
	return trustZone, nil
}

func (lds *LocalDataSource) DestroyTrustZone(name string) error {
	// Fail if any clusters exist in the trust zone.
	if len(lds.config.GetClustersByTrustZone(name)) > 0 {
		return fmt.Errorf("one or more clusters exist in trust zone %s in local config", name)
	}
	// Deleting the trust zone also implicitly removes any attestation policies and federations
	// bound to it because they are stored in the trust zone message.
	// Federations in other trust zones that reference this trust zone need to be cleaned up.
	for _, trustZone := range lds.config.TrustZones {
		if trustZone.Name != name {
			// nolint:staticcheck
			trustZone.Federations = slices.DeleteFunc(
				// nolint:staticcheck
				trustZone.Federations,
				func(federation *federation_proto.Federation) bool {
					// nolint:staticcheck
					return federation.To == name
				},
			)
		}
	}
	for i, trustZone := range lds.config.TrustZones {
		if trustZone.Name == name {
			lds.config.TrustZones = append(lds.config.TrustZones[:i], lds.config.TrustZones[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove trust zone from local config: %s", err)
			}
			return nil
		}
	}
	return fmt.Errorf("failed to find trust zone %s in local config", name)
}

func (lds *LocalDataSource) GetTrustZone(name string) (*trust_zone_proto.TrustZone, error) {
	trustZone, ok := lds.config.GetTrustZoneByName(name)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", name)
	}

	return proto.CloneTrustZone(trustZone)
}

func (lds *LocalDataSource) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	trustZones := []*trust_zone_proto.TrustZone{}
	for _, trustZone := range lds.config.TrustZones {
		trustZone, err := proto.CloneTrustZone(trustZone)
		if err != nil {
			return nil, err
		}
		trustZones = append(trustZones, trustZone)
	}
	return trustZones, nil
}

func (lds *LocalDataSource) UpdateTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	for i, current := range lds.config.TrustZones {
		if current.Name == trustZone.Name {
			if err := validateTrustZoneUpdate(current, trustZone); err != nil {
				return nil, err
			}

			trustZone, err := proto.CloneTrustZone(trustZone)
			if err != nil {
				return nil, err
			}

			lds.config.TrustZones[i] = trustZone

			if err := lds.updateDataFile(); err != nil {
				return nil, fmt.Errorf("failed to update trust zone %s in local config: %s", trustZone.Name, err)
			}

			return proto.CloneTrustZone(trustZone)
		}
	}

	return nil, fmt.Errorf("failed to find trust zone %s in local config", trustZone.Name)
}

func validateTrustZoneUpdate(current, new *trust_zone_proto.TrustZone) error {
	if new.Name != current.Name {
		return fmt.Errorf("cannot update name for existing trust zone %s", current.Name)
	}
	if new.TrustDomain != current.TrustDomain {
		return fmt.Errorf("cannot update trust domain for existing trust zone %s", current.Name)
	}
	// The following should be updated though other means.
	// nolint:staticcheck
	if !slices.EqualFunc(new.Federations, current.Federations, proto.FederationsEqual) {
		return fmt.Errorf("cannot update federations for existing trust zone %s", current.Name)
	}
	// nolint:staticcheck
	if !slices.EqualFunc(new.AttestationPolicies, current.AttestationPolicies, proto.APBindingsEqual) {
		return fmt.Errorf("cannot update attestation policies for existing trust zone %s", current.Name)
	}
	return nil
}

func (lds *LocalDataSource) AddCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	name := cluster.GetName()
	trustZone := cluster.GetTrustZone()

	if cluster.GetId() != "" {
		return nil, fmt.Errorf("cluster %s should not have an ID set, this will be auto generated", cluster.GetId())
	}
	id, err := generateId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID for cluster: %w", err)
	}
	cluster.Id = id

	if _, ok := lds.config.GetClusterByName(name, trustZone); ok {
		return nil, fmt.Errorf("cluster %s already exists in trust zone %s in local config", name, trustZone)
	}

	if len(lds.config.GetClustersByTrustZone(trustZone)) != 0 {
		return nil, fmt.Errorf("trust zone %s already has a cluster", trustZone)
	}

	cluster, err = proto.CloneCluster(cluster)
	if err != nil {
		return nil, err
	}

	lds.config.Clusters = append(lds.config.Clusters, cluster)
	if err := lds.updateDataFile(); err != nil {
		return nil, fmt.Errorf("failed to add cluster %s in trust zone %s to local config: %s", name, trustZone, err)
	}
	return cluster, nil
}

func (lds *LocalDataSource) DestroyCluster(name, trustZoneName string) error {
	for i, cluster := range lds.config.Clusters {
		if cluster.GetName() == name && cluster.GetTrustZone() == trustZoneName {
			lds.config.Clusters = append(lds.config.Clusters[:i], lds.config.Clusters[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove cluster from local config: %s", err)
			}
			return nil
		}
	}
	return fmt.Errorf("failed to find cluster %s in trust zone %s in local config", name, trustZoneName)
}

func (lds *LocalDataSource) GetCluster(name, trustZone string) (*clusterpb.Cluster, error) {
	cluster, ok := lds.config.GetClusterByName(name, trustZone)
	if !ok {
		return nil, fmt.Errorf("failed to find cluster %s in trust zone %s in local config", name, trustZone)
	}

	return proto.CloneCluster(cluster)
}

func (lds *LocalDataSource) ListClusters(trustZone string) ([]*clusterpb.Cluster, error) {
	clusters := []*clusterpb.Cluster{}
	for _, cluster := range lds.config.GetClustersByTrustZone(trustZone) {
		cluster, err := proto.CloneCluster(cluster)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

func (lds *LocalDataSource) UpdateCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	name := cluster.GetName()
	trustZone := cluster.GetTrustZone()

	for i, current := range lds.config.Clusters {
		if current.GetName() == name {
			if err := validateClusterUpdate(current, cluster); err != nil {
				return nil, err
			}

			cluster, err := proto.CloneCluster(cluster)
			if err != nil {
				return nil, err
			}

			lds.config.Clusters[i] = cluster

			if err := lds.updateDataFile(); err != nil {
				return nil, fmt.Errorf("failed to update cluster %s in trust zone %s in local config: %s", name, trustZone, err)
			}

			return proto.CloneCluster(cluster)
		}
	}

	return nil, fmt.Errorf("failed to find cluster %s in trust zone %s in local config", name, trustZone)
}

func validateClusterUpdate(current, new *clusterpb.Cluster) error {
	name := current.GetName()
	trustZone := current.GetTrustZone()

	if new.GetName() != current.GetName() {
		return fmt.Errorf("cannot update name for existing cluster %s in trust zone %s", name, trustZone)
	}

	if new.GetTrustZone() != current.GetTrustZone() {
		return fmt.Errorf("cannot update trust zone for existing cluster %s in trust zone %s", name, trustZone)
	}

	if new.GetProfile() != current.GetProfile() {
		return fmt.Errorf("cannot update profile for existing cluster %s in trust zone %s", name, trustZone)
	}

	if err := validateTrustProviderUpdate(current.GetName(), current.GetTrustZone(), current.TrustProvider, new.TrustProvider); err != nil {
		return err
	}

	return nil
}

func validateTrustProviderUpdate(cluster, tzName string, current, new *trust_provider_proto.TrustProvider) error {
	if current == nil {
		return fmt.Errorf("no trust provider in existing cluster %s in trust zone %s", cluster, tzName)
	}
	if new == nil {
		return fmt.Errorf("cannot remove trust provider for cluster %s in trust zone %s", cluster, tzName)
	}
	if new.GetKind() != current.GetKind() {
		return fmt.Errorf("cannot update trust provider kind for existing cluster %s in trust zone %s", cluster, tzName)
	}
	return nil
}

func (lds *LocalDataSource) AddAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) (*attestation_policy_proto.AttestationPolicy, error) {
	if policy.GetId() != "" {
		return nil, fmt.Errorf("attestation policy %s should not have an ID set, this will be auto generated", *policy.Id)
	}

	id, err := generateId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID for attestation policy: %w", err)
	}
	policy.Id = id

	if _, ok := lds.config.GetAttestationPolicyByName(policy.Name); ok {
		return nil, fmt.Errorf("attestation policy %s already exists in local config", policy.Name)
	}
	policy, err = proto.CloneAttestationPolicy(policy)
	if err != nil {
		return nil, err
	}
	lds.config.AttestationPolicies = append(lds.config.AttestationPolicies, policy)
	if err := lds.updateDataFile(); err != nil {
		return nil, fmt.Errorf("failed to add attestation policy to local config: %s", err)
	}
	return proto.CloneAttestationPolicy(policy)
}

func (lds *LocalDataSource) DestroyAttestationPolicy(name string) error {
	// Fail if the policy is bound to any trust zones.
	for _, trustZone := range lds.config.TrustZones {
		// nolint:staticcheck
		for _, binding := range trustZone.AttestationPolicies {
			// nolint:staticcheck
			if binding.Policy == name {
				return fmt.Errorf("attestation policy %s is bound to trust zone %s in local config", name, trustZone.Name)
			}
		}
	}
	for i, policy := range lds.config.AttestationPolicies {
		if policy.Name == name {
			lds.config.AttestationPolicies = append(lds.config.AttestationPolicies[:i], lds.config.AttestationPolicies[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove attestation policy from local config: %s", err)
			}
			return nil
		}
	}
	return fmt.Errorf("failed to find attestation policy %s in local config", name)
}

func (lds *LocalDataSource) GetAttestationPolicy(name string) (*attestation_policy_proto.AttestationPolicy, error) {
	if policy, ok := lds.config.GetAttestationPolicyByName(name); ok {
		return proto.CloneAttestationPolicy(policy)
	} else {
		return nil, fmt.Errorf("failed to find attestation policy %s in local config", name)
	}
}

func (lds *LocalDataSource) ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error) {
	policies := []*attestation_policy_proto.AttestationPolicy{}
	for _, policy := range lds.config.AttestationPolicies {
		policy, err := proto.CloneAttestationPolicy(policy)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

func (lds *LocalDataSource) AddAPBinding(binding *ap_binding_proto.APBinding) (*ap_binding_proto.APBinding, error) {
	if binding.GetId() != "" {
		return nil, fmt.Errorf("attestation policy binding %s should not have an ID set, this will be auto generated", *binding.Id)
	}

	id, err := generateId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID for attestation policy binding: %w", err)
	}
	binding.Id = id

	// nolint:staticcheck
	localTrustZone, ok := lds.config.GetTrustZoneByName(binding.TrustZone)
	if !ok {
		// nolint:staticcheck
		return nil, fmt.Errorf("failed to find trust zone %s in local config", binding.TrustZone)
	}

	// nolint:staticcheck
	_, ok = lds.config.GetAttestationPolicyByName(binding.Policy)
	if !ok {
		// nolint:staticcheck
		return nil, fmt.Errorf("failed to find attestation policy %s in local config", binding.Policy)
	}

	// nolint:staticcheck
	for _, apb := range localTrustZone.AttestationPolicies {
		// nolint:staticcheck
		if apb.Policy == binding.Policy {
			// nolint:staticcheck
			return nil, fmt.Errorf("attestation policy %s is already bound to trust zone %s", binding.Policy, binding.TrustZone)
		}
	}

	remoteTzs := map[string]bool{}
	// nolint:staticcheck
	for _, federation := range localTrustZone.Federations {
		// nolint:staticcheck
		remoteTzs[federation.To] = true
	}
	// nolint:staticcheck
	for _, remoteTz := range binding.FederatesWith {
		// nolint:staticcheck
		if remoteTz == binding.TrustZone {
			// Is this a problem?
			// nolint:staticcheck
			return nil, fmt.Errorf("attestation policy %s federates with its own trust zone %s", binding.Policy, binding.TrustZone)
		}
		if _, ok := remoteTzs[remoteTz]; !ok {
			if _, ok := lds.config.GetTrustZoneByName(remoteTz); !ok {
				return nil, fmt.Errorf("attestation policy %s federates with unknown trust zone %s", binding.Policy, remoteTz)
			} else {
				return nil, fmt.Errorf("attestation policy %s federates with %s but trust zone %s does not", binding.Policy, remoteTz, binding.TrustZone)
			}
		}
	}

	binding, err = proto.CloneAPBinding(binding)
	if err != nil {
		return nil, err
	}
	// nolint:staticcheck
	localTrustZone.AttestationPolicies = append(localTrustZone.AttestationPolicies, binding)
	if err := lds.updateDataFile(); err != nil {
		return nil, fmt.Errorf("failed to add attestation policy to local config: %w", err)
	}
	return proto.CloneAPBinding(binding)
}

func (lds *LocalDataSource) DestroyAPBinding(binding *ap_binding_proto.APBinding) error {
	// nolint:staticcheck
	trustZone, ok := lds.config.GetTrustZoneByName(binding.TrustZone)
	if !ok {
		// nolint:staticcheck
		return fmt.Errorf("failed to find trust zone %s in local config", binding.TrustZone)
	}

	// nolint:staticcheck
	for i, tzBinding := range trustZone.AttestationPolicies {
		// nolint:staticcheck
		if tzBinding.Policy == binding.Policy {
			trustZone.AttestationPolicies = append(trustZone.AttestationPolicies[:i], trustZone.AttestationPolicies[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove attestation policy binding from local config: %w", err)
			}
			return nil
		}
	}

	// nolint:staticcheck
	return fmt.Errorf("failed to find attestation policy binding for %s in trust zone %s", binding.Policy, binding.TrustZone)
}

func (lds *LocalDataSource) ListAPBindings(filter *datasourcepb.ListAPBindingsRequest_Filter) ([]*ap_binding_proto.APBinding, error) {
	var trustZones []*trust_zone_proto.TrustZone
	if filter != nil && filter.TrustZoneName != nil {
		trustZone, ok := lds.config.GetTrustZoneByName(filter.GetTrustZoneName())
		if !ok {
			return nil, fmt.Errorf("failed to find trust zone %s in local config", filter.GetTrustZoneName())
		}
		trustZones = []*trust_zone_proto.TrustZone{trustZone}
	} else {
		trustZones = lds.config.TrustZones
	}
	bindings := []*ap_binding_proto.APBinding{}
	for _, trustZone := range trustZones {
		// nolint:staticcheck
		for _, binding := range trustZone.AttestationPolicies {
			// nolint:staticcheck
			if filter != nil && filter.PolicyName != nil && binding.Policy != filter.GetPolicyName() {
				continue
			}

			binding, err := proto.CloneAPBinding(binding)
			if err != nil {
				return nil, err
			}
			bindings = append(bindings, binding)
		}
	}
	return bindings, nil
}

func (lds *LocalDataSource) AddFederation(federationProto *federation_proto.Federation) (*federation_proto.Federation, error) {
	if federationProto.Id == nil || *federationProto.Id == "" {
		id, err := generateId()
		if err != nil {
			return nil, fmt.Errorf("failed to generate UUID for federation: %w", err)
		}
		federationProto.Id = id
	}
	// nolint:staticcheck
	fromTrustZone, ok := lds.config.GetTrustZoneByName(federationProto.From)
	if !ok {
		// nolint:staticcheck
		return nil, fmt.Errorf("failed to find trust zone %s in local config", federationProto.From)
	}

	// nolint:staticcheck
	_, ok = lds.config.GetTrustZoneByName(federationProto.To)
	if !ok {
		// nolint:staticcheck
		return nil, fmt.Errorf("failed to find trust zone %s in local config", federationProto.To)
	}

	// nolint:staticcheck
	if federationProto.From == federationProto.To {
		// nolint:staticcheck
		return nil, fmt.Errorf("cannot federate trust zone %s with itself", federationProto.From)
	}

	// nolint:staticcheck
	for _, federation := range fromTrustZone.Federations {
		// nolint:staticcheck
		if federation.To == federationProto.To {
			// nolint:staticcheck
			return nil, fmt.Errorf("federation already exists between %s and %s", federationProto.From, federationProto.To)
		}
	}

	federationProto, err := proto.CloneFederation(federationProto)
	if err != nil {
		return nil, err
	}

	// nolint:staticcheck
	fromTrustZone.Federations = append(fromTrustZone.Federations, federationProto)

	if err := lds.updateDataFile(); err != nil {
		return nil, fmt.Errorf("failed to add federation to local config: %s", err)
	}
	return proto.CloneFederation(federationProto)
}

func (lds *LocalDataSource) DestroyFederation(federation *federation_proto.Federation) error {
	// nolint:staticcheck
	trustZone, ok := lds.config.GetTrustZoneByName(federation.From)
	if !ok {
		// nolint:staticcheck
		return fmt.Errorf("failed to find trust zone %s in local config", federation.From)
	}

	// nolint:staticcheck
	for i, fed := range trustZone.Federations {
		// We cannot compare the protos directly here because the ones in the config have IDs.
		// nolint:staticcheck
		if fed.From == federation.From && fed.To == federation.To {
			// nolint:staticcheck
			trustZone.Federations = append(trustZone.Federations[:i], trustZone.Federations[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove federation from local config: %s", err)
			}
			return nil
		}
	}
	// nolint:staticcheck
	return fmt.Errorf("failed to find federation for trust zone %s in local config", federation.From)
}

func (lds *LocalDataSource) ListFederations() ([]*federation_proto.Federation, error) {
	// federations are expressed in-line with the trust zone(s) so we need to iterate the trust zones
	federations := []*federation_proto.Federation{}
	for _, trustZone := range lds.config.TrustZones {
		// nolint:staticcheck
		for _, federation := range trustZone.Federations {
			federation, err := proto.CloneFederation(federation)
			if err != nil {
				return nil, err
			}
			federations = append(federations, federation)
		}
	}
	return federations, nil
}

func (lds *LocalDataSource) ListFederationsByTrustZone(tzName string) ([]*federation_proto.Federation, error) {
	trustZone, ok := lds.config.GetTrustZoneByName(tzName)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", tzName)
	}

	var federations []*federation_proto.Federation
	// nolint:staticcheck
	for _, federation := range trustZone.Federations {
		federation, err := proto.CloneFederation(federation)
		if err != nil {
			return nil, err
		}
		federations = append(federations, federation)
	}
	return federations, nil
}
