import * as vscode from 'vscode';
import { listNamespaces } from '../k8s/namespaces';
import { listNodes, nodeStatus, nodeRoles } from '../k8s/nodes';
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
