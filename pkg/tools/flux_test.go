package tools

import (
	"context"
	"strings"
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	imagev1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/flux"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// callFluxTool sets up a server with Flux tools and calls the named tool.
func callFluxTool(t *testing.T, fluxObjs []client.Object, k8sObjs []corev1.Pod, toolName string, args map[string]any) (string, bool) {
	t.Helper()

	fluxClient := flux.NewFluxClientForTesting(fluxObjs...)

	fakeK8s := fake.NewSimpleClientset()
	for i := range k8sObjs {
		fakeK8s = fake.NewSimpleClientset(&k8sObjs[i])
	}
	k8sClient := k8s.NewClusterClientForTesting(fakeK8s, nil)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "kube-doctor-test",
		Version: "test",
	}, nil)

	registerFluxTools(server, fluxClient, k8sClient)

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("Server connect: %v", err)
	}
	defer serverSession.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "test",
	}, nil)

	clientSession, err := mcpClient.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("Client connect: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s) error: %v", toolName, err)
	}

	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text, result.IsError
		}
	}
	t.Fatal("no text content in result")
	return "", false
}

// --- list_flux_kustomizations tests ---

func TestListFluxKustomizations(t *testing.T) {
	objs := []client.Object{
		&kustomizev1.Kustomization{
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
				LastAppliedRevision: "main@sha1:abc123def456",
			},
		},
		&kustomizev1.Kustomization{
			ObjectMeta: metav1.ObjectMeta{Name: "monitoring", Namespace: "flux-system"},
			Spec: kustomizev1.KustomizationSpec{
				Path: "./monitoring",
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind: "GitRepository",
					Name: "flux-system",
				},
				Suspend: true,
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "list_flux_kustomizations", map[string]any{"namespace": "flux-system"})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "app") {
		t.Error("expected 'app' in output")
	}
	if !strings.Contains(text, "monitoring") {
		t.Error("expected 'monitoring' in output")
	}
	if !strings.Contains(text, "Ready") {
		t.Error("expected 'Ready' status in output")
	}
	if !strings.Contains(text, "Suspended") {
		t.Error("expected 'Suspended' status in output")
	}
	if !strings.Contains(text, "Found 2 Kustomizations") {
		t.Error("expected count in output")
	}
}

func TestListFluxKustomizations_Empty(t *testing.T) {
	text, isErr := callFluxTool(t, nil, nil, "list_flux_kustomizations", nil)
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "(none)") {
		t.Error("expected '(none)' for empty list")
	}
}

// --- list_flux_helm_releases tests ---

func TestListFluxHelmReleases(t *testing.T) {
	objs := []client.Object{
		&helmv2.HelmRelease{
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
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "list_flux_helm_releases", map[string]any{"namespace": "default"})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "nginx") {
		t.Error("expected 'nginx' in output")
	}
	if !strings.Contains(text, "Found 1 HelmReleases") {
		t.Error("expected count in output")
	}
}

// --- list_flux_sources tests ---

func TestListFluxSources(t *testing.T) {
	objs := []client.Object{
		&sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "flux-system", Namespace: "flux-system"},
			Spec: sourcev1.GitRepositorySpec{
				URL: "https://github.com/org/repo",
			},
			Status: sourcev1.GitRepositoryStatus{
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
		&sourcev1.HelmRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "bitnami", Namespace: "flux-system"},
			Spec: sourcev1.HelmRepositorySpec{
				URL: "https://charts.bitnami.com/bitnami",
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "list_flux_sources", nil)
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "GitRepository") {
		t.Error("expected 'GitRepository' in output")
	}
	if !strings.Contains(text, "HelmRepository") {
		t.Error("expected 'HelmRepository' in output")
	}
	if !strings.Contains(text, "Found 2 sources") {
		t.Error("expected count in output")
	}
}

func TestListFluxSources_Filtered(t *testing.T) {
	objs := []client.Object{
		&sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "repo", Namespace: "flux-system"},
			Spec:       sourcev1.GitRepositorySpec{URL: "https://github.com/org/repo"},
		},
		&sourcev1.HelmRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "bitnami", Namespace: "flux-system"},
			Spec:       sourcev1.HelmRepositorySpec{URL: "https://charts.bitnami.com"},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "list_flux_sources", map[string]any{"source_type": "git"})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "GitRepository") {
		t.Error("expected 'GitRepository' in output")
	}
	if strings.Contains(text, "HelmRepository") {
		t.Error("did not expect 'HelmRepository' when filtered to git")
	}
}

// --- list_flux_image_policies tests ---

func TestListFluxImagePolicies(t *testing.T) {
	objs := []client.Object{
		&imagev1beta2.ImageRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "flux-system"},
			Spec: imagev1beta2.ImageRepositorySpec{
				Image: "ghcr.io/org/myapp",
			},
		},
		&imagev1beta2.ImagePolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "myapp-policy", Namespace: "flux-system"},
			Spec: imagev1beta2.ImagePolicySpec{
				ImageRepositoryRef: fluxmeta.NamespacedObjectReference{
					Name: "myapp",
				},
			},
			Status: imagev1beta2.ImagePolicyStatus{
				LatestRef: &imagev1beta2.ImageRef{
					Name: "ghcr.io/org/myapp",
					Tag:  "v1.2.3",
				},
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "list_flux_image_policies", nil)
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "Image Repositories") {
		t.Error("expected 'Image Repositories' section")
	}
	if !strings.Contains(text, "Image Policies") {
		t.Error("expected 'Image Policies' section")
	}
	if !strings.Contains(text, "ghcr.io/org/myapp:v1.2.3") {
		t.Error("expected latest image in output")
	}
}

// --- diagnose_flux_kustomization tests ---

func TestDiagnoseFluxKustomization_Healthy(t *testing.T) {
	objs := []client.Object{
		&kustomizev1.Kustomization{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "flux-system", Generation: 1},
			Spec: kustomizev1.KustomizationSpec{
				Path: "./clusters/prod",
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind: "GitRepository",
					Name: "flux-system",
				},
			},
			Status: kustomizev1.KustomizationStatus{
				ObservedGeneration:  1,
				LastAppliedRevision: "main@sha1:abc123",
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue, Reason: "Succeeded"},
				},
			},
		},
		&sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "flux-system", Namespace: "flux-system", Generation: 1},
			Spec:       sourcev1.GitRepositorySpec{URL: "https://github.com/org/repo"},
			Status: sourcev1.GitRepositoryStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "diagnose_flux_kustomization", map[string]any{
		"namespace": "flux-system",
		"name":      "app",
	})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "STATUS:") {
		t.Error("expected STATUS field")
	}
	if !strings.Contains(text, "Ready") {
		t.Error("expected Ready status")
	}
	if !strings.Contains(text, "No issues found") {
		t.Error("expected healthy message")
	}
}

func TestDiagnoseFluxKustomization_Failed(t *testing.T) {
	objs := []client.Object{
		&kustomizev1.Kustomization{
			ObjectMeta: metav1.ObjectMeta{Name: "broken", Namespace: "flux-system", Generation: 2},
			Spec: kustomizev1.KustomizationSpec{
				Path: "./broken",
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind: "GitRepository",
					Name: "flux-system",
				},
			},
			Status: kustomizev1.KustomizationStatus{
				ObservedGeneration: 2,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionFalse, Reason: "BuildFailed", Message: "kustomize build failed"},
				},
			},
		},
		&sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "flux-system", Namespace: "flux-system", Generation: 1},
			Spec:       sourcev1.GitRepositorySpec{URL: "https://github.com/org/repo"},
			Status: sourcev1.GitRepositoryStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "diagnose_flux_kustomization", map[string]any{
		"namespace": "flux-system",
		"name":      "broken",
	})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "CRITICAL") {
		t.Error("expected CRITICAL finding for failed kustomization")
	}
	if !strings.Contains(text, "kustomize build failed") {
		t.Error("expected failure message")
	}
	if !strings.Contains(text, "SUGGESTED ACTIONS") {
		t.Error("expected suggested actions")
	}
}

// --- diagnose_flux_helm_release tests ---

func TestDiagnoseFluxHelmRelease_Healthy(t *testing.T) {
	objs := []client.Object{
		&helmv2.HelmRelease{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default", Generation: 1},
			Spec: helmv2.HelmReleaseSpec{
				Chart: &helmv2.HelmChartTemplate{
					Spec: helmv2.HelmChartTemplateSpec{
						Chart:   "nginx",
						Version: ">=1.0.0",
						SourceRef: helmv2.CrossNamespaceObjectReference{
							Kind: "HelmRepository",
							Name: "bitnami",
						},
					},
				},
			},
			Status: helmv2.HelmReleaseStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue, Reason: "Succeeded"},
				},
			},
		},
		&sourcev1.HelmRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "bitnami", Namespace: "default", Generation: 1},
			Spec:       sourcev1.HelmRepositorySpec{URL: "https://charts.bitnami.com/bitnami"},
			Status: sourcev1.HelmRepositoryStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "diagnose_flux_helm_release", map[string]any{
		"namespace": "default",
		"name":      "nginx",
	})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "Ready") {
		t.Error("expected Ready status")
	}
	if !strings.Contains(text, "No issues found") {
		t.Error("expected healthy message")
	}
}

// --- diagnose_flux_system tests ---

func TestDiagnoseFluxSystem(t *testing.T) {
	fluxObjs := []client.Object{
		&kustomizev1.Kustomization{
			ObjectMeta: metav1.ObjectMeta{Name: "infra", Namespace: "flux-system", Generation: 1},
			Spec: kustomizev1.KustomizationSpec{
				Path: "./infra",
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind: "GitRepository",
					Name: "flux-system",
				},
			},
			Status: kustomizev1.KustomizationStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
		&sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "flux-system", Namespace: "flux-system", Generation: 1},
			Spec:       sourcev1.GitRepositorySpec{URL: "https://github.com/org/repo"},
			Status: sourcev1.GitRepositoryStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
	}

	k8sPods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "source-controller-abc", Namespace: "flux-system"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "manager"}}},
		},
	}

	text, isErr := callFluxTool(t, fluxObjs, k8sPods, "diagnose_flux_system", nil)
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "FluxCD System Health Report") {
		t.Error("expected header")
	}
	if !strings.Contains(text, "Kustomization Health") {
		t.Error("expected Kustomization Health section")
	}
	if !strings.Contains(text, "Source Health") {
		t.Error("expected Source Health section")
	}
	if !strings.Contains(text, "mermaid") {
		t.Error("expected Mermaid diagram")
	}
}

// --- get_flux_resource_tree tests ---

func TestGetFluxResourceTree_Kustomization(t *testing.T) {
	objs := []client.Object{
		&kustomizev1.Kustomization{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "flux-system", Generation: 1},
			Spec: kustomizev1.KustomizationSpec{
				Path: "./app",
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind: "GitRepository",
					Name: "flux-system",
				},
				DependsOn: []kustomizev1.DependencyReference{
					{Name: "infra"},
				},
			},
			Status: kustomizev1.KustomizationStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
		&kustomizev1.Kustomization{
			ObjectMeta: metav1.ObjectMeta{Name: "infra", Namespace: "flux-system", Generation: 1},
			Spec: kustomizev1.KustomizationSpec{
				Path: "./infra",
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind: "GitRepository",
					Name: "flux-system",
				},
			},
			Status: kustomizev1.KustomizationStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "get_flux_resource_tree", map[string]any{
		"namespace": "flux-system",
		"name":      "app",
	})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "Resource Tree") {
		t.Error("expected Resource Tree header")
	}
	if !strings.Contains(text, "infra") {
		t.Error("expected dependency 'infra' in tree")
	}
	if !strings.Contains(text, "mermaid") {
		t.Error("expected Mermaid diagram")
	}
}

func TestGetFluxResourceTree_HelmRelease(t *testing.T) {
	objs := []client.Object{
		&helmv2.HelmRelease{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default", Generation: 1},
			Spec: helmv2.HelmReleaseSpec{
				Chart: &helmv2.HelmChartTemplate{
					Spec: helmv2.HelmChartTemplateSpec{
						Chart:   "nginx",
						Version: "1.0.0",
						SourceRef: helmv2.CrossNamespaceObjectReference{
							Kind: "HelmRepository",
							Name: "bitnami",
						},
					},
				},
			},
			Status: helmv2.HelmReleaseStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue},
				},
			},
		},
	}

	text, isErr := callFluxTool(t, objs, nil, "get_flux_resource_tree", map[string]any{
		"namespace":     "default",
		"name":          "nginx",
		"resource_kind": "HelmRelease",
	})
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "HelmRelease") {
		t.Error("expected HelmRelease in tree")
	}
	if !strings.Contains(text, "nginx") {
		t.Error("expected 'nginx' in output")
	}
}

// --- Helper tests ---

func TestTruncateRevision(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"", "<none>"},
		{"main@sha1:abc123def456789", "main@sha1:abc123def456"},
		{"short", "short"},
	}
	for _, tc := range tests {
		got := truncateRevision(tc.input)
		if got != tc.expected {
			t.Errorf("truncateRevision(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestHelmChartInfo(t *testing.T) {
	hr := &helmv2.HelmRelease{
		Spec: helmv2.HelmReleaseSpec{
			Chart: &helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   "nginx",
					Version: ">=1.0.0",
				},
			},
		},
	}
	chart, version := helmChartInfo(hr)
	if chart != "nginx" {
		t.Errorf("expected chart 'nginx', got %q", chart)
	}
	if version != ">=1.0.0" {
		t.Errorf("expected version '>=1.0.0', got %q", version)
	}
}

func TestSanitizeMermaidID(t *testing.T) {
	got := sanitizeMermaidID("flux-system/my-app.v2:latest")
	if strings.ContainsAny(got, "/-.:") {
		t.Errorf("sanitized ID still contains invalid chars: %q", got)
	}
}
