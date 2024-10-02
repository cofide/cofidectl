package plugin

import (
	"context"

	"github.com/cofide/cofide-connect/pkg/api"
)

type DataSource interface {
	GetTrustZones() ([]*api.TrustZone, error)
}

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

func (lds *LocalDataSource) GetTrustZones(ctx context.Context) ([]*api.TrustZone, error) {
	trustzones := make([]*api.TrustZone, 0)
	return trustzones, nil
}
