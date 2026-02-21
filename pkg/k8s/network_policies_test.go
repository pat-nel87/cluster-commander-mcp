package k8s

import (
	"context"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListNetworkPolicies(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "deny-all", Namespace: "default"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-web", Namespace: "default"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{},
				},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	policies, err := client.ListNetworkPolicies(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListNetworkPolicies() error = %v", err)
	}
	if len(policies) != 2 {
		t.Errorf("expected 2 network policies, got %d", len(policies))
	}
}

func TestListNetworkPoliciesAllNamespaces(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "np1", Namespace: "ns1"},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "np2", Namespace: "ns2"},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	policies, err := client.ListNetworkPolicies(context.Background(), "", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListNetworkPolicies() error = %v", err)
	}
	if len(policies) != 2 {
		t.Errorf("expected 2 network policies across all namespaces, got %d", len(policies))
	}
}
