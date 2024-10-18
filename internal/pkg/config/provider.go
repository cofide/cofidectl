package config

import (
	"github.com/cofide/cofidectl/internal/pkg/attestationpolicy"
	"github.com/cofide/cofidectl/internal/pkg/trustzone"
)

type ConfigProvider interface {
	GetConfig() (*Config, error)
	GetPlugins() ([]string, error)
}

type Config struct {
	Plugins             []string                                        `yaml:"plugins,omitempty"`
	TrustZones          map[string]*trustzone.TrustZone                 `yaml:"trust_zones,omitempty"`
	AttestationPolicies map[string]*attestationpolicy.AttestationPolicy `yaml:"attestation_policies,omitempty"`
}
