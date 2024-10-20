package local

import (
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin"
)

type YAMLConfigProvider struct {
	DataSource *plugin.LocalDataSource
}

func (ycp *YAMLConfigProvider) GetConfig() (*config.Config, error) {
	return ycp.DataSource.Config, nil
}

func (ycp *YAMLConfigProvider) GetPlugins() ([]string, error) {
	return nil, nil
}
