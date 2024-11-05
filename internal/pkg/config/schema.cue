#Plugin: string

#TrustZone: {
	name!: string
	trust_domain!: string
	kubernetes_cluster!: string
	kubernetes_context!: string
	trust_provider!: #TrustProvider
	bundle_endpoint_url?: string
	bundle?: string
	federations: [...#Federation]
	attestation_policies: [...#APBinding]
}

#TrustProvider: {
	name?: string
	kind!: string
}

#APBinding: {
	trust_zone!: string
	policy!: string
	federates_with: [...string]
}

#AttestationPolicy: {
	name!: string
	#APKubernetes
}

#APKubernetes: {
	kubernetes!: {
		namespace_selector?: #APLabelSelector
		pod_selector?: #APLabelSelector
	}
}

#APLabelSelector: {
	match_labels?:
		[string]: string
	match_expressions?: [...#APMatchExpression]
}

#APMatchExpression: {
	key!: string
	operator!: string
	values: [...string]
}

#Federation: {
	left!: string
	right!: string
}

#Config: {
	plugins: [...#Plugin]
	trust_zones: [...#TrustZone]
	attestation_policies: [...#AttestationPolicy]
}

#Config
