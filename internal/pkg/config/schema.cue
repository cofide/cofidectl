#DataSource: string

#TrustZone: {
	id?: string
	name!: string
	trust_domain!: string
	bundle_endpoint_url?: string
	bundle?: #Bundle
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
	id?: string
	name!: string
	trust_zone?: string
	trust_zone_id!: string
	kubernetes_context!: string
	trust_provider!: #TrustProvider
	profile!: string
	extra_helm_values?: #HelmValues
	external_server?: bool
	oidc_issuer_url?: string
	oidc_issuer_ca_cert?: string
}

#TrustProvider: {
	name?: string
	kind!: string
}

#APBinding: {
	id?: string
	trust_zone_id!: string
	policy_id!: string
	federations: [...#APBFederation]
	federates_with: [...string]
}

#APBFederation: {
	trust_zone_id!: string
}

#AttestationPolicy: {
	id?: string
	name!: string
	#APKubernetes | #APStatic | #APTPMNode
}

#APKubernetes: {
	kubernetes?: {
		namespace_selector?: #APLabelSelector
		pod_selector?: #APLabelSelector
		spiffe_id_path_template?: string
		dns_name_templates?: [...string]
	}
}

#APStatic: {
	static?: {
		spiffe_id_path!: string
		parent_id_path!: string
		selectors!: [...#APSelector]
		dns_names?: [...string]
	}
}

#APTPMNode: {
	tpm_node?: {
		attestation!: #TPMAttestation
		selector_values!: [...string]
	}
}

#TPMAttestation: {
	ek_hash?: string
}

#APSelector: {
	type!: string
	value!: string
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
	id?: string
	trust_zone_id!: string
	remote_trust_zone_id!: string
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
	ap_bindings: [...#APBinding]
	federations: [...#Federation]
	plugin_config?: #PluginConfig
	plugins!: #Plugins
}

#Config
