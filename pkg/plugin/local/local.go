// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"fmt"
	"slices"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/proto/ap_binding/v1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/proto"
)

type LocalDataSource struct {
	loader config.Loader
	config *config.Config
	loaded bool
}

func NewLocalDataSource(loader config.Loader) (*LocalDataSource, error) {
	cfg := config.NewConfig()
	lds := &LocalDataSource{
		loader: loader,
		config: cfg,
	}

	dataFileExists, err := lds.loader.Exists()
	if err != nil {
		return nil, err
	}

	if dataFileExists {
		err = lds.loadState()
		if err != nil {
			return nil, err
		}
	}

	return lds, nil
}

func (lds *LocalDataSource) Init() error {
	if !lds.loaded {
		if err := lds.createDataFile(); err != nil {
			return err
		}
	} else {
		fmt.Println("the config file already exists.")
	}

	if err := lds.loadState(); err != nil {
		return err
	}

	return nil
}

func (lds *LocalDataSource) Validate() error {
	if !lds.loaded {
		return fmt.Errorf("the config file doesn't exist. Please run cofidectl init")
	}

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
	lds.loaded = true
	return nil
}

func (lds *LocalDataSource) createDataFile() error {
	fmt.Println("initialising local config file")
	return lds.loader.Write(config.NewConfig())
}

func (lds *LocalDataSource) updateDataFile() error {
	if !lds.loaded {
		return fmt.Errorf("the config file doesn't exist. Please run cofidectl init")
	}

	return lds.loader.Write(lds.config)
}

func (lds *LocalDataSource) CreateTrustZone(trustZone *trust_zone_proto.TrustZone) (*trust_zone_proto.TrustZone, error) {
	if _, ok := lds.config.GetTrustZoneByName(trustZone.Name); ok {
		return nil, fmt.Errorf("trust zone %s already exists in local config", trustZone.Name)
	}
	trustZone, err := proto.CloneTrustZone(trustZone)
	if err != nil {
		return nil, err
	}

	lds.config.TrustZones = append(lds.config.TrustZones, trustZone)
	if err := lds.updateDataFile(); err != nil {
		return nil, fmt.Errorf("failed to add trust zone %s to local config: %s", trustZone.Name, err)
	}
	return trustZone, nil
}

func (lds *LocalDataSource) GetTrustZone(id string) (*trust_zone_proto.TrustZone, error) {
	trustZone, ok := lds.config.GetTrustZoneByName(id)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", id)
	}

	return proto.CloneTrustZone(trustZone)
}

func (lds *LocalDataSource) UpdateTrustZone(trustZone *trust_zone_proto.TrustZone) error {
	for i, current := range lds.config.TrustZones {
		if current.Name == trustZone.Name {
			if err := validateTrustZoneUpdate(current, trustZone); err != nil {
				return err
			}

			trustZone, err := proto.CloneTrustZone(trustZone)
			if err != nil {
				return err
			}

			lds.config.TrustZones[i] = trustZone

			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to update trust zone %s in local config: %s", trustZone.Name, err)
			}

			return nil
		}
	}

	return fmt.Errorf("failed to find trust zone %s in local config", trustZone.Name)
}

func validateTrustZoneUpdate(current, new *trust_zone_proto.TrustZone) error {
	if new.Name != current.Name {
		return fmt.Errorf("cannot update name for existing trust zone %s", current.Name)
	}
	if new.TrustDomain != current.TrustDomain {
		return fmt.Errorf("cannot update trust domain for existing trust zone %s", current.Name)
	}
	if err := validateTrustProviderUpdate(current.Name, current.TrustProvider, new.TrustProvider); err != nil {
		return err
	}
	// The following should be updated though other means.
	if !slices.EqualFunc(new.Federations, current.Federations, proto.FederationsEqual) {
		return fmt.Errorf("cannot update federations for existing trust zone %s", current.Name)
	}
	if !slices.EqualFunc(new.AttestationPolicies, current.AttestationPolicies, proto.APBindingsEqual) {
		return fmt.Errorf("cannot update attestation policies for existing trust zone %s", current.Name)
	}
	return nil
}

func validateTrustProviderUpdate(tzName string, current, new *trust_provider_proto.TrustProvider) error {
	if new.Kind != current.Kind {
		return fmt.Errorf("cannot update trust provider kind for existing trust zone %s", tzName)
	}
	return nil
}

func (lds *LocalDataSource) AddAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) error {
	if _, ok := lds.config.GetAttestationPolicyByName(policy.Name); ok {
		return fmt.Errorf("attestation policy %s already exists in local config", policy.Name)
	}
	policy, err := proto.CloneAttestationPolicy(policy)
	if err != nil {
		return err
	}
	lds.config.AttestationPolicies = append(lds.config.AttestationPolicies, policy)
	if err := lds.updateDataFile(); err != nil {
		return fmt.Errorf("failed to add attestation policy to local config: %s", err)
	}
	return nil
}

func (lds *LocalDataSource) GetAttestationPolicy(id string) (*attestation_policy_proto.AttestationPolicy, error) {
	if policy, ok := lds.config.GetAttestationPolicyByName(id); ok {
		return proto.CloneAttestationPolicy(policy)
	} else {
		return nil, fmt.Errorf("failed to find attestation policy %s in local config", id)
	}
}

func (lds *LocalDataSource) AddAPBinding(binding *ap_binding_proto.APBinding) error {
	localTrustZone, ok := lds.config.GetTrustZoneByName(binding.TrustZone)
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", binding.TrustZone)
	}

	_, ok = lds.config.GetAttestationPolicyByName(binding.Policy)
	if !ok {
		return fmt.Errorf("attestation policy %s does not exist in local config", binding.Policy)
	}

	for _, apb := range localTrustZone.AttestationPolicies {
		if apb.Policy == binding.Policy {
			return fmt.Errorf("attestation policy %s is already bound to trust zone %s", binding.Policy, binding.TrustZone)
		}
	}

	remoteTzs := map[string]bool{}
	for _, federation := range localTrustZone.Federations {
		remoteTzs[federation.To] = true
	}
	for _, remoteTz := range binding.FederatesWith {
		if remoteTz == binding.TrustZone {
			// Is this a problem?
			return fmt.Errorf("attestation policy %s federates with its own trust zone %s", binding.Policy, binding.TrustZone)
		}
		if _, ok := remoteTzs[remoteTz]; !ok {
			if _, ok := lds.config.GetTrustZoneByName(remoteTz); !ok {
				return fmt.Errorf("attestation policy %s federates with unknown trust zone %s", binding.Policy, remoteTz)
			} else {
				return fmt.Errorf("attestation policy %s federates with %s but trust zone %s does not", binding.Policy, remoteTz, binding.TrustZone)
			}
		}
	}

	binding, err := proto.CloneAPBinding(binding)
	if err != nil {
		return err
	}
	localTrustZone.AttestationPolicies = append(localTrustZone.AttestationPolicies, binding)
	if err := lds.updateDataFile(); err != nil {
		return fmt.Errorf("failed to add attestation policy to local config: %w", err)
	}
	return nil
}

func (lds *LocalDataSource) ListAPBindingsByTrustZone(name string) ([]*ap_binding_proto.APBinding, error) {
	trustZone, ok := lds.config.GetTrustZoneByName(name)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", name)
	}

	var bindings []*ap_binding_proto.APBinding
	for _, binding := range trustZone.AttestationPolicies {
		binding, err := proto.CloneAPBinding(binding)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}
	return bindings, nil
}

func (lds *LocalDataSource) AddFederation(federationProto *federation_proto.Federation) error {
	fromTrustZone, ok := lds.config.GetTrustZoneByName(federationProto.From)
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", federationProto.From)
	}

	_, ok = lds.config.GetTrustZoneByName(federationProto.To)
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", federationProto.To)
	}

	if federationProto.From == federationProto.To {
		return fmt.Errorf("cannot federate trust zone %s with itself", federationProto.From)
	}

	for _, federation := range fromTrustZone.Federations {
		if federation.To == federationProto.To {
			return fmt.Errorf("federation already exists between %s and %s", federationProto.From, federationProto.To)
		}
	}

	federationProto, err := proto.CloneFederation(federationProto)
	if err != nil {
		return err
	}

	fromTrustZone.Federations = append(fromTrustZone.Federations, federationProto)

	if err := lds.updateDataFile(); err != nil {
		return fmt.Errorf("failed to add federation to local config: %s", err)
	}
	return nil
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

func (lds *LocalDataSource) ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error) {
	var policies []*attestation_policy_proto.AttestationPolicy
	for _, policy := range lds.config.AttestationPolicies {
		policy, err := proto.CloneAttestationPolicy(policy)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

func (lds *LocalDataSource) ListFederations() ([]*federation_proto.Federation, error) {
	// federations are expressed in-line with the trust zone(s) so we need to iterate the trust zones
	var federations []*federation_proto.Federation
	for _, trustZone := range lds.config.TrustZones {
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
		return nil, fmt.Errorf("failed to find trust zone %s in local config", trustZone.Name)
	}

	var federations []*federation_proto.Federation
	for _, federation := range trustZone.Federations {
		federation, err := proto.CloneFederation(federation)
		if err != nil {
			return nil, err
		}
		federations = append(federations, federation)
	}
	return federations, nil
}
