package resend

import "github.com/testcontainers/testcontainers-go"

// resendOptions holds the configuration for the Resend module.
type resendOptions struct {
	specURL string
}

func defaultOptions() resendOptions {
	return resendOptions{
		specURL: defaultSpecURL,
	}
}

// Compiler check to ensure that Option implements the testcontainers.ContainerCustomizer interface.
var _ testcontainers.ContainerCustomizer = (Option)(nil)

// Option is a function that configures the Resend module.
type Option func(*resendOptions) error

// Customize is a NOOP. It's defined to satisfy the testcontainers.ContainerCustomizer interface.
func (o Option) Customize(*testcontainers.GenericContainerRequest) error {
	// NOOP to satisfy interface.
	return nil
}

// WithSpecURL sets a custom URL to fetch the Resend OpenAPI spec from.
// If the URL is unreachable, the module falls back to the embedded spec.
func WithSpecURL(url string) Option {
	return func(o *resendOptions) error {
		o.specURL = url
		return nil
	}
}
