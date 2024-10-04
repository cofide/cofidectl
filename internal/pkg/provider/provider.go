package provider

// SPIREProvider is the interface to downstream SPIRE installation methodologies for the Cofide stack
type SPIREProvider interface {
	Execute() error
}
