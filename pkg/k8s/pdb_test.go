package k8s

import (
	"context"
	"testing"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListPodDisruptionBudgets(t *testing.T) {
	minAvail := intstr.FromInt32(2)
	fakeClient := fake.NewSimpleClientset(
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{Name: "web-pdb", Namespace: "default"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvail,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
			},
			Status: policyv1.PodDisruptionBudgetStatus{
				CurrentHealthy:     3,
				ExpectedPods:       3,
				DisruptionsAllowed: 1,
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	pdbs, err := client.ListPodDisruptionBudgets(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListPodDisruptionBudgets() error = %v", err)
	}
	if len(pdbs) != 1 {
		t.Errorf("expected 1 PDB, got %d", len(pdbs))
	}
	if pdbs[0].Status.DisruptionsAllowed != 1 {
		t.Errorf("expected 1 disruption allowed, got %d", pdbs[0].Status.DisruptionsAllowed)
	}
}
