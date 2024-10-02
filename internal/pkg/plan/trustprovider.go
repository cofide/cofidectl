package plan

type TrustProvider struct {
	Name         string                    `yaml:"name"`
	Kind         string                    `yaml:"kind"`
	AgentConfig  TrustProviderAgentConfig  `yaml:"agentConfig"`
	ServerConfig TrustProviderServerConfig `yaml:"serverConfig"`
}

type TrustProviderAgentConfig struct {
	WorkloadAttestor        string                 `yaml:"workloadAttestor"`
	WorkloadAttestorEnabled bool                   `yaml:"workloadAttestorEnabled"`
	WorkloadAttestorConfig  map[string]interface{} `yaml:"workloadAttestorConfig"`
	NodeAttestor            string                 `yaml:"nodeAttestor"`
	NodeAttestorEnabled     bool                   `yaml:"nodeAttestorEnabled"`
}

type TrustProviderServerConfig struct {
	NodeAttestor        string                 `yaml:"nodeAttestor"`
	NodeAttestorEnabled bool                   `yaml:"nodeAttestorEnabled"`
	NodeAttestorConfig  map[string]interface{} `yaml:"nodeAttestorConfig"`
}
