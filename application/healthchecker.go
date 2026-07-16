package application

import "context"

// Healthchecker represents a service that can perform health checks.
type Healthchecker interface {
	// Healthcheck returns JSON-serializable health data for the service.
	// The returned value must not be mutated after Healthcheck returns.
	Healthcheck(context.Context) any
}
