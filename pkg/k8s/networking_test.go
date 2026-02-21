package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListServices(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "default"},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: "10.0.0.1",
				Ports: []corev1.ServicePort{
					{Port: 80, Protocol: corev1.ProtocolTCP},
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	services, err := client.ListServices(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListServices() error = %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 service, got %d", len(services))
	}
	if services[0].Spec.ClusterIP != "10.0.0.1" {
		t.Errorf("expected ClusterIP '10.0.0.1', got %q", services[0].Spec.ClusterIP)
	}
}

func TestListIngresses(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	fakeClient := fake.NewSimpleClientset(
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "web-ingress", Namespace: "default"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{Path: "/", PathType: &pathType},
								},
							},
						},
					},
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	ingresses, err := client.ListIngresses(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListIngresses() error = %v", err)
	}
	if len(ingresses) != 1 {
		t.Errorf("expected 1 ingress, got %d", len(ingresses))
	}
}

func TestGetEndpoints(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "default"},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{IP: "10.244.0.5"},
						{IP: "10.244.0.6"},
					},
					Ports: []corev1.EndpointPort{
						{Port: 8080},
					},
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	ep, err := client.GetEndpoints(context.Background(), "default", "web-svc")
	if err != nil {
		t.Fatalf("GetEndpoints() error = %v", err)
	}
	if len(ep.Subsets) != 1 {
		t.Errorf("expected 1 subset, got %d", len(ep.Subsets))
	}
	if len(ep.Subsets[0].Addresses) != 2 {
		t.Errorf("expected 2 addresses, got %d", len(ep.Subsets[0].Addresses))
	}
}
