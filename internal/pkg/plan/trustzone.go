package plan

type TrustZone struct {
	// Name is the name of the TrustZone
	Name string `yaml:"name"`
	// Domain is the trust domain of this TrustZone
	Domain string `yaml:"domain"`
	// Cluster is the cluster this TrustZone is deployed to
	Cluster string `yaml:"cluster"`
	// Context is the Kubernetes context of the Cluster
	Context string `yaml:"context"`

	TrustProvider *TrustProvider `yaml:"trustProvider"`

	AttestationPolicies []AttestationPolicy `yaml:"attestationPolicies"`
}
