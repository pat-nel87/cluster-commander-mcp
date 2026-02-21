import * as vscode from 'vscode';
import { listPods, getPod, podPhaseReason, podContainerSummary } from '../k8s/pods';
import { getPodLogs } from '../k8s/logs';
import { getEventsForObject } from '../k8s/events';
import { formatAge, formatTable, formatLabels, formatError } from '../util/formatting';

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

// ---- get_pod_detail ----

interface GetPodDetailInput { namespace: string; name: string; }

export class GetPodDetailTool implements vscode.LanguageModelTool<GetPodDetailInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<GetPodDetailInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Getting details for pod ${options.input.namespace}/${options.input.name}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<GetPodDetailInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, name } = options.input;
            const pod = await getPod(namespace, name);
            const lines: string[] = [];

            lines.push(`=== Pod: ${pod.metadata?.name} (namespace: ${pod.metadata?.namespace}) ===`);
            lines.push(`Status: ${pod.status?.phase || 'Unknown'}`);
            lines.push(`Node: ${pod.spec?.nodeName || 'unassigned'}`);
            lines.push(`IP: ${pod.status?.podIP || ''}`);
            lines.push(`Age: ${formatAge(pod.metadata?.creationTimestamp)}`);
            lines.push(`Labels: ${formatLabels(pod.metadata?.labels)}`);
            if (pod.spec?.serviceAccountName) {
                lines.push(`Service Account: ${pod.spec.serviceAccountName}`);
            }

            // Containers
            lines.push('');
            lines.push('--- Containers ---');
            for (const c of pod.spec?.containers || []) {
                lines.push(`\n  Container: ${c.name}`);
                lines.push(`    Image: ${c.image || ''}`);
                if (c.resources?.requests) {
                    lines.push(`    Requests: cpu=${c.resources.requests['cpu'] || '-'}, memory=${c.resources.requests['memory'] || '-'}`);
                }
                if (c.resources?.limits) {
                    lines.push(`    Limits:   cpu=${c.resources.limits['cpu'] || '-'}, memory=${c.resources.limits['memory'] || '-'}`);
                }
                if (c.ports && c.ports.length > 0) {
                    const ports = c.ports.map(p => `${p.containerPort}/${p.protocol || 'TCP'}`).join(', ');
                    lines.push(`    Ports: ${ports}`);
                }
            }

            // Container statuses
            lines.push('');
            lines.push('--- Container Statuses ---');
            for (const cs of pod.status?.containerStatuses || []) {
                lines.push(`\n  ${cs.name}: ready=${cs.ready}, restarts=${cs.restartCount || 0}`);
                if (cs.state?.running) {
                    lines.push(`    State: Running (since ${formatAge(cs.state.running.startedAt)})`);
                }
                if (cs.state?.waiting) {
                    lines.push(`    State: Waiting (${cs.state.waiting.reason || ''}: ${cs.state.waiting.message || ''})`);
                }
                if (cs.state?.terminated) {
                    lines.push(`    State: Terminated (${cs.state.terminated.reason || ''}, exit code ${cs.state.terminated.exitCode})`);
                }
                if (cs.lastState?.terminated) {
                    const t = cs.lastState.terminated;
                    lines.push(`    Last Termination: ${t.reason || ''} (exit code ${t.exitCode}, ${formatAge(t.finishedAt)})`);
                }
            }

            // Conditions
            lines.push('');
            lines.push('--- Conditions ---');
            for (const cond of pod.status?.conditions || []) {
                lines.push(`  ${(cond.type || '').padEnd(20)} ${cond.status}${cond.reason ? ` (${cond.reason})` : ''}`);
            }

            // Events
            try {
                const events = await getEventsForObject(namespace, name);
                if (events.length > 0) {
                    lines.push('');
                    lines.push('--- Recent Events ---');
                    for (const e of events) {
                        const count = (e.count || 1) > 1 ? ` (x${e.count})` : '';
                        lines.push(`  ${(e.type || '').padEnd(8)} ${(e.reason || '').padEnd(20)} ${e.message || ''}${count}`);
                    }
                }
            } catch { /* best-effort */ }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError(`getting pod detail`, err))]);
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
