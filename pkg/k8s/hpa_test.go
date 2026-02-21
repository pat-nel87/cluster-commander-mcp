package k8s

import (
	"context"
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListHPAs(t *testing.T) {
	minReplicas := int32(2)
	fakeClient := fake.NewSimpleClientset(
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: "web-hpa", Namespace: "default"},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					Kind: "Deployment",
					Name: "web",
				},
				MinReplicas: &minReplicas,
				MaxReplicas: 10,
			},
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{
				CurrentReplicas: 3,
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	hpas, err := client.ListHPAs(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}
	if len(hpas) != 1 {
		t.Errorf("expected 1 HPA, got %d", len(hpas))
	}
	if hpas[0].Spec.MaxReplicas != 10 {
		t.Errorf("expected MaxReplicas 10, got %d", hpas[0].Spec.MaxReplicas)
	}
}

func TestListHPAsAllNamespaces(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: "hpa1", Namespace: "ns1"},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				MaxReplicas: 5,
			},
		},
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: "hpa2", Namespace: "ns2"},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				MaxReplicas: 8,
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	hpas, err := client.ListHPAs(context.Background(), "", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}
	if len(hpas) != 2 {
		t.Errorf("expected 2 HPAs, got %d", len(hpas))
	}
}
