package config

import (
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

// Config describes the cofide.yaml configuration file format.
type Config struct {
	Plugins             []string `yaml:"plugins,omitempty"`
	TrustZones          []*trust_zone_proto.TrustZone
	AttestationPolicies []*attestation_policy_proto.AttestationPolicy
}

func NewConfig() *Config {
	return &Config{
		Plugins:             []string{},
		TrustZones:          []*trust_zone_proto.TrustZone{},
		AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
	}
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
