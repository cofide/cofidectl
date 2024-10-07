#Plugins: {
	name: string
}
#TrustZone: {
	name: string
	trust_domain: string
}

#Config: {
	plugins: [...#TrustZone]
	trust_zones: [...#TrustZone]
}

config: #Config
