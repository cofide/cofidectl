// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"buf.build/go/protoyaml"
	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	config_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/config/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	pluginspb "github.com/cofide/cofide-api-sdk/gen/go/proto/plugins/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"google.golang.org/protobuf/types/known/structpb"
)

// Config describes the cofide.yaml configuration file format.
type Config struct {
	TrustZones          []*trust_zone_proto.TrustZone
	Clusters            []*clusterpb.Cluster
	AttestationPolicies []*attestation_policy_proto.AttestationPolicy
	APBindings          []*ap_binding_proto.APBinding
	Federations         []*federation_proto.Federation
	PluginConfig        map[string]*structpb.Struct
	Plugins             *pluginspb.Plugins
}

func NewConfig() *Config {
	return &Config{
		TrustZones:          []*trust_zone_proto.TrustZone{},
		Clusters:            []*clusterpb.Cluster{},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		APBindings:          []*ap_binding_proto.APBinding{},
		Federations:         []*federation_proto.Federation{},
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
		Clusters:            proto.Clusters,
		AttestationPolicies: proto.AttestationPolicies,
		APBindings:          proto.ApBindings,
		Federations:         proto.Federations,
		PluginConfig:        proto.PluginConfig,
		Plugins:             plugins,
	}
}

func (c *Config) toProto() *config_proto.Config {
	return &config_proto.Config{
		TrustZones:          c.TrustZones,
		Clusters:            c.Clusters,
		AttestationPolicies: c.AttestationPolicies,
		ApBindings:          c.APBindings,
		Federations:         c.Federations,
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
		Clusters:            []*clusterpb.Cluster{},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		ApBindings:          []*ap_binding_proto.APBinding{},
		Federations:         []*federation_proto.Federation{},
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

func (c *Config) GetTrustZoneByID(id string) (*trust_zone_proto.TrustZone, bool) {
	for _, tz := range c.TrustZones {
		if tz.GetId() == id {
			return tz, true
		}
	}
	return nil, false
}

func (c *Config) GetClusterByName(name, trustZoneID string) (*clusterpb.Cluster, bool) {
	for _, cluster := range c.Clusters {
		if cluster.GetName() == name && cluster.GetTrustZoneId() == trustZoneID {
			return cluster, true
		}
	}
	return nil, false
}

func (c *Config) GetClusterByID(id string) (*clusterpb.Cluster, bool) {
	for _, cluster := range c.Clusters {
		if cluster.GetId() == id {
			return cluster, true
		}
	}
	return nil, false
}

func (c *Config) GetClustersByTrustZone(trustZoneID string) []*clusterpb.Cluster {
	clusters := []*clusterpb.Cluster{}
	for _, cluster := range c.Clusters {
		if cluster.GetTrustZoneId() == trustZoneID {
			clusters = append(clusters, cluster)
		}
	}
	return clusters
}

func (c *Config) GetAttestationPolicyByName(name string) (*attestation_policy_proto.AttestationPolicy, bool) {
	for _, ap := range c.AttestationPolicies {
		if ap.Name == name {
			return ap, true
		}
	}
	return nil, false
}

func (c *Config) GetAttestationPolicyByID(id string) (*attestation_policy_proto.AttestationPolicy, bool) {
	for _, ap := range c.AttestationPolicies {
		if ap.GetId() == id {
			return ap, true
		}
	}
	return nil, false
}
