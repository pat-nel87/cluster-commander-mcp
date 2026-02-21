package flux

import (
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FluxHealthStatus represents the reconciliation health of a Flux resource.
type FluxHealthStatus string

const (
	HealthReady       FluxHealthStatus = "Ready"
	HealthReconciling FluxHealthStatus = "Reconciling"
	HealthStalled     FluxHealthStatus = "Stalled"
	HealthFailed      FluxHealthStatus = "Failed"
	HealthSuspended   FluxHealthStatus = "Suspended"
	HealthUnknown     FluxHealthStatus = "Unknown"
)

// GetFluxHealth interprets Flux status conditions into a single health status.
// The evaluation order follows Flux conventions: Suspended > Stalled > Reconciling > Ready.
func GetFluxHealth(conditions []metav1.Condition, generation, observedGeneration int64, suspended bool) FluxHealthStatus {
	if suspended {
		return HealthSuspended
	}

	stalled := findCondition(conditions, fluxmeta.StalledCondition)
	if stalled != nil && stalled.Status == metav1.ConditionTrue {
		return HealthStalled
	}

	reconciling := findCondition(conditions, fluxmeta.ReconcilingCondition)
	if reconciling != nil && reconciling.Status == metav1.ConditionTrue {
		return HealthReconciling
	}

	ready := findCondition(conditions, fluxmeta.ReadyCondition)
	if ready == nil {
		return HealthUnknown
	}

	if ready.Status == metav1.ConditionTrue {
		if generation != observedGeneration {
			return HealthReconciling
		}
		return HealthReady
	}

	if ready.Status == metav1.ConditionFalse {
		return HealthFailed
	}

	return HealthUnknown
}

// GetConditionMessage returns the message from a condition of the given type, or empty string.
func GetConditionMessage(conditions []metav1.Condition, condType string) string {
	c := findCondition(conditions, condType)
	if c == nil {
		return ""
	}
	return c.Message
}

// GetConditionReason returns the reason from a condition of the given type, or empty string.
func GetConditionReason(conditions []metav1.Condition, condType string) string {
	c := findCondition(conditions, condType)
	if c == nil {
		return ""
	}
	return c.Reason
}

// findCondition returns the condition with the given type, or nil.
func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}
