package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/k8s"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

type analyzeResourceAllocationInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (empty for cluster-wide)"`
}

type listLimitRangesInput struct {
	Namespace string `json:"namespace" jsonschema:"Kubernetes namespace (use 'all' for all namespaces)"`
}

type getWorkloadDependenciesInput struct {
	Namespace    string `json:"namespace" jsonschema:"required,Kubernetes namespace"`
	WorkloadName string `json:"workload_name" jsonschema:"required,Name of the Deployment, StatefulSet, or Pod"`
	WorkloadKind string `json:"workload_kind,omitempty" jsonschema:"Kind: Deployment, StatefulSet, or Pod (default: Deployment)"`
}

func registerResourceTools(server *mcp.Server, client *k8s.ClusterClient) {
	// analyze_resource_allocation
	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze_resource_allocation",
		Description: "Analyze CPU and memory resource allocation: requests vs limits vs node allocatable capacity. If metrics-server is available, includes actual usage. Produces a Mermaid bar chart. Use namespace for namespace-scoped or leave empty for cluster-wide.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input analyzeResourceAllocationInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)
		scope := displayNS(input.Namespace)

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Resource Allocation Analysis (scope: %s)", scope)))
		sb.WriteString("\n\n")

		// Get pods
		pods, err := client.ListPods(ctx, ns, metav1.ListOptions{})
		if err != nil {
			return util.HandleK8sError("listing pods", err), nil, nil
		}

		// Sum requests and limits
		var cpuRequests, cpuLimits, memRequests, memLimits int64
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				continue
			}
			for _, c := range pod.Spec.Containers {
				if c.Resources.Requests != nil {
					cpuRequests += c.Resources.Requests.Cpu().MilliValue()
					memRequests += c.Resources.Requests.Memory().Value()
				}
				if c.Resources.Limits != nil {
					cpuLimits += c.Resources.Limits.Cpu().MilliValue()
					memLimits += c.Resources.Limits.Memory().Value()
				}
			}
		}

		sb.WriteString(util.FormatSubHeader("Resource Summary"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  Active Pods:    %d\n", len(pods)))
		sb.WriteString(fmt.Sprintf("  CPU Requests:   %dm\n", cpuRequests))
		sb.WriteString(fmt.Sprintf("  CPU Limits:     %dm\n", cpuLimits))
		sb.WriteString(fmt.Sprintf("  Memory Requests: %s\n", formatBytes(memRequests)))
		sb.WriteString(fmt.Sprintf("  Memory Limits:   %s\n", formatBytes(memLimits)))

		// Node capacity (cluster-wide or all nodes)
		var cpuAllocatable, memAllocatable int64
		nodes, err := client.ListNodes(ctx, metav1.ListOptions{})
		if err == nil && len(nodes) > 0 {
			for _, n := range nodes {
				cpuAllocatable += n.Status.Allocatable.Cpu().MilliValue()
				memAllocatable += n.Status.Allocatable.Memory().Value()
			}
			sb.WriteString(fmt.Sprintf("\n  Node Allocatable (total across %d nodes):\n", len(nodes)))
			sb.WriteString(fmt.Sprintf("    CPU:    %dm\n", cpuAllocatable))
			sb.WriteString(fmt.Sprintf("    Memory: %s\n", formatBytes(memAllocatable)))

			if cpuAllocatable > 0 {
				cpuReqPct := float64(cpuRequests) / float64(cpuAllocatable) * 100
				cpuLimPct := float64(cpuLimits) / float64(cpuAllocatable) * 100
				sb.WriteString(fmt.Sprintf("\n  CPU Utilization: requests=%.1f%%, limits=%.1f%%\n", cpuReqPct, cpuLimPct))
				if cpuReqPct > float64(util.ResourceUsageWarningPercent) {
					sb.WriteString(util.FormatFinding("WARNING", "CPU requests exceed 80% of allocatable capacity"))
					sb.WriteString("\n")
				}
				if cpuLimPct > 200 {
					sb.WriteString(util.FormatFinding("WARNING", "CPU limits are >200% of allocatable (heavy overcommit)"))
					sb.WriteString("\n")
				}
			}
			if memAllocatable > 0 {
				memReqPct := float64(memRequests) / float64(memAllocatable) * 100
				memLimPct := float64(memLimits) / float64(memAllocatable) * 100
				sb.WriteString(fmt.Sprintf("  Memory Utilization: requests=%.1f%%, limits=%.1f%%\n", memReqPct, memLimPct))
				if memReqPct > float64(util.ResourceUsageWarningPercent) {
					sb.WriteString(util.FormatFinding("WARNING", "Memory requests exceed 80% of allocatable capacity"))
					sb.WriteString("\n")
				}
			}
		}

		// Actual usage via metrics
		var cpuUsage, memUsage int64
		hasMetrics := false
		podMetrics, err := client.GetPodMetrics(ctx, ns, metav1.ListOptions{})
		if err == nil && len(podMetrics) > 0 {
			hasMetrics = true
			for _, pm := range podMetrics {
				for _, c := range pm.Containers {
					cpuUsage += c.Usage.Cpu().MilliValue()
					memUsage += c.Usage.Memory().Value()
				}
			}
			sb.WriteString(fmt.Sprintf("\n  Actual Usage (from metrics-server):\n"))
			sb.WriteString(fmt.Sprintf("    CPU:    %dm\n", cpuUsage))
			sb.WriteString(fmt.Sprintf("    Memory: %s\n", formatBytes(memUsage)))
		}

		// Mermaid chart
		sb.WriteString("\nRESOURCE ALLOCATION CHART:\n")
		var mermaidLines []string
		mermaidLines = append(mermaidLines, "xychart-beta")
		mermaidLines = append(mermaidLines, fmt.Sprintf("    title \"Resource Allocation (%s)\"", scope))

		if hasMetrics {
			mermaidLines = append(mermaidLines, "    x-axis [\"CPU (m)\", \"Memory (Mi)\"]")
			cpuRow := fmt.Sprintf("    bar [%d, %d]", cpuRequests, memRequests/(1024*1024))
			mermaidLines = append(mermaidLines, cpuRow)
		} else {
			mermaidLines = append(mermaidLines, "    x-axis [\"CPU Req (m)\", \"CPU Lim (m)\", \"Mem Req (Mi)\", \"Mem Lim (Mi)\"]")
			mermaidLines = append(mermaidLines, fmt.Sprintf("    bar [%d, %d, %d, %d]", cpuRequests, cpuLimits, memRequests/(1024*1024), memLimits/(1024*1024)))
		}

		sb.WriteString(util.FormatMermaidBlock(strings.Join(mermaidLines, "\n")))
		sb.WriteString("\n")

		return util.SuccessResult(sb.String()), nil, nil
	})

	// list_limit_ranges
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_limit_ranges",
		Description: "List LimitRange rules in a namespace showing type, resource, default/defaultRequest, min, and max values. Use namespace='all' for all namespaces.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listLimitRangesInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)
		limitRanges, err := client.ListLimitRanges(ctx, ns, metav1.ListOptions{})
		if err != nil {
			return util.HandleK8sError("listing limit ranges", err), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Limit Ranges (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n")

		if len(limitRanges) == 0 {
			sb.WriteString("(none)\n")
			return util.SuccessResult(sb.String()), nil, nil
		}

		for _, lr := range limitRanges {
			sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatSubHeader(fmt.Sprintf("%s/%s", lr.Namespace, lr.Name))))

			headers := []string{"TYPE", "RESOURCE", "DEFAULT", "DEFAULT-REQUEST", "MIN", "MAX"}
			rows := make([][]string, 0)

			for _, item := range lr.Spec.Limits {
				resources := collectLimitRangeResources(item)
				for _, res := range resources {
					rows = append(rows, []string{
						string(item.Type),
						res,
						quantityStr(item.Default, corev1.ResourceName(res)),
						quantityStr(item.DefaultRequest, corev1.ResourceName(res)),
						quantityStr(item.Min, corev1.ResourceName(res)),
						quantityStr(item.Max, corev1.ResourceName(res)),
					})
				}
			}
			sb.WriteString(util.FormatTable(headers, rows))
		}

		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("limit ranges", len(limitRanges))))

		return util.SuccessResult(sb.String()), nil, nil
	})

	// get_workload_dependencies
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workload_dependencies",
		Description: "Map all dependencies for a Deployment, StatefulSet, or Pod: ConfigMaps, Secrets, PVCs, ServiceAccounts from volumes and envFrom/env valueFrom. Finds Services whose selector matches. Returns a Mermaid dependency graph.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getWorkloadDependenciesInput) (*mcp.CallToolResult, any, error) {
		kind := input.WorkloadKind
		if kind == "" {
			kind = "Deployment"
		}

		var podSpec *corev1.PodSpec
		var podLabels map[string]string
		var displayName string

		switch kind {
		case "Deployment":
			deploy, err := client.GetDeployment(ctx, input.Namespace, input.WorkloadName)
			if err != nil {
				return util.HandleK8sError(fmt.Sprintf("getting deployment %s/%s", input.Namespace, input.WorkloadName), err), nil, nil
			}
			podSpec = &deploy.Spec.Template.Spec
			podLabels = deploy.Spec.Template.Labels
			displayName = fmt.Sprintf("Deployment/%s", deploy.Name)

		case "StatefulSet":
			statefulSets, err := client.ListStatefulSets(ctx, input.Namespace, metav1.ListOptions{})
			if err != nil {
				return util.HandleK8sError("listing statefulsets", err), nil, nil
			}
			var found bool
			for i := range statefulSets {
				if statefulSets[i].Name == input.WorkloadName {
					podSpec = &statefulSets[i].Spec.Template.Spec
					podLabels = statefulSets[i].Spec.Template.Labels
					displayName = fmt.Sprintf("StatefulSet/%s", statefulSets[i].Name)
					found = true
					break
				}
			}
			if !found {
				return util.ErrorResult("StatefulSet '%s' not found in namespace '%s'", input.WorkloadName, input.Namespace), nil, nil
			}

		case "Pod":
			pod, err := client.GetPod(ctx, input.Namespace, input.WorkloadName)
			if err != nil {
				return util.HandleK8sError(fmt.Sprintf("getting pod %s/%s", input.Namespace, input.WorkloadName), err), nil, nil
			}
			podSpec = &pod.Spec
			podLabels = pod.Labels
			displayName = fmt.Sprintf("Pod/%s", pod.Name)

		default:
			return util.ErrorResult("Unsupported workload kind: %s (use Deployment, StatefulSet, or Pod)", kind), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Workload Dependencies: %s (namespace: %s)", displayName, input.Namespace)))
		sb.WriteString("\n\n")

		// Extract dependencies
		configMaps := make(map[string]bool)
		secrets := make(map[string]bool)
		pvcs := make(map[string]bool)
		serviceAccountName := podSpec.ServiceAccountName

		// From volumes
		for _, vol := range podSpec.Volumes {
			if vol.ConfigMap != nil {
				configMaps[vol.ConfigMap.Name] = true
			}
			if vol.Secret != nil {
				secrets[vol.Secret.SecretName] = true
			}
			if vol.PersistentVolumeClaim != nil {
				pvcs[vol.PersistentVolumeClaim.ClaimName] = true
			}
			if vol.Projected != nil {
				for _, src := range vol.Projected.Sources {
					if src.ConfigMap != nil {
						configMaps[src.ConfigMap.Name] = true
					}
					if src.Secret != nil {
						secrets[src.Secret.Name] = true
					}
				}
			}
		}

		// From containers env/envFrom
		allContainers := append(podSpec.InitContainers, podSpec.Containers...)
		for _, c := range allContainers {
			for _, envFrom := range c.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					configMaps[envFrom.ConfigMapRef.Name] = true
				}
				if envFrom.SecretRef != nil {
					secrets[envFrom.SecretRef.Name] = true
				}
			}
			for _, env := range c.Env {
				if env.ValueFrom != nil {
					if env.ValueFrom.ConfigMapKeyRef != nil {
						configMaps[env.ValueFrom.ConfigMapKeyRef.Name] = true
					}
					if env.ValueFrom.SecretKeyRef != nil {
						secrets[env.ValueFrom.SecretKeyRef.Name] = true
					}
				}
			}
		}

		// List dependencies
		if serviceAccountName != "" {
			sb.WriteString(util.FormatKeyValue("ServiceAccount", serviceAccountName))
			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("\nConfigMaps (%d):\n", len(configMaps)))
		for name := range configMaps {
			sb.WriteString(fmt.Sprintf("  - %s\n", name))
		}
		if len(configMaps) == 0 {
			sb.WriteString("  (none)\n")
		}

		sb.WriteString(fmt.Sprintf("\nSecrets (%d):\n", len(secrets)))
		for name := range secrets {
			sb.WriteString(fmt.Sprintf("  - %s\n", name))
		}
		if len(secrets) == 0 {
			sb.WriteString("  (none)\n")
		}

		sb.WriteString(fmt.Sprintf("\nPVCs (%d):\n", len(pvcs)))
		for name := range pvcs {
			sb.WriteString(fmt.Sprintf("  - %s\n", name))
		}
		if len(pvcs) == 0 {
			sb.WriteString("  (none)\n")
		}

		// Find matching services
		matchingServices := make([]string, 0)
		if len(podLabels) > 0 {
			services, err := client.ListServices(ctx, input.Namespace, metav1.ListOptions{})
			if err == nil {
				for _, svc := range services {
					if len(svc.Spec.Selector) == 0 {
						continue
					}
					match := true
					for k, v := range svc.Spec.Selector {
						if podLabels[k] != v {
							match = false
							break
						}
					}
					if match {
						matchingServices = append(matchingServices, svc.Name)
					}
				}
			}
		}

		sb.WriteString(fmt.Sprintf("\nMatching Services (%d):\n", len(matchingServices)))
		for _, name := range matchingServices {
			sb.WriteString(fmt.Sprintf("  - %s\n", name))
		}
		if len(matchingServices) == 0 {
			sb.WriteString("  (none)\n")
		}

		// Mermaid dependency graph
		sb.WriteString("\nDEPENDENCY GRAPH:\n")
		var mermaidLines []string
		mermaidLines = append(mermaidLines, "graph TD")
		workloadID := "WL"
		mermaidLines = append(mermaidLines, fmt.Sprintf("    %s[%s]", workloadID, displayName))

		if serviceAccountName != "" {
			mermaidLines = append(mermaidLines, fmt.Sprintf("    %s --> SA[ServiceAccount: %s]", workloadID, serviceAccountName))
		}

		i := 0
		for name := range configMaps {
			mermaidLines = append(mermaidLines, fmt.Sprintf("    %s --> CM%d[ConfigMap: %s]", workloadID, i, name))
			i++
		}
		i = 0
		for name := range secrets {
			mermaidLines = append(mermaidLines, fmt.Sprintf("    %s --> SEC%d[Secret: %s]", workloadID, i, name))
			i++
		}
		i = 0
		for name := range pvcs {
			mermaidLines = append(mermaidLines, fmt.Sprintf("    %s --> PVC%d[PVC: %s]", workloadID, i, name))
			i++
		}
		for j, name := range matchingServices {
			mermaidLines = append(mermaidLines, fmt.Sprintf("    SVC%d[Service: %s] --> %s", j, name, workloadID))
		}

		sb.WriteString(util.FormatMermaidBlock(strings.Join(mermaidLines, "\n")))
		sb.WriteString("\n")

		return util.SuccessResult(sb.String()), nil, nil
	})
}

func collectLimitRangeResources(item corev1.LimitRangeItem) []string {
	seen := make(map[string]bool)
	for res := range item.Default {
		seen[string(res)] = true
	}
	for res := range item.DefaultRequest {
		seen[string(res)] = true
	}
	for res := range item.Min {
		seen[string(res)] = true
	}
	for res := range item.Max {
		seen[string(res)] = true
	}
	result := make([]string, 0, len(seen))
	for res := range seen {
		result = append(result, res)
	}
	return result
}

func quantityStr(list corev1.ResourceList, name corev1.ResourceName) string {
	if list == nil {
		return "-"
	}
	q, ok := list[name]
	if !ok {
		return "-"
	}
	return q.String()
}
