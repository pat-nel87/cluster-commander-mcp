package k8s

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListRoleBindings(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "admin-binding", Namespace: "default"},
			RoleRef: rbacv1.RoleRef{
				Kind: "Role",
				Name: "admin",
			},
			Subjects: []rbacv1.Subject{
				{Kind: "ServiceAccount", Name: "my-sa", Namespace: "default"},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	bindings, err := client.ListRoleBindings(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListRoleBindings() error = %v", err)
	}
	if len(bindings) != 1 {
		t.Errorf("expected 1 role binding, got %d", len(bindings))
	}
	if bindings[0].Subjects[0].Name != "my-sa" {
		t.Errorf("expected subject 'my-sa', got %q", bindings[0].Subjects[0].Name)
	}
}

func TestListClusterRoleBindings(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-admin-binding"},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "cluster-admin",
			},
			Subjects: []rbacv1.Subject{
				{Kind: "User", Name: "admin@example.com"},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	bindings, err := client.ListClusterRoleBindings(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListClusterRoleBindings() error = %v", err)
	}
	if len(bindings) != 1 {
		t.Errorf("expected 1 cluster role binding, got %d", len(bindings))
	}
}

func TestListRoles(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-reader", Namespace: "default"},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	roles, err := client.ListRoles(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListRoles() error = %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(roles))
	}
}
