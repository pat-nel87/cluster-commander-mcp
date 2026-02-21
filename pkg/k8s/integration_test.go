//go:build integration

package k8s

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Run with: go test ./pkg/k8s/ -tags integration -v
func TestLiveCluster(t *testing.T) {
	client, err := NewClusterClient("")
	if err != nil {
		t.Fatalf("Failed to connect to cluster: %v", err)
	}

	t.Run("ListNamespaces", func(t *testing.T) {
		namespaces, err := client.ListNamespaces(context.Background())
		if err != nil {
			t.Fatalf("ListNamespaces: %v", err)
		}
		if len(namespaces) == 0 {
			t.Fatal("expected at least 1 namespace")
		}
		for _, ns := range namespaces {
			fmt.Printf("  namespace: %s (%s)\n", ns.Name, ns.Status.Phase)
		}
	})

	t.Run("ListNodes", func(t *testing.T) {
		nodes, err := client.ListNodes(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("ListNodes: %v", err)
		}
		if len(nodes) == 0 {
			t.Fatal("expected at least 1 node")
		}
		for _, n := range nodes {
			fmt.Printf("  node: %s\n", n.Name)
		}
	})

	t.Run("ListPods_AllNamespaces", func(t *testing.T) {
		pods, err := client.ListPods(context.Background(), "", metav1.ListOptions{})
		if err != nil {
			t.Fatalf("ListPods: %v", err)
		}
		fmt.Printf("  total pods: %d\n", len(pods))
		for _, p := range pods {
			fmt.Printf("  %s/%s: %s\n", p.Namespace, p.Name, p.Status.Phase)
		}
	})

	t.Run("ListEvents", func(t *testing.T) {
		events, err := client.ListEvents(context.Background(), "", metav1.ListOptions{})
		if err != nil {
			t.Fatalf("ListEvents: %v", err)
		}
		fmt.Printf("  total events: %d\n", len(events))
		for i, e := range events {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s: %s %s\n", e.Type, e.Reason, e.Message)
		}
	})

	t.Run("ListDeployments_KubeSystem", func(t *testing.T) {
		deployments, err := client.ListDeployments(context.Background(), "kube-system", metav1.ListOptions{})
		if err != nil {
			t.Fatalf("ListDeployments: %v", err)
		}
		fmt.Printf("  kube-system deployments: %d\n", len(deployments))
		for _, d := range deployments {
			desired := int32(0)
			if d.Spec.Replicas != nil {
				desired = *d.Spec.Replicas
			}
			fmt.Printf("  %s: %d/%d ready\n", d.Name, d.Status.ReadyReplicas, desired)
		}
	})

	t.Run("ListServices_AllNamespaces", func(t *testing.T) {
		services, err := client.ListServices(context.Background(), "", metav1.ListOptions{})
		if err != nil {
			t.Fatalf("ListServices: %v", err)
		}
		fmt.Printf("  total services: %d\n", len(services))
		for _, s := range services {
			fmt.Printf("  %s/%s: %s %s\n", s.Namespace, s.Name, s.Spec.Type, s.Spec.ClusterIP)
		}
	})
}
