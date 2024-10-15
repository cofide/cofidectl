package attestationpolicy

type AttestationPolicy struct {
	Kind AttestationPolicyKind
	Opts *AttestationPolicyOpts
}

type AttestationPolicyKind string

const (
	Annotated = "annotated"
	Namespace = "namespace"
)

type AttestationPolicyOpts struct {
	// Annotated
	PodKey   string
	PodValue string

	// Namespace
	Namespace string
}
