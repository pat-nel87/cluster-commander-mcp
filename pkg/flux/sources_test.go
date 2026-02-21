package flux

import (
	"context"
	"testing"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListGitRepositories(t *testing.T) {
	gr := &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "flux-system", Namespace: "flux-system"},
		Spec: sourcev1.GitRepositorySpec{
			URL: "https://github.com/org/repo",
		},
	}

	fc := NewFluxClientForTesting(gr)
	ctx := context.Background()

	items, err := fc.ListGitRepositories(ctx, "")
	if err != nil {
		t.Fatalf("ListGitRepositories() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if items[0].Spec.URL != "https://github.com/org/repo" {
		t.Errorf("expected URL, got %q", items[0].Spec.URL)
	}
}

func TestListOCIRepositories(t *testing.T) {
	oci := &sourcev1beta2.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: "flux-system"},
		Spec: sourcev1beta2.OCIRepositorySpec{
			URL: "oci://ghcr.io/stefanprodan/manifests/podinfo",
		},
	}

	fc := NewFluxClientForTesting(oci)
	ctx := context.Background()

	items, err := fc.ListOCIRepositories(ctx, "flux-system")
	if err != nil {
		t.Fatalf("ListOCIRepositories() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestListHelmRepositories(t *testing.T) {
	hr := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "bitnami", Namespace: "flux-system"},
		Spec: sourcev1.HelmRepositorySpec{
			URL: "https://charts.bitnami.com/bitnami",
		},
	}

	fc := NewFluxClientForTesting(hr)
	ctx := context.Background()

	items, err := fc.ListHelmRepositories(ctx, "")
	if err != nil {
		t.Fatalf("ListHelmRepositories() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestListHelmCharts(t *testing.T) {
	hc := &sourcev1.HelmChart{
		ObjectMeta: metav1.ObjectMeta{Name: "flux-system-nginx", Namespace: "flux-system"},
		Spec: sourcev1.HelmChartSpec{
			Chart:   "nginx",
			Version: ">=1.0.0",
		},
	}

	fc := NewFluxClientForTesting(hc)
	ctx := context.Background()

	items, err := fc.ListHelmCharts(ctx, "flux-system")
	if err != nil {
		t.Fatalf("ListHelmCharts() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestListBuckets(t *testing.T) {
	b := &sourcev1.Bucket{
		ObjectMeta: metav1.ObjectMeta{Name: "my-bucket", Namespace: "flux-system"},
		Spec: sourcev1.BucketSpec{
			BucketName: "my-s3-bucket",
			Endpoint:   "s3.amazonaws.com",
		},
	}

	fc := NewFluxClientForTesting(b)
	ctx := context.Background()

	items, err := fc.ListBuckets(ctx, "")
	if err != nil {
		t.Fatalf("ListBuckets() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestListGitRepositories_Empty(t *testing.T) {
	fc := NewFluxClientForTesting()
	ctx := context.Background()

	items, err := fc.ListGitRepositories(ctx, "flux-system")
	if err != nil {
		t.Fatalf("ListGitRepositories() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestListGitRepositories_NamespaceScoped(t *testing.T) {
	gr1 := &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "repo1", Namespace: "flux-system"},
		Spec:       sourcev1.GitRepositorySpec{URL: "https://github.com/org/repo1"},
	}
	gr2 := &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "repo2", Namespace: "other"},
		Spec:       sourcev1.GitRepositorySpec{URL: "https://github.com/org/repo2"},
	}

	fc := NewFluxClientForTesting(gr1, gr2)
	ctx := context.Background()

	items, err := fc.ListGitRepositories(ctx, "flux-system")
	if err != nil {
		t.Fatalf("ListGitRepositories(flux-system) error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item in flux-system, got %d", len(items))
	}
}
