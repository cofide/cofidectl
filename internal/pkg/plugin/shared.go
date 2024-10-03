package plugin

import (
	"github.com/hashicorp/go-plugin"
)

var PluginMap = map[string]plugin.Plugin{
	"data_source": &DataSourcePlugin{},
}
