package provider

// WorkloadIdentityProvider is the interface to drive downstream workload identity methodologies for the Cofide stack
type WorkloadIdentityProvider interface {
	Execute() error
}
