package flux

import (
	"context"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListKustomizations returns all Kustomizations in the given namespace (empty = all namespaces).
func (fc *FluxClient) ListKustomizations(ctx context.Context, namespace string) ([]kustomizev1.Kustomization, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var list kustomizev1.KustomizationList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := fc.Client.List(ctx, &list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetKustomization returns a single Kustomization by namespace and name.
func (fc *FluxClient) GetKustomization(ctx context.Context, namespace, name string) (*kustomizev1.Kustomization, error) {
	ctx, cancel := context.WithTimeout(ctx, util.DefaultTimeout)
	defer cancel()

	var ks kustomizev1.Kustomization
	if err := fc.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &ks); err != nil {
		return nil, err
	}
	return &ks, nil
}

// KustomizationHealth returns the health status of a Kustomization.
func KustomizationHealth(ks *kustomizev1.Kustomization) FluxHealthStatus {
	return GetFluxHealth(
		ks.Status.Conditions,
		ks.Generation,
		ks.Status.ObservedGeneration,
		ks.Spec.Suspend,
	)
}

// KustomizationAge returns the time since creation.
func KustomizationAge(ks *kustomizev1.Kustomization) time.Duration {
	return time.Since(ks.CreationTimestamp.Time)
}
