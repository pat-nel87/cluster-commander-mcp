package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListLimitRanges(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.LimitRange{
			ObjectMeta: metav1.ObjectMeta{Name: "default-limits", Namespace: "default"},
			Spec: corev1.LimitRangeSpec{
				Limits: []corev1.LimitRangeItem{
					{
						Type: corev1.LimitTypeContainer,
						Default: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
						DefaultRequest: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	limitRanges, err := client.ListLimitRanges(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListLimitRanges() error = %v", err)
	}
	if len(limitRanges) != 1 {
		t.Errorf("expected 1 limit range, got %d", len(limitRanges))
	}
	if len(limitRanges[0].Spec.Limits) != 1 {
		t.Errorf("expected 1 limit item, got %d", len(limitRanges[0].Spec.Limits))
	}
}

func TestListLimitRangesEmpty(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	client := NewClusterClientForTesting(fakeClient, nil)

	limitRanges, err := client.ListLimitRanges(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListLimitRanges() error = %v", err)
	}
	if len(limitRanges) != 0 {
		t.Errorf("expected 0 limit ranges, got %d", len(limitRanges))
	}
}
