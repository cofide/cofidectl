package plugin

import (
	"context"

	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

type LocalDataSource struct {
	FilePath string
}

func NewLocalDataSource(filePath string) (*LocalDataSource, error) {
	lds := &LocalDataSource{
		FilePath: filePath,
	}
	if err := lds.loadState(); err != nil {
		return nil, err
	}
	return lds, nil
}

func (lds *LocalDataSource) loadState() error {
	// load file from disk
	return nil
}

func (lds *LocalDataSource) ListTrustZones(ctx context.Context) ([]*trust_zone_proto.TrustZone, error) {
	trustzones := make([]*trust_zone_proto.TrustZone, 0)
	return trustzones, nil
}
