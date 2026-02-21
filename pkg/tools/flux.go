package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/flux"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/k8s"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

// --- input structs ---

type listFluxKustomizationsInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Namespace (empty for all namespaces)"`
}

type listFluxHelmReleasesInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Namespace (empty for all namespaces)"`
}

type listFluxSourcesInput struct {
	Namespace  string `json:"namespace,omitempty" jsonschema:"Namespace (empty for all namespaces)"`
	SourceType string `json:"source_type,omitempty" jsonschema:"Filter by source type: git, oci, helm, helmchart, bucket (empty for all)"`
}

type listFluxImagePoliciesInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Namespace (empty for all namespaces)"`
}

type diagnoseFluxKustomizationInput struct {
	Namespace string `json:"namespace" jsonschema:"required,Kubernetes namespace"`
	Name      string `json:"name" jsonschema:"required,Kustomization name"`
}

type diagnoseFluxHelmReleaseInput struct {
	Namespace string `json:"namespace" jsonschema:"required,Kubernetes namespace"`
	Name      string `json:"name" jsonschema:"required,HelmRelease name"`
}

type diagnoseFluxSystemInput struct{}

type getFluxResourceTreeInput struct {
	Namespace    string `json:"namespace" jsonschema:"required,Kubernetes namespace"`
	Name         string `json:"name" jsonschema:"required,Resource name"`
	ResourceKind string `json:"resource_kind,omitempty" jsonschema:"Resource kind: Kustomization or HelmRelease (default: Kustomization)"`
}

// registerFluxTools registers all 8 FluxCD diagnostic tools.
func registerFluxTools(server *mcp.Server, fluxClient *flux.FluxClient, k8sClient *k8s.ClusterClient) {
	registerListFluxKustomizations(server, fluxClient)
	registerListFluxHelmReleases(server, fluxClient)
	registerListFluxSources(server, fluxClient)
	registerListFluxImagePolicies(server, fluxClient)
	registerDiagnoseFluxKustomization(server, fluxClient, k8sClient)
	registerDiagnoseFluxHelmRelease(server, fluxClient, k8sClient)
	registerDiagnoseFluxSystem(server, fluxClient, k8sClient)
	registerGetFluxResourceTree(server, fluxClient)
}

// --- Tool 1: list_flux_kustomizations ---

func registerListFluxKustomizations(server *mcp.Server, fluxClient *flux.FluxClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_flux_kustomizations",
		Description: "List FluxCD Kustomizations with reconciliation status, source reference, applied revision, and suspend state. Use this to see what Flux is deploying and whether reconciliation is healthy.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listFluxKustomizationsInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)

		items, err := fluxClient.ListKustomizations(ctx, ns)
		if err != nil {
			return handleFluxError("listing Flux Kustomizations", err), nil, nil
		}

		headers := []string{"NAME", "NAMESPACE", "SOURCE", "PATH", "STATUS", "REVISION", "SUSPENDED", "AGE"}
		rows := make([][]string, 0, len(items))
		for i := range items {
			ks := &items[i]
			sourceRef := fmt.Sprintf("%s/%s", ks.Spec.SourceRef.Kind, ks.Spec.SourceRef.Name)
			health := flux.KustomizationHealth(ks)
			revision := ks.Status.LastAppliedRevision
			if revision == "" {
				revision = "<none>"
			}
			suspended := "false"
			if ks.Spec.Suspend {
				suspended = "true"
			}
			rows = append(rows, []string{
				ks.Name,
				ks.Namespace,
				sourceRef,
				ks.Spec.Path,
				string(health),
				truncateRevision(revision),
				suspended,
				util.FormatAge(ks.CreationTimestamp.Time),
			})
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Flux Kustomizations (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n")
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("Kustomizations", len(items))))

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Tool 2: list_flux_helm_releases ---

func registerListFluxHelmReleases(server *mcp.Server, fluxClient *flux.FluxClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_flux_helm_releases",
		Description: "List FluxCD HelmReleases with chart, version, reconciliation status, and remediation config. Use this to see Helm-based deployments managed by Flux.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listFluxHelmReleasesInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)

		items, err := fluxClient.ListHelmReleases(ctx, ns)
		if err != nil {
			return handleFluxError("listing Flux HelmReleases", err), nil, nil
		}

		headers := []string{"NAME", "NAMESPACE", "CHART", "VERSION", "STATUS", "REMEDIATION", "SUSPENDED", "AGE"}
		rows := make([][]string, 0, len(items))
		for i := range items {
			hr := &items[i]
			chart, version := helmChartInfo(hr)
			health := flux.HelmReleaseHealth(hr)
			remediation := helmRemediationSummary(hr)
			suspended := "false"
			if hr.Spec.Suspend {
				suspended = "true"
			}
			rows = append(rows, []string{
				hr.Name,
				hr.Namespace,
				chart,
				version,
				string(health),
				remediation,
				suspended,
				util.FormatAge(hr.CreationTimestamp.Time),
			})
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Flux HelmReleases (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n")
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("HelmReleases", len(items))))

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Tool 3: list_flux_sources ---

func registerListFluxSources(server *mcp.Server, fluxClient *flux.FluxClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_flux_sources",
		Description: "List FluxCD source objects — GitRepositories, OCIRepositories, HelmRepositories, HelmCharts, and Buckets. Filter by source_type (git/oci/helm/helmchart/bucket). Use this to see where Flux pulls manifests and charts from.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listFluxSourcesInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)

		headers := []string{"TYPE", "NAME", "NAMESPACE", "URL", "REVISION", "STATUS", "AGE"}
		rows := make([][]string, 0)
		filter := strings.ToLower(input.SourceType)

		if filter == "" || filter == "git" {
			gitRepos, err := fluxClient.ListGitRepositories(ctx, ns)
			if err == nil {
				for i := range gitRepos {
					gr := &gitRepos[i]
					revision := "<none>"
					if gr.Status.Artifact != nil {
						revision = truncateRevision(gr.Status.Artifact.Revision)
					}
					health := flux.GetFluxHealth(gr.Status.Conditions, gr.Generation, gr.Status.ObservedGeneration, gr.Spec.Suspend)
					rows = append(rows, []string{
						"GitRepository", gr.Name, gr.Namespace, gr.Spec.URL, revision, string(health), util.FormatAge(gr.CreationTimestamp.Time),
					})
				}
			}
		}

		if filter == "" || filter == "oci" {
			ociRepos, err := fluxClient.ListOCIRepositories(ctx, ns)
			if err == nil {
				for i := range ociRepos {
					or := &ociRepos[i]
					revision := "<none>"
					if or.Status.Artifact != nil {
						revision = truncateRevision(or.Status.Artifact.Revision)
					}
					health := flux.GetFluxHealth(or.Status.Conditions, or.Generation, or.Status.ObservedGeneration, or.Spec.Suspend)
					rows = append(rows, []string{
						"OCIRepository", or.Name, or.Namespace, or.Spec.URL, revision, string(health), util.FormatAge(or.CreationTimestamp.Time),
					})
				}
			}
		}

		if filter == "" || filter == "helm" {
			helmRepos, err := fluxClient.ListHelmRepositories(ctx, ns)
			if err == nil {
				for i := range helmRepos {
					hr := &helmRepos[i]
					revision := "<none>"
					if hr.Status.Artifact != nil {
						revision = truncateRevision(hr.Status.Artifact.Revision)
					}
					health := flux.GetFluxHealth(hr.Status.Conditions, hr.Generation, hr.Status.ObservedGeneration, hr.Spec.Suspend)
					rows = append(rows, []string{
						"HelmRepository", hr.Name, hr.Namespace, hr.Spec.URL, revision, string(health), util.FormatAge(hr.CreationTimestamp.Time),
					})
				}
			}
		}

		if filter == "" || filter == "helmchart" {
			helmCharts, err := fluxClient.ListHelmCharts(ctx, ns)
			if err == nil {
				for i := range helmCharts {
					hc := &helmCharts[i]
					revision := "<none>"
					if hc.Status.Artifact != nil {
						revision = truncateRevision(hc.Status.Artifact.Revision)
					}
					health := flux.GetFluxHealth(hc.Status.Conditions, hc.Generation, hc.Status.ObservedGeneration, hc.Spec.Suspend)
					rows = append(rows, []string{
						"HelmChart", hc.Name, hc.Namespace, hc.Spec.Chart, revision, string(health), util.FormatAge(hc.CreationTimestamp.Time),
					})
				}
			}
		}

		if filter == "" || filter == "bucket" {
			buckets, err := fluxClient.ListBuckets(ctx, ns)
			if err == nil {
				for i := range buckets {
					b := &buckets[i]
					revision := "<none>"
					if b.Status.Artifact != nil {
						revision = truncateRevision(b.Status.Artifact.Revision)
					}
					health := flux.GetFluxHealth(b.Status.Conditions, b.Generation, b.Status.ObservedGeneration, b.Spec.Suspend)
					rows = append(rows, []string{
						"Bucket", b.Name, b.Namespace, b.Spec.Endpoint, revision, string(health), util.FormatAge(b.CreationTimestamp.Time),
					})
				}
			}
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Flux Sources (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n")
		if input.SourceType != "" {
			sb.WriteString(fmt.Sprintf("Filter: %s\n", input.SourceType))
		}
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("sources", len(rows))))

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Tool 4: list_flux_image_policies ---

func registerListFluxImagePolicies(server *mcp.Server, fluxClient *flux.FluxClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_flux_image_policies",
		Description: "List FluxCD ImageRepositories and ImagePolicies for image automation. Shows which container images Flux scans and the policies selecting versions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listFluxImagePoliciesInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)
		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Flux Image Automation (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n\n")

		// ImageRepositories
		sb.WriteString(util.FormatSubHeader("Image Repositories"))
		sb.WriteString("\n")
		imageRepos, err := fluxClient.ListImageRepositories(ctx, ns)
		if err != nil {
			return handleFluxError("listing Flux ImageRepositories", err), nil, nil
		}
		irHeaders := []string{"NAME", "NAMESPACE", "IMAGE", "LATEST TAG", "STATUS", "AGE"}
		irRows := make([][]string, 0, len(imageRepos))
		for i := range imageRepos {
			ir := &imageRepos[i]
			health := flux.GetFluxHealth(ir.Status.Conditions, ir.Generation, ir.Status.ObservedGeneration, ir.Spec.Suspend)
			latestTag := "<none>"
			if ir.Status.LastScanResult != nil && ir.Status.LastScanResult.LatestTags != nil && len(ir.Status.LastScanResult.LatestTags) > 0 {
				latestTag = ir.Status.LastScanResult.LatestTags[0]
			}
			irRows = append(irRows, []string{
				ir.Name, ir.Namespace, ir.Spec.Image, latestTag, string(health), util.FormatAge(ir.CreationTimestamp.Time),
			})
		}
		sb.WriteString(util.FormatTable(irHeaders, irRows))

		// ImagePolicies
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Image Policies"))
		sb.WriteString("\n")
		policies, err := fluxClient.ListImagePolicies(ctx, ns)
		if err != nil {
			return handleFluxError("listing Flux ImagePolicies", err), nil, nil
		}
		ipHeaders := []string{"NAME", "NAMESPACE", "IMAGE REPO", "LATEST IMAGE", "STATUS", "AGE"}
		ipRows := make([][]string, 0, len(policies))
		for i := range policies {
			ip := &policies[i]
			health := flux.GetFluxHealth(ip.Status.Conditions, ip.Generation, ip.Status.ObservedGeneration, false)
			latestImage := "<none>"
			if ip.Status.LatestRef != nil {
				latestImage = fmt.Sprintf("%s:%s", ip.Status.LatestRef.Name, ip.Status.LatestRef.Tag)
			}
			repoRef := ip.Spec.ImageRepositoryRef.Name
			ipRows = append(ipRows, []string{
				ip.Name, ip.Namespace, repoRef, latestImage, string(health), util.FormatAge(ip.CreationTimestamp.Time),
			})
		}
		sb.WriteString(util.FormatTable(ipHeaders, ipRows))

		sb.WriteString(fmt.Sprintf("\nTotal: %d image repositories, %d image policies\n", len(imageRepos), len(policies)))

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Tool 5: diagnose_flux_kustomization ---

func registerDiagnoseFluxKustomization(server *mcp.Server, fluxClient *flux.FluxClient, k8sClient *k8s.ClusterClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "diagnose_flux_kustomization",
		Description: "Deep diagnosis of a FluxCD Kustomization. Checks reconciliation status, source health, dependency chain, managed resources from inventory, and recent events. Use this when a Kustomization is failing or stuck.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input diagnoseFluxKustomizationInput) (*mcp.CallToolResult, any, error) {
		ks, err := fluxClient.GetKustomization(ctx, input.Namespace, input.Name)
		if err != nil {
			return handleFluxError(fmt.Sprintf("getting Kustomization %s/%s", input.Namespace, input.Name), err), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Flux Kustomization Diagnosis: %s (namespace: %s)", ks.Name, ks.Namespace)))
		sb.WriteString("\n\n")

		// Basic info
		health := flux.KustomizationHealth(ks)
		sb.WriteString(util.FormatKeyValue("STATUS", string(health)))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("SOURCE", fmt.Sprintf("%s/%s", ks.Spec.SourceRef.Kind, ks.Spec.SourceRef.Name)))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("PATH", ks.Spec.Path))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("INTERVAL", ks.Spec.Interval.Duration.String()))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("SUSPENDED", fmt.Sprintf("%v", ks.Spec.Suspend)))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("APPLIED REVISION", valueOrNone(ks.Status.LastAppliedRevision)))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("ATTEMPTED REVISION", valueOrNone(ks.Status.LastAttemptedRevision)))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("AGE", util.FormatAge(ks.CreationTimestamp.Time)))
		sb.WriteString("\n")

		// Conditions
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Conditions"))
		sb.WriteString("\n")
		for _, c := range ks.Status.Conditions {
			sb.WriteString(fmt.Sprintf("  %-15s %-6s  %s", c.Type, c.Status, c.Message))
			if c.Reason != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", c.Reason))
			}
			sb.WriteString("\n")
		}

		// Findings
		sb.WriteString("\nFINDINGS:\n")
		findings := 0

		if ks.Spec.Suspend {
			sb.WriteString(util.FormatFinding("INFO", "Kustomization is suspended — reconciliation paused"))
			sb.WriteString("\n")
			findings++
		}

		if health == flux.HealthFailed {
			msg := flux.GetConditionMessage(ks.Status.Conditions, fluxmeta.ReadyCondition)
			sb.WriteString(util.FormatFinding("CRITICAL", fmt.Sprintf("Reconciliation failed: %s", msg)))
			sb.WriteString("\n")
			findings++
		}

		if health == flux.HealthStalled {
			msg := flux.GetConditionMessage(ks.Status.Conditions, fluxmeta.StalledCondition)
			sb.WriteString(util.FormatFinding("CRITICAL", fmt.Sprintf("Reconciliation stalled: %s", msg)))
			sb.WriteString("\n")
			findings++
		}

		if ks.Status.LastAppliedRevision != ks.Status.LastAttemptedRevision && ks.Status.LastAttemptedRevision != "" {
			sb.WriteString(util.FormatFinding("WARNING", fmt.Sprintf("Applied revision (%s) differs from attempted revision (%s)",
				truncateRevision(ks.Status.LastAppliedRevision), truncateRevision(ks.Status.LastAttemptedRevision))))
			sb.WriteString("\n")
			findings++
		}

		// Check source health
		sourceHealth := checkSourceHealth(ctx, fluxClient, ks.Spec.SourceRef.Kind, ks.Spec.SourceRef.Name, resolveNamespace(ks.Spec.SourceRef.Namespace, ks.Namespace))
		if sourceHealth != "" {
			sb.WriteString(sourceHealth)
			findings++
		}

		// Check dependencies
		if len(ks.Spec.DependsOn) > 0 {
			sb.WriteString("\n")
			sb.WriteString(util.FormatSubHeader("Dependencies"))
			sb.WriteString("\n")
			for _, dep := range ks.Spec.DependsOn {
				depNS := dep.Namespace
				if depNS == "" {
					depNS = ks.Namespace
				}
				depKs, err := fluxClient.GetKustomization(ctx, depNS, dep.Name)
				if err != nil {
					sb.WriteString(fmt.Sprintf("  %s/%s: (could not fetch: %v)\n", depNS, dep.Name, err))
					findings++
				} else {
					depHealth := flux.KustomizationHealth(depKs)
					sb.WriteString(fmt.Sprintf("  %s/%s: %s\n", depNS, dep.Name, depHealth))
					if depHealth != flux.HealthReady {
						sb.WriteString(fmt.Sprintf("  %s\n", util.FormatFinding("WARNING", fmt.Sprintf("Dependency %s is not Ready", dep.Name))))
						findings++
					}
				}
			}
		}

		// Inventory
		if ks.Status.Inventory != nil && len(ks.Status.Inventory.Entries) > 0 {
			sb.WriteString("\n")
			sb.WriteString(util.FormatSubHeader(fmt.Sprintf("Managed Resources (%d)", len(ks.Status.Inventory.Entries))))
			sb.WriteString("\n")
			limit := 20
			for i, entry := range ks.Status.Inventory.Entries {
				if i >= limit {
					sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(ks.Status.Inventory.Entries)-limit))
					break
				}
				sb.WriteString(fmt.Sprintf("  %s (v%s)\n", entry.ID, entry.Version))
			}
		}

		// Events from K8s
		if k8sClient != nil {
			events, err := k8sClient.GetEventsForObject(ctx, ks.Namespace, ks.Name)
			if err == nil && len(events) > 0 {
				sb.WriteString("\n")
				sb.WriteString(util.FormatSubHeader("Recent Events"))
				sb.WriteString("\n")
				for _, e := range events {
					sb.WriteString(fmt.Sprintf("  %-8s %-25s %s", e.Type, e.Reason, e.Message))
					if e.Count > 1 {
						sb.WriteString(fmt.Sprintf(" (x%d)", e.Count))
					}
					sb.WriteString("\n")
				}
			}
		}

		if findings == 0 {
			sb.WriteString("  No issues found — Kustomization appears healthy.\n")
		}

		// Suggested actions
		sb.WriteString("\nSUGGESTED ACTIONS:\n")
		actionNum := 1
		if health == flux.HealthFailed {
			reason := flux.GetConditionReason(ks.Status.Conditions, fluxmeta.ReadyCondition)
			switch reason {
			case "BuildFailed":
				sb.WriteString(fmt.Sprintf("%d. Check the Kustomize overlay at path '%s' for YAML/kustomization errors\n", actionNum, ks.Spec.Path))
				actionNum++
			case "HealthCheckFailed":
				sb.WriteString(fmt.Sprintf("%d. Inspect managed resources for readiness issues (use diagnose_pod on failing pods)\n", actionNum))
				actionNum++
			case "DependencyNotReady":
				sb.WriteString(fmt.Sprintf("%d. Fix failing dependencies before this Kustomization can reconcile\n", actionNum))
				actionNum++
			default:
				sb.WriteString(fmt.Sprintf("%d. Check Flux controller logs for more details\n", actionNum))
				actionNum++
			}
		}
		if ks.Spec.Suspend {
			sb.WriteString(fmt.Sprintf("%d. Resume reconciliation: flux resume kustomization %s -n %s\n", actionNum, ks.Name, ks.Namespace))
			actionNum++
		}
		if actionNum == 1 {
			sb.WriteString("  No specific actions needed — Kustomization is healthy.\n")
		}

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Tool 6: diagnose_flux_helm_release ---

func registerDiagnoseFluxHelmRelease(server *mcp.Server, fluxClient *flux.FluxClient, k8sClient *k8s.ClusterClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "diagnose_flux_helm_release",
		Description: "Deep diagnosis of a FluxCD HelmRelease. Checks reconciliation status, chart source health, release history, remediation config, and recent events. Use this when a HelmRelease is failing.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input diagnoseFluxHelmReleaseInput) (*mcp.CallToolResult, any, error) {
		hr, err := fluxClient.GetHelmRelease(ctx, input.Namespace, input.Name)
		if err != nil {
			return handleFluxError(fmt.Sprintf("getting HelmRelease %s/%s", input.Namespace, input.Name), err), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Flux HelmRelease Diagnosis: %s (namespace: %s)", hr.Name, hr.Namespace)))
		sb.WriteString("\n\n")

		health := flux.HelmReleaseHealth(hr)
		chart, version := helmChartInfo(hr)

		sb.WriteString(util.FormatKeyValue("STATUS", string(health)))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("CHART", chart))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("VERSION", version))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("INTERVAL", hr.Spec.Interval.Duration.String()))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("SUSPENDED", fmt.Sprintf("%v", hr.Spec.Suspend)))
		sb.WriteString("\n")
		sb.WriteString(util.FormatKeyValue("AGE", util.FormatAge(hr.CreationTimestamp.Time)))
		sb.WriteString("\n")

		// Remediation config
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Remediation Config"))
		sb.WriteString("\n")
		if hr.Spec.Install != nil {
			retries := hr.Spec.Install.Remediation.GetRetries()
			sb.WriteString(fmt.Sprintf("  Install retries: %d\n", retries))
		}
		if hr.Spec.Upgrade != nil && hr.Spec.Upgrade.Remediation != nil {
			retries := hr.Spec.Upgrade.Remediation.GetRetries()
			sb.WriteString(fmt.Sprintf("  Upgrade retries: %d\n", retries))
			if hr.Spec.Upgrade.Remediation.Strategy != nil {
				sb.WriteString(fmt.Sprintf("  Upgrade strategy: %s\n", *hr.Spec.Upgrade.Remediation.Strategy))
			}
		}

		// Conditions
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Conditions"))
		sb.WriteString("\n")
		for _, c := range hr.Status.Conditions {
			sb.WriteString(fmt.Sprintf("  %-15s %-6s  %s", c.Type, c.Status, c.Message))
			if c.Reason != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", c.Reason))
			}
			sb.WriteString("\n")
		}

		// Findings
		sb.WriteString("\nFINDINGS:\n")
		findings := 0

		if hr.Spec.Suspend {
			sb.WriteString(util.FormatFinding("INFO", "HelmRelease is suspended — reconciliation paused"))
			sb.WriteString("\n")
			findings++
		}

		if health == flux.HealthFailed {
			msg := flux.GetConditionMessage(hr.Status.Conditions, fluxmeta.ReadyCondition)
			sb.WriteString(util.FormatFinding("CRITICAL", fmt.Sprintf("Reconciliation failed: %s", msg)))
			sb.WriteString("\n")
			findings++
		}

		if health == flux.HealthStalled {
			msg := flux.GetConditionMessage(hr.Status.Conditions, fluxmeta.StalledCondition)
			sb.WriteString(util.FormatFinding("CRITICAL", fmt.Sprintf("Reconciliation stalled: %s", msg)))
			sb.WriteString("\n")
			findings++
		}

		// Check Released condition
		releasedMsg := flux.GetConditionMessage(hr.Status.Conditions, "Released")
		releasedReason := flux.GetConditionReason(hr.Status.Conditions, "Released")
		if releasedReason != "" && releasedReason != "Succeeded" {
			sb.WriteString(util.FormatFinding("WARNING", fmt.Sprintf("Release issue: %s — %s", releasedReason, releasedMsg)))
			sb.WriteString("\n")
			findings++
		}

		// Check test condition
		testMsg := flux.GetConditionMessage(hr.Status.Conditions, "TestSuccess")
		testReason := flux.GetConditionReason(hr.Status.Conditions, "TestSuccess")
		if testReason == "Failed" {
			sb.WriteString(util.FormatFinding("WARNING", fmt.Sprintf("Helm tests failed: %s", testMsg)))
			sb.WriteString("\n")
			findings++
		}

		// Release history
		if len(hr.Status.History) > 0 {
			sb.WriteString("\n")
			sb.WriteString(util.FormatSubHeader("Release History"))
			sb.WriteString("\n")
			for _, snap := range hr.Status.History {
				sb.WriteString(fmt.Sprintf("  v%d: %s (chart: %s, app: %s)\n",
					snap.Version, snap.Status, snap.ChartVersion, snap.AppVersion))
			}
		}

		// Check chart source
		if hr.Spec.Chart != nil {
			sourceKind := "HelmRepository"
			sourceName := ""
			sourceNS := hr.Namespace
			if hr.Spec.Chart.Spec.SourceRef.Kind != "" {
				sourceKind = hr.Spec.Chart.Spec.SourceRef.Kind
			}
			sourceName = hr.Spec.Chart.Spec.SourceRef.Name
			if hr.Spec.Chart.Spec.SourceRef.Namespace != "" {
				sourceNS = hr.Spec.Chart.Spec.SourceRef.Namespace
			}
			sourceCheck := checkSourceHealth(ctx, fluxClient, sourceKind, sourceName, sourceNS)
			if sourceCheck != "" {
				sb.WriteString(sourceCheck)
				findings++
			}
		}

		// Events
		if k8sClient != nil {
			events, err := k8sClient.GetEventsForObject(ctx, hr.Namespace, hr.Name)
			if err == nil && len(events) > 0 {
				sb.WriteString("\n")
				sb.WriteString(util.FormatSubHeader("Recent Events"))
				sb.WriteString("\n")
				for _, e := range events {
					sb.WriteString(fmt.Sprintf("  %-8s %-25s %s", e.Type, e.Reason, e.Message))
					if e.Count > 1 {
						sb.WriteString(fmt.Sprintf(" (x%d)", e.Count))
					}
					sb.WriteString("\n")
				}
			}
		}

		if findings == 0 {
			sb.WriteString("  No issues found — HelmRelease appears healthy.\n")
		}

		// Suggested actions
		sb.WriteString("\nSUGGESTED ACTIONS:\n")
		actionNum := 1
		if health == flux.HealthFailed {
			reason := flux.GetConditionReason(hr.Status.Conditions, fluxmeta.ReadyCondition)
			switch {
			case strings.Contains(reason, "Install"):
				sb.WriteString(fmt.Sprintf("%d. Check Helm chart values and templates for install errors\n", actionNum))
				actionNum++
			case strings.Contains(reason, "Upgrade"):
				sb.WriteString(fmt.Sprintf("%d. Check Helm chart changes — upgrade may have failed. Review release history above\n", actionNum))
				actionNum++
			default:
				sb.WriteString(fmt.Sprintf("%d. Check Flux helm-controller logs for more details\n", actionNum))
				actionNum++
			}
		}
		if hr.Spec.Suspend {
			sb.WriteString(fmt.Sprintf("%d. Resume reconciliation: flux resume helmrelease %s -n %s\n", actionNum, hr.Name, hr.Namespace))
			actionNum++
		}
		if actionNum == 1 {
			sb.WriteString("  No specific actions needed — HelmRelease is healthy.\n")
		}

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Tool 7: diagnose_flux_system ---

func registerDiagnoseFluxSystem(server *mcp.Server, fluxClient *flux.FluxClient, k8sClient *k8s.ClusterClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "diagnose_flux_system",
		Description: "Comprehensive FluxCD system health check. Checks flux-system pods, tallies Kustomization/HelmRelease/Source health across the cluster, lists warning events, and generates a Mermaid topology diagram. Use this for a broad Flux health overview.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input diagnoseFluxSystemInput) (*mcp.CallToolResult, any, error) {
		var sb strings.Builder
		sb.WriteString(util.FormatHeader("FluxCD System Health Report"))
		sb.WriteString("\n\n")

		findings := 0

		// 1. Flux controller pods
		sb.WriteString(util.FormatSubHeader("Flux Controllers (flux-system namespace)"))
		sb.WriteString("\n")
		if k8sClient != nil {
			pods, err := k8sClient.ListPods(ctx, "flux-system", metav1.ListOptions{})
			if err != nil {
				sb.WriteString(fmt.Sprintf("  (could not list pods: %v)\n", err))
			} else if len(pods) == 0 {
				sb.WriteString(util.FormatFinding("CRITICAL", "No pods found in flux-system namespace — FluxCD may not be installed"))
				sb.WriteString("\n")
				findings++
			} else {
				healthy := 0
				for i := range pods {
					if isPodHealthy(&pods[i]) {
						healthy++
					}
				}
				sb.WriteString(fmt.Sprintf("  Pods: %d/%d healthy\n", healthy, len(pods)))
				if healthy < len(pods) {
					for i := range pods {
						p := &pods[i]
						if !isPodHealthy(p) {
							sb.WriteString(fmt.Sprintf("  %s\n", util.FormatFinding("CRITICAL", fmt.Sprintf("Controller pod '%s' is unhealthy: %s", p.Name, podPhaseReason(p)))))
							findings++
						}
					}
				}
			}
		}

		// 2. Kustomization health tally
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Kustomization Health"))
		sb.WriteString("\n")
		ksTally := map[flux.FluxHealthStatus]int{}
		ksList, err := fluxClient.ListKustomizations(ctx, "")
		if err != nil {
			sb.WriteString(fmt.Sprintf("  (could not list: %v)\n", err))
		} else {
			for i := range ksList {
				h := flux.KustomizationHealth(&ksList[i])
				ksTally[h]++
			}
			sb.WriteString(fmt.Sprintf("  Total: %d  ", len(ksList)))
			writeHealthTally(&sb, ksTally)
			sb.WriteString("\n")
			failedCount := ksTally[flux.HealthFailed] + ksTally[flux.HealthStalled]
			if failedCount > 0 {
				sb.WriteString(fmt.Sprintf("  %s\n", util.FormatFinding("WARNING", fmt.Sprintf("%d Kustomizations not healthy", failedCount))))
				for i := range ksList {
					h := flux.KustomizationHealth(&ksList[i])
					if h == flux.HealthFailed || h == flux.HealthStalled {
						sb.WriteString(fmt.Sprintf("    - %s/%s: %s\n", ksList[i].Namespace, ksList[i].Name, h))
					}
				}
				findings++
			}
		}

		// 3. HelmRelease health tally
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("HelmRelease Health"))
		sb.WriteString("\n")
		hrTally := map[flux.FluxHealthStatus]int{}
		hrList, err := fluxClient.ListHelmReleases(ctx, "")
		if err != nil {
			sb.WriteString(fmt.Sprintf("  (could not list: %v)\n", err))
		} else {
			for i := range hrList {
				h := flux.HelmReleaseHealth(&hrList[i])
				hrTally[h]++
			}
			sb.WriteString(fmt.Sprintf("  Total: %d  ", len(hrList)))
			writeHealthTally(&sb, hrTally)
			sb.WriteString("\n")
			failedCount := hrTally[flux.HealthFailed] + hrTally[flux.HealthStalled]
			if failedCount > 0 {
				sb.WriteString(fmt.Sprintf("  %s\n", util.FormatFinding("WARNING", fmt.Sprintf("%d HelmReleases not healthy", failedCount))))
				for i := range hrList {
					h := flux.HelmReleaseHealth(&hrList[i])
					if h == flux.HealthFailed || h == flux.HealthStalled {
						sb.WriteString(fmt.Sprintf("    - %s/%s: %s\n", hrList[i].Namespace, hrList[i].Name, h))
					}
				}
				findings++
			}
		}

		// 4. Source health tally
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Source Health"))
		sb.WriteString("\n")
		srcCount := 0
		srcFailed := 0
		gitRepos, err := fluxClient.ListGitRepositories(ctx, "")
		if err == nil {
			for i := range gitRepos {
				srcCount++
				h := flux.GetFluxHealth(gitRepos[i].Status.Conditions, gitRepos[i].Generation, gitRepos[i].Status.ObservedGeneration, gitRepos[i].Spec.Suspend)
				if h == flux.HealthFailed || h == flux.HealthStalled {
					srcFailed++
					sb.WriteString(fmt.Sprintf("  %s\n", util.FormatFinding("WARNING", fmt.Sprintf("GitRepository %s/%s: %s", gitRepos[i].Namespace, gitRepos[i].Name, h))))
				}
			}
		}
		helmRepos, err := fluxClient.ListHelmRepositories(ctx, "")
		if err == nil {
			for i := range helmRepos {
				srcCount++
				h := flux.GetFluxHealth(helmRepos[i].Status.Conditions, helmRepos[i].Generation, helmRepos[i].Status.ObservedGeneration, helmRepos[i].Spec.Suspend)
				if h == flux.HealthFailed || h == flux.HealthStalled {
					srcFailed++
					sb.WriteString(fmt.Sprintf("  %s\n", util.FormatFinding("WARNING", fmt.Sprintf("HelmRepository %s/%s: %s", helmRepos[i].Namespace, helmRepos[i].Name, h))))
				}
			}
		}
		ociRepos, err := fluxClient.ListOCIRepositories(ctx, "")
		if err == nil {
			for i := range ociRepos {
				srcCount++
				h := flux.GetFluxHealth(ociRepos[i].Status.Conditions, ociRepos[i].Generation, ociRepos[i].Status.ObservedGeneration, ociRepos[i].Spec.Suspend)
				if h == flux.HealthFailed || h == flux.HealthStalled {
					srcFailed++
					sb.WriteString(fmt.Sprintf("  %s\n", util.FormatFinding("WARNING", fmt.Sprintf("OCIRepository %s/%s: %s", ociRepos[i].Namespace, ociRepos[i].Name, h))))
				}
			}
		}
		if srcFailed > 0 {
			findings++
		}
		sb.WriteString(fmt.Sprintf("  Total sources: %d, Unhealthy: %d\n", srcCount, srcFailed))

		// 5. Warning events in flux-system
		if k8sClient != nil {
			events, err := k8sClient.ListEvents(ctx, "flux-system", metav1.ListOptions{})
			if err == nil {
				oneHourAgo := time.Now().Add(-1 * time.Hour)
				warningCount := 0
				for _, e := range events {
					eventTime := e.LastTimestamp.Time
					if eventTime.IsZero() {
						eventTime = e.CreationTimestamp.Time
					}
					if e.Type == "Warning" && eventTime.After(oneHourAgo) {
						warningCount++
					}
				}
				if warningCount > 0 {
					sb.WriteString(fmt.Sprintf("\n  %s\n", util.FormatFinding("WARNING", fmt.Sprintf("%d warning events in flux-system in the last hour", warningCount))))
					findings++
				}
			}
		}

		// Overall
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Overall Assessment"))
		sb.WriteString("\n")
		if findings == 0 {
			sb.WriteString("  FluxCD system appears healthy. No issues found.\n")
		} else {
			sb.WriteString(fmt.Sprintf("  %d issue(s) found. Review findings above.\n", findings))
		}

		// Mermaid diagram
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Topology"))
		sb.WriteString("\n")
		mermaid := generateFluxTopologyMermaid(ksList, hrList, gitRepos, helmRepos)
		sb.WriteString(util.FormatMermaidBlock(mermaid))
		sb.WriteString("\n")

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Tool 8: get_flux_resource_tree ---

func registerGetFluxResourceTree(server *mcp.Server, fluxClient *flux.FluxClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_flux_resource_tree",
		Description: "Trace a FluxCD resource's dependency tree — source, dependencies, and managed resources from inventory. Generates a text tree and Mermaid dependency graph. Use resource_kind=Kustomization (default) or resource_kind=HelmRelease.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getFluxResourceTreeInput) (*mcp.CallToolResult, any, error) {
		kind := input.ResourceKind
		if kind == "" {
			kind = "Kustomization"
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Flux Resource Tree: %s/%s (%s)", input.Namespace, input.Name, kind)))
		sb.WriteString("\n\n")

		var mermaidLines []string

		switch kind {
		case "Kustomization":
			ks, err := fluxClient.GetKustomization(ctx, input.Namespace, input.Name)
			if err != nil {
				return handleFluxError(fmt.Sprintf("getting Kustomization %s/%s", input.Namespace, input.Name), err), nil, nil
			}

			rootID := fmt.Sprintf("%s/%s", ks.Namespace, ks.Name)
			health := flux.KustomizationHealth(ks)

			// Source
			sourceRef := fmt.Sprintf("%s/%s", ks.Spec.SourceRef.Kind, ks.Spec.SourceRef.Name)
			sb.WriteString(fmt.Sprintf("Source: %s\n", sourceRef))
			sb.WriteString(fmt.Sprintf("  -> Kustomization: %s [%s]\n", rootID, health))
			mermaidLines = append(mermaidLines, fmt.Sprintf("  %s[%s] --> %s[%s\\n%s]", sanitizeMermaidID(sourceRef), sourceRef, sanitizeMermaidID(rootID), rootID, health))

			// Dependencies (depth limit 10)
			seen := map[string]bool{rootID: true}
			buildDependencyTree(ctx, fluxClient, ks, &sb, &mermaidLines, seen, 2, 10)

			// Managed resources
			if ks.Status.Inventory != nil && len(ks.Status.Inventory.Entries) > 0 {
				sb.WriteString(fmt.Sprintf("  Managed Resources (%d):\n", len(ks.Status.Inventory.Entries)))
				limit := 30
				for i, entry := range ks.Status.Inventory.Entries {
					if i >= limit {
						sb.WriteString(fmt.Sprintf("    ... and %d more\n", len(ks.Status.Inventory.Entries)-limit))
						break
					}
					sb.WriteString(fmt.Sprintf("    %s\n", entry.ID))
				}
			}

		case "HelmRelease":
			hr, err := fluxClient.GetHelmRelease(ctx, input.Namespace, input.Name)
			if err != nil {
				return handleFluxError(fmt.Sprintf("getting HelmRelease %s/%s", input.Namespace, input.Name), err), nil, nil
			}

			rootID := fmt.Sprintf("%s/%s", hr.Namespace, hr.Name)
			health := flux.HelmReleaseHealth(hr)
			chart, version := helmChartInfo(hr)

			sb.WriteString(fmt.Sprintf("Chart: %s@%s\n", chart, version))
			sb.WriteString(fmt.Sprintf("  -> HelmRelease: %s [%s]\n", rootID, health))

			if hr.Spec.Chart != nil {
				chartSourceRef := fmt.Sprintf("%s/%s", hr.Spec.Chart.Spec.SourceRef.Kind, hr.Spec.Chart.Spec.SourceRef.Name)
				mermaidLines = append(mermaidLines, fmt.Sprintf("  %s[%s] --> %s[%s\\n%s]", sanitizeMermaidID(chartSourceRef), chartSourceRef, sanitizeMermaidID(rootID), rootID, health))
			}

			// Managed resources
			if hr.Status.Inventory != nil && len(hr.Status.Inventory.Entries) > 0 {
				sb.WriteString(fmt.Sprintf("  Managed Resources (%d):\n", len(hr.Status.Inventory.Entries)))
				limit := 30
				for i, entry := range hr.Status.Inventory.Entries {
					if i >= limit {
						sb.WriteString(fmt.Sprintf("    ... and %d more\n", len(hr.Status.Inventory.Entries)-limit))
						break
					}
					sb.WriteString(fmt.Sprintf("    %s\n", entry.ID))
				}
			}

		default:
			return util.ErrorResult("Unsupported resource_kind: %s (use Kustomization or HelmRelease)", kind), nil, nil
		}

		// Mermaid diagram
		if len(mermaidLines) > 0 {
			sb.WriteString("\n")
			var mermaid strings.Builder
			mermaid.WriteString("graph LR\n")
			for _, line := range mermaidLines {
				mermaid.WriteString(line)
				mermaid.WriteString("\n")
			}
			sb.WriteString(util.FormatMermaidBlock(mermaid.String()))
			sb.WriteString("\n")
		}

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// --- Helpers ---

// handleFluxError handles errors from Flux API calls, including CRD-not-found detection.
func handleFluxError(action string, err error) *mcp.CallToolResult {
	errStr := err.Error()
	if strings.Contains(errStr, "no matches for kind") ||
		strings.Contains(errStr, "no kind is registered") ||
		strings.Contains(errStr, "the server could not find the requested resource") {
		return util.SuccessResult("FluxCD is not installed in this cluster. The required Custom Resource Definitions were not found.\n\nTo install FluxCD: https://fluxcd.io/flux/installation/")
	}
	return util.HandleK8sError(action, err)
}

// truncateRevision shortens a Flux revision string for display.
func truncateRevision(rev string) string {
	if rev == "" {
		return "<none>"
	}
	// Git revisions often have format: main@sha1:abc123...
	if idx := strings.Index(rev, ":"); idx >= 0 {
		prefix := rev[:idx+1]
		hash := rev[idx+1:]
		if len(hash) > 12 {
			hash = hash[:12]
		}
		return prefix + hash
	}
	if len(rev) > 40 {
		return rev[:40] + "..."
	}
	return rev
}

// helmChartInfo extracts chart name and version from a HelmRelease.
func helmChartInfo(hr *helmv2.HelmRelease) (chart, version string) {
	if hr.Spec.ChartRef != nil {
		return fmt.Sprintf("%s/%s", hr.Spec.ChartRef.Kind, hr.Spec.ChartRef.Name), "<chartref>"
	}
	if hr.Spec.Chart != nil {
		return hr.Spec.Chart.Spec.Chart, hr.Spec.Chart.Spec.Version
	}
	return "<unknown>", "<unknown>"
}

// helmRemediationSummary returns a brief remediation config summary.
func helmRemediationSummary(hr *helmv2.HelmRelease) string {
	parts := []string{}
	if hr.Spec.Install != nil {
		retries := hr.Spec.Install.Remediation.GetRetries()
		if retries > 0 {
			parts = append(parts, fmt.Sprintf("install:%d", retries))
		}
	}
	if hr.Spec.Upgrade != nil && hr.Spec.Upgrade.Remediation != nil {
		retries := hr.Spec.Upgrade.Remediation.GetRetries()
		if retries > 0 {
			parts = append(parts, fmt.Sprintf("upgrade:%d", retries))
		}
	}
	if len(parts) == 0 {
		return "default"
	}
	return strings.Join(parts, ", ")
}

// checkSourceHealth checks the health of a Flux source and returns a finding string or empty.
func checkSourceHealth(ctx context.Context, fluxClient *flux.FluxClient, kind, name, namespace string) string {
	var conditions []metav1.Condition
	var generation, observedGeneration int64
	var suspended bool

	switch kind {
	case "GitRepository":
		gr, err := fluxClient.GetGitRepository(ctx, namespace, name)
		if err != nil {
			return fmt.Sprintf("%s\n", util.FormatFinding("WARNING", fmt.Sprintf("Cannot fetch source %s/%s: %v", kind, name, err)))
		}
		conditions = gr.Status.Conditions
		generation = gr.Generation
		observedGeneration = gr.Status.ObservedGeneration
		suspended = gr.Spec.Suspend
	case "OCIRepository":
		or, err := fluxClient.GetOCIRepository(ctx, namespace, name)
		if err != nil {
			return fmt.Sprintf("%s\n", util.FormatFinding("WARNING", fmt.Sprintf("Cannot fetch source %s/%s: %v", kind, name, err)))
		}
		conditions = or.Status.Conditions
		generation = or.Generation
		observedGeneration = or.Status.ObservedGeneration
		suspended = or.Spec.Suspend
	case "HelmRepository":
		hr, err := fluxClient.GetHelmRepository(ctx, namespace, name)
		if err != nil {
			return fmt.Sprintf("%s\n", util.FormatFinding("WARNING", fmt.Sprintf("Cannot fetch source %s/%s: %v", kind, name, err)))
		}
		conditions = hr.Status.Conditions
		generation = hr.Generation
		observedGeneration = hr.Status.ObservedGeneration
		suspended = hr.Spec.Suspend
	case "Bucket":
		b, err := fluxClient.GetBucket(ctx, namespace, name)
		if err != nil {
			return fmt.Sprintf("%s\n", util.FormatFinding("WARNING", fmt.Sprintf("Cannot fetch source %s/%s: %v", kind, name, err)))
		}
		conditions = b.Status.Conditions
		generation = b.Generation
		observedGeneration = b.Status.ObservedGeneration
		suspended = b.Spec.Suspend
	default:
		return ""
	}

	health := flux.GetFluxHealth(conditions, generation, observedGeneration, suspended)
	if health != flux.HealthReady && health != flux.HealthSuspended {
		msg := flux.GetConditionMessage(conditions, fluxmeta.ReadyCondition)
		return fmt.Sprintf("%s\n", util.FormatFinding("WARNING", fmt.Sprintf("Source %s/%s is %s: %s", kind, name, health, msg)))
	}
	return ""
}

// resolveNamespace returns namespace or defaultNS if empty.
func resolveNamespace(namespace, defaultNS string) string {
	if namespace != "" {
		return namespace
	}
	return defaultNS
}

// valueOrNone returns the string or "<none>" if empty.
func valueOrNone(s string) string {
	if s == "" {
		return "<none>"
	}
	return s
}

// writeHealthTally writes a compact health tally to a builder.
func writeHealthTally(sb *strings.Builder, tally map[flux.FluxHealthStatus]int) {
	statuses := []flux.FluxHealthStatus{flux.HealthReady, flux.HealthReconciling, flux.HealthFailed, flux.HealthStalled, flux.HealthSuspended, flux.HealthUnknown}
	parts := []string{}
	for _, s := range statuses {
		if count, ok := tally[s]; ok && count > 0 {
			parts = append(parts, fmt.Sprintf("%s: %d", s, count))
		}
	}
	sb.WriteString(strings.Join(parts, ", "))
}

// buildDependencyTree recursively traces Kustomization dependencies.
func buildDependencyTree(ctx context.Context, fluxClient *flux.FluxClient, ks *kustomizev1.Kustomization, sb *strings.Builder, mermaidLines *[]string, seen map[string]bool, depth, maxDepth int) {
	if depth > maxDepth || len(ks.Spec.DependsOn) == 0 {
		return
	}

	indent := strings.Repeat("  ", depth)
	rootID := fmt.Sprintf("%s/%s", ks.Namespace, ks.Name)

	for _, dep := range ks.Spec.DependsOn {
		depNS := dep.Namespace
		if depNS == "" {
			depNS = ks.Namespace
		}
		depID := fmt.Sprintf("%s/%s", depNS, dep.Name)

		if seen[depID] {
			sb.WriteString(fmt.Sprintf("%s-> %s (circular ref, skipped)\n", indent, depID))
			continue
		}
		seen[depID] = true

		depKs, err := fluxClient.GetKustomization(ctx, depNS, dep.Name)
		if err != nil {
			sb.WriteString(fmt.Sprintf("%s-> %s (not found)\n", indent, depID))
			continue
		}

		depHealth := flux.KustomizationHealth(depKs)
		sb.WriteString(fmt.Sprintf("%s-> %s [%s]\n", indent, depID, depHealth))
		*mermaidLines = append(*mermaidLines, fmt.Sprintf("  %s --> %s[%s\\n%s]", sanitizeMermaidID(rootID), sanitizeMermaidID(depID), depID, depHealth))

		buildDependencyTree(ctx, fluxClient, depKs, sb, mermaidLines, seen, depth+1, maxDepth)
	}
}

// generateFluxTopologyMermaid generates a Mermaid diagram of the Flux resource topology.
func generateFluxTopologyMermaid(ksList []kustomizev1.Kustomization, hrList []helmv2.HelmRelease, gitRepos []sourcev1.GitRepository, helmRepos []sourcev1.HelmRepository) string {
	var sb strings.Builder
	sb.WriteString("graph LR\n")

	// Source nodes
	for i := range gitRepos {
		id := sanitizeMermaidID(fmt.Sprintf("git_%s_%s", gitRepos[i].Namespace, gitRepos[i].Name))
		sb.WriteString(fmt.Sprintf("  %s[\"GitRepo: %s\"]\n", id, gitRepos[i].Name))
	}
	for i := range helmRepos {
		id := sanitizeMermaidID(fmt.Sprintf("helmrepo_%s_%s", helmRepos[i].Namespace, helmRepos[i].Name))
		sb.WriteString(fmt.Sprintf("  %s[\"HelmRepo: %s\"]\n", id, helmRepos[i].Name))
	}

	// Kustomization nodes and edges
	for i := range ksList {
		ks := &ksList[i]
		ksID := sanitizeMermaidID(fmt.Sprintf("ks_%s_%s", ks.Namespace, ks.Name))
		health := flux.KustomizationHealth(ks)
		sb.WriteString(fmt.Sprintf("  %s[\"%s\\n%s\"]\n", ksID, ks.Name, health))

		srcID := sanitizeMermaidID(fmt.Sprintf("git_%s_%s", resolveNamespace(ks.Spec.SourceRef.Namespace, ks.Namespace), ks.Spec.SourceRef.Name))
		if ks.Spec.SourceRef.Kind == "OCIRepository" {
			srcID = sanitizeMermaidID(fmt.Sprintf("oci_%s_%s", resolveNamespace(ks.Spec.SourceRef.Namespace, ks.Namespace), ks.Spec.SourceRef.Name))
		} else if ks.Spec.SourceRef.Kind == "Bucket" {
			srcID = sanitizeMermaidID(fmt.Sprintf("bucket_%s_%s", resolveNamespace(ks.Spec.SourceRef.Namespace, ks.Namespace), ks.Spec.SourceRef.Name))
		}
		sb.WriteString(fmt.Sprintf("  %s --> %s\n", srcID, ksID))

		for _, dep := range ks.Spec.DependsOn {
			depNS := dep.Namespace
			if depNS == "" {
				depNS = ks.Namespace
			}
			depID := sanitizeMermaidID(fmt.Sprintf("ks_%s_%s", depNS, dep.Name))
			sb.WriteString(fmt.Sprintf("  %s -.-> %s\n", depID, ksID))
		}
	}

	// HelmRelease nodes and edges
	for i := range hrList {
		hr := &hrList[i]
		hrID := sanitizeMermaidID(fmt.Sprintf("hr_%s_%s", hr.Namespace, hr.Name))
		health := flux.HelmReleaseHealth(hr)
		sb.WriteString(fmt.Sprintf("  %s[\"%s\\n%s\"]\n", hrID, hr.Name, health))

		if hr.Spec.Chart != nil {
			srcNS := resolveNamespace(hr.Spec.Chart.Spec.SourceRef.Namespace, hr.Namespace)
			srcID := sanitizeMermaidID(fmt.Sprintf("helmrepo_%s_%s", srcNS, hr.Spec.Chart.Spec.SourceRef.Name))
			sb.WriteString(fmt.Sprintf("  %s --> %s\n", srcID, hrID))
		}
	}

	return sb.String()
}

// sanitizeMermaidID removes characters that are invalid in Mermaid node IDs.
func sanitizeMermaidID(id string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_", ".", "_", ":", "_")
	return replacer.Replace(id)
}
