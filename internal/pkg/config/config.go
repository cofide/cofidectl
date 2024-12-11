// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"buf.build/go/protoyaml"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	config_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/config/v1alpha1"
	pluginspb "github.com/cofide/cofide-api-sdk/gen/go/proto/plugins/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"google.golang.org/protobuf/types/known/structpb"
)

// Config describes the cofide.yaml configuration file format.
type Config struct {
	TrustZones          []*trust_zone_proto.TrustZone
	AttestationPolicies []*attestation_policy_proto.AttestationPolicy
	PluginConfig        map[string]*structpb.Struct
	Plugins             *pluginspb.Plugins
}

func NewConfig() *Config {
	return &Config{
		TrustZones:          []*trust_zone_proto.TrustZone{},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		PluginConfig:        map[string]*structpb.Struct{},
		Plugins:             &pluginspb.Plugins{},
	}
}

func newConfigFromProto(proto *config_proto.Config) *Config {
	plugins := proto.GetPlugins()
	if plugins == nil {
		plugins = &pluginspb.Plugins{}
	}
	return &Config{
		TrustZones:          proto.TrustZones,
		AttestationPolicies: proto.AttestationPolicies,
		PluginConfig:        proto.PluginConfig,
		Plugins:             plugins,
	}
}

func (c *Config) toProto() *config_proto.Config {
	return &config_proto.Config{
		TrustZones:          c.TrustZones,
		AttestationPolicies: c.AttestationPolicies,
		PluginConfig:        c.PluginConfig,
		Plugins:             c.Plugins,
	}
}

func (c *Config) marshalYAML() ([]byte, error) {
	// Convert the Config to the config_proto.Config message to allow marshalling with protoyaml.
	proto := c.toProto()
	options := protoyaml.MarshalOptions{UseProtoNames: true}
	return options.Marshal(proto)

}

func unmarshalYAML(data []byte) (*Config, error) {
	proto := config_proto.Config{
		TrustZones:          []*trust_zone_proto.TrustZone{},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		PluginConfig:        map[string]*structpb.Struct{},
	}
	err := protoyaml.Unmarshal(data, &proto)
	if err != nil {
		return nil, err
	}
	return newConfigFromProto(&proto), nil
}

func (c *Config) GetTrustZoneByName(name string) (*trust_zone_proto.TrustZone, bool) {
	for _, tz := range c.TrustZones {
		if tz.Name == name {
			return tz, true
		}
	}
	return nil, false
}

func (c *Config) GetAttestationPolicyByName(name string) (*attestation_policy_proto.AttestationPolicy, bool) {
	for _, ap := range c.AttestationPolicies {
		if ap.Name == name {
			return ap, true
		}
	}
	return nil, false
}
