#DataSource: string

#TrustZone: {
	name!: string
	trust_domain!: string
	bundle_endpoint_url?: string
	bundle?: #Bundle
	federations: [...#Federation]
	attestation_policies: [...#APBinding]
	jwt_issuer?: string
	bundle_endpoint_profile?: #BundleEndpointProfile
}

#Bundle: {
	trust_domain?: string
	x509_authorities?: [...#X509Certificate]
	jwt_authorities?: [...#JWTKey]
	refresh_hint?: string
	sequence_number?: string
}

#X509Certificate: {
	asn1!: string
	tainted?: bool
}

#JWTKey: {
	public_key!: string
	key_id?: string
	expires_at?: string
	tainted?: bool
}

#Cluster: {
	name!: string
	trust_zone!: string
	kubernetes_context!: string
	trust_provider!: #TrustProvider
	profile!: string
	extra_helm_values?: #HelmValues
	external_server?: bool
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

#BundleEndpointProfile: string & =~"BUNDLE_ENDPOINT_PROFILE_.*"

#PluginConfig: {
	[string]: _
}

#Plugins: {
	data_source?: string
	provision?: string
}

#Config: {
	trust_zones: [...#TrustZone]
	clusters: [...#Cluster]
	attestation_policies: [...#AttestationPolicy]
	plugin_config?: #PluginConfig
	plugins!: #Plugins
}

#Config
