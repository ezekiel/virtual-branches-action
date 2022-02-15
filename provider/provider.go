package provider

import "context"

// VirtualBranchConfig is the configuration we're expecting to be supplied to us.
// Based on this configuration, we're going to create the virtual branches.
type VirtualBranchConfig struct {
	Target string
	Base   string
	Track  []string
}

// Provider defines the expected capabilities of any service wanting to be a provider for this.
type Provider interface {
	// GetConfigurations gets the slice of VirtualBranchConfig using provider specific tooling.
	GetConfigurations(ctx context.Context) ([]VirtualBranchConfig, error)

	// ApplyConfigurations will apply the configurations retrieved from GetConfigurations to the provider.
	// This method returns a slice of errors, if the error happened with a specific VirtualBranchConfig,
	// or a second error as the second return value if the error happened outside the VirtualBranchConfig
	// processing loop.
	ApplyConfigurations(ctx context.Context, configs []VirtualBranchConfig) ([]error, error)
}
