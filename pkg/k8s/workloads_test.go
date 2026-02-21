package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func int32Ptr(i int32) *int32 { return &i }

func TestListDeployments(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     3,
				AvailableReplicas: 3,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(2),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	deployments, err := client.ListDeployments(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListDeployments() error = %v", err)
	}
	if len(deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(deployments))
	}
}

func TestListStatefulSets(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: 3,
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	sets, err := client.ListStatefulSets(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListStatefulSets() error = %v", err)
	}
	if len(sets) != 1 {
		t.Errorf("expected 1 statefulset, got %d", len(sets))
	}
}

func TestListJobs(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "migration", Namespace: "default"},
			Spec: batchv1.JobSpec{
				Completions: int32Ptr(1),
			},
			Status: batchv1.JobStatus{
				Succeeded: 1,
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	jobs, err := client.ListJobs(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}
