# kube-doctor-mcp

Kubernetes cluster diagnostics for AI assistants. Two implementations, one goal: give Copilot (or any LLM) the ability to inspect and diagnose your Kubernetes cluster.

## What's Inside

| Project | Path | Approach | Tools |
|---------|------|----------|-------|
| **Go MCP Server** | root (`main.go`) | Standalone binary, stdio transport (MCP protocol) | 48 |
| **VS Code Extension** | `kube-doctor-vscode/` | Native Language Model Tools API (no MCP dependency) | 15 |

Both connect directly to the Kubernetes API via your kubeconfig. No kubectl dependency.

### When to Use Which

- **Go MCP Server** — Full-featured. Works with any MCP-compatible host (VS Code, Claude Code, Claude Desktop, etc.). Requires MCP to be enabled in your environment.
- **VS Code Extension** — Subset of tools packaged as a VS Code extension. Works in corporate/restricted environments where MCP servers may be blocked by policy. Installs as a `.vsix` sideload.

---

## Go MCP Server

### Prerequisites

- Go 1.23+
- Access to a Kubernetes cluster (kubeconfig)

### Build

```bash
go build -o kube-doctor .
```

### Run with MCP Inspector

The [MCP Inspector](https://github.com/modelcontextprotocol/inspector) is a standalone web UI for testing MCP servers:

```bash
npx @modelcontextprotocol/inspector ./kube-doctor
```

### VS Code Setup (Copilot Agent Mode)

Add to your workspace `.vscode/mcp.json`:

```json
{
  "servers": {
    "kube-doctor": {
      "type": "stdio",
      "command": "/absolute/path/to/kube-doctor",
      "env": {
        "KUBECONFIG": "/path/to/.kube/config"
      }
    }
  }
}
```

Then open the Copilot Chat panel, select **Agent** mode, and ask questions like:
- "What namespaces are in my cluster?"
- "Are there any unhealthy pods?"
- "Diagnose the crasher pod in the default namespace"

### Claude Desktop Setup

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "kube-doctor": {
      "command": "/absolute/path/to/kube-doctor",
      "env": {
        "KUBECONFIG": "/path/to/.kube/config"
      }
    }
  }
}
```

### All 48 Tools

| Category | Tool | Description |
|----------|------|-------------|
| **Cluster** | `list_contexts` | List kubeconfig contexts |
| | `list_namespaces` | Namespaces with status and age |
| | `cluster_info` | Cluster version, node/pod/service counts |
| **Pods** | `list_pods` | Pods with status, restarts, node |
| | `get_pod_detail` | Full pod spec, conditions, events |
| | `get_pod_logs` | Container logs with tail/previous/since |
| **Events** | `get_events` | Events filtered by type/namespace/object |
| **Workloads** | `list_deployments` | Deployments with replica status |
| | `get_deployment_detail` | Rollout status, conditions, RS history |
| | `list_statefulsets` | StatefulSets with replica status |
| | `list_daemonsets` | DaemonSets with node scheduling |
| | `list_jobs` | Jobs/CronJobs with completion status |
| **Nodes** | `list_nodes` | Nodes with status, roles, capacity |
| | `get_node_detail` | Conditions, taints, allocatable resources |
| **Networking** | `list_services` | Services with type, IPs, ports |
| | `list_ingresses` | Ingresses with hosts, paths, TLS |
| | `get_endpoints` | Service endpoints (backing pod IPs) |
| **Storage** | `list_pvcs` | PVCs with status, capacity, storage class |
| | `list_pvs` | PVs with reclaim policy, class |
| **Metrics** | `get_node_metrics` | Node CPU/memory usage |
| | `get_pod_metrics` | Pod CPU/memory usage |
| | `top_resource_consumers` | Top N pods by CPU or memory |
| **Policy** | `list_network_policies` | Network policies with selectors and rules |
| | `analyze_pod_connectivity` | Pod traffic analysis with Mermaid diagram |
| | `list_hpas` | Horizontal Pod Autoscalers |
| | `list_pdbs` | Pod Disruption Budgets |
| **Security** | `analyze_pod_security` | Pod/container SecurityContext audit |
| | `list_rbac_bindings` | Role bindings with subject filter |
| | `audit_namespace_security` | Composite security score with Mermaid |
| **Resources** | `analyze_resource_allocation` | CPU/memory requests vs limits vs capacity with Mermaid |
| | `list_limit_ranges` | LimitRange rules |
| | `get_workload_dependencies` | ConfigMap/Secret/PVC/Service dependency map with Mermaid |
| **Discovery** | `list_crds` | Custom Resource Definitions |
| | `get_api_resources` | Available API resource types |
| | `list_webhook_configs` | Mutating/validating webhooks with failure policies |
| **Doctor** | `diagnose_pod` | Comprehensive pod diagnosis |
| | `diagnose_namespace` | Namespace health check |
| | `diagnose_cluster` | Cluster-wide health report |
| | `find_unhealthy_pods` | Find all unhealthy pods |
| | `check_resource_quotas` | Quota usage and warnings |
| **FluxCD** | `list_flux_kustomizations` | Kustomizations with source, path, status, revision |
| | `list_flux_helm_releases` | HelmReleases with chart, version, remediation |
| | `list_flux_sources` | All source types (Git, OCI, Helm, Bucket) |
| | `list_flux_image_policies` | ImageRepositories and ImagePolicies |
| | `diagnose_flux_kustomization` | Deep Kustomization diagnosis |
| | `diagnose_flux_helm_release` | Deep HelmRelease diagnosis |
| | `diagnose_flux_system` | Flux system health with Mermaid topology |
| | `get_flux_resource_tree` | Dependency tree with Mermaid graph |

### Testing

```bash
# Unit tests (no cluster required — uses fake clientset)
go test ./...

# Integration tests (requires a live cluster)
go test -tags=integration ./pkg/k8s/ -v
go test -tags=integration ./pkg/tools/ -v
```

---

## VS Code Extension

### Prerequisites

- Node.js 18+
- VS Code 1.99+ with GitHub Copilot Chat extension

### Build and Install

```bash
cd kube-doctor-vscode

# Install dependencies
npm install

# Build
npm run compile

# Package as .vsix
npx @vscode/vsce package

# Install in VS Code
code --install-extension kube-doctor-0.1.0.vsix
```

### Usage

After installation, open the Copilot Chat panel in **Agent** mode. The extension registers tools directly with the Language Model Tools API — no MCP configuration needed.

The tools use your current kubeconfig context automatically.

### Tool Reference Names

You can reference tools directly in Copilot Chat with `#`:

| Reference | What it does |
|-----------|--------------|
| `#kubeListNamespaces` | List namespaces |
| `#kubeListPods` | List pods in a namespace |
| `#kubeGetPodLogs` | Get pod container logs |
| `#kubeGetEvents` | Get cluster events |
| `#kubeListDeployments` | List deployments |
| `#kubeListNodes` | List nodes |
| `#kubeListServices` | List services |
| `#kubeDiagnosePod` | Diagnose a specific pod |
| `#kubeDiagnoseNamespace` | Diagnose a namespace |
| `#kubeDiagnoseCluster` | Cluster-wide health check |
| `#kubeFindUnhealthyPods` | Find unhealthy pods |
| `#kubeListNetworkPolicies` | List network policies |
| `#kubeAnalyzePodConnectivity` | Analyze pod network connectivity (Mermaid) |
| `#kubeAnalyzePodSecurity` | Analyze pod security posture |
| `#kubeGetWorkloadDependencies` | Map workload dependencies (Mermaid) |

### Why a VS Code Extension?

Some corporate environments restrict or block MCP servers via policy. The VS Code Language Model Tools API is a first-party extension point that doesn't require MCP to be enabled, letting you bring Kubernetes diagnostics to Copilot even in locked-down environments.

---

## Architecture

Both implementations follow the same pattern:

```
K8s Client Layer   →   Tool Handlers   →   Transport
(API queries)          (formatting)        (MCP stdio / VS Code LM API)
```

- **K8s layer** — Thin wrappers around `client-go` (Go) or `@kubernetes/client-node` (TypeScript). Read-only. All calls use a 30-second timeout.
- **Tool handlers** — Format Kubernetes API responses into structured text with headers, tables, and severity-tagged findings (`[CRITICAL]`, `[WARNING]`, `[INFO]`).
- **Transport** — Go server uses MCP stdio. VS Code extension registers tools directly with the LM API.

### Output Format

All tools produce structured human-readable text (not JSON):

```
=== Pod Diagnosis: crasher (namespace: default) ===
STATUS: CrashLoopBackOff
RESTARTS: 42
NODE: k3d-test-server-0
AGE: 2h

FINDINGS:
[CRITICAL] Container 'crasher' is in CrashLoopBackOff
  - Last termination reason: Error
  - Exit code: 1
[WARNING] Container 'crasher' has high restart count: 42

SUGGESTED ACTIONS:
1. Check application logs for container 'crasher'
```

---

## Quick Start with k3d

To test locally with a disposable cluster:

```bash
# Create a cluster
k3d cluster create kube-doctor-test

# Deploy some test workloads
kubectl create deployment nginx --image=nginx --replicas=3
kubectl create deployment broken --image=nginx:does-not-exist
kubectl run crasher --image=busybox -- sh -c "exit 1"

# Build and test
go build -o kube-doctor .
npx @modelcontextprotocol/inspector ./kube-doctor

# Clean up
k3d cluster delete kube-doctor-test
```

---

## Project Structure

```
kube-doctor-mcp/
├── README.md
├── CLAUDE.md                              ← AI assistant instructions
├── main.go                                ← Server entry point
├── go.mod
├── .github/agents/kube-doctor.agent.md    ← GitHub Copilot agent
├── pkg/
│   ├── k8s/                               ← Kubernetes client wrappers
│   ├── flux/                              ← FluxCD client wrappers (controller-runtime)
│   ├── tools/                             ← MCP tool handlers (14 files)
│   └── util/                              ← Formatting, filters, error helpers
├── .vscode/mcp.json                       ← VS Code MCP config
└── kube-doctor-vscode/                    ← VS Code extension (15 tools)
    ├── package.json
    ├── src/
    │   ├── extension.ts
    │   ├── k8s/                           ← Kubernetes client wrappers
    │   ├── tools/                         ← Language Model Tool implementations
    │   └── util/                          ← Formatting helpers
    ├── kube-doctor-0.1.0.vsix             ← Pre-built extension
    └── dist/extension.js                  ← Bundled output
```
