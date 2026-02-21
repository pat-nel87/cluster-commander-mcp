package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/k8s"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

type analyzePodSecurityInput struct {
	Namespace string `json:"namespace" jsonschema:"required,Kubernetes namespace"`
	PodName   string `json:"pod_name" jsonschema:"required,Pod name to analyze"`
}

type listRBACBindingsInput struct {
	Namespace     string `json:"namespace" jsonschema:"required,Kubernetes namespace"`
	SubjectFilter string `json:"subject_filter,omitempty" jsonschema:"Filter by subject name (user, group, or service account)"`
}

type auditNamespaceSecurityInput struct {
	Namespace string `json:"namespace" jsonschema:"required,Kubernetes namespace to audit"`
}

func registerSecurityTools(server *mcp.Server, client *k8s.ClusterClient) {
	// analyze_pod_security
	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze_pod_security",
		Description: "Analyze security posture of a specific pod. Checks SecurityContext at pod and container level: root user, privilege escalation, capabilities, readOnlyRootFilesystem, hostNetwork/PID/IPC, and seccomp profile. Returns severity-tagged findings and suggested actions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input analyzePodSecurityInput) (*mcp.CallToolResult, any, error) {
		pod, err := client.GetPod(ctx, input.Namespace, input.PodName)
		if err != nil {
			return util.HandleK8sError(fmt.Sprintf("getting pod %s/%s", input.Namespace, input.PodName), err), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Pod Security Analysis: %s (namespace: %s)", pod.Name, pod.Namespace)))
		sb.WriteString("\n\n")

		findings := 0
		var actions []string

		// Pod-level security context
		sb.WriteString(util.FormatSubHeader("Pod-Level Security"))
		sb.WriteString("\n")

		if pod.Spec.HostNetwork {
			sb.WriteString(util.FormatFinding("CRITICAL", "Pod uses hostNetwork — shares node's network namespace"))
			sb.WriteString("\n")
			actions = append(actions, "Remove hostNetwork unless absolutely required (e.g., CNI plugins)")
			findings++
		}
		if pod.Spec.HostPID {
			sb.WriteString(util.FormatFinding("CRITICAL", "Pod uses hostPID — can see all processes on the node"))
			sb.WriteString("\n")
			actions = append(actions, "Remove hostPID to prevent process visibility across the node")
			findings++
		}
		if pod.Spec.HostIPC {
			sb.WriteString(util.FormatFinding("WARNING", "Pod uses hostIPC — shares node's IPC namespace"))
			sb.WriteString("\n")
			actions = append(actions, "Remove hostIPC unless inter-process communication with host is required")
			findings++
		}

		podSC := pod.Spec.SecurityContext
		if podSC != nil {
			if podSC.RunAsUser != nil && *podSC.RunAsUser == 0 {
				sb.WriteString(util.FormatFinding("CRITICAL", "Pod runAsUser is 0 (root)"))
				sb.WriteString("\n")
				actions = append(actions, "Set runAsUser to a non-zero UID (e.g., 1000)")
				findings++
			}
			if podSC.RunAsNonRoot != nil && !*podSC.RunAsNonRoot {
				sb.WriteString(util.FormatFinding("WARNING", "Pod runAsNonRoot is explicitly set to false"))
				sb.WriteString("\n")
				findings++
			}
			if podSC.SeccompProfile != nil {
				sb.WriteString(fmt.Sprintf("  Seccomp Profile: %s\n", podSC.SeccompProfile.Type))
			} else {
				sb.WriteString(util.FormatFinding("INFO", "No seccomp profile set at pod level"))
				sb.WriteString("\n")
				findings++
			}
		} else {
			sb.WriteString(util.FormatFinding("INFO", "No pod-level SecurityContext defined"))
			sb.WriteString("\n")
			findings++
		}

		// Container-level security contexts
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Container Security"))
		sb.WriteString("\n")

		allContainers := append(pod.Spec.InitContainers, pod.Spec.Containers...)
		for _, c := range allContainers {
			sb.WriteString(fmt.Sprintf("\n  Container: %s\n", c.Name))
			sc := c.SecurityContext
			if sc == nil {
				sb.WriteString(fmt.Sprintf("    %s\n", util.FormatFinding("WARNING", "No SecurityContext defined")))
				actions = append(actions, fmt.Sprintf("Add SecurityContext to container '%s'", c.Name))
				findings++
				continue
			}

			if sc.Privileged != nil && *sc.Privileged {
				sb.WriteString(fmt.Sprintf("    %s\n", util.FormatFinding("CRITICAL", "Container runs in privileged mode")))
				actions = append(actions, fmt.Sprintf("Remove privileged mode from container '%s'", c.Name))
				findings++
			}
			if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
				sb.WriteString(fmt.Sprintf("    %s\n", util.FormatFinding("WARNING", "allowPrivilegeEscalation is not explicitly disabled")))
				findings++
			}
			if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
				sb.WriteString(fmt.Sprintf("    %s\n", util.FormatFinding("CRITICAL", "Container runAsUser is 0 (root)")))
				findings++
			}
			if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
				sb.WriteString(fmt.Sprintf("    %s\n", util.FormatFinding("INFO", "readOnlyRootFilesystem is not enabled")))
				findings++
			}
			if sc.Capabilities != nil {
				if len(sc.Capabilities.Add) > 0 {
					caps := make([]string, 0, len(sc.Capabilities.Add))
					for _, cap := range sc.Capabilities.Add {
						caps = append(caps, string(cap))
					}
					severity := "WARNING"
					for _, cap := range sc.Capabilities.Add {
						if cap == "SYS_ADMIN" || cap == "NET_ADMIN" || cap == "ALL" {
							severity = "CRITICAL"
							break
						}
					}
					sb.WriteString(fmt.Sprintf("    %s\n", util.FormatFinding(severity, fmt.Sprintf("Added capabilities: %s", strings.Join(caps, ", ")))))
					findings++
				}
				if len(sc.Capabilities.Drop) > 0 {
					caps := make([]string, 0, len(sc.Capabilities.Drop))
					for _, cap := range sc.Capabilities.Drop {
						caps = append(caps, string(cap))
					}
					sb.WriteString(fmt.Sprintf("    Dropped capabilities: %s\n", strings.Join(caps, ", ")))
				}
			} else {
				sb.WriteString(fmt.Sprintf("    %s\n", util.FormatFinding("INFO", "No capabilities configuration (consider dropping ALL and adding only needed)")))
				findings++
			}
		}

		// Summary
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Summary"))
		sb.WriteString("\n")
		if findings == 0 {
			sb.WriteString("  Pod security posture looks good. No issues found.\n")
		} else {
			sb.WriteString(fmt.Sprintf("  %d security finding(s) identified.\n", findings))
		}

		if len(actions) > 0 {
			sb.WriteString("\nSUGGESTED ACTIONS:\n")
			for i, action := range actions {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, action))
			}
		}

		return util.SuccessResult(sb.String()), nil, nil
	})

	// list_rbac_bindings
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_rbac_bindings",
		Description: "List RBAC role bindings in a namespace showing subject → role mapping. Includes both RoleBindings and ClusterRoleBindings that apply. Optional subject filter to find bindings for a specific user, group, or service account.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listRBACBindingsInput) (*mcp.CallToolResult, any, error) {
		// Get namespace-scoped role bindings
		roleBindings, err := client.ListRoleBindings(ctx, input.Namespace, metav1.ListOptions{})
		if err != nil {
			return util.HandleK8sError("listing role bindings", err), nil, nil
		}

		headers := []string{"BINDING", "SCOPE", "ROLE", "SUBJECT-KIND", "SUBJECT-NAME", "SUBJECT-NS"}
		rows := make([][]string, 0)

		for _, rb := range roleBindings {
			for _, subject := range rb.Subjects {
				if input.SubjectFilter != "" && !strings.Contains(strings.ToLower(subject.Name), strings.ToLower(input.SubjectFilter)) {
					continue
				}
				rows = append(rows, []string{
					rb.Name,
					"Namespace",
					fmt.Sprintf("%s/%s", rb.RoleRef.Kind, rb.RoleRef.Name),
					subject.Kind,
					subject.Name,
					subject.Namespace,
				})
			}
		}

		// Also get cluster role bindings that may grant permissions in this namespace
		clusterBindings, err := client.ListClusterRoleBindings(ctx, metav1.ListOptions{})
		if err == nil {
			for _, crb := range clusterBindings {
				for _, subject := range crb.Subjects {
					if input.SubjectFilter != "" && !strings.Contains(strings.ToLower(subject.Name), strings.ToLower(input.SubjectFilter)) {
						continue
					}
					// Include ClusterRoleBindings that reference ServiceAccounts in this namespace
					if subject.Kind == "ServiceAccount" && subject.Namespace != "" && subject.Namespace != input.Namespace {
						continue
					}
					rows = append(rows, []string{
						crb.Name,
						"Cluster",
						fmt.Sprintf("%s/%s", crb.RoleRef.Kind, crb.RoleRef.Name),
						subject.Kind,
						subject.Name,
						subject.Namespace,
					})
				}
			}
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("RBAC Bindings (namespace: %s)", input.Namespace)))
		sb.WriteString("\n")
		if input.SubjectFilter != "" {
			sb.WriteString(fmt.Sprintf("Filter: %s\n", input.SubjectFilter))
		}
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("bindings", len(rows))))

		return util.SuccessResult(sb.String()), nil, nil
	})

	// audit_namespace_security
	mcp.AddTool(server, &mcp.Tool{
		Name:        "audit_namespace_security",
		Description: "Comprehensive security audit for a namespace. Checks network policies, pod disruption budgets, pod security contexts, RBAC bindings, and resource quotas. Returns an overall security score and a Mermaid policy coverage diagram.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input auditNamespaceSecurityInput) (*mcp.CallToolResult, any, error) {
		var sb strings.Builder
		sb.WriteString(util.FormatHeader(fmt.Sprintf("Namespace Security Audit: %s", input.Namespace)))
		sb.WriteString("\n\n")

		score := 100
		findings := 0

		// 1. Network Policies
		sb.WriteString(util.FormatSubHeader("Network Policies"))
		sb.WriteString("\n")
		netPols, err := client.ListNetworkPolicies(ctx, input.Namespace, metav1.ListOptions{})
		hasNetPol := false
		if err != nil {
			sb.WriteString("  (could not check network policies)\n")
		} else if len(netPols) == 0 {
			sb.WriteString(util.FormatFinding("WARNING", "No network policies — all pod traffic is unrestricted"))
			sb.WriteString("\n")
			score -= 20
			findings++
		} else {
			sb.WriteString(fmt.Sprintf("  %d network policies defined\n", len(netPols)))
			hasNetPol = true
		}

		// 2. Pod Disruption Budgets
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Pod Disruption Budgets"))
		sb.WriteString("\n")
		hasPDB := false
		pdbs, err := client.ListPodDisruptionBudgets(ctx, input.Namespace, metav1.ListOptions{})
		if err != nil {
			sb.WriteString("  (could not check PDBs)\n")
		} else if len(pdbs) == 0 {
			sb.WriteString(util.FormatFinding("INFO", "No PDBs — workloads have no disruption protection"))
			sb.WriteString("\n")
			score -= 5
			findings++
		} else {
			sb.WriteString(fmt.Sprintf("  %d PDBs defined\n", len(pdbs)))
			hasPDB = true
			for _, pdb := range pdbs {
				if pdb.Status.DisruptionsAllowed == 0 && pdb.Status.ExpectedPods > 0 {
					sb.WriteString(util.FormatFinding("WARNING", fmt.Sprintf("PDB '%s' has 0 disruptions allowed", pdb.Name)))
					sb.WriteString("\n")
					findings++
				}
			}
		}

		// 3. Pod Security scan
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Pod Security"))
		sb.WriteString("\n")
		pods, err := client.ListPods(ctx, input.Namespace, metav1.ListOptions{})
		podsScanned := false
		privilegedPods := 0
		rootPods := 0
		noSecCtxPods := 0
		if err != nil {
			sb.WriteString("  (could not list pods)\n")
		} else if len(pods) == 0 {
			sb.WriteString("  No pods in namespace\n")
		} else {
			podsScanned = true
			for _, pod := range pods {
				hasAnySecCtx := pod.Spec.SecurityContext != nil
				for _, c := range pod.Spec.Containers {
					if c.SecurityContext == nil {
						if !hasAnySecCtx {
							noSecCtxPods++
						}
						continue
					}
					hasAnySecCtx = true
					if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
						privilegedPods++
					}
					if c.SecurityContext.RunAsUser != nil && *c.SecurityContext.RunAsUser == 0 {
						rootPods++
					}
				}
				if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.RunAsUser != nil && *pod.Spec.SecurityContext.RunAsUser == 0 {
					rootPods++
				}
			}
			sb.WriteString(fmt.Sprintf("  %d pods scanned\n", len(pods)))
			if privilegedPods > 0 {
				sb.WriteString(util.FormatFinding("CRITICAL", fmt.Sprintf("%d pod(s) running in privileged mode", privilegedPods)))
				sb.WriteString("\n")
				score -= 15
				findings++
			}
			if rootPods > 0 {
				sb.WriteString(util.FormatFinding("WARNING", fmt.Sprintf("%d pod(s) running as root", rootPods)))
				sb.WriteString("\n")
				score -= 10
				findings++
			}
			if noSecCtxPods > 0 {
				sb.WriteString(util.FormatFinding("INFO", fmt.Sprintf("%d pod(s) with no SecurityContext", noSecCtxPods)))
				sb.WriteString("\n")
				score -= 5
				findings++
			}
		}

		// 4. RBAC
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("RBAC"))
		sb.WriteString("\n")
		hasRBAC := false
		bindings, err := client.ListRoleBindings(ctx, input.Namespace, metav1.ListOptions{})
		if err != nil {
			sb.WriteString("  (could not check RBAC)\n")
		} else {
			sb.WriteString(fmt.Sprintf("  %d role bindings\n", len(bindings)))
			if len(bindings) > 0 {
				hasRBAC = true
			}
		}

		// 5. Resource Quotas
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Resource Quotas"))
		sb.WriteString("\n")
		hasQuota := false
		quotas, err := client.ListResourceQuotas(ctx, input.Namespace)
		if err != nil {
			sb.WriteString("  (could not check resource quotas)\n")
		} else if len(quotas) == 0 {
			sb.WriteString(util.FormatFinding("INFO", "No resource quotas — resource consumption is unrestricted"))
			sb.WriteString("\n")
			score -= 5
			findings++
		} else {
			sb.WriteString(fmt.Sprintf("  %d resource quotas defined\n", len(quotas)))
			hasQuota = true
		}

		// Clamp score
		if score < 0 {
			score = 0
		}

		// Overall Score
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Overall Security Score"))
		sb.WriteString("\n")
		grade := "A"
		switch {
		case score >= 90:
			grade = "A"
		case score >= 80:
			grade = "B"
		case score >= 70:
			grade = "C"
		case score >= 60:
			grade = "D"
		default:
			grade = "F"
		}
		sb.WriteString(fmt.Sprintf("  Score: %d/100 (Grade: %s)\n", score, grade))
		sb.WriteString(fmt.Sprintf("  %d finding(s) identified\n", findings))

		// Mermaid policy coverage diagram
		sb.WriteString("\nPOLICY COVERAGE:\n")
		var mermaidLines []string
		mermaidLines = append(mermaidLines, "graph TD")
		mermaidLines = append(mermaidLines, fmt.Sprintf("    NS[Namespace: %s]", input.Namespace))

		netPolStatus := "none"
		if hasNetPol {
			netPolStatus = fmt.Sprintf("%d policies", len(netPols))
		}
		pdbStatus := "none"
		if hasPDB {
			pdbStatus = fmt.Sprintf("%d PDBs", len(pdbs))
		}
		secStatus := "not scanned"
		if podsScanned {
			if privilegedPods == 0 && rootPods == 0 {
				secStatus = "good"
			} else {
				secStatus = fmt.Sprintf("%d issues", privilegedPods+rootPods)
			}
		}
		rbacStatus := "none"
		if hasRBAC {
			rbacStatus = fmt.Sprintf("%d bindings", len(bindings))
		}
		quotaStatus := "none"
		if hasQuota {
			quotaStatus = fmt.Sprintf("%d quotas", len(quotas))
		}

		netPolStyle := ":::warning"
		if hasNetPol {
			netPolStyle = ":::success"
		}
		pdbStyle := ":::info"
		if hasPDB {
			pdbStyle = ":::success"
		}
		secStyle := ":::warning"
		if podsScanned && privilegedPods == 0 && rootPods == 0 {
			secStyle = ":::success"
		}

		mermaidLines = append(mermaidLines, fmt.Sprintf("    NS --> NP[NetworkPolicies: %s]%s", netPolStatus, netPolStyle))
		mermaidLines = append(mermaidLines, fmt.Sprintf("    NS --> PDB[PDBs: %s]%s", pdbStatus, pdbStyle))
		mermaidLines = append(mermaidLines, fmt.Sprintf("    NS --> SEC[Pod Security: %s]%s", secStatus, secStyle))
		mermaidLines = append(mermaidLines, fmt.Sprintf("    NS --> RBAC[RBAC: %s]", rbacStatus))
		mermaidLines = append(mermaidLines, fmt.Sprintf("    NS --> QUOTA[Quotas: %s]", quotaStatus))

		sb.WriteString(util.FormatMermaidBlock(strings.Join(mermaidLines, "\n")))
		sb.WriteString("\n")

		return util.SuccessResult(sb.String()), nil, nil
	})
}
