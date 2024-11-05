// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package plugin

import go_plugin "github.com/hashicorp/go-plugin"

// HandshakeConfig is a common handshake that is shared by plugin and host.
var HandshakeConfig = go_plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "COFIDECTL_PLUGIN",
	MagicCookieValue: "config",
}
