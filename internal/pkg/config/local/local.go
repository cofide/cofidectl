package local

import (
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin"
)

type YAMLConfigProvider struct {
	DataSource plugin.DataSource
}

func (ycp *YAMLConfigProvider) GetConfig() (*config.Config, error) {
	return nil, nil
}

func (ycp *YAMLConfigProvider) GetPlugins() ([]string, error) {
	return nil, nil
}
