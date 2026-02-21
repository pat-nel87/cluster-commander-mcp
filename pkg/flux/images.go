package flux

import (
	"context"

	imagev1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListImageRepositories returns all ImageRepositories in the given namespace (empty = all).
func (fc *FluxClient) ListImageRepositories(ctx context.Context, namespace string) ([]imagev1beta2.ImageRepository, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list imagev1beta2.ImageRepositoryList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListImagePolicies returns all ImagePolicies in the given namespace (empty = all).
func (fc *FluxClient) ListImagePolicies(ctx context.Context, namespace string) ([]imagev1beta2.ImagePolicy, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list imagev1beta2.ImagePolicyList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}
