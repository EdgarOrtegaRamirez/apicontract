package validator

import "time"

// ValidateOption configures validation behavior.
type ValidateOption func(*validateConfig)

type validateConfig struct {
	Timeout    time.Duration
	Headers    map[string]string
	PathParams map[string]string
}

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 30 * time.Second

// WithTimeout sets the HTTP request timeout.
func WithTimeout(t time.Duration) ValidateOption {
	return func(c *validateConfig) {
		c.Timeout = t
	}
}

// WithHeaders sets additional request headers.
func WithHeaders(h map[string]string) ValidateOption {
	return func(c *validateConfig) {
		c.Headers = h
	}
}

// WithPathParams sets path parameter values.
func WithPathParams(p map[string]string) ValidateOption {
	return func(c *validateConfig) {
		c.PathParams = p
	}
}

func applyOptions(opts []ValidateOption) validateConfig {
	cfg := validateConfig{
		Timeout:    DefaultTimeout,
		Headers:    make(map[string]string),
		PathParams: make(map[string]string),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
