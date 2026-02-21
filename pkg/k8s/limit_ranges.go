package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

// ListLimitRanges returns limit ranges in the given namespace.
func (c *ClusterClient) ListLimitRanges(ctx context.Context, namespace string, opts metav1.ListOptions) ([]corev1.LimitRange, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.CoreV1().LimitRanges(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
