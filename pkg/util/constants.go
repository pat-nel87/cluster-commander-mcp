package util

import "time"

const (
	// DefaultTimeout is the default timeout for Kubernetes API calls.
	DefaultTimeout = 30 * time.Second

	// MaxLogBytes is the maximum size of pod logs to return (50KB).
	MaxLogBytes = 50 * 1024

	// MaxEvents is the maximum number of events to return.
	MaxEvents = 50

	// MaxPods is the maximum number of pods to return in a list.
	MaxPods = 200

	// DefaultTailLines is the default number of log lines to tail.
	DefaultTailLines int64 = 100

	// DefaultTopLimit is the default number of top resource consumers.
	DefaultTopLimit = 10

	// HighRestartThreshold is the restart count that triggers a warning.
	HighRestartThreshold int32 = 5

	// ResourceUsageWarningPercent is the threshold for resource usage warnings.
	ResourceUsageWarningPercent = 80

	// MaxNetworkPolicies is the maximum number of network policies to return.
	MaxNetworkPolicies = 100

	// MaxRBACBindings is the maximum number of RBAC bindings to return.
	MaxRBACBindings = 200

	// MaxFluxResources is the maximum number of Flux resources to return in a list.
	MaxFluxResources = 200
)
