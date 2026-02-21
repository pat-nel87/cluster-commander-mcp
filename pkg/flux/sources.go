package flux

import (
	"context"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListGitRepositories returns all GitRepositories in the given namespace (empty = all).
func (fc *FluxClient) ListGitRepositories(ctx context.Context, namespace string) ([]sourcev1.GitRepository, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list sourcev1.GitRepositoryList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetGitRepository returns a single GitRepository by namespace and name.
func (fc *FluxClient) GetGitRepository(ctx context.Context, namespace, name string) (*sourcev1.GitRepository, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var obj sourcev1.GitRepository
	if err := fc.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// ListOCIRepositories returns all OCIRepositories in the given namespace (empty = all).
func (fc *FluxClient) ListOCIRepositories(ctx context.Context, namespace string) ([]sourcev1beta2.OCIRepository, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list sourcev1beta2.OCIRepositoryList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetOCIRepository returns a single OCIRepository by namespace and name.
func (fc *FluxClient) GetOCIRepository(ctx context.Context, namespace, name string) (*sourcev1beta2.OCIRepository, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var obj sourcev1beta2.OCIRepository
	if err := fc.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// ListHelmRepositories returns all HelmRepositories in the given namespace (empty = all).
func (fc *FluxClient) ListHelmRepositories(ctx context.Context, namespace string) ([]sourcev1.HelmRepository, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list sourcev1.HelmRepositoryList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetHelmRepository returns a single HelmRepository by namespace and name.
func (fc *FluxClient) GetHelmRepository(ctx context.Context, namespace, name string) (*sourcev1.HelmRepository, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var obj sourcev1.HelmRepository
	if err := fc.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// ListHelmCharts returns all HelmCharts in the given namespace (empty = all).
func (fc *FluxClient) ListHelmCharts(ctx context.Context, namespace string) ([]sourcev1.HelmChart, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list sourcev1.HelmChartList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetHelmChart returns a single HelmChart by namespace and name.
func (fc *FluxClient) GetHelmChart(ctx context.Context, namespace, name string) (*sourcev1.HelmChart, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var obj sourcev1.HelmChart
	if err := fc.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// ListBuckets returns all Buckets in the given namespace (empty = all).
func (fc *FluxClient) ListBuckets(ctx context.Context, namespace string) ([]sourcev1.Bucket, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list sourcev1.BucketList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetBucket returns a single Bucket by namespace and name.
func (fc *FluxClient) GetBucket(ctx context.Context, namespace, name string) (*sourcev1.Bucket, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var obj sourcev1.Bucket
	if err := fc.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}
