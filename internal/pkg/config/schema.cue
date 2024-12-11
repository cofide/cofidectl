#DataSource: string

#TrustZone: {
	name!: string
	trust_domain!: string
	kubernetes_cluster!: string
	kubernetes_context!: string
	trust_provider!: #TrustProvider
	profile!: string
	bundle_endpoint_url?: string
	bundle?: string
	federations: [...#Federation]
	attestation_policies: [...#APBinding]
	jwt_issuer?: string
	extra_helm_values?: #HelmValues
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
	from!: string
	to!: string
}

#HelmValues: {
	[string]: _
}

#PluginConfig: {
	[string]: _
}

#Config: {
	data_source!: #DataSource
	trust_zones: [...#TrustZone]
	attestation_policies: [...#AttestationPolicy]
	plugin_config?: #PluginConfig
}

#Config
