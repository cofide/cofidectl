package plugin

import (
<<<<<<< HEAD
=======
	"encoding/json"
>>>>>>> aec3d398f6ccc64a548bb60de1698cf2a1c2eda0
	"fmt"
	"log/slog"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
<<<<<<< HEAD
	cue_yaml "cuelang.org/go/encoding/yaml"

	"gopkg.in/yaml.v3"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
=======
	"cuelang.org/go/cue/load"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
	"google.golang.org/protobuf/encoding/protojson"
>>>>>>> aec3d398f6ccc64a548bb60de1698cf2a1c2eda0
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

#Config: {
	plugins: [...#Plugins]
	trust_zones: [...#TrustZone]
}

config: #Config
`

type Config struct {
	Plugins     []string                      `yaml:"plugins"`
	Trust_Zones []*trust_zone_proto.TrustZone `yaml:"trust_zones"`
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

	//slog.Info("Cofide configuration has been successfully validated")

	return nil
}

func (lds *LocalDataSource) GetConfig() (Config, error) {
	return lds.config, nil
}

func (lds *LocalDataSource) GetPlugins() ([]string, error) {
	return lds.config.Plugins, nil
}

func (lds *LocalDataSource) AddTrustZone(trustZone *trust_zone_proto.TrustZone) error {
	lds.config.Trust_Zones = append(lds.config.Trust_Zones, trustZone)
	if err := lds.UpdateDataFile(); err != nil {
		return fmt.Errorf("failed to add trust zone %s to local config: %s", trustZone.TrustDomain, err)
	}
	//slog.Info("Successfully updated local config", "trust_zone", trustZone.TrustDomain)
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
	return lds.config.Trust_Zones, nil
}
