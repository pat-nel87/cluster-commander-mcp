import * as vscode from 'vscode';
import { getNodeMetrics, getPodMetrics, parseCPU, parseMemory, formatBytes } from '../k8s/metrics';
import { listNodes } from '../k8s/nodes';
import { formatTable, formatError } from '../util/formatting';

// ---- get_node_metrics ----

export class GetNodeMetricsTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Getting node metrics...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const metrics = await getNodeMetrics();
            // Get node capacity for %
            const capacityMap: Record<string, { cpu: number; mem: number }> = {};
            try {
                const nodes = await listNodes();
                for (const n of nodes) {
                    const cpuCap = n.status?.capacity?.['cpu'] || '0';
                    const memCap = n.status?.capacity?.['memory'] || '0';
                    capacityMap[n.metadata?.name || ''] = {
                        cpu: parseCPU(cpuCap) * 1000, // capacity is in cores, convert to millicores
                        mem: parseMemory(memCap),
                    };
                }
            } catch { /* proceed without capacity */ }

            const headers = ['NODE', 'CPU USAGE', 'CPU %', 'MEMORY USAGE', 'MEMORY %'];
            const rows = metrics.map(m => {
                const cpuUsage = parseCPU(m.usage.cpu);
                const memUsage = parseMemory(m.usage.memory);
                let cpuPct = 'N/A';
                let memPct = 'N/A';
                const cap = capacityMap[m.name];
                if (cap) {
                    if (cap.cpu > 0) { cpuPct = `${(cpuUsage / cap.cpu * 100).toFixed(1)}%`; }
                    if (cap.mem > 0) { memPct = `${(memUsage / cap.mem * 100).toFixed(1)}%`; }
                }
                return [m.name, `${cpuUsage}m`, cpuPct, formatBytes(memUsage), memPct];
            });
            const output = `=== Node Metrics ===\n${formatTable(headers, rows)}\n\nFound ${metrics.length} nodes with metrics`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting node metrics (requires metrics-server)', err))]);
        }
    }
}

// ---- get_pod_metrics ----

interface GetPodMetricsInput { namespace: string; }

export class GetPodMetricsTool implements vscode.LanguageModelTool<GetPodMetricsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<GetPodMetricsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Getting pod metrics in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<GetPodMetricsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const metrics = await getPodMetrics(options.input.namespace);
            const headers = ['POD', 'NAMESPACE', 'CONTAINER', 'CPU', 'MEMORY'];
            const rows: string[][] = [];
            for (const m of metrics) {
                for (const c of m.containers) {
                    rows.push([
                        m.name, m.namespace, c.name,
                        `${parseCPU(c.usage.cpu)}m`, formatBytes(parseMemory(c.usage.memory)),
                    ]);
                }
            }
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== Pod Metrics (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${metrics.length} pods with metrics`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting pod metrics (requires metrics-server)', err))]);
        }
    }
}

// ---- top_resource_consumers ----

interface TopResourceConsumersInput { namespace?: string; resource: string; limit?: number; }

export class TopResourceConsumersTool implements vscode.LanguageModelTool<TopResourceConsumersInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<TopResourceConsumersInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Finding top ${options.input.resource} consumers...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<TopResourceConsumersInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, resource, limit: inputLimit } = options.input;
            const limitN = inputLimit && inputLimit > 0 ? inputLimit : 10;
            const metrics = await getPodMetrics(namespace);

            let usages = metrics.map(m => {
                let totalCPU = 0;
                let totalMem = 0;
                for (const c of m.containers) {
                    totalCPU += parseCPU(c.usage.cpu);
                    totalMem += parseMemory(c.usage.memory);
                }
                return { name: m.name, namespace: m.namespace, cpu: totalCPU, memory: totalMem };
            });

            // Sort
            const res = (resource || 'cpu').toLowerCase();
            if (res === 'memory' || res === 'mem') {
                usages.sort((a, b) => b.memory - a.memory);
            } else {
                usages.sort((a, b) => b.cpu - a.cpu);
            }
            usages = usages.slice(0, limitN);

            const headers = ['#', 'POD', 'NAMESPACE', 'CPU', 'MEMORY'];
            const rows = usages.map((u, i) => [
                `${i + 1}`, u.name, u.namespace, `${u.cpu}m`, formatBytes(u.memory),
            ]);
            const output = `=== Top ${limitN} Resource Consumers by ${resource || 'cpu'} ===\n${formatTable(headers, rows)}`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting top resource consumers (requires metrics-server)', err))]);
        }
    }
}
