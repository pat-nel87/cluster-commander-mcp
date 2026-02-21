import * as vscode from 'vscode';
import { listDeployments, getDeployment, listReplicaSets, listStatefulSets, listDaemonSets, listJobs } from '../k8s/workloads';
import { getEventsForObject } from '../k8s/events';
import { formatAge, formatTable, formatLabels, formatError } from '../util/formatting';

interface ListDeploymentsInput { namespace: string; }

export class ListDeploymentsTool implements vscode.LanguageModelTool<ListDeploymentsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListDeploymentsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing deployments in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListDeploymentsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const deployments = await listDeployments(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'READY', 'UP-TO-DATE', 'AVAILABLE', 'AGE'];
            const rows = deployments.map(d => {
                const desired = d.spec?.replicas ?? 0;
                return [
                    d.metadata?.name || '', d.metadata?.namespace || '',
                    `${d.status?.readyReplicas || 0}/${desired}`, `${d.status?.updatedReplicas || 0}`,
                    `${d.status?.availableReplicas || 0}`, formatAge(d.metadata?.creationTimestamp),
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

// ---- get_deployment_detail ----

interface GetDeploymentDetailInput { namespace: string; name: string; }

export class GetDeploymentDetailTool implements vscode.LanguageModelTool<GetDeploymentDetailInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<GetDeploymentDetailInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Getting deployment ${options.input.namespace}/${options.input.name} details...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<GetDeploymentDetailInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, name } = options.input;
            const deploy = await getDeployment(namespace, name);
            const lines: string[] = [];
            const desired = deploy.spec?.replicas ?? 0;

            lines.push(`=== Deployment: ${deploy.metadata?.name} (namespace: ${deploy.metadata?.namespace}) ===`);
            lines.push(`Replicas: ${desired} desired, ${deploy.status?.readyReplicas || 0} ready, ${deploy.status?.availableReplicas || 0} available, ${deploy.status?.updatedReplicas || 0} updated`);
            lines.push(`Strategy: ${deploy.spec?.strategy?.type || 'RollingUpdate'}`);
            lines.push(`Age: ${formatAge(deploy.metadata?.creationTimestamp)}`);
            lines.push(`Labels: ${formatLabels(deploy.metadata?.labels)}`);
            if (deploy.spec?.selector?.matchLabels) {
                lines.push(`Selector: ${formatLabels(deploy.spec.selector.matchLabels)}`);
            }

            // Conditions
            lines.push('');
            lines.push('--- Conditions ---');
            for (const cond of deploy.status?.conditions || []) {
                lines.push(`  ${(cond.type || '').padEnd(20)} ${(cond.status || '').padEnd(6)} ${cond.message || ''}`);
            }

            // Pod template
            lines.push('');
            lines.push('--- Pod Template ---');
            for (const c of deploy.spec?.template?.spec?.containers || []) {
                lines.push(`  Container: ${c.name}`);
                lines.push(`    Image: ${c.image || ''}`);
                if (c.resources?.requests) {
                    lines.push(`    Requests: cpu=${c.resources.requests['cpu'] || '-'}, memory=${c.resources.requests['memory'] || '-'}`);
                }
                if (c.resources?.limits) {
                    lines.push(`    Limits:   cpu=${c.resources.limits['cpu'] || '-'}, memory=${c.resources.limits['memory'] || '-'}`);
                }
            }

            // ReplicaSets
            try {
                const selectorLabels = deploy.spec?.selector?.matchLabels || {};
                const labelSelector = Object.entries(selectorLabels).map(([k, v]) => `${k}=${v}`).join(',');
                if (labelSelector) {
                    const rsList = await listReplicaSets(namespace, labelSelector);
                    if (rsList.length > 0) {
                        lines.push('');
                        lines.push('--- ReplicaSets ---');
                        for (const rs of rsList) {
                            const rsDesired = rs.spec?.replicas ?? 0;
                            const revision = rs.metadata?.annotations?.['deployment.kubernetes.io/revision'] || '';
                            lines.push(`  ${rs.metadata?.name}: ${rs.status?.readyReplicas || 0}/${rsDesired} ready, revision=${revision}`);
                        }
                    }
                }
            } catch { /* best-effort */ }

            // Events
            try {
                const events = await getEventsForObject(namespace, name);
                if (events.length > 0) {
                    lines.push('');
                    lines.push('--- Recent Events ---');
                    for (const e of events) {
                        const count = (e.count || 1) > 1 ? ` (x${e.count})` : '';
                        lines.push(`  ${(e.type || '').padEnd(8)} ${(e.reason || '').padEnd(25)} ${e.message || ''}${count}`);
                    }
                }
            } catch { /* best-effort */ }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting deployment detail', err))]);
        }
    }
}

// ---- list_statefulsets ----

interface ListStatefulSetsInput { namespace: string; }

export class ListStatefulSetsTool implements vscode.LanguageModelTool<ListStatefulSetsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListStatefulSetsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing statefulsets in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListStatefulSetsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const sets = await listStatefulSets(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'READY', 'AGE'];
            const rows = sets.map(s => [
                s.metadata?.name || '', s.metadata?.namespace || '',
                `${s.status?.readyReplicas || 0}/${s.spec?.replicas ?? 0}`,
                formatAge(s.metadata?.creationTimestamp),
            ]);
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== StatefulSets (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${sets.length} statefulsets`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing statefulsets', err))]);
        }
    }
}

// ---- list_daemonsets ----

interface ListDaemonSetsInput { namespace: string; }

export class ListDaemonSetsTool implements vscode.LanguageModelTool<ListDaemonSetsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListDaemonSetsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing daemonsets in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListDaemonSetsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const sets = await listDaemonSets(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'DESIRED', 'READY', 'UP-TO-DATE', 'AVAILABLE', 'AGE'];
            const rows = sets.map(d => [
                d.metadata?.name || '', d.metadata?.namespace || '',
                `${d.status?.desiredNumberScheduled || 0}`, `${d.status?.numberReady || 0}`,
                `${d.status?.updatedNumberScheduled || 0}`, `${d.status?.numberAvailable || 0}`,
                formatAge(d.metadata?.creationTimestamp),
            ]);
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== DaemonSets (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${sets.length} daemonsets`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing daemonsets', err))]);
        }
    }
}

// ---- list_jobs ----

interface ListJobsInput { namespace: string; }

export class ListJobsTool implements vscode.LanguageModelTool<ListJobsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListJobsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing jobs in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListJobsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const jobs = await listJobs(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'COMPLETIONS', 'ACTIVE', 'SUCCEEDED', 'FAILED', 'AGE'];
            const rows = jobs.map(j => {
                const completions = j.spec?.completions ?? 1;
                return [
                    j.metadata?.name || '', j.metadata?.namespace || '',
                    `${j.status?.succeeded || 0}/${completions}`, `${j.status?.active || 0}`,
                    `${j.status?.succeeded || 0}`, `${j.status?.failed || 0}`,
                    formatAge(j.metadata?.creationTimestamp),
                ];
            });
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== Jobs (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${jobs.length} jobs`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing jobs', err))]);
        }
    }
}
