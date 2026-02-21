package flux

import (
	"context"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListHelmReleases returns all HelmReleases in the given namespace (empty = all namespaces).
func (fc *FluxClient) ListHelmReleases(ctx context.Context, namespace string) ([]helmv2.HelmRelease, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list helmv2.HelmReleaseList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetHelmRelease returns a single HelmRelease by namespace and name.
func (fc *FluxClient) GetHelmRelease(ctx context.Context, namespace, name string) (*helmv2.HelmRelease, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var hr helmv2.HelmRelease
	if err := fc.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &hr); err != nil {
		return nil, err
	}
	return &hr, nil
}

// HelmReleaseHealth returns the health status of a HelmRelease.
func HelmReleaseHealth(hr *helmv2.HelmRelease) FluxHealthStatus {
	return GetFluxHealth(
		hr.Status.Conditions,
		hr.Generation,
		hr.Status.ObservedGeneration,
		hr.Spec.Suspend,
	)
}

// HelmReleaseAge returns the time since creation.
func HelmReleaseAge(hr *helmv2.HelmRelease) time.Duration {
	return time.Since(hr.CreationTimestamp.Time)
}
