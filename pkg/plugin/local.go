package plugin

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/trust_zone/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const schemaCue = `
#Plugins: {
	name: string
}
#TrustZone: {
	name: string
	trust_domain: string
}

#Config: {
	plugins: [...#TrustZone]
	trust_zones: [...#TrustZone]
}

config: #Config
`

type LocalDataSource struct {
	filePath    string
	plugins     []string
	trustZones  []*trust_zone_proto.TrustZone
	schemaValue cue.Value
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
	// load file from disk
	ctx := cuecontext.New()
	instances := load.Instances([]string{lds.filePath}, nil)
	if len(instances) == 0 {
		return fmt.Errorf("no Cue instances found")
	}

	dataValue := ctx.BuildInstance(instances[0])
	if dataValue.Err() != nil {
		return fmt.Errorf("error building Cue instance: %s", dataValue.Err())
	}

	schemaValue := ctx.CompileString(schemaCue)
	if schemaValue.Err() != nil {
		return fmt.Errorf("error compiling schema: %s", schemaValue.Err())
	}

	lds.schemaValue = schemaValue.Unify(dataValue)
	if lds.schemaValue.Err() != nil {
		return fmt.Errorf("error unifying schema and data: %s", lds.schemaValue.Err())
	}

	return nil
}

func (lds *LocalDataSource) getConfig(key string) (cue.Value, error) {
	value := lds.schemaValue.LookupPath(cue.ParsePath(fmt.Sprintf("config.%s", key)))
	return value, nil
}

func (lds *LocalDataSource) GetPlugins() ([]string, error) {
	pluginValues, err := lds.getConfig("plugins")
	if err != nil {
		return nil, err
	}

	err = pluginValues.Decode(&lds.plugins)
	if err != nil {
		return nil, err
	}
	return lds.plugins, nil
}

func (lds *LocalDataSource) GetTrustZones() ([]*trust_zone_proto.TrustZone, error) {
	trustZoneValues, err := lds.getConfig("trust_zones")
	if err != nil {
		return nil, err
	}

	jsonBytes, err := trustZoneValues.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error marshaling to JSON: %s", err)
	}

	// unmarshal JSON
	var rawTrustZones []map[string]interface{}
	err = json.Unmarshal(jsonBytes, &rawTrustZones)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling to raw: %s", err)
	}

	for _, rawTrustZone := range rawTrustZones {
		trustZoneJson, err := json.Marshal(rawTrustZone)
		if err != nil {
			slog.Error("error marshaling individual trust zone", "error", err)
			continue
		}

		trustZone := &trust_zone_proto.TrustZone{}
		err = protojson.Unmarshal(trustZoneJson, trustZone)
		if err != nil {
			slog.Error("error unmarshaling to protocol buffer", "error", err)
			continue
		}

		lds.trustZones = append(lds.trustZones, trustZone)
	}

	return lds.trustZones, nil
}
