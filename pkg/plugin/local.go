package plugin

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cue_yaml "cuelang.org/go/encoding/yaml"

	"gopkg.in/yaml.v3"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/proto"
)

// TODO: use Go embedding eg //go:embed cofidectl-schema.cue
const schemaCue = `
#Plugins: {
	name: string
}

#TrustZone: {
	name: string
	trust_domain: string
	attestation_policies: [...string]
}

#AttestationPolicy: {
	name: string
	kind: string
	namespace: string
	pod_key: string
	pod_value: string
}

#Federation: {
	left: string
	right: string
}

#Config: {
	plugins: [...#Plugins]
	trust_zones: [...#TrustZone]
	attestation_policies: [...#AttestationPolicy]
	federation: [...#Federation]
}

config: #Config
`

type LocalDataSource struct {
	filePath   string
	Config     *config.Config
	cueContext *cue.Context
	loaded     bool
}

func (lds *LocalDataSource) Init() error {
	if !lds.loaded {
		fmt.Printf("initialising local config file: %v\n", lds.filePath)
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

func NewLocalDataSource(filePath string) (*LocalDataSource, error) {
	trustZones := &trust_zone_proto.TrustZoneList{}
	attestationPolicies := &attestation_policy_proto.AttestationPolicyList{}

	cfg := &config.Config{TrustZones: trustZones, AttestationPolicies: attestationPolicies}
	lds := &LocalDataSource{
		filePath: filePath,
		Config:   cfg,
	}

	dataFileExists, err := lds.dataFileExists()
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

func (lds *LocalDataSource) loadState() error {
	dataFileExists, err := lds.dataFileExists()
	if err != nil {
		return err
	}

	if !dataFileExists {
		return fmt.Errorf("the config file doesn't exist. Please run cofidectl init")
	}

	// load YAML config file from disk
	yamlData, err := os.ReadFile(lds.filePath)
	if err != nil {
		return fmt.Errorf("error reading YAML file: %s", err)
	}

	lds.cueContext = cuecontext.New()

	// validate the YAML using the Cue schema
	schema := lds.cueContext.CompileString(schemaCue)
	if schema.Err() != nil {
		return fmt.Errorf("error compiling Cue schema: %s", err)
	}

	if err = cue_yaml.Validate(yamlData, schema); err != nil {
		return fmt.Errorf("error validating YAML: %s", err)
	}

	if err := yaml.Unmarshal(yamlData, &lds.Config); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %s", err)
	}

	lds.loaded = true
	return nil
}

func (lds *LocalDataSource) createDataFile() error {
	err := os.WriteFile(lds.filePath, []byte{}, 0600)
	if err != nil {
		return fmt.Errorf("error creating config: %v", err)
	}

	return nil
}

func (lds *LocalDataSource) updateDataFile() error {
	if !lds.loaded {
		return fmt.Errorf("the config file doesn't exist. Please run cofidectl init")
	}

	data, err := yaml.Marshal(lds.Config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %v", err)
	}

	err = os.WriteFile(lds.filePath, data, 0600)
	if err != nil {
		return fmt.Errorf("error updating config: %v", err)
	}

	return nil
}

func (lds *LocalDataSource) dataFileExists() (bool, error) {
	if _, err := os.Stat(lds.filePath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func (lds *LocalDataSource) AddTrustZone(trustZone *trust_zone_proto.TrustZone) error {
	if _, ok := lds.Config.GetTrustZoneByName(trustZone.Name); ok {
		return fmt.Errorf("trust zone %s already exists in local config", trustZone.Name)
	}
	trustZone = proto.CloneTrustZone(trustZone)
	lds.Config.TrustZones.TrustZones = append(lds.Config.TrustZones.TrustZones, trustZone)
	if err := lds.updateDataFile(); err != nil {
		return fmt.Errorf("failed to add trust zone %s to local config: %s", trustZone.TrustDomain, err)
	}
	return nil
}

func (lds *LocalDataSource) GetTrustZone(id string) (*trust_zone_proto.TrustZone, error) {
	trustZone, ok := lds.Config.GetTrustZoneByName(id)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", id)
	}

	trustZone = proto.CloneTrustZone(trustZone)
	return trustZone, nil
}

func (lds *LocalDataSource) UpdateTrustZone(trustZone *trust_zone_proto.TrustZone) error {
	for i, current := range lds.Config.TrustZones.TrustZones {
		if current.Name == trustZone.Name {
			if err := validateTrustZoneUpdate(current, trustZone); err != nil {
				return err
			}

			lds.Config.TrustZones.TrustZones[i] = proto.CloneTrustZone(trustZone)

			if err := lds.updateDataFile(); err != nil {
				return fmt.Errorf("failed to update trust zone %s in local config: %s", trustZone.TrustDomain, err)
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
	if err := validateTrustProviderUpdate(current.TrustProvider, new.TrustProvider); err != nil {
		return err
	}
	// The following should be updated though other means.
	if !slices.EqualFunc(new.Federations, current.Federations, proto.FederationsEqual) {
		return fmt.Errorf("cannot update federations for existing trust zone %s", current.Name)
	}
	if !slices.EqualFunc(new.AttestationPolicies, current.AttestationPolicies, proto.AttestationPoliciesEqual) {
		return fmt.Errorf("cannot update attestation policies for existing trust zone %s", current.Name)
	}
	return nil
}

func validateTrustProviderUpdate(current, new *trust_provider_proto.TrustProvider) error {
	if new.Name != current.Name {
		return fmt.Errorf("cannot update trust provider name for existing trust zone %s", current.Name)
	}
	if new.Kind != current.Kind {
		return fmt.Errorf("cannot update trust provider kind for existing trust zone %s", current.Name)
	}
	return nil
}

func (lds *LocalDataSource) AddAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) error {
	if _, ok := lds.Config.GetAttestationPolicyByName(policy.Name); ok {
		return fmt.Errorf("attestation policy %s already exists in local config", policy.Name)
	}
	policy = proto.CloneAttestationPolicy(policy)
	lds.Config.AttestationPolicies.Policies = append(lds.Config.AttestationPolicies.Policies, policy)
	if err := lds.updateDataFile(); err != nil {
		return fmt.Errorf("failed to add attestation policy to local config: %s", err)
	}
	return nil
}

func (lds *LocalDataSource) BindAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy, trustZone *trust_zone_proto.TrustZone) error {
	localTrustZone, ok := lds.Config.GetTrustZoneByName(trustZone.Name)
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", trustZone.Name)
	}

	if _, ok := lds.Config.GetAttestationPolicyByName(policy.Name); !ok {
		return fmt.Errorf("attestation policy %s does not exist in local config", policy.Name)
	}

	for _, ap := range localTrustZone.AttestationPolicies {
		if ap.Name == policy.Name {
			return fmt.Errorf("attestation policy %s is already bound to trust zone %s", policy.Name, trustZone.Name)
		}
	}

	policy = proto.CloneAttestationPolicy(policy)
	localTrustZone.AttestationPolicies = append(localTrustZone.AttestationPolicies, policy)
	if err := lds.updateDataFile(); err != nil {
		return fmt.Errorf("failed to add attestation policy to local config: %w", err)
	}
	return nil
}

func (lds *LocalDataSource) GetAttestationPolicy(id string) (*attestation_policy_proto.AttestationPolicy, error) {
	if policy, ok := lds.Config.GetAttestationPolicyByName(id); ok {
		policy = proto.CloneAttestationPolicy(policy)
		return policy, nil
	} else {
		return nil, fmt.Errorf("failed to find attestation policy %s in local config", id)
	}
}

func (lds *LocalDataSource) AddFederation(federationProto *federation_proto.Federation) error {
	leftTrustZone, ok := lds.Config.GetTrustZoneByName(federationProto.Left)
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", federationProto.Left)
	}

	_, ok = lds.Config.GetTrustZoneByName(federationProto.Right)
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", federationProto.Right)
	}

	for _, federation := range leftTrustZone.Federations {
		if federation.Right == federationProto.Right {
			return fmt.Errorf("federation already exists between %s and %s", federationProto.Left, federationProto.Right)
		}
	}

	federationProto = proto.CloneFederation(federationProto)
	leftTrustZone.Federations = append(leftTrustZone.Federations, federationProto)

	if err := lds.updateDataFile(); err != nil {
		return fmt.Errorf("failed to add federation to local config: %s", err)
	}
	return nil
}

func (lds *LocalDataSource) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	var trustZones []*trust_zone_proto.TrustZone
	for _, trustZone := range lds.Config.TrustZones.TrustZones {
		trustZones = append(trustZones, proto.CloneTrustZone(trustZone))
	}
	return trustZones, nil
}

func (lds *LocalDataSource) ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error) {
	var policies []*attestation_policy_proto.AttestationPolicy
	for _, policy := range lds.Config.AttestationPolicies.Policies {
		policies = append(policies, proto.CloneAttestationPolicy(policy))
	}
	return policies, nil
}

func (lds *LocalDataSource) ListFederations() ([]*federation_proto.Federation, error) {
	// federations are expressed in-line with the trust zone(s) so we need to iterate the trust zones
	var federations []*federation_proto.Federation
	for _, trustZone := range lds.Config.TrustZones.TrustZones {
		for _, federation := range trustZone.Federations {
			federation = proto.CloneFederation(federation)
			federations = append(federations, federation)
		}
	}
	return federations, nil
}

func (lds *LocalDataSource) ListFederationsByTrustZone(tzName string) ([]*federation_proto.Federation, error) {
	trustZone, ok := lds.Config.GetTrustZoneByName(tzName)
	if !ok {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", trustZone.TrustDomain)
	}

	var federations []*federation_proto.Federation
	for _, federation := range trustZone.Federations {
		federation = proto.CloneFederation(federation)
		federations = append(federations, federation)
	}
	return federations, nil
}
