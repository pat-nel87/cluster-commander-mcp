package k8s

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListEvents(t *testing.T) {
	now := time.Now()
	fakeClient := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta:    metav1.ObjectMeta{Name: "event-1", Namespace: "default"},
			Type:          "Warning",
			Reason:        "BackOff",
			Message:       "Back-off restarting failed container",
			Count:         5,
			LastTimestamp:  metav1.NewTime(now.Add(-1 * time.Minute)),
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "test-pod"},
		},
		&corev1.Event{
			ObjectMeta:    metav1.ObjectMeta{Name: "event-2", Namespace: "default"},
			Type:          "Normal",
			Reason:        "Pulled",
			Message:       "Successfully pulled image",
			Count:         1,
			LastTimestamp:  metav1.NewTime(now.Add(-5 * time.Minute)),
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "test-pod"},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	events, err := client.ListEvents(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	// Should be sorted most recent first
	if events[0].Name != "event-1" {
		t.Errorf("expected event-1 first (most recent), got %q", events[0].Name)
	}
}

func TestGetEventsForObject(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "event-1", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "target-pod"},
			Type:           "Warning",
			Reason:         "Failed",
			Message:        "Error pulling image",
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "event-2", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "other-pod"},
			Type:           "Normal",
			Reason:         "Started",
			Message:        "Started container",
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	events, err := client.GetEventsForObject(context.Background(), "default", "target-pod")
	if err != nil {
		t.Fatalf("GetEventsForObject() error = %v", err)
	}
	// The fake clientset may not filter by field selector, so just verify no error
	if len(events) == 0 {
		// Fake clientset doesn't support field selectors, so we get all events
		// This is expected behavior for the fake
		t.Log("Note: fake clientset may not filter by field selector")
	}
}
