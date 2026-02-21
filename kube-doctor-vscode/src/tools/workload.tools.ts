import * as vscode from 'vscode';
import { listDeployments } from '../k8s/workloads';
import { formatAge, formatTable, formatError } from '../util/formatting';

interface ListDeploymentsInput {
    namespace: string;
}

export class ListDeploymentsTool implements vscode.LanguageModelTool<ListDeploymentsInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<ListDeploymentsInput>
    ): Promise<vscode.PreparedToolInvocation> {
        const ns = options.input.namespace || 'all namespaces';
        return { invocationMessage: `Listing deployments in ${ns}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<ListDeploymentsInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const deployments = await listDeployments(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'READY', 'UP-TO-DATE', 'AVAILABLE', 'AGE'];
            const rows = deployments.map(d => {
                const desired = d.spec?.replicas ?? 0;
                return [
                    d.metadata?.name || '',
                    d.metadata?.namespace || '',
                    `${d.status?.readyReplicas || 0}/${desired}`,
                    `${d.status?.updatedReplicas || 0}`,
                    `${d.status?.availableReplicas || 0}`,
                    formatAge(d.metadata?.creationTimestamp),
                ];
            });

            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== Deployments (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${deployments.length} deployments`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing deployments', err))]);
        }
    }
}
