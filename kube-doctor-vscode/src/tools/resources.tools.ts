import * as vscode from 'vscode';
import { listPods } from '../k8s/pods';
import { listNodes } from '../k8s/nodes';
import { listLimitRanges } from '../k8s/quotas';
import { getPodMetrics, parseCPU, parseMemory, formatBytes } from '../k8s/metrics';
import { formatTable, formatError } from '../util/formatting';

// ---- analyze_resource_allocation ----

interface AnalyzeResourceAllocationInput { namespace?: string; }

export class AnalyzeResourceAllocationTool implements vscode.LanguageModelTool<AnalyzeResourceAllocationInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<AnalyzeResourceAllocationInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Analyzing resource allocation in ${options.input.namespace || 'cluster-wide'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<AnalyzeResourceAllocationInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const ns = options.input.namespace || 'all';
            const scope = !ns || ns === 'all' ? 'all' : ns;
            const lines: string[] = [];
            lines.push(`=== Resource Allocation Analysis (scope: ${scope}) ===`);
            lines.push('');

            const pods = await listPods(ns);
            let cpuRequests = 0, cpuLimits = 0, memRequests = 0, memLimits = 0;

            for (const pod of pods) {
                if (pod.status?.phase === 'Succeeded' || pod.status?.phase === 'Failed') { continue; }
                for (const c of pod.spec?.containers || []) {
                    if (c.resources?.requests) {
                        cpuRequests += parseCPU(c.resources.requests['cpu'] || '0');
                        memRequests += parseMemory(c.resources.requests['memory'] || '0');
                    }
                    if (c.resources?.limits) {
                        cpuLimits += parseCPU(c.resources.limits['cpu'] || '0');
                        memLimits += parseMemory(c.resources.limits['memory'] || '0');
                    }
                }
            }

            lines.push('--- Resource Summary ---');
            lines.push(`  Active Pods:     ${pods.length}`);
            lines.push(`  CPU Requests:    ${cpuRequests}m`);
            lines.push(`  CPU Limits:      ${cpuLimits}m`);
            lines.push(`  Memory Requests: ${formatBytes(memRequests)}`);
            lines.push(`  Memory Limits:   ${formatBytes(memLimits)}`);

            // Node capacity
            try {
                const nodes = await listNodes();
                let cpuAlloc = 0, memAlloc = 0;
                for (const n of nodes) {
                    cpuAlloc += parseCPU(n.status?.allocatable?.['cpu'] || '0') * 1000;
                    memAlloc += parseMemory(n.status?.allocatable?.['memory'] || '0');
                }
                lines.push(`\n  Node Allocatable (total across ${nodes.length} nodes):`);
                lines.push(`    CPU:    ${cpuAlloc}m`);
                lines.push(`    Memory: ${formatBytes(memAlloc)}`);

                if (cpuAlloc > 0) {
                    const cpuReqPct = (cpuRequests / cpuAlloc * 100).toFixed(1);
                    const cpuLimPct = (cpuLimits / cpuAlloc * 100).toFixed(1);
                    lines.push(`\n  CPU Utilization: requests=${cpuReqPct}%, limits=${cpuLimPct}%`);
                    if (parseFloat(cpuReqPct) > 80) {
                        lines.push('[WARNING] CPU requests exceed 80% of allocatable capacity');
                    }
                }
                if (memAlloc > 0) {
                    const memReqPct = (memRequests / memAlloc * 100).toFixed(1);
                    const memLimPct = (memLimits / memAlloc * 100).toFixed(1);
                    lines.push(`  Memory Utilization: requests=${memReqPct}%, limits=${memLimPct}%`);
                    if (parseFloat(memReqPct) > 80) {
                        lines.push('[WARNING] Memory requests exceed 80% of allocatable capacity');
                    }
                }
            } catch { /* proceed without node data */ }

            // Actual usage via metrics
            try {
                const podMetrics = await getPodMetrics(ns);
                if (podMetrics.length > 0) {
                    let cpuUsage = 0, memUsage = 0;
                    for (const pm of podMetrics) {
                        for (const c of pm.containers) {
                            cpuUsage += parseCPU(c.usage.cpu);
                            memUsage += parseMemory(c.usage.memory);
                        }
                    }
                    lines.push(`\n  Actual Usage (from metrics-server):`);
                    lines.push(`    CPU:    ${cpuUsage}m`);
                    lines.push(`    Memory: ${formatBytes(memUsage)}`);
                }
            } catch { /* metrics not available */ }

            // Mermaid chart
            lines.push('\nRESOURCE ALLOCATION CHART:');
            lines.push('```mermaid');
            lines.push('xychart-beta');
            lines.push(`    title "Resource Allocation (${scope})"`);
            lines.push('    x-axis ["CPU Req (m)", "CPU Lim (m)", "Mem Req (Mi)", "Mem Lim (Mi)"]');
            lines.push(`    bar [${cpuRequests}, ${cpuLimits}, ${Math.round(memRequests / (1024 * 1024))}, ${Math.round(memLimits / (1024 * 1024))}]`);
            lines.push('```');

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('analyzing resource allocation', err))]);
        }
    }
}

// ---- list_limit_ranges ----

interface ListLimitRangesInput { namespace: string; }

export class ListLimitRangesTool implements vscode.LanguageModelTool<ListLimitRangesInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListLimitRangesInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing limit ranges in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListLimitRangesInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const limitRanges = await listLimitRanges(options.input.namespace);
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const lines: string[] = [];
            lines.push(`=== Limit Ranges (namespace: ${displayNs}) ===`);

            if (limitRanges.length === 0) {
                lines.push('(none)');
                return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
            }

            for (const lr of limitRanges) {
                lines.push(`\n--- ${lr.metadata?.namespace}/${lr.metadata?.name} ---`);
                const headers = ['TYPE', 'RESOURCE', 'DEFAULT', 'DEFAULT-REQUEST', 'MIN', 'MAX'];
                const rows: string[][] = [];
                for (const item of lr.spec?.limits || []) {
                    const resources = new Set<string>();
                    for (const r of Object.keys(item._default || {})) { resources.add(r); }
                    for (const r of Object.keys(item.defaultRequest || {})) { resources.add(r); }
                    for (const r of Object.keys(item.min || {})) { resources.add(r); }
                    for (const r of Object.keys(item.max || {})) { resources.add(r); }
                    for (const res of resources) {
                        rows.push([
                            item.type || '', res,
                            (item._default as any)?.[res] || '-',
                            (item.defaultRequest as any)?.[res] || '-',
                            (item.min as any)?.[res] || '-',
                            (item.max as any)?.[res] || '-',
                        ]);
                    }
                }
                lines.push(formatTable(headers, rows));
            }

            lines.push(`\nFound ${limitRanges.length} limit ranges`);
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing limit ranges', err))]);
        }
    }
}
