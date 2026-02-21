package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

// ListRoles returns roles in the given namespace.
func (c *ClusterClient) ListRoles(ctx context.Context, namespace string, opts metav1.ListOptions) ([]rbacv1.Role, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.RbacV1().Roles(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListClusterRoles returns all cluster roles.
func (c *ClusterClient) ListClusterRoles(ctx context.Context, opts metav1.ListOptions) ([]rbacv1.ClusterRole, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.RbacV1().ClusterRoles().List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListRoleBindings returns role bindings in the given namespace.
func (c *ClusterClient) ListRoleBindings(ctx context.Context, namespace string, opts metav1.ListOptions) ([]rbacv1.RoleBinding, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.RbacV1().RoleBindings(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(list.Items) > util.MaxRBACBindings {
		return list.Items[:util.MaxRBACBindings], nil
	}
	return list.Items, nil
}

// ListClusterRoleBindings returns all cluster role bindings.
func (c *ClusterClient) ListClusterRoleBindings(ctx context.Context, opts metav1.ListOptions) ([]rbacv1.ClusterRoleBinding, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.RbacV1().ClusterRoleBindings().List(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(list.Items) > util.MaxRBACBindings {
		return list.Items[:util.MaxRBACBindings], nil
	}
	return list.Items, nil
}
