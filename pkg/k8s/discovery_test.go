package k8s

import (
	"context"
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListMutatingWebhookConfigurations(t *testing.T) {
	failPolicy := admissionregistrationv1.Fail
	fakeClient := fake.NewSimpleClientset(
		&admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: "inject-sidecar"},
			Webhooks: []admissionregistrationv1.MutatingWebhook{
				{
					Name:          "sidecar.example.com",
					FailurePolicy: &failPolicy,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: "webhook-system",
							Name:      "sidecar-injector",
						},
					},
					AdmissionReviewVersions: []string{"v1"},
					SideEffects:             sideEffectNone(),
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	configs, err := client.ListMutatingWebhookConfigurations(context.Background())
	if err != nil {
		t.Fatalf("ListMutatingWebhookConfigurations() error = %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("expected 1 mutating webhook config, got %d", len(configs))
	}
	if len(configs[0].Webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(configs[0].Webhooks))
	}
}

func TestListValidatingWebhookConfigurations(t *testing.T) {
	failPolicy := admissionregistrationv1.Ignore
	fakeClient := fake.NewSimpleClientset(
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: "validate-policy"},
			Webhooks: []admissionregistrationv1.ValidatingWebhook{
				{
					Name:          "policy.example.com",
					FailurePolicy: &failPolicy,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: "policy-system",
							Name:      "policy-validator",
						},
					},
					AdmissionReviewVersions: []string{"v1"},
					SideEffects:             sideEffectNone(),
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	configs, err := client.ListValidatingWebhookConfigurations(context.Background())
	if err != nil {
		t.Fatalf("ListValidatingWebhookConfigurations() error = %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("expected 1 validating webhook config, got %d", len(configs))
	}
}

func TestListCRDsNilClient(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	client := NewClusterClientForTesting(fakeClient, nil)

	// ApiextensionsClient is nil by default in test
	_, err := client.ListCRDs(context.Background())
	if err == nil {
		t.Error("expected error for nil ApiextensionsClient, got nil")
	}
}

func sideEffectNone() *admissionregistrationv1.SideEffectClass {
	se := admissionregistrationv1.SideEffectClassNone
	return &se
}
