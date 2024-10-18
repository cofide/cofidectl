package plugin

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cue_yaml "cuelang.org/go/encoding/yaml"

	"gopkg.in/yaml.v3"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
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
	config     *config.Config
	cueContext *cue.Context
}

func NewLocalDataSource(filePath string) (*LocalDataSource, error) {
	trustZones := make(map[string]*trustzone.TrustZone)
	attestationPolicies := make(map[string]*attestationpolicy.AttestationPolicy)

	cfg := &config.Config{TrustZones: trustZones, AttestationPolicies: attestationPolicies}
	lds := &LocalDataSource{
		filePath: filePath,
		config:   cfg,
	}
	if err := lds.loadState(); err != nil {
		return nil, err
	}

	return lds, nil
}

func (lds *LocalDataSource) loadState() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error determining current working directory: %s", "error")
	}
	cfgFile := cwd + "/cofide.yaml"
	// check to see if the configuration file exists
	if _, err := os.Stat(lds.filePath); errors.Is(err, os.ErrNotExist) {
		// no existing configuration here
		slog.Info("initialising Cofide configuration", "config_file", cfgFile)
		return lds.UpdateDataFile()
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

	if err := yaml.Unmarshal(yamlData, &lds.config); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %s", err)
	}

	return nil
}

func (lds *LocalDataSource) AddTrustZone(trustZone *trust_zone_proto.TrustZone) error {
	if _, ok := lds.config.TrustZones[trustZone.Name]; ok {
		return fmt.Errorf("trust zone %s already exists in local config", trustZone.Name)
	}
	lds.config.TrustZones[trustZone.Name] = trustzone.NewTrustZone(trustZone)
	if err := lds.UpdateDataFile(); err != nil {
		return fmt.Errorf("failed to add trust zone %s to local config: %s", trustZone.TrustDomain, err)
	}
	return nil
}

func (lds *LocalDataSource) GetTrustZone(id string) (*trust_zone_proto.TrustZone, error) {
	var trustZone *trust_zone_proto.TrustZone

	if tz, ok := lds.config.TrustZones[id]; ok {
		trustZone = tz.TrustZoneProto
	} else {
		return nil, fmt.Errorf("failed to find trust zone %s in local config", id)
	}

	return trustZone, nil
}

func (lds *LocalDataSource) AddAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) error {
	if _, ok := lds.config.AttestationPolicies[policy.Name]; ok {
		return fmt.Errorf("attestation policy %s already exists in local config", policy.Name)
	}
	lds.config.AttestationPolicies[policy.Name] = attestationpolicy.NewAttestationPolicy(policy)
	if err := lds.UpdateDataFile(); err != nil {
		return fmt.Errorf("failed to add attestation policy to local config: %s", err)
	}
	return nil
}

func (lds *LocalDataSource) BindAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy, trustZone *trust_zone_proto.TrustZone) error {
	localTrustZone, ok := lds.config.TrustZones[trustZone.Name]
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", trustZone.Name)
	}

	if _, ok := lds.config.AttestationPolicies[policy.Name]; !ok {
		return fmt.Errorf("attestation policy %s does not exist in local config", policy.Name)
	}

	localTrustZone.AttestationPolicies = append(localTrustZone.AttestationPolicies, policy.Name)
	if err := lds.UpdateDataFile(); err != nil {
		return fmt.Errorf("failed to add attestation policy to local config: %w", err)
	}
	return nil
}

func (lds *LocalDataSource) GetAttestationPolicy(id string) (*attestation_policy_proto.AttestationPolicy, error) {
	var attestationPolicy *attestation_policy_proto.AttestationPolicy

	if ap, ok := lds.config.AttestationPolicies[id]; ok {
		attestationPolicy = ap.AttestationPolicyProto
		return attestationPolicy, nil
	} else {
		return nil, fmt.Errorf("failed to find attestation policy %s in local config", id)
	}
}

func (lds *LocalDataSource) AddFederation(federation *federation_proto.Federation) error {
	leftTrustZone, ok := lds.config.TrustZones[federation.Left.Name]
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", federation.Left.Name)
	}

	_, ok = lds.config.TrustZones[federation.Right.Name]
	if !ok {
		return fmt.Errorf("failed to find trust zone %s in local config", federation.Right.Name)
	}

	leftTrustZone.Federations = append(leftTrustZone.Federations, federation.Right.Name)
	if err := lds.UpdateDataFile(); err != nil {
		return fmt.Errorf("failed to add federation to local config: %s", err)
	}
	return nil
}

func (lds *LocalDataSource) UpdateDataFile() error {
	data, err := yaml.Marshal(lds.config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %v", err)
	}
	os.WriteFile(lds.filePath, data, 0644)

	return nil
}

func (lds *LocalDataSource) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	trustZoneAsProtos := make([]*trust_zone_proto.TrustZone, 0, len(lds.config.TrustZones))
	for _, trustZone := range lds.config.TrustZones {
		trustZoneAsProtos = append(trustZoneAsProtos, trustZone.TrustZoneProto)
	}
	return trustZoneAsProtos, nil
}

func (lds *LocalDataSource) ListAttestationPolicies() ([]*attestation_policy_proto.AttestationPolicy, error) {
	attestationPoliciesAsProtos := make([]*attestation_policy_proto.AttestationPolicy, 0, len(lds.config.AttestationPolicies))
	for _, attestationPolicy := range lds.config.AttestationPolicies {
		attestationPoliciesAsProtos = append(attestationPoliciesAsProtos, attestationPolicy.AttestationPolicyProto)
	}

	return attestationPoliciesAsProtos, nil
}

func (lds *LocalDataSource) ListFederation() ([]*federation_proto.Federation, error) {
	// federations are expressed in-line with the trust zone(s) so we need to iterate the trust zones
	federationsAsProto := make([]*federation_proto.Federation, 0)
	for _, trustZone := range lds.config.TrustZones {
		for _, federation := range trustZone.Federations {
			rightTrustZone, err := lds.GetTrustZone(federation)
			if err != nil {
				return nil, err
			}
			federationsAsProto = append(federationsAsProto, &federation_proto.Federation{Left: trustZone.TrustZoneProto, Right: rightTrustZone})
		}
	}
	return federationsAsProto, nil
}
