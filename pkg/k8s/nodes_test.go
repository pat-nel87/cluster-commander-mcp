package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListNodes(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "node-1",
				Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
				Capacity: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
				NodeInfo: corev1.NodeSystemInfo{
					KubeletVersion: "v1.28.0",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
				Capacity: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("16Gi"),
				},
				NodeInfo: corev1.NodeSystemInfo{
					KubeletVersion: "v1.28.0",
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	nodes, err := client.ListNodes(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestGetNode(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue, Message: "kubelet is posting ready status"},
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	node, err := client.GetNode(context.Background(), "test-node")
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}
	if node.Name != "test-node" {
		t.Errorf("expected node name 'test-node', got %q", node.Name)
	}
}

func TestGetNodeNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	client := NewClusterClientForTesting(fakeClient, nil)

	_, err := client.GetNode(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent node")
	}
}
