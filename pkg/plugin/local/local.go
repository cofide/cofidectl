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
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
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

func (lds *LocalDataSource) DestroyTrustZone(id string) error {
	// Fail if any clusters exist in the trust zone.
	if len(lds.config.GetClustersByTrustZone(id)) > 0 {
		return fmt.Errorf("one or more clusters exist in trust zone %s in local config", id)
	}
	// Deleting the trust zone also implicitly removes any attestation policies and federations
	// bound to it because they are stored in the trust zone message.
	// Federations in other trust zones that reference this trust zone need to be cleaned up.
	for _, trustZone := range lds.config.TrustZones {
		if trustZone.GetId() != id {
			// nolint:staticcheck
			trustZone.Federations = slices.DeleteFunc(
				// nolint:staticcheck
				trustZone.Federations,
				func(federation *federation_proto.Federation) bool {
					return federation.GetRemoteTrustZoneId() == id
				},
			)
		}
	}
	for i, trustZone := range lds.config.TrustZones {
		if trustZone.GetId() == id {
			lds.config.TrustZones = append(lds.config.TrustZones[:i], lds.config.TrustZones[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove trust zone from local config: %s", err)
			}
			return nil
		}
	}
	return fmt.Errorf("failed to find trust zone %s in local config", id)
}

func (lds *LocalDataSource) GetTrustZone(id string) (*trust_zone_proto.TrustZone, error) {
	trustZone, ok := lds.config.GetTrustZoneByID(id)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", id)
	}

	return proto.CloneTrustZone(trustZone)
}

func (lds *LocalDataSource) GetTrustZoneByName(name string) (*trust_zone_proto.TrustZone, error) {
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
		if current.GetId() == trustZone.GetId() {
			if err := validateTrustZoneUpdate(current, trustZone); err != nil {
				return nil, err
			}

			trustZone, err := proto.CloneTrustZone(trustZone)
			if err != nil {
				return nil, err
			}

			lds.config.TrustZones[i] = trustZone

			if err := lds.updateDataFile(); err != nil {
				return nil, fmt.Errorf("failed to update trust zone %s in local config: %s", trustZone.GetId(), err)
			}

			return proto.CloneTrustZone(trustZone)
		}
	}

	return nil, fmt.Errorf("failed to find trust zone %s in local config", trustZone.GetId())
}

func validateTrustZoneUpdate(current, new *trust_zone_proto.TrustZone) error {
	if new.GetId() != current.GetId() {
		return fmt.Errorf("cannot update id for existing trust zone %s", *current.Id)
	}
	if new.Name != current.Name {
		return fmt.Errorf("cannot update name for existing trust zone %s", *current.Id)
	}
	if new.TrustDomain != current.TrustDomain {
		return fmt.Errorf("cannot update trust domain for existing trust zone %s", *current.Id)
	}
	// The following should be updated though other means.
	// nolint:staticcheck
	if !slices.EqualFunc(new.Federations, current.Federations, proto.FederationsEqual) {
		return fmt.Errorf("cannot update federations for existing trust zone %s", *current.Id)
	}
	// nolint:staticcheck
	if !slices.EqualFunc(new.AttestationPolicies, current.AttestationPolicies, proto.APBindingsEqual) {
		return fmt.Errorf("cannot update attestation policies for existing trust zone %s", *current.Id)
	}
	return nil
}

func (lds *LocalDataSource) AddCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	name := cluster.GetName()
	trustZoneID := cluster.GetTrustZoneId()

	if cluster.GetId() != "" {
		return nil, fmt.Errorf("cluster %s should not have an ID set, this will be auto generated", cluster.GetId())
	}
	id, err := generateId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID for cluster: %w", err)
	}
	cluster.Id = id

	if _, ok := lds.config.GetClusterByName(name, trustZoneID); ok {
		return nil, fmt.Errorf("cluster %s already exists in trust zone %s in local config", name, trustZoneID)
	}

	if len(lds.config.GetClustersByTrustZone(trustZoneID)) != 0 {
		return nil, fmt.Errorf("trust zone %s already has a cluster", trustZoneID)
	}

	cluster, err = proto.CloneCluster(cluster)
	if err != nil {
		return nil, err
	}

	lds.config.Clusters = append(lds.config.Clusters, cluster)
	if err := lds.updateDataFile(); err != nil {
		return nil, fmt.Errorf("failed to add cluster %s in trust zone %s to local config: %s", name, trustZoneID, err)
	}
	return cluster, nil
}

func (lds *LocalDataSource) DestroyCluster(id string) error {
	for i, cluster := range lds.config.Clusters {
		if cluster.GetId() == id {
			lds.config.Clusters = append(lds.config.Clusters[:i], lds.config.Clusters[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove cluster from local config: %s", err)
			}
			return nil
		}
	}
	return fmt.Errorf("failed to find cluster %s in local config", id)
}

func (lds *LocalDataSource) GetCluster(id string) (*clusterpb.Cluster, error) {
	cluster, ok := lds.config.GetClusterByID(id)
	if !ok {
		return nil, fmt.Errorf("failed to find cluster %s in local config", id)
	}

	return proto.CloneCluster(cluster)
}

func (lds *LocalDataSource) GetClusterByName(name, trustZoneID string) (*clusterpb.Cluster, error) {
	cluster, ok := lds.config.GetClusterByName(name, trustZoneID)
	if !ok {
		return nil, fmt.Errorf("failed to find cluster %s in trust zone %s in local config", name, trustZoneID)
	}

	return proto.CloneCluster(cluster)
}

func (lds *LocalDataSource) ListClusters(filter *datasourcepb.ListClustersRequest_Filter) ([]*clusterpb.Cluster, error) {
	clusters := []*clusterpb.Cluster{}
	if filter != nil && filter.GetTrustZoneId() != "" {
		for _, cluster := range lds.config.GetClustersByTrustZone(filter.GetTrustZoneId()) {
			cluster, err := proto.CloneCluster(cluster)
			if err != nil {
				return nil, err
			}
			clusters = append(clusters, cluster)
		}
	} else {
		for _, cluster := range lds.config.Clusters {
			cluster, err := proto.CloneCluster(cluster)
			if err != nil {
				return nil, err
			}
			clusters = append(clusters, cluster)
		}
	}
	return clusters, nil
}

func (lds *LocalDataSource) UpdateCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	id := cluster.GetId()
	trustZoneId := cluster.GetTrustZoneId()

	for i, current := range lds.config.Clusters {
		if current.GetId() == id {
			if err := validateClusterUpdate(current, cluster); err != nil {
				return nil, err
			}

			cluster, err := proto.CloneCluster(cluster)
			if err != nil {
				return nil, err
			}

			lds.config.Clusters[i] = cluster

			if err := lds.updateDataFile(); err != nil {
				return nil, fmt.Errorf("failed to update cluster %s in trust zone %s in local config: %s", id, trustZoneId, err)
			}

			return proto.CloneCluster(cluster)
		}
	}

	return nil, fmt.Errorf("failed to find cluster %s in trust zone %s in local config", id, trustZoneId)
}

func validateClusterUpdate(current, new *clusterpb.Cluster) error {
	id := current.GetId()
	trustZoneID := current.GetTrustZoneId()

	if new.GetId() != current.GetId() {
		return fmt.Errorf("cannot update id for existing cluster %s in trust zone %s", id, trustZoneID)
	}

	if new.GetName() != current.GetName() {
		return fmt.Errorf("cannot update name for existing cluster %s in trust zone %s", id, trustZoneID)
	}

	if new.GetTrustZoneId() != current.GetTrustZoneId() {
		return fmt.Errorf("cannot update trust zone for existing cluster %s in trust zone %s", id, trustZoneID)
	}

	if new.GetProfile() != current.GetProfile() {
		return fmt.Errorf("cannot update profile for existing cluster %s in trust zone %s", id, trustZoneID)
	}

	if err := validateTrustProviderUpdate(current.GetId(), current.GetTrustZoneId(), current.TrustProvider, new.TrustProvider); err != nil {
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

	if _, ok := lds.config.GetAttestationPolicyByID(policy.GetId()); ok {
		return nil, fmt.Errorf("attestation policy %s already exists in local config", policy.GetId())
	}

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

func (lds *LocalDataSource) DestroyAttestationPolicy(id string) error {
	// Fail if the policy is bound to any trust zones.
	for _, trustZone := range lds.config.TrustZones {
		// nolint:staticcheck
		for _, binding := range trustZone.AttestationPolicies {
			// nolint:staticcheck
			if binding.GetPolicyId() == id {
				return fmt.Errorf("attestation policy %s is bound to trust zone %s in local config", id, trustZone.Name)
			}
		}
	}
	for i, policy := range lds.config.AttestationPolicies {
		if policy.GetId() == id {
			lds.config.AttestationPolicies = append(lds.config.AttestationPolicies[:i], lds.config.AttestationPolicies[i+1:]...)
			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to remove attestation policy from local config: %s", err)
			}
			return nil
		}
	}
	return fmt.Errorf("failed to find attestation policy %s in local config", id)
}

func (lds *LocalDataSource) GetAttestationPolicy(id string) (*attestation_policy_proto.AttestationPolicy, error) {
	if policy, ok := lds.config.GetAttestationPolicyByID(id); ok {
		return proto.CloneAttestationPolicy(policy)
	} else {
		return nil, fmt.Errorf("failed to find attestation policy %s in local config", id)
	}
}

func (lds *LocalDataSource) GetAttestationPolicyByName(name string) (*attestation_policy_proto.AttestationPolicy, error) {
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

	localTrustZone, ok := lds.config.GetTrustZoneByID(binding.GetTrustZoneId())
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", binding.GetTrustZoneId())
	}

	_, ok = lds.config.GetAttestationPolicyByID(binding.GetPolicyId())
	if !ok {
		return nil, fmt.Errorf("failed to find attestation policy %s in local config", binding.GetPolicyId())
	}

	// nolint:staticcheck
	for _, apb := range localTrustZone.AttestationPolicies {
		if apb.GetPolicyId() == binding.GetPolicyId() {
			return nil, fmt.Errorf("attestation policy %s is already bound to trust zone %s", binding.GetPolicyId(), binding.GetTrustZoneId())
		}
	}

	remoteTzs := map[string]bool{}
	// nolint:staticcheck
	for _, federation := range localTrustZone.Federations {
		remoteTzs[federation.GetRemoteTrustZoneId()] = true
	}

	for _, remoteTz := range binding.Federations {
		if remoteTz.GetTrustZoneId() == binding.GetTrustZoneId() {
			// Is this a problem?
			return nil, fmt.Errorf("attestation policy %s federates with its own trust zone %s", binding.GetPolicyId(), binding.GetTrustZoneId())
		}
		if _, ok := remoteTzs[remoteTz.GetTrustZoneId()]; !ok {
			if _, ok := lds.config.GetTrustZoneByID(remoteTz.GetTrustZoneId()); !ok {
				return nil, fmt.Errorf("attestation policy %s federates with unknown trust zone %s", binding.GetPolicyId(), remoteTz.GetTrustZoneId())
			} else {
				return nil, fmt.Errorf("attestation policy %s federates with %s but trust zone %s does not", binding.GetPolicyId(), remoteTz.GetTrustZoneId(), binding.GetTrustZoneId())
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

func (lds *LocalDataSource) DestroyAPBinding(id string) error {
	for _, trustZone := range lds.config.TrustZones {
		// nolint:staticcheck
		for i, tzBinding := range trustZone.AttestationPolicies {
			if tzBinding.GetId() == id {
				// nolint:staticcheck
				trustZone.AttestationPolicies = append(trustZone.AttestationPolicies[:i], trustZone.AttestationPolicies[i+1:]...)
				if err := lds.updateDataFile(); err != nil {
					return fmt.Errorf("failed to remove attestation policy binding from local config: %w", err)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("failed to find attestation policy binding %s in local config", id)
}

func (lds *LocalDataSource) ListAPBindings(filter *datasourcepb.ListAPBindingsRequest_Filter) ([]*ap_binding_proto.APBinding, error) {
	var trustZones []*trust_zone_proto.TrustZone
	if filter != nil && filter.TrustZoneId != nil {
		trustZone, ok := lds.config.GetTrustZoneByID(filter.GetTrustZoneId())
		if !ok {
			return nil, fmt.Errorf("failed to find trust zone %s in local config", filter.GetTrustZoneId())
		}
		trustZones = []*trust_zone_proto.TrustZone{trustZone}
	} else {
		trustZones = lds.config.TrustZones
	}
	bindings := []*ap_binding_proto.APBinding{}
	for _, trustZone := range trustZones {
		// nolint:staticcheck
		for _, binding := range trustZone.AttestationPolicies {
			if filter != nil && filter.GetPolicyId() != "" && binding.GetPolicyId() != filter.GetPolicyId() {
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
	if federationProto.GetId() == "" {
		id, err := generateId()
		if err != nil {
			return nil, fmt.Errorf("failed to generate UUID for federation: %w", err)
		}
		federationProto.Id = id
	}
	fromTrustZone, ok := lds.config.GetTrustZoneByID(federationProto.GetTrustZoneId())
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", federationProto.GetTrustZoneId())
	}

	_, ok = lds.config.GetTrustZoneByID(federationProto.GetRemoteTrustZoneId())
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", federationProto.GetRemoteTrustZoneId())
	}

	if federationProto.GetTrustZoneId() == federationProto.GetRemoteTrustZoneId() {
		return nil, fmt.Errorf("cannot federate trust zone %s with itself", federationProto.GetTrustZoneId())
	}

	federations, err := lds.ListFederationsByTrustZone(federationProto.GetTrustZoneId())
	if err != nil {
		return nil, fmt.Errorf("failed to list federations for trust zone %s: %w", federationProto.GetTrustZoneId(), err)
	}
	for _, federation := range federations {
		if federation.GetTrustZoneId() == federationProto.GetTrustZoneId() && federation.GetRemoteTrustZoneId() == federationProto.GetRemoteTrustZoneId() {
			return nil, fmt.Errorf("federation already exists between %s and %s", federationProto.GetTrustZoneId(), federationProto.GetRemoteTrustZoneId())
		}
	}

	federationProto, err = proto.CloneFederation(federationProto)
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

func (lds *LocalDataSource) DestroyFederation(id string) error {
	for _, trustZone := range lds.config.TrustZones {
		// nolint:staticcheck
		for i, fed := range trustZone.Federations {
			if fed.GetId() == id {
				// nolint:staticcheck
				trustZone.Federations = append(trustZone.Federations[:i], trustZone.Federations[i+1:]...)
				if err := lds.updateDataFile(); err != nil {
					return fmt.Errorf("failed to remove federation from local config: %s", err)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("failed to find federation %s in local config", id)
}

func (lds *LocalDataSource) ListFederations(filter *datasourcepb.ListFederationsRequest_Filter) ([]*federation_proto.Federation, error) {
	// federations are expressed in-line with the trust zone(s) so we need to iterate the trust zones
	federations := []*federation_proto.Federation{}
	for _, trustZone := range lds.config.TrustZones {
		// nolint:staticcheck
		for _, federation := range trustZone.Federations {
			include := true
			if filter != nil {
				if filter.TrustZoneId != nil && federation.GetTrustZoneId() != filter.GetTrustZoneId() {
					include = false
				}
			}
			if include {
				federation, err := proto.CloneFederation(federation)
				if err != nil {
					return nil, err
				}
				federations = append(federations, federation)
			}
		}
	}
	return federations, nil
}

func (lds *LocalDataSource) ListFederationsByTrustZone(tzID string) ([]*federation_proto.Federation, error) {
	trustZone, ok := lds.config.GetTrustZoneByID(tzID)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", tzID)
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
