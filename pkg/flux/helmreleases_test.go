package flux

import (
	"context"
	"testing"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListHelmReleases(t *testing.T) {
	hr1 := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default"},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   "nginx",
					Version: ">=1.0.0",
				},
			},
		},
		Status: helmv2.HelmReleaseStatus{
			Conditions: []metav1.Condition{
				{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
			},
		},
	}
	hr2 := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{Name: "prometheus", Namespace: "monitoring"},
	}

	fc := NewFluxClientForTesting(hr1, hr2)
	ctx := context.Background()

	// All namespaces
	items, err := fc.ListHelmReleases(ctx, "")
	if err != nil {
		t.Fatalf("ListHelmReleases() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Namespace scoped
	items, err = fc.ListHelmReleases(ctx, "default")
	if err != nil {
		t.Fatalf("ListHelmReleases(default) error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item in default, got %d", len(items))
	}
}

func TestGetHelmRelease(t *testing.T) {
	hr := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default"},
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart: "nginx",
				},
			},
		},
	}

	fc := NewFluxClientForTesting(hr)
	ctx := context.Background()

	got, err := fc.GetHelmRelease(ctx, "default", "nginx")
	if err != nil {
		t.Fatalf("GetHelmRelease() error = %v", err)
	}
	if got.Name != "nginx" {
		t.Errorf("expected name 'nginx', got %q", got.Name)
	}
}

func TestGetHelmRelease_NotFound(t *testing.T) {
	fc := NewFluxClientForTesting()
	ctx := context.Background()

	_, err := fc.GetHelmRelease(ctx, "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent helm release")
	}
}

func TestListHelmReleases_Empty(t *testing.T) {
	fc := NewFluxClientForTesting()
	ctx := context.Background()

	items, err := fc.ListHelmReleases(ctx, "")
	if err != nil {
		t.Fatalf("ListHelmReleases() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}
