package context

import (
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/pkg/plugin/manager"
)

type CommandContext struct {
	PluginManager *manager.PluginManager
	ConfigLoader  config.Loader
}
