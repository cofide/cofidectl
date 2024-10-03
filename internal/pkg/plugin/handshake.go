package plugin

import "github.com/hashicorp/go-plugin"

// Handshake is a common handshake that is shared by plugin and host.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "COFIDECTL_PLUGIN",
	MagicCookieValue: "config",
}
