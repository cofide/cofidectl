package plugin

import (
	"context"

	"github.com/cofide/cofide-connect/pkg/api"
)

type DataSource interface {
	GetTrustZones() ([]*api.TrustZone, error)
}

type FileDataSource struct {
	FilePath string
}

func NewFileDataSource(filePath string) (*FileDataSource, error) {
	fds := &FileDataSource{
		FilePath: filePath,
	}
	if err := fds.loadState(); err != nil {
		return nil, err
	}
	return fds, nil
}

func (f *FileDataSource) loadState() error {
	// load file from disk
	return nil
}

func (f *FileDataSource) GetTrustZone(ctx context.Context) ([]*api.TrustZone, error) {
	trustzones := make([]*api.TrustZone, 0)
	return trustzones, nil
}
