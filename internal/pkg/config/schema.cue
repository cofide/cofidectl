#Plugin: string

#TrustZone: {
	name: string
	trustdomain: string
	kubernetescluster: string
	kubernetescontext: string
	trustprovider: #TrustProvider
	bundleendpointurl: string
	bundle: string
	federations: [...#Federation]
	attestationpolicies: [...#APBinding]
}

#TrustProvider: {
	name: string
	kind: string
}

#APBinding: {
	trustzone: string
	policy: string
	federateswith: [...string]
}

#AttestationPolicy: {
	name: string
	kind: int
	namespace: string
	podkey: string
	podvalue: string
}

#Federation: {
	left: string
	right: string
}

#Config: {
	plugins: [...#Plugin]
	trustzones: [...#TrustZone]
	attestationpolicies: [...#AttestationPolicy]
}

#Config
