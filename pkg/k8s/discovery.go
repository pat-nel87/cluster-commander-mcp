package k8s

import (
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

// ListCRDs returns custom resource definitions from the cluster.
func (c *ClusterClient) ListCRDs(ctx context.Context) ([]apiextensionsv1.CustomResourceDefinition, error) {
	if c.ApiextensionsClient == nil {
		return nil, fmt.Errorf("apiextensions client not available")
	}

	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.ApiextensionsClient.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListMutatingWebhookConfigurations returns mutating webhook configurations.
func (c *ClusterClient) ListMutatingWebhookConfigurations(ctx context.Context) ([]admissionregistrationv1.MutatingWebhookConfiguration, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListValidatingWebhookConfigurations returns validating webhook configurations.
func (c *ClusterClient) ListValidatingWebhookConfigurations(ctx context.Context) ([]admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	list, err := c.Clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetAPIResources returns server API resources grouped by API group.
func (c *ClusterClient) GetAPIResources(ctx context.Context) ([]*metav1.APIResourceList, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	_, resourceLists, err := c.Clientset.Discovery().ServerGroupsAndResources()
	if err != nil {
		// Partial results are common; return what we have
		if resourceLists != nil {
			return resourceLists, nil
		}
		return nil, err
	}
	return resourceLists, nil
}
