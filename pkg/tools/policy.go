package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/k8s"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

type listNetworkPoliciesInput struct {
	Namespace string `json:"namespace" jsonschema:"Kubernetes namespace (use 'all' for all namespaces)"`
}

type analyzePodConnectivityInput struct {
	Namespace string `json:"namespace" jsonschema:"required,Kubernetes namespace"`
	PodName   string `json:"pod_name" jsonschema:"required,Pod name to analyze connectivity for"`
}

type listHPAsInput struct {
	Namespace string `json:"namespace" jsonschema:"Kubernetes namespace (use 'all' for all namespaces)"`
}

type listPDBsInput struct {
	Namespace string `json:"namespace" jsonschema:"Kubernetes namespace (use 'all' for all namespaces)"`
}

func registerPolicyTools(server *mcp.Server, client *k8s.ClusterClient) {
	// list_network_policies
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_network_policies",
		Description: "List network policies with pod selectors, ingress/egress rule counts, and policy types. Use namespace='all' for all namespaces. Useful for understanding network segmentation.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listNetworkPoliciesInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)
		policies, err := client.ListNetworkPolicies(ctx, ns, metav1.ListOptions{})
		if err != nil {
			return util.HandleK8sError("listing network policies", err), nil, nil
		}

		headers := []string{"NAME", "NAMESPACE", "POD-SELECTOR", "INGRESS-RULES", "EGRESS-RULES", "POLICY-TYPES", "AGE"}
		rows := make([][]string, 0, len(policies))
		for _, np := range policies {
			policyTypes := make([]string, 0, len(np.Spec.PolicyTypes))
			for _, pt := range np.Spec.PolicyTypes {
				policyTypes = append(policyTypes, string(pt))
			}
			if len(policyTypes) == 0 {
				policyTypes = []string{"Ingress"}
			}

			rows = append(rows, []string{
				np.Name,
				np.Namespace,
				formatLabelSelector(&np.Spec.PodSelector),
				fmt.Sprintf("%d", len(np.Spec.Ingress)),
				fmt.Sprintf("%d", len(np.Spec.Egress)),
				strings.Join(policyTypes, ","),
				util.FormatAge(np.CreationTimestamp.Time),
			})
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Network Policies (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n")
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("network policies", len(policies))))

		// Detail section for each policy
		if len(policies) > 0 {
			sb.WriteString("\n")
			sb.WriteString(util.FormatSubHeader("Policy Details"))
			sb.WriteString("\n")
			for _, np := range policies {
				sb.WriteString(fmt.Sprintf("\n  %s/%s:\n", np.Namespace, np.Name))
				sb.WriteString(fmt.Sprintf("    Pod Selector: %s\n", formatLabelSelector(&np.Spec.PodSelector)))
				for i, rule := range np.Spec.Ingress {
					sb.WriteString(fmt.Sprintf("    Ingress Rule %d: ", i+1))
					parts := describeIngressRule(rule)
					sb.WriteString(strings.Join(parts, "; "))
					sb.WriteString("\n")
				}
				for i, rule := range np.Spec.Egress {
					sb.WriteString(fmt.Sprintf("    Egress Rule %d: ", i+1))
					parts := describeEgressRule(rule)
					sb.WriteString(strings.Join(parts, "; "))
					sb.WriteString("\n")
				}
			}
		}

		return util.SuccessResult(sb.String()), nil, nil
	})

	// analyze_pod_connectivity
	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze_pod_connectivity",
		Description: "Analyze network connectivity for a specific pod by matching its labels against all network policies in the namespace. Produces a Mermaid flowchart showing allowed and denied traffic directions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input analyzePodConnectivityInput) (*mcp.CallToolResult, any, error) {
		pod, err := client.GetPod(ctx, input.Namespace, input.PodName)
		if err != nil {
			return util.HandleK8sError(fmt.Sprintf("getting pod %s/%s", input.Namespace, input.PodName), err), nil, nil
		}

		policies, err := client.ListNetworkPolicies(ctx, input.Namespace, metav1.ListOptions{})
		if err != nil {
			return util.HandleK8sError("listing network policies", err), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Pod Connectivity Analysis: %s (namespace: %s)", pod.Name, pod.Namespace)))
		sb.WriteString("\n\n")

		sb.WriteString(util.FormatKeyValue("Pod Labels", util.FormatLabels(pod.Labels)))
		sb.WriteString("\n\n")

		// Find policies that select this pod
		matchingPolicies := make([]networkingv1.NetworkPolicy, 0)
		for _, np := range policies {
			selector, err := metav1.LabelSelectorAsSelector(&np.Spec.PodSelector)
			if err != nil {
				continue
			}
			if selector.Matches(labels.Set(pod.Labels)) {
				matchingPolicies = append(matchingPolicies, np)
			}
		}

		if len(matchingPolicies) == 0 {
			sb.WriteString(util.FormatFinding("INFO", "No network policies select this pod — all traffic is allowed by default"))
			sb.WriteString("\n\n")
			sb.WriteString("CONNECTIVITY DIAGRAM:\n")
			mermaid := fmt.Sprintf("graph LR\n    ANY[Any Source] -->|allowed| POD[Pod: %s]\n    POD -->|allowed| ANY2[Any Destination]", pod.Name)
			sb.WriteString(util.FormatMermaidBlock(mermaid))
			sb.WriteString("\n")
			return util.SuccessResult(sb.String()), nil, nil
		}

		sb.WriteString(fmt.Sprintf("Matching Policies: %d\n\n", len(matchingPolicies)))

		// Determine what traffic is allowed
		hasIngressPolicy := false
		hasEgressPolicy := false
		var ingressSources []string
		var egressDests []string

		for _, np := range matchingPolicies {
			for _, pt := range np.Spec.PolicyTypes {
				if pt == networkingv1.PolicyTypeIngress {
					hasIngressPolicy = true
				}
				if pt == networkingv1.PolicyTypeEgress {
					hasEgressPolicy = true
				}
			}
			// If no policy types specified, default is Ingress
			if len(np.Spec.PolicyTypes) == 0 {
				hasIngressPolicy = true
			}

			for _, rule := range np.Spec.Ingress {
				sources := describeIngressSources(rule)
				ingressSources = append(ingressSources, sources...)
			}
			for _, rule := range np.Spec.Egress {
				dests := describeEgressDests(rule)
				egressDests = append(egressDests, dests...)
			}
		}

		// Findings
		sb.WriteString("FINDINGS:\n")
		for _, np := range matchingPolicies {
			sb.WriteString(fmt.Sprintf("  Policy '%s':\n", np.Name))
			for _, pt := range np.Spec.PolicyTypes {
				if pt == networkingv1.PolicyTypeIngress {
					if len(np.Spec.Ingress) == 0 {
						sb.WriteString(util.FormatFinding("WARNING", "  Ingress policy with no rules — all ingress DENIED"))
						sb.WriteString("\n")
					} else {
						sb.WriteString(fmt.Sprintf("    %d ingress rules defined\n", len(np.Spec.Ingress)))
					}
				}
				if pt == networkingv1.PolicyTypeEgress {
					if len(np.Spec.Egress) == 0 {
						sb.WriteString(util.FormatFinding("WARNING", "  Egress policy with no rules — all egress DENIED"))
						sb.WriteString("\n")
					} else {
						sb.WriteString(fmt.Sprintf("    %d egress rules defined\n", len(np.Spec.Egress)))
					}
				}
			}
		}

		// Build Mermaid diagram
		sb.WriteString("\nCONNECTIVITY DIAGRAM:\n")
		var mermaidLines []string
		mermaidLines = append(mermaidLines, "graph LR")
		podNode := fmt.Sprintf("POD[Pod: %s]", pod.Name)

		if hasIngressPolicy {
			if len(ingressSources) > 0 {
				for i, src := range dedupe(ingressSources) {
					srcID := fmt.Sprintf("SRC%d", i)
					mermaidLines = append(mermaidLines, fmt.Sprintf("    %s[%s] -->|allowed| %s", srcID, src, podNode))
				}
			} else {
				mermaidLines = append(mermaidLines, fmt.Sprintf("    BLOCKED1[All Sources] -.->|denied| %s", podNode))
			}
		} else {
			mermaidLines = append(mermaidLines, fmt.Sprintf("    ANY_IN[Any Source] -->|allowed| %s", podNode))
		}

		if hasEgressPolicy {
			if len(egressDests) > 0 {
				for i, dst := range dedupe(egressDests) {
					dstID := fmt.Sprintf("DST%d", i)
					mermaidLines = append(mermaidLines, fmt.Sprintf("    %s -->|allowed| %s[%s]", podNode, dstID, dst))
				}
			} else {
				mermaidLines = append(mermaidLines, fmt.Sprintf("    %s -.->|denied| BLOCKED2[All Destinations]", podNode))
			}
		} else {
			mermaidLines = append(mermaidLines, fmt.Sprintf("    %s -->|allowed| ANY_OUT[Any Destination]", podNode))
		}

		sb.WriteString(util.FormatMermaidBlock(strings.Join(mermaidLines, "\n")))
		sb.WriteString("\n")

		return util.SuccessResult(sb.String()), nil, nil
	})

	// list_hpas
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_hpas",
		Description: "List Horizontal Pod Autoscalers with target reference, current/target metrics, min/max/current replicas, and conditions. Use namespace='all' for all namespaces.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listHPAsInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)
		hpas, err := client.ListHPAs(ctx, ns, metav1.ListOptions{})
		if err != nil {
			return util.HandleK8sError("listing HPAs", err), nil, nil
		}

		headers := []string{"NAME", "NAMESPACE", "REFERENCE", "MIN", "MAX", "CURRENT", "AGE"}
		rows := make([][]string, 0, len(hpas))
		for _, hpa := range hpas {
			minReplicas := int32(1)
			if hpa.Spec.MinReplicas != nil {
				minReplicas = *hpa.Spec.MinReplicas
			}
			rows = append(rows, []string{
				hpa.Name,
				hpa.Namespace,
				fmt.Sprintf("%s/%s", hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name),
				fmt.Sprintf("%d", minReplicas),
				fmt.Sprintf("%d", hpa.Spec.MaxReplicas),
				fmt.Sprintf("%d", hpa.Status.CurrentReplicas),
				util.FormatAge(hpa.CreationTimestamp.Time),
			})
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Horizontal Pod Autoscalers (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n")
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("HPAs", len(hpas))))

		// Detail: metrics and conditions
		if len(hpas) > 0 {
			sb.WriteString("\n")
			sb.WriteString(util.FormatSubHeader("HPA Details"))
			sb.WriteString("\n")
			for _, hpa := range hpas {
				sb.WriteString(fmt.Sprintf("\n  %s/%s:\n", hpa.Namespace, hpa.Name))
				for _, metric := range hpa.Spec.Metrics {
					switch metric.Type {
					case "Resource":
						if metric.Resource != nil {
							target := "n/a"
							if metric.Resource.Target.AverageUtilization != nil {
								target = fmt.Sprintf("%d%%", *metric.Resource.Target.AverageUtilization)
							} else if metric.Resource.Target.AverageValue != nil {
								target = metric.Resource.Target.AverageValue.String()
							}
							sb.WriteString(fmt.Sprintf("    Metric: %s (target: %s)\n", metric.Resource.Name, target))
						}
					case "Pods":
						if metric.Pods != nil {
							sb.WriteString(fmt.Sprintf("    Metric: %s (target avg: %s)\n", metric.Pods.Metric.Name, metric.Pods.Target.AverageValue.String()))
						}
					default:
						sb.WriteString(fmt.Sprintf("    Metric: %s type\n", metric.Type))
					}
				}
				for _, cond := range hpa.Status.Conditions {
					status := string(cond.Status)
					sb.WriteString(fmt.Sprintf("    Condition: %s=%s (%s)\n", cond.Type, status, cond.Reason))
				}
			}
		}

		return util.SuccessResult(sb.String()), nil, nil
	})

	// list_pdbs
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_pdbs",
		Description: "List Pod Disruption Budgets with min-available, max-unavailable, current/expected pods, and disruptions allowed. Warns when disruptions allowed is 0. Use namespace='all' for all namespaces.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listPDBsInput) (*mcp.CallToolResult, any, error) {
		ns := util.NamespaceOrAll(input.Namespace)
		pdbs, err := client.ListPodDisruptionBudgets(ctx, ns, metav1.ListOptions{})
		if err != nil {
			return util.HandleK8sError("listing PDBs", err), nil, nil
		}

		headers := []string{"NAME", "NAMESPACE", "MIN-AVAILABLE", "MAX-UNAVAILABLE", "CURRENT", "EXPECTED", "ALLOWED-DISRUPTIONS", "AGE"}
		rows := make([][]string, 0, len(pdbs))
		for _, pdb := range pdbs {
			minAvail := "N/A"
			if pdb.Spec.MinAvailable != nil {
				minAvail = pdb.Spec.MinAvailable.String()
			}
			maxUnavail := "N/A"
			if pdb.Spec.MaxUnavailable != nil {
				maxUnavail = pdb.Spec.MaxUnavailable.String()
			}
			rows = append(rows, []string{
				pdb.Name,
				pdb.Namespace,
				minAvail,
				maxUnavail,
				fmt.Sprintf("%d", pdb.Status.CurrentHealthy),
				fmt.Sprintf("%d", pdb.Status.ExpectedPods),
				fmt.Sprintf("%d", pdb.Status.DisruptionsAllowed),
				util.FormatAge(pdb.CreationTimestamp.Time),
			})
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Pod Disruption Budgets (namespace: %s)", displayNS(input.Namespace))))
		sb.WriteString("\n")
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("PDBs", len(pdbs))))

		// Warn on zero disruptions allowed
		for _, pdb := range pdbs {
			if pdb.Status.DisruptionsAllowed == 0 && pdb.Status.ExpectedPods > 0 {
				sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatFinding("WARNING", fmt.Sprintf("PDB '%s/%s' has 0 disruptions allowed — voluntary disruptions (node drains, rolling updates) will be blocked", pdb.Namespace, pdb.Name))))
			}
		}

		return util.SuccessResult(sb.String()), nil, nil
	})
}

// formatLabelSelector returns a human-readable label selector string.
func formatLabelSelector(sel *metav1.LabelSelector) string {
	if sel == nil {
		return "<all pods>"
	}
	if len(sel.MatchLabels) == 0 && len(sel.MatchExpressions) == 0 {
		return "<all pods>"
	}
	parts := make([]string, 0)
	for k, v := range sel.MatchLabels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(parts)
	for _, expr := range sel.MatchExpressions {
		parts = append(parts, fmt.Sprintf("%s %s (%s)", expr.Key, expr.Operator, strings.Join(expr.Values, ",")))
	}
	return strings.Join(parts, ", ")
}

func describeIngressRule(rule networkingv1.NetworkPolicyIngressRule) []string {
	var parts []string
	if len(rule.From) == 0 {
		parts = append(parts, "from: all sources")
	} else {
		for _, from := range rule.From {
			if from.PodSelector != nil {
				parts = append(parts, fmt.Sprintf("from pods: %s", formatLabelSelector(from.PodSelector)))
			}
			if from.NamespaceSelector != nil {
				parts = append(parts, fmt.Sprintf("from namespaces: %s", formatLabelSelector(from.NamespaceSelector)))
			}
			if from.IPBlock != nil {
				cidr := from.IPBlock.CIDR
				if len(from.IPBlock.Except) > 0 {
					cidr += fmt.Sprintf(" (except %s)", strings.Join(from.IPBlock.Except, ","))
				}
				parts = append(parts, fmt.Sprintf("from CIDR: %s", cidr))
			}
		}
	}
	if len(rule.Ports) > 0 {
		portParts := make([]string, 0, len(rule.Ports))
		for _, p := range rule.Ports {
			proto := "TCP"
			if p.Protocol != nil {
				proto = string(*p.Protocol)
			}
			if p.Port != nil {
				portParts = append(portParts, fmt.Sprintf("%s/%s", p.Port.String(), proto))
			}
		}
		parts = append(parts, fmt.Sprintf("ports: %s", strings.Join(portParts, ",")))
	}
	return parts
}

func describeEgressRule(rule networkingv1.NetworkPolicyEgressRule) []string {
	var parts []string
	if len(rule.To) == 0 {
		parts = append(parts, "to: all destinations")
	} else {
		for _, to := range rule.To {
			if to.PodSelector != nil {
				parts = append(parts, fmt.Sprintf("to pods: %s", formatLabelSelector(to.PodSelector)))
			}
			if to.NamespaceSelector != nil {
				parts = append(parts, fmt.Sprintf("to namespaces: %s", formatLabelSelector(to.NamespaceSelector)))
			}
			if to.IPBlock != nil {
				cidr := to.IPBlock.CIDR
				if len(to.IPBlock.Except) > 0 {
					cidr += fmt.Sprintf(" (except %s)", strings.Join(to.IPBlock.Except, ","))
				}
				parts = append(parts, fmt.Sprintf("to CIDR: %s", cidr))
			}
		}
	}
	if len(rule.Ports) > 0 {
		portParts := make([]string, 0, len(rule.Ports))
		for _, p := range rule.Ports {
			proto := "TCP"
			if p.Protocol != nil {
				proto = string(*p.Protocol)
			}
			if p.Port != nil {
				portParts = append(portParts, fmt.Sprintf("%s/%s", p.Port.String(), proto))
			}
		}
		parts = append(parts, fmt.Sprintf("ports: %s", strings.Join(portParts, ",")))
	}
	return parts
}

func describeIngressSources(rule networkingv1.NetworkPolicyIngressRule) []string {
	if len(rule.From) == 0 {
		return []string{"All Sources"}
	}
	var sources []string
	for _, from := range rule.From {
		if from.PodSelector != nil {
			sources = append(sources, fmt.Sprintf("Pods: %s", formatLabelSelector(from.PodSelector)))
		}
		if from.NamespaceSelector != nil {
			sources = append(sources, fmt.Sprintf("NS: %s", formatLabelSelector(from.NamespaceSelector)))
		}
		if from.IPBlock != nil {
			sources = append(sources, fmt.Sprintf("CIDR: %s", from.IPBlock.CIDR))
		}
	}
	return sources
}

func describeEgressDests(rule networkingv1.NetworkPolicyEgressRule) []string {
	if len(rule.To) == 0 {
		return []string{"All Destinations"}
	}
	var dests []string
	for _, to := range rule.To {
		if to.PodSelector != nil {
			dests = append(dests, fmt.Sprintf("Pods: %s", formatLabelSelector(to.PodSelector)))
		}
		if to.NamespaceSelector != nil {
			dests = append(dests, fmt.Sprintf("NS: %s", formatLabelSelector(to.NamespaceSelector)))
		}
		if to.IPBlock != nil {
			dests = append(dests, fmt.Sprintf("CIDR: %s", to.IPBlock.CIDR))
		}
	}
	return dests
}

func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
