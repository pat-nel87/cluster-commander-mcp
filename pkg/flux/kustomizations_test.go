package flux

import (
	"context"
	"testing"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListKustomizations(t *testing.T) {
	ks1 := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "flux-system"},
		Spec: kustomizev1.KustomizationSpec{
			Path: "./clusters/prod",
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: "flux-system",
			},
		},
		Status: kustomizev1.KustomizationStatus{
			Conditions: []metav1.Condition{
				{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
			},
		},
	}
	ks2 := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{Name: "monitoring", Namespace: "flux-system"},
		Spec: kustomizev1.KustomizationSpec{
			Path: "./monitoring",
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: "flux-system",
			},
		},
	}
	ks3 := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "other-ns"},
	}

	fc := NewFluxClientForTesting(ks1, ks2, ks3)
	ctx := context.Background()

	// All namespaces
	items, err := fc.ListKustomizations(ctx, "")
	if err != nil {
		t.Fatalf("ListKustomizations() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	// Namespace scoped
	items, err = fc.ListKustomizations(ctx, "flux-system")
	if err != nil {
		t.Fatalf("ListKustomizations(flux-system) error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items in flux-system, got %d", len(items))
	}
}

func TestGetKustomization(t *testing.T) {
	ks := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "flux-system"},
		Spec: kustomizev1.KustomizationSpec{
			Path: "./clusters/prod",
		},
	}

	fc := NewFluxClientForTesting(ks)
	ctx := context.Background()

	got, err := fc.GetKustomization(ctx, "flux-system", "app")
	if err != nil {
		t.Fatalf("GetKustomization() error = %v", err)
	}
	if got.Name != "app" {
		t.Errorf("expected name 'app', got %q", got.Name)
	}
	if got.Spec.Path != "./clusters/prod" {
		t.Errorf("expected path './clusters/prod', got %q", got.Spec.Path)
	}
}

func TestGetKustomization_NotFound(t *testing.T) {
	fc := NewFluxClientForTesting()
	ctx := context.Background()

	_, err := fc.GetKustomization(ctx, "flux-system", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent kustomization")
	}
}

func TestListKustomizations_Empty(t *testing.T) {
	fc := NewFluxClientForTesting()
	ctx := context.Background()

	items, err := fc.ListKustomizations(ctx, "flux-system")
	if err != nil {
		t.Fatalf("ListKustomizations() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}
