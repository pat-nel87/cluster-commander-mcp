import * as vscode from 'vscode';
import { listNamespaces } from '../k8s/namespaces';
import { listNodes, nodeStatus, nodeRoles, getNode } from '../k8s/nodes';
import { listPods, podPhaseReason, isPodHealthy } from '../k8s/pods';
import { listServices } from '../k8s/networking';
import { getKubeConfig, getCurrentContext } from '../k8s/client';
import { formatAge, formatTable, formatLabels, formatError } from '../util/formatting';

export class ListNamespacesTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Listing Kubernetes namespaces...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const namespaces = await listNamespaces();
            const headers = ['NAME', 'STATUS', 'AGE', 'LABELS'];
            const rows = namespaces.map(ns => [
                ns.metadata?.name || '',
                ns.status?.phase || '',
                formatAge(ns.metadata?.creationTimestamp),
                formatLabels(ns.metadata?.labels),
            ]);
            const output = `=== Namespaces ===\n${formatTable(headers, rows)}\n\nTotal: ${namespaces.length} namespaces`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing namespaces', err))]);
        }
    }
}

// ---- list_contexts ----

export class ListContextsTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Listing Kubernetes contexts...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const kc = getKubeConfig();
            const currentCtx = getCurrentContext();
            const contexts = kc.getContexts();
            const lines: string[] = [];
            lines.push('=== Kubernetes Contexts ===');
            lines.push(`Current context: ${currentCtx}`);
            lines.push('');
            for (const ctx of contexts) {
                const marker = ctx.name === currentCtx ? '* ' : '  ';
                lines.push(`${marker}${ctx.name}`);
            }
            lines.push(`\nTotal: ${contexts.length} contexts`);
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing contexts', err))]);
        }
    }
}

// ---- cluster_info ----

export class ClusterInfoTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Getting cluster information...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const lines: string[] = [];
            lines.push('=== Cluster Information ===');

            // Nodes
            try {
                const nodes = await listNodes();
                const readyCount = nodes.filter(n => nodeStatus(n) === 'Ready').length;
                lines.push(`Nodes: ${nodes.length} total, ${readyCount} ready`);
            } catch { lines.push('Nodes: (error)'); }

            // Namespaces
            try {
                const ns = await listNamespaces();
                lines.push(`Namespaces: ${ns.length}`);
            } catch { lines.push('Namespaces: (error)'); }

            // Pods
            try {
                const pods = await listPods('all');
                const running = pods.filter(p => p.status?.phase === 'Running').length;
                lines.push(`Pods: ${pods.length} total, ${running} running`);
            } catch { lines.push('Pods: (error)'); }

            // Services
            try {
                const svcs = await listServices('all');
                lines.push(`Services: ${svcs.length}`);
            } catch { lines.push('Services: (error)'); }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting cluster info', err))]);
        }
    }
}

// ---- get_node_detail ----

interface GetNodeDetailInput { name: string; }

export class GetNodeDetailTool implements vscode.LanguageModelTool<GetNodeDetailInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<GetNodeDetailInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Getting details for node ${options.input.name}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<GetNodeDetailInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const node = await getNode(options.input.name);
            const lines: string[] = [];
            lines.push(`=== Node: ${node.metadata?.name} ===`);
            lines.push(`Status: ${nodeStatus(node)}`);
            lines.push(`Roles: ${nodeRoles(node)}`);
            lines.push(`Version: ${node.status?.nodeInfo?.kubeletVersion || ''}`);
            lines.push(`OS: ${node.status?.nodeInfo?.osImage || ''}`);
            lines.push(`Kernel: ${node.status?.nodeInfo?.kernelVersion || ''}`);
            lines.push(`Container Runtime: ${node.status?.nodeInfo?.containerRuntimeVersion || ''}`);
            lines.push(`Age: ${formatAge(node.metadata?.creationTimestamp)}`);
            lines.push(`Labels: ${formatLabels(node.metadata?.labels)}`);

            // Allocatable
            lines.push('');
            lines.push('--- Allocatable ---');
            const alloc = node.status?.allocatable || {};
            lines.push(`  CPU: ${alloc['cpu'] || 'N/A'}`);
            lines.push(`  Memory: ${alloc['memory'] || 'N/A'}`);
            lines.push(`  Pods: ${alloc['pods'] || 'N/A'}`);

            // Conditions
            lines.push('');
            lines.push('--- Conditions ---');
            for (const cond of node.status?.conditions || []) {
                lines.push(`  ${(cond.type || '').padEnd(20)} ${cond.status}${cond.reason ? ` (${cond.reason})` : ''}`);
            }

            // Taints
            if (node.spec?.taints && node.spec.taints.length > 0) {
                lines.push('');
                lines.push('--- Taints ---');
                for (const taint of node.spec.taints) {
                    lines.push(`  ${taint.key}=${taint.value || ''}:${taint.effect}`);
                }
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError(`getting node ${options.input.name}`, err))]);
        }
    }
}

export class ListNodesTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Listing Kubernetes nodes...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const nodes = await listNodes();
            const headers = ['NAME', 'STATUS', 'ROLES', 'VERSION', 'CPU', 'MEMORY', 'AGE'];
            const rows = nodes.map(n => [
                n.metadata?.name || '',
                nodeStatus(n),
                nodeRoles(n),
                n.status?.nodeInfo?.kubeletVersion || '',
                n.status?.capacity?.['cpu'] || '',
                n.status?.capacity?.['memory'] || '',
                formatAge(n.metadata?.creationTimestamp),
            ]);
            const output = `=== Nodes ===\n${formatTable(headers, rows)}\n\nFound ${nodes.length} nodes`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing nodes', err))]);
        }
    }
}
