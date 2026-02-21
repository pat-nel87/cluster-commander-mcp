package flux

import (
	"context"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	autov1beta2 "github.com/fluxcd/image-automation-controller/api/v1beta2"
	imagev1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notifv1 "github.com/fluxcd/notification-controller/api/v1"
	notifv1beta3 "github.com/fluxcd/notification-controller/api/v1beta3"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// FluxClient wraps a controller-runtime client configured for Flux CRDs.
type FluxClient struct {
	Client client.Client
}

// newScheme builds a runtime.Scheme with all Flux API types registered.
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = sourcev1.AddToScheme(scheme)
	_ = sourcev1beta2.AddToScheme(scheme)
	_ = kustomizev1.AddToScheme(scheme)
	_ = helmv2.AddToScheme(scheme)
	_ = notifv1.AddToScheme(scheme)
	_ = notifv1beta3.AddToScheme(scheme)
	_ = imagev1beta2.AddToScheme(scheme)
	_ = autov1beta2.AddToScheme(scheme)
	return scheme
}

// NewFluxClient creates a FluxClient from a rest.Config.
func NewFluxClient(config *rest.Config) (*FluxClient, error) {
	c, err := client.New(config, client.Options{Scheme: newScheme()})
	if err != nil {
		return nil, err
	}
	return &FluxClient{Client: c}, nil
}

// NewFluxClientForTesting creates a FluxClient backed by a fake client for unit tests.
func NewFluxClientForTesting(objects ...client.Object) *FluxClient {
	scheme := newScheme()
	builder := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(objects...)
	return &FluxClient{Client: builder.Build()}
}

// IsFluxInstalled probes the cluster to detect whether Flux CRDs are present.
func (fc *FluxClient) IsFluxInstalled(ctx context.Context) bool {
	var list kustomizev1.KustomizationList
	err := fc.Client.List(ctx, &list, client.Limit(1))
	return err == nil
}
