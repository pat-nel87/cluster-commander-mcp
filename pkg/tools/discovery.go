package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"

	"github.com/pat-nel87/kube-doctor-mcp/pkg/k8s"
	"github.com/pat-nel87/kube-doctor-mcp/pkg/util"
)

type listCRDsInput struct {
	GroupFilter string `json:"group_filter,omitempty" jsonschema:"Filter by API group (substring match)"`
}

type getAPIResourcesInput struct {
	GroupFilter string `json:"group_filter,omitempty" jsonschema:"Filter by API group (substring match)"`
}

type listWebhookConfigsInput struct{}

func registerDiscoveryTools(server *mcp.Server, client *k8s.ClusterClient) {
	// list_crds
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_crds",
		Description: "List Custom Resource Definitions with group, version, scope, and age. Optional group filter to narrow results. Useful for discovering what CRDs are installed in the cluster.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listCRDsInput) (*mcp.CallToolResult, any, error) {
		crds, err := client.ListCRDs(ctx)
		if err != nil {
			return util.HandleK8sError("listing CRDs", err), nil, nil
		}

		headers := []string{"NAME", "GROUP", "VERSION", "SCOPE", "AGE"}
		rows := make([][]string, 0, len(crds))
		for _, crd := range crds {
			if input.GroupFilter != "" && !strings.Contains(crd.Spec.Group, input.GroupFilter) {
				continue
			}
			version := ""
			for _, v := range crd.Spec.Versions {
				if v.Served {
					if version != "" {
						version += ","
					}
					version += v.Name
					if v.Storage {
						version += "*"
					}
				}
			}
			rows = append(rows, []string{
				crd.Name,
				crd.Spec.Group,
				version,
				string(crd.Spec.Scope),
				util.FormatAge(crd.CreationTimestamp.Time),
			})
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader("Custom Resource Definitions"))
		sb.WriteString("\n")
		if input.GroupFilter != "" {
			sb.WriteString(fmt.Sprintf("Filter: group contains '%s'\n", input.GroupFilter))
		}
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("CRDs", len(rows))))

		return util.SuccessResult(sb.String()), nil, nil
	})

	// get_api_resources
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_api_resources",
		Description: "List API resources available in the cluster with group/version, namespaced scope, and supported verbs. Optional group filter. Useful for understanding what resource types exist.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getAPIResourcesInput) (*mcp.CallToolResult, any, error) {
		resourceLists, err := client.GetAPIResources(ctx)
		if err != nil {
			return util.HandleK8sError("getting API resources", err), nil, nil
		}

		headers := []string{"NAME", "API-GROUP", "NAMESPACED", "VERBS"}
		rows := make([][]string, 0)

		for _, rl := range resourceLists {
			groupVersion := rl.GroupVersion
			if input.GroupFilter != "" && !strings.Contains(groupVersion, input.GroupFilter) {
				continue
			}
			for _, r := range rl.APIResources {
				// Skip sub-resources (contain /)
				if strings.Contains(r.Name, "/") {
					continue
				}
				namespaced := "true"
				if !r.Namespaced {
					namespaced = "false"
				}
				verbs := strings.Join(r.Verbs, ",")
				rows = append(rows, []string{
					r.Name,
					groupVersion,
					namespaced,
					verbs,
				})
			}
		}

		var sb strings.Builder
		sb.WriteString(util.FormatHeader("API Resources"))
		sb.WriteString("\n")
		if input.GroupFilter != "" {
			sb.WriteString(fmt.Sprintf("Filter: group contains '%s'\n", input.GroupFilter))
		}
		sb.WriteString(util.FormatTable(headers, rows))
		sb.WriteString(fmt.Sprintf("\n%s\n", util.FormatCount("API resources", len(rows))))

		return util.SuccessResult(sb.String()), nil, nil
	})

	// list_webhook_configs
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_webhook_configs",
		Description: "List mutating and validating webhook configurations with service endpoints, failure policies, rules, and timeouts. Warns when failurePolicy is Fail, which can block cluster operations if the webhook is down.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listWebhookConfigsInput) (*mcp.CallToolResult, any, error) {
		var sb strings.Builder
		sb.WriteString(util.FormatHeader("Webhook Configurations"))
		sb.WriteString("\n\n")

		// Mutating webhooks
		sb.WriteString(util.FormatSubHeader("Mutating Webhooks"))
		sb.WriteString("\n")
		mutating, err := client.ListMutatingWebhookConfigurations(ctx)
		if err != nil {
			sb.WriteString(fmt.Sprintf("  (could not list: %v)\n", err))
		} else if len(mutating) == 0 {
			sb.WriteString("  (none)\n")
		} else {
			for _, mwc := range mutating {
				sb.WriteString(fmt.Sprintf("\n  %s:\n", mwc.Name))
				for _, wh := range mwc.Webhooks {
					failPolicy := "Ignore"
					if wh.FailurePolicy != nil {
						failPolicy = string(*wh.FailurePolicy)
					}
					timeout := int32(10)
					if wh.TimeoutSeconds != nil {
						timeout = *wh.TimeoutSeconds
					}
					endpoint := formatWebhookEndpoint(wh.ClientConfig.Service, wh.ClientConfig.URL)
					sb.WriteString(fmt.Sprintf("    Webhook: %s\n", wh.Name))
					sb.WriteString(fmt.Sprintf("      Endpoint: %s\n", endpoint))
					sb.WriteString(fmt.Sprintf("      Failure Policy: %s\n", failPolicy))
					sb.WriteString(fmt.Sprintf("      Timeout: %ds\n", timeout))
					if len(wh.Rules) > 0 {
						for _, rule := range wh.Rules {
							ops := operationStrings(rule.Operations)
							sb.WriteString(fmt.Sprintf("      Rule: %s %s %s\n",
								strings.Join(ops, ","),
								strings.Join(rule.APIGroups, ","),
								strings.Join(rule.Resources, ",")))
						}
					}
					if failPolicy == "Fail" {
						sb.WriteString(fmt.Sprintf("      %s\n", util.FormatFinding("WARNING", "failurePolicy=Fail — webhook outage will block matching API requests")))
					}
				}
			}
		}

		// Validating webhooks
		sb.WriteString("\n")
		sb.WriteString(util.FormatSubHeader("Validating Webhooks"))
		sb.WriteString("\n")
		validating, err := client.ListValidatingWebhookConfigurations(ctx)
		if err != nil {
			sb.WriteString(fmt.Sprintf("  (could not list: %v)\n", err))
		} else if len(validating) == 0 {
			sb.WriteString("  (none)\n")
		} else {
			for _, vwc := range validating {
				sb.WriteString(fmt.Sprintf("\n  %s:\n", vwc.Name))
				for _, wh := range vwc.Webhooks {
					failPolicy := "Ignore"
					if wh.FailurePolicy != nil {
						failPolicy = string(*wh.FailurePolicy)
					}
					timeout := int32(10)
					if wh.TimeoutSeconds != nil {
						timeout = *wh.TimeoutSeconds
					}
					endpoint := formatWebhookEndpoint(wh.ClientConfig.Service, wh.ClientConfig.URL)
					sb.WriteString(fmt.Sprintf("    Webhook: %s\n", wh.Name))
					sb.WriteString(fmt.Sprintf("      Endpoint: %s\n", endpoint))
					sb.WriteString(fmt.Sprintf("      Failure Policy: %s\n", failPolicy))
					sb.WriteString(fmt.Sprintf("      Timeout: %ds\n", timeout))
					if len(wh.Rules) > 0 {
						for _, rule := range wh.Rules {
							ops := operationStrings(rule.Operations)
							sb.WriteString(fmt.Sprintf("      Rule: %s %s %s\n",
								strings.Join(ops, ","),
								strings.Join(rule.APIGroups, ","),
								strings.Join(rule.Resources, ",")))
						}
					}
					if failPolicy == "Fail" {
						sb.WriteString(fmt.Sprintf("      %s\n", util.FormatFinding("WARNING", "failurePolicy=Fail — webhook outage will block matching API requests")))
					}
				}
			}
		}

		totalMut := 0
		for _, m := range mutating {
			totalMut += len(m.Webhooks)
		}
		totalVal := 0
		for _, v := range validating {
			totalVal += len(v.Webhooks)
		}
		sb.WriteString(fmt.Sprintf("\nTotal: %d mutating, %d validating webhooks\n", totalMut, totalVal))

		return util.SuccessResult(sb.String()), nil, nil
	})
}

func formatWebhookEndpoint(svc *admissionregistrationv1.ServiceReference, url *string) string {
	if url != nil {
		return *url
	}
	if svc != nil {
		path := "/"
		if svc.Path != nil {
			path = *svc.Path
		}
		port := int32(443)
		if svc.Port != nil {
			port = *svc.Port
		}
		return fmt.Sprintf("%s/%s:%d%s", svc.Namespace, svc.Name, port, path)
	}
	return "<unknown>"
}

func operationStrings(ops []admissionregistrationv1.OperationType) []string {
	result := make([]string, len(ops))
	for i, op := range ops {
		result[i] = string(op)
	}
	return result
}
