package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListPods(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default", Labels: map[string]string{"app": "web"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default", Labels: map[string]string{"app": "api"}},
			Status:     corev1.PodStatus{Phase: corev1.PodPending},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-3", Namespace: "kube-system"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	// List all pods
	pods, err := client.ListPods(context.Background(), "", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListPods() error = %v", err)
	}
	if len(pods) != 3 {
		t.Errorf("expected 3 pods, got %d", len(pods))
	}

	// List pods in default namespace
	pods, err = client.ListPods(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListPods() error = %v", err)
	}
	if len(pods) != 2 {
		t.Errorf("expected 2 pods in default, got %d", len(pods))
	}
}

func TestGetPod(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "nginx:latest"},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	pod, err := client.GetPod(context.Background(), "default", "test-pod")
	if err != nil {
		t.Fatalf("GetPod() error = %v", err)
	}
	if pod.Name != "test-pod" {
		t.Errorf("expected pod name 'test-pod', got %q", pod.Name)
	}
	if len(pod.Spec.Containers) != 1 {
		t.Errorf("expected 1 container, got %d", len(pod.Spec.Containers))
	}
}

func TestGetPodNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	client := NewClusterClientForTesting(fakeClient, nil)

	_, err := client.GetPod(context.Background(), "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent pod")
	}
}
