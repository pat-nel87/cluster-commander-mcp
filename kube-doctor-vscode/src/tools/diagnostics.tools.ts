import * as vscode from 'vscode';
import { getPod, listPods, podPhaseReason, podContainerSummary, isPodHealthy } from '../k8s/pods';
import { getPodLogs } from '../k8s/logs';
import { getEventsForObject, listEvents } from '../k8s/events';
import { listDeployments } from '../k8s/workloads';
import { listNodes, nodeStatus } from '../k8s/nodes';
import { formatAge, formatTable, formatError } from '../util/formatting';
import { getCoreApi } from '../k8s/client';

// ---- diagnose_pod ----

interface DiagnosePodInput { namespace: string; name: string; }

export class DiagnosePodTool implements vscode.LanguageModelTool<DiagnosePodInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<DiagnosePodInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Diagnosing pod ${options.input.namespace}/${options.input.name}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<DiagnosePodInput>): Promise<vscode.LanguageModelToolResult> {
        const { namespace, name } = options.input;
        const lines: string[] = [];
        try {
            const pod = await getPod(namespace, name);
            const { restarts } = podContainerSummary(pod);
            const phase = podPhaseReason(pod);

            lines.push(`=== Pod Diagnosis: ${name} (namespace: ${namespace}) ===`);
            lines.push(`STATUS: ${phase}`);
            lines.push(`RESTARTS: ${restarts}`);
            lines.push(`NODE: ${pod.spec?.nodeName || 'unassigned'}`);
            lines.push(`AGE: ${formatAge(pod.metadata?.creationTimestamp)}`);
            lines.push('');
            lines.push('FINDINGS:');

            let findings = 0;

            // Check container statuses
            for (const cs of pod.status?.containerStatuses || []) {
                if (cs.state?.waiting) {
                    const reason = cs.state.waiting.reason || 'Unknown';
                    if (reason === 'CrashLoopBackOff') {
                        lines.push(`[CRITICAL] Container '${cs.name}' is in CrashLoopBackOff`);
                        if (cs.lastState?.terminated) {
                            lines.push(`  - Last termination reason: ${cs.lastState.terminated.reason || 'Unknown'}`);
                            lines.push(`  - Exit code: ${cs.lastState.terminated.exitCode}`);
                            if (cs.lastState.terminated.reason === 'OOMKilled') {
                                lines.push('  - Container was killed due to out-of-memory');
                            }
                        }
                        findings++;
                    } else if (reason === 'ImagePullBackOff' || reason === 'ErrImagePull') {
                        lines.push(`[CRITICAL] Container '${cs.name}' cannot pull image: ${cs.state.waiting.message || ''}`);
                        findings++;
                    } else {
                        lines.push(`[WARNING] Container '${cs.name}' is waiting: ${reason}`);
                        findings++;
                    }
                }
                if (cs.state?.terminated && cs.state.terminated.exitCode !== 0) {
                    lines.push(`[WARNING] Container '${cs.name}' terminated with exit code ${cs.state.terminated.exitCode} (${cs.state.terminated.reason || ''})`);
                    findings++;
                }
                if ((cs.restartCount || 0) > 5) {
                    lines.push(`[WARNING] Container '${cs.name}' has high restart count: ${cs.restartCount}`);
                    findings++;
                }
            }

            // Check conditions
            for (const cond of pod.status?.conditions || []) {
                if (cond.status === 'False' && cond.type === 'PodScheduled') {
                    lines.push(`[CRITICAL] Pod not scheduled: ${cond.message || ''}`);
                    findings++;
                }
            }

            // Resource limits
            for (const c of pod.spec?.containers || []) {
                if (!c.resources?.limits?.['memory']) {
                    lines.push(`[INFO] Container '${c.name}' has no memory limit set`);
                    findings++;
                }
            }

            if (findings === 0) {
                lines.push('  No issues found - pod appears healthy.');
            }

            // Warning events
            try {
                const events = await getEventsForObject(namespace, name);
                const warnings = events.filter(e => e.type === 'Warning');
                if (warnings.length > 0) {
                    lines.push('');
                    lines.push(`[WARNING] ${warnings.length} Warning events in recent history`);
                    for (const e of warnings.slice(0, 10)) {
                        lines.push(`  - ${e.reason}: ${e.message}${(e.count || 1) > 1 ? ` (x${e.count})` : ''}`);
                    }
                }
            } catch { /* events are best-effort */ }

            // Fetch logs from crashing containers
            for (const cs of pod.status?.containerStatuses || []) {
                if (cs.state?.waiting?.reason === 'CrashLoopBackOff') {
                    lines.push('');
                    lines.push(`RECENT LOGS (container '${cs.name}', previous instance):`);
                    try {
                        const logs = await getPodLogs(namespace, name, cs.name, 50, true);
                        lines.push(logs || '(no logs available)');
                    } catch {
                        lines.push('(could not fetch logs)');
                    }
                }
            }

            // Suggested actions
            lines.push('');
            lines.push('SUGGESTED ACTIONS:');
            let actionNum = 1;
            for (const cs of pod.status?.containerStatuses || []) {
                if (cs.lastState?.terminated?.reason === 'OOMKilled') {
                    const container = pod.spec?.containers?.find(c => c.name === cs.name);
                    const memLimit = container?.resources?.limits?.['memory'] || 'unknown';
                    lines.push(`${actionNum++}. Increase memory limit for container '${cs.name}' (currently ${memLimit}, OOMKilled)`);
                }
                if (cs.state?.waiting?.reason === 'ImagePullBackOff' || cs.state?.waiting?.reason === 'ErrImagePull') {
                    lines.push(`${actionNum++}. Check image name and registry credentials for container '${cs.name}'`);
                }
                if (cs.state?.waiting?.reason === 'CrashLoopBackOff') {
                    lines.push(`${actionNum++}. Check application logs for container '${cs.name}'`);
                }
            }
            if (pod.status?.phase === 'Pending') {
                lines.push(`${actionNum++}. Check cluster capacity and node selectors/tolerations`);
            }
            if (actionNum === 1) {
                lines.push('  No specific actions needed - pod is healthy.');
            }
        } catch (err) {
            lines.push(formatError(`diagnosing pod ${namespace}/${name}`, err));
        }

        return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
    }
}

// ---- diagnose_namespace ----

interface DiagnoseNamespaceInput { namespace: string; }

export class DiagnoseNamespaceTool implements vscode.LanguageModelTool<DiagnoseNamespaceInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<DiagnoseNamespaceInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Diagnosing namespace ${options.input.namespace}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<DiagnoseNamespaceInput>): Promise<vscode.LanguageModelToolResult> {
        const { namespace } = options.input;
        const lines: string[] = [];
        let findings = 0;

        try {
            lines.push(`=== Namespace Diagnosis: ${namespace} ===`);
            lines.push('');

            // 1. Pod health
            const pods = await listPods(namespace);
            const unhealthy = pods.filter(p => !isPodHealthy(p));
            const highRestarts = pods.filter(p => podContainerSummary(p).restarts > 5);

            lines.push('--- Pod Summary ---');
            lines.push(`  Total: ${pods.length}, Unhealthy: ${unhealthy.length}, High Restarts: ${highRestarts.length}`);

            if (unhealthy.length > 0) {
                lines.push('');
                lines.push(`[CRITICAL] ${unhealthy.length} unhealthy pods`);
                for (const p of unhealthy) {
                    lines.push(`  - ${p.metadata?.name}: ${podPhaseReason(p)} (restarts: ${podContainerSummary(p).restarts})`);
                }
                findings++;
            }

            if (highRestarts.length > 0) {
                lines.push('');
                lines.push(`[WARNING] ${highRestarts.length} pods with >5 restarts`);
                for (const p of highRestarts) {
                    lines.push(`  - ${p.metadata?.name}: ${podContainerSummary(p).restarts} restarts`);
                }
                findings++;
            }

            // 2. Deployment health
            try {
                const deployments = await listDeployments(namespace);
                const failing = deployments.filter(d => (d.status?.availableReplicas || 0) < (d.spec?.replicas || 0));
                if (failing.length > 0) {
                    lines.push('');
                    lines.push(`[WARNING] ${failing.length} deployments with unavailable replicas`);
                    for (const d of failing) {
                        lines.push(`  - ${d.metadata?.name}: ${d.status?.availableReplicas || 0}/${d.spec?.replicas || 0} available`);
                    }
                    findings++;
                }
            } catch { /* best effort */ }

            // 3. Warning events in last hour
            try {
                const events = await listEvents(namespace);
                const oneHourAgo = Date.now() - 3600000;
                const warnings = events.filter(e => {
                    const t = e.lastTimestamp ? new Date(e.lastTimestamp).getTime() : (e.metadata?.creationTimestamp ? new Date(e.metadata.creationTimestamp).getTime() : 0);
                    return e.type === 'Warning' && t > oneHourAgo;
                });
                if (warnings.length > 0) {
                    lines.push('');
                    lines.push(`[WARNING] ${warnings.length} warning events in the last hour`);
                    findings++;
                }
            } catch { /* best effort */ }

            // 4. Pending PVCs
            try {
                const api = getCoreApi();
                const pvcs = await api.listNamespacedPersistentVolumeClaim({ namespace });
                const pending = pvcs.items.filter(p => p.status?.phase !== 'Bound');
                if (pending.length > 0) {
                    lines.push('');
                    lines.push(`[WARNING] ${pending.length} PVCs not bound`);
                    for (const p of pending) {
                        lines.push(`  - ${p.metadata?.name}: ${p.status?.phase}`);
                    }
                    findings++;
                }
            } catch { /* best effort */ }

            lines.push('');
            lines.push('--- Overall Assessment ---');
            if (findings === 0) {
                lines.push('  Namespace appears healthy. No issues found.');
            } else {
                lines.push(`  ${findings} issue(s) found. Review findings above.`);
            }
        } catch (err) {
            lines.push(formatError(`diagnosing namespace ${namespace}`, err));
        }

        return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
    }
}

// ---- diagnose_cluster ----

export class DiagnoseClusterTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Running cluster-wide health check...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        const lines: string[] = [];
        let findings = 0;

        try {
            lines.push('=== Cluster Health Report ===');
            lines.push('');

            // 1. Node health
            const nodes = await listNodes();
            lines.push('--- Node Health ---');
            let notReady = 0;
            for (const n of nodes) {
                const status = nodeStatus(n);
                if (status !== 'Ready') {
                    lines.push(`  [CRITICAL] Node '${n.metadata?.name}' is ${status}`);
                    notReady++;
                    findings++;
                }
                for (const cond of n.status?.conditions || []) {
                    if (['MemoryPressure', 'DiskPressure', 'PIDPressure'].includes(cond.type || '') && cond.status === 'True') {
                        lines.push(`  [WARNING] Node '${n.metadata?.name}' has ${cond.type}`);
                        findings++;
                    }
                }
            }
            if (notReady === 0) {
                lines.push(`  All ${nodes.length} nodes healthy.`);
            }

            // 2. Pod summary
            const pods = await listPods('all');
            const phases: Record<string, number> = {};
            let unhealthyCount = 0;
            for (const p of pods) {
                const phase = p.status?.phase || 'Unknown';
                phases[phase] = (phases[phase] || 0) + 1;
                if (!isPodHealthy(p)) { unhealthyCount++; }
            }

            lines.push('');
            lines.push('--- Pod Summary (all namespaces) ---');
            lines.push(`  Total: ${pods.length}`);
            for (const [phase, count] of Object.entries(phases)) {
                lines.push(`  ${phase}: ${count}`);
            }
            if (unhealthyCount > 0) {
                lines.push(`  [WARNING] ${unhealthyCount} unhealthy pods cluster-wide`);
                findings++;
            }

            // 3. Warning events
            try {
                const events = await listEvents(undefined);
                const oneHourAgo = Date.now() - 3600000;
                const warnings = events.filter(e => {
                    const t = e.lastTimestamp ? new Date(e.lastTimestamp).getTime() : 0;
                    return e.type === 'Warning' && t > oneHourAgo;
                });
                lines.push('');
                lines.push('--- Recent Events ---');
                if (warnings.length > 0) {
                    lines.push(`  [WARNING] ${warnings.length} warning events in the last hour`);
                    findings++;
                } else {
                    lines.push('  No warning events in the last hour.');
                }
            } catch { /* best effort */ }

            // 4. kube-system health
            try {
                const ksPods = await listPods('kube-system');
                const ksUnhealthy = ksPods.filter(p => !isPodHealthy(p));
                lines.push('');
                lines.push('--- kube-system Health ---');
                if (ksUnhealthy.length > 0) {
                    lines.push(`  [CRITICAL] ${ksUnhealthy.length} unhealthy pods in kube-system`);
                    for (const p of ksUnhealthy) {
                        lines.push(`  - ${p.metadata?.name}: ${podPhaseReason(p)}`);
                    }
                    findings++;
                } else {
                    lines.push(`  All ${ksPods.length} kube-system pods healthy.`);
                }
            } catch { /* best effort */ }

            lines.push('');
            lines.push('--- Overall Assessment ---');
            if (findings === 0) {
                lines.push('  Cluster appears healthy. No issues found.');
            } else {
                lines.push(`  ${findings} issue(s) found. Review findings above.`);
            }
        } catch (err) {
            lines.push(formatError('diagnosing cluster', err));
        }

        return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
    }
}

// ---- find_unhealthy_pods ----

interface FindUnhealthyPodsInput { namespace?: string; }

export class FindUnhealthyPodsTool implements vscode.LanguageModelTool<FindUnhealthyPodsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<FindUnhealthyPodsInput>): Promise<vscode.PreparedToolInvocation> {
        const ns = options.input.namespace || 'all namespaces';
        return { invocationMessage: `Finding unhealthy pods in ${ns}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<FindUnhealthyPodsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const ns = options.input.namespace || 'all';
            const pods = await listPods(ns);
            const unhealthy = pods.filter(p => !isPodHealthy(p));

            const headers = ['NAME', 'NAMESPACE', 'STATUS', 'RESTARTS', 'AGE', 'NODE'];
            const rows = unhealthy.map(p => [
                p.metadata?.name || '',
                p.metadata?.namespace || '',
                podPhaseReason(p),
                `${podContainerSummary(p).restarts}`,
                formatAge(p.metadata?.creationTimestamp),
                p.spec?.nodeName || '',
            ]);

            const displayNs = ns === 'all' || !ns ? 'all' : ns;
            let output = `=== Unhealthy Pods (namespace: ${displayNs}) ===\n`;
            if (rows.length === 0) {
                output += 'No unhealthy pods found.';
            } else {
                output += `${formatTable(headers, rows)}\n\nFound ${unhealthy.length} unhealthy pods out of ${pods.length} total`;
            }
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('finding unhealthy pods', err))]);
        }
    }
}
