import * as vscode from 'vscode';
import { listPods, podPhaseReason, podContainerSummary } from '../k8s/pods';
import { getPodLogs } from '../k8s/logs';
import { formatAge, formatTable, formatError } from '../util/formatting';

interface ListPodsInput {
    namespace: string;
    labelSelector?: string;
}

interface GetPodLogsInput {
    namespace: string;
    name: string;
    container?: string;
    tailLines?: number;
    previous?: boolean;
}

export class ListPodsTool implements vscode.LanguageModelTool<ListPodsInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<ListPodsInput>
    ): Promise<vscode.PreparedToolInvocation> {
        const ns = options.input.namespace || 'all namespaces';
        return { invocationMessage: `Listing pods in ${ns}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<ListPodsInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, labelSelector } = options.input;
            const pods = await listPods(namespace, labelSelector);

            const headers = ['NAME', 'NAMESPACE', 'STATUS', 'READY', 'RESTARTS', 'AGE', 'NODE'];
            const rows = pods.map(p => {
                const { ready, total, restarts } = podContainerSummary(p);
                return [
                    p.metadata?.name || '',
                    p.metadata?.namespace || '',
                    podPhaseReason(p),
                    `${ready}/${total}`,
                    `${restarts}`,
                    formatAge(p.metadata?.creationTimestamp),
                    p.spec?.nodeName || '',
                ];
            });

            const displayNs = !namespace || namespace === 'all' ? 'all' : namespace;
            const output = `=== Pods (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${pods.length} pods`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing pods', err))]);
        }
    }
}

export class GetPodLogsTool implements vscode.LanguageModelTool<GetPodLogsInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<GetPodLogsInput>
    ): Promise<vscode.PreparedToolInvocation> {
        const { namespace, name, previous } = options.input;
        const msg = `Getting logs from ${namespace}/${name}${previous ? ' (previous)' : ''}...`;
        return { invocationMessage: msg };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<GetPodLogsInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, name, container, tailLines, previous } = options.input;
            const logs = await getPodLogs(namespace, name, container, tailLines, previous);

            let header = `=== Logs: ${namespace}/${name} ===`;
            if (container) { header += ` (container: ${container})`; }
            if (previous) { header += ' [previous]'; }

            const output = logs ? `${header}\n\n${logs}` : `${header}\n\n(no logs available)`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting pod logs', err))]);
        }
    }
}
