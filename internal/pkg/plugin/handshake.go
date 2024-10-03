package plugin

import go_plugin "github.com/hashicorp/go-plugin"

// Handshake is a common handshake that is shared by plugin and host.
var HandshakeConfig = go_plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "COFIDECTL_PLUGIN",
	MagicCookieValue: "config",
}
