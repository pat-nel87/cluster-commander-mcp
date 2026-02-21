package k8s

import (
	"context"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

// ListPodDisruptionBudgets returns PDBs in the given namespace.
func (c *ClusterClient) ListPodDisruptionBudgets(ctx context.Context, namespace string, opts metav1.ListOptions) ([]policyv1.PodDisruptionBudget, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
