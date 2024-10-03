package plugin

import (
	go_plugin "github.com/hashicorp/go-plugin"
)

func main() {
	go_plugin.Serve(&go_plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		GRPCServer:      go_plugin.DefaultGRPCServer,
	})
}
