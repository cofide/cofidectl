package plugin

import (
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
)

// TODO: use Go embedding eg //go:embed cofidectl-schema.cue
const schemaCue = `
#Plugins: {
	name: string
}

#TrustZone: {
	name: string
	trust_domain: string
}

#AttestationPolicy: {
	trust_zone: string
	namespace: string
	pod_annotation_key: string
	pod_annotation_value: string
}


#Config: {
	plugins: [...#Plugins]
	trust_zones: [...#TrustZone]
	attestation_policy: [...#AttestationPolicy]
}

config: #Config
`

type Config struct {
	Plugins           []string                                      `yaml:"plugins"`
	TrustZones        []*trust_zone_proto.TrustZone                 `yaml:"trust_zones"`
	AttestationPolicy []*attestation_policy_proto.AttestationPolicy `yaml:"attestation_policy"`
	Federations       []*federation_proto.Federation                `yaml:"federations"`
}

type LocalDataSource struct {
	filePath   string
	config     Config
	cueContext *cue.Context
}

func NewLocalDataSource(filePath string) (*LocalDataSource, error) {
	lds := &LocalDataSource{
		filePath: filePath,
	}
	if err := lds.loadState(); err != nil {
		return nil, err
	}
	return lds, nil
}

func (lds *LocalDataSource) loadState() error {
	// load YAML file from disk
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

func (lds *LocalDataSource) GetConfig() (Config, error) {
	return lds.config, nil
}

func (lds *LocalDataSource) GetPlugins() ([]string, error) {
	return lds.config.Plugins, nil
}

func (lds *LocalDataSource) AddTrustZone(trustZone *trust_zone_proto.TrustZone) error {
	lds.config.TrustZones = append(lds.config.TrustZones, trustZone)
	if err := lds.UpdateDataFile(); err != nil {
		return fmt.Errorf("failed to add trust zone %s to local config: %s", trustZone.TrustDomain, err)
	}
	return nil
}

func (lds *LocalDataSource) AddAttestationPolicy(policy *attestation_policy_proto.AttestationPolicy) error {
	lds.config.AttestationPolicy = append(lds.config.AttestationPolicy, policy)
	if err := lds.UpdateDataFile(); err != nil {
		return fmt.Errorf("failed to add attestation policy to local config: %s", err)
	}
	return nil
}

func (lds *LocalDataSource) AddFederation(federation *federation_proto.Federation) error {
	lds.config.Federations = append(lds.config.Federations, federation)
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

	slog.Info("Successfully added new trust zone", "trust_zone", lds.filePath)

	return nil
}

func (lds *LocalDataSource) ListTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	return lds.config.TrustZones, nil
}

func (lds *LocalDataSource) ListAttestationPolicy() ([]*attestation_policy_proto.AttestationPolicy, error) {
	return lds.config.AttestationPolicy, nil
}

func (lds *LocalDataSource) ListFederation() ([]*federation_proto.Federation, error) {
	return lds.config.Federations, nil
}
