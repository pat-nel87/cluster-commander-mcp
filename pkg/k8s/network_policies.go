package k8s

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

// ListNetworkPolicies returns network policies in the given namespace.
func (c *ClusterClient) ListNetworkPolicies(ctx context.Context, namespace string, opts metav1.ListOptions) ([]networkingv1.NetworkPolicy, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(list.Items) > util.MaxNetworkPolicies {
		return list.Items[:util.MaxNetworkPolicies], nil
	}
	return list.Items, nil
}
