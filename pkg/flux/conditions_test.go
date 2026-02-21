package flux

import (
	"testing"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetFluxHealth_Ready(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
	}
	status := GetFluxHealth(conditions, 1, 1, false)
	if status != HealthReady {
		t.Errorf("expected Ready, got %s", status)
	}
}

func TestGetFluxHealth_Suspended(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
	}
	status := GetFluxHealth(conditions, 1, 1, true)
	if status != HealthSuspended {
		t.Errorf("expected Suspended, got %s", status)
	}
}

func TestGetFluxHealth_Stalled(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.StalledCondition, Status: metav1.ConditionTrue, Reason: "DependencyNotReady"},
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionFalse},
	}
	status := GetFluxHealth(conditions, 1, 1, false)
	if status != HealthStalled {
		t.Errorf("expected Stalled, got %s", status)
	}
}

func TestGetFluxHealth_Reconciling(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.ReconcilingCondition, Status: metav1.ConditionTrue},
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionFalse},
	}
	status := GetFluxHealth(conditions, 2, 1, false)
	if status != HealthReconciling {
		t.Errorf("expected Reconciling, got %s", status)
	}
}

func TestGetFluxHealth_Failed(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionFalse, Reason: "BuildFailed"},
	}
	status := GetFluxHealth(conditions, 1, 1, false)
	if status != HealthFailed {
		t.Errorf("expected Failed, got %s", status)
	}
}

func TestGetFluxHealth_Unknown(t *testing.T) {
	status := GetFluxHealth(nil, 1, 0, false)
	if status != HealthUnknown {
		t.Errorf("expected Unknown, got %s", status)
	}
}

func TestGetFluxHealth_GenerationMismatch(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
	}
	status := GetFluxHealth(conditions, 3, 2, false)
	if status != HealthReconciling {
		t.Errorf("expected Reconciling on generation mismatch, got %s", status)
	}
}

func TestGetConditionMessage(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionFalse, Message: "build failed: invalid path"},
	}
	msg := GetConditionMessage(conditions, fluxmeta.ReadyCondition)
	if msg != "build failed: invalid path" {
		t.Errorf("expected message, got %q", msg)
	}
}

func TestGetConditionMessage_Missing(t *testing.T) {
	msg := GetConditionMessage(nil, fluxmeta.ReadyCondition)
	if msg != "" {
		t.Errorf("expected empty, got %q", msg)
	}
}

func TestGetConditionReason(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: fluxmeta.ReadyCondition, Reason: "BuildFailed"},
	}
	reason := GetConditionReason(conditions, fluxmeta.ReadyCondition)
	if reason != "BuildFailed" {
		t.Errorf("expected BuildFailed, got %q", reason)
	}
}
