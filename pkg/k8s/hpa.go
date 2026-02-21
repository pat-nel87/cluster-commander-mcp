package k8s

import (
	"context"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

// ListHPAs returns horizontal pod autoscalers in the given namespace.
func (c *ClusterClient) ListHPAs(ctx context.Context, namespace string, opts metav1.ListOptions) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
