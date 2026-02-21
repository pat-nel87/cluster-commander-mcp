import * as vscode from 'vscode';
import {
    listKustomizations, getKustomization,
    listHelmReleases, getHelmRelease,
    listGitRepositories, listOCIRepositories, listHelmRepositories, listHelmCharts, listBuckets,
    listImageRepositories, listImagePolicies,
    getSource, getFluxHealthStatus, getConditionMessage, truncateRevision, formatFluxAge,
    isFluxNotInstalled, FluxCondition,
} from '../k8s/flux';
import { listPods, isPodHealthy, podPhaseReason } from '../k8s/pods';
import { listEvents } from '../k8s/events';
import { formatTable, formatError } from '../util/formatting';

function fluxNotInstalledResult(): vscode.LanguageModelToolResult {
    return new vscode.LanguageModelToolResult([
        new vscode.LanguageModelTextPart('FluxCD is not installed in this cluster. Flux CRDs were not found.'),
    ]);
}

function handleFluxError(action: string, err: any): vscode.LanguageModelToolResult {
    if (isFluxNotInstalled(err)) { return fluxNotInstalledResult(); }
    return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError(action, err))]);
}

// ---- list_flux_kustomizations ----

interface FluxNamespaceInput { namespace?: string; }

export class ListFluxKustomizationsTool implements vscode.LanguageModelTool<FluxNamespaceInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<FluxNamespaceInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing Flux Kustomizations in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<FluxNamespaceInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const items = await listKustomizations(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'SOURCE', 'PATH', 'STATUS', 'REVISION', 'SUSPENDED', 'AGE'];
            const rows = items.map((ks: any) => {
                const conditions: FluxCondition[] = ks.status?.conditions || [];
                const sourceRef = ks.spec?.sourceRef;
                const source = sourceRef ? `${sourceRef.kind}/${sourceRef.name}` : '';
                return [
                    ks.metadata?.name || '', ks.metadata?.namespace || '', source,
                    ks.spec?.path || './',
                    getFluxHealthStatus(conditions, ks.spec?.suspend),
                    truncateRevision(ks.status?.lastAppliedRevision),
                    ks.spec?.suspend ? 'true' : 'false',
                    formatFluxAge(ks.metadata?.creationTimestamp),
                ];
            });
            const displayNs = options.input.namespace || 'all';
            const output = `=== Flux Kustomizations (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${items.length} kustomizations`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return handleFluxError('listing Flux Kustomizations', err);
        }
    }
}

// ---- list_flux_helm_releases ----

export class ListFluxHelmReleasesTool implements vscode.LanguageModelTool<FluxNamespaceInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<FluxNamespaceInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing Flux HelmReleases in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<FluxNamespaceInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const items = await listHelmReleases(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'CHART', 'VERSION', 'STATUS', 'SUSPENDED', 'AGE'];
            const rows = items.map((hr: any) => {
                const conditions: FluxCondition[] = hr.status?.conditions || [];
                const chart = hr.spec?.chart?.spec?.chart || hr.spec?.chartRef?.name || '';
                const version = hr.spec?.chart?.spec?.version || '';
                return [
                    hr.metadata?.name || '', hr.metadata?.namespace || '', chart, version,
                    getFluxHealthStatus(conditions, hr.spec?.suspend),
                    hr.spec?.suspend ? 'true' : 'false',
                    formatFluxAge(hr.metadata?.creationTimestamp),
                ];
            });
            const displayNs = options.input.namespace || 'all';
            const output = `=== Flux HelmReleases (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${items.length} helm releases`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return handleFluxError('listing Flux HelmReleases', err);
        }
    }
}

// ---- list_flux_sources ----

interface ListFluxSourcesInput { namespace?: string; sourceType?: string; }

export class ListFluxSourcesTool implements vscode.LanguageModelTool<ListFluxSourcesInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListFluxSourcesInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing Flux sources in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListFluxSourcesInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, sourceType } = options.input;
            const headers = ['TYPE', 'NAME', 'NAMESPACE', 'URL', 'REVISION', 'STATUS', 'AGE'];
            const rows: string[][] = [];

            const shouldInclude = (type: string) => !sourceType || sourceType.toLowerCase() === type.toLowerCase();

            if (shouldInclude('git')) {
                try {
                    const items = await listGitRepositories(namespace);
                    for (const item of items) {
                        const conditions: FluxCondition[] = item.status?.conditions || [];
                        rows.push(['GitRepository', item.metadata?.name || '', item.metadata?.namespace || '',
                            item.spec?.url || '', truncateRevision(item.status?.artifact?.revision),
                            getFluxHealthStatus(conditions, item.spec?.suspend), formatFluxAge(item.metadata?.creationTimestamp)]);
                    }
                } catch { /* skip if CRD missing */ }
            }

            if (shouldInclude('oci')) {
                try {
                    const items = await listOCIRepositories(namespace);
                    for (const item of items) {
                        const conditions: FluxCondition[] = item.status?.conditions || [];
                        rows.push(['OCIRepository', item.metadata?.name || '', item.metadata?.namespace || '',
                            item.spec?.url || '', truncateRevision(item.status?.artifact?.revision),
                            getFluxHealthStatus(conditions, item.spec?.suspend), formatFluxAge(item.metadata?.creationTimestamp)]);
                    }
                } catch { /* skip */ }
            }

            if (shouldInclude('helm')) {
                try {
                    const items = await listHelmRepositories(namespace);
                    for (const item of items) {
                        const conditions: FluxCondition[] = item.status?.conditions || [];
                        rows.push(['HelmRepository', item.metadata?.name || '', item.metadata?.namespace || '',
                            item.spec?.url || '', truncateRevision(item.status?.artifact?.revision),
                            getFluxHealthStatus(conditions, item.spec?.suspend), formatFluxAge(item.metadata?.creationTimestamp)]);
                    }
                } catch { /* skip */ }
            }

            if (shouldInclude('bucket')) {
                try {
                    const items = await listBuckets(namespace);
                    for (const item of items) {
                        const conditions: FluxCondition[] = item.status?.conditions || [];
                        rows.push(['Bucket', item.metadata?.name || '', item.metadata?.namespace || '',
                            item.spec?.endpoint || '', truncateRevision(item.status?.artifact?.revision),
                            getFluxHealthStatus(conditions, item.spec?.suspend), formatFluxAge(item.metadata?.creationTimestamp)]);
                    }
                } catch { /* skip */ }
            }

            if (rows.length === 0) {
                return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart('No Flux sources found. FluxCD may not be installed or no sources are configured.')]);
            }

            const displayNs = namespace || 'all';
            const output = `=== Flux Sources (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${rows.length} sources`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return handleFluxError('listing Flux sources', err);
        }
    }
}

// ---- list_flux_image_policies ----

export class ListFluxImagePoliciesTool implements vscode.LanguageModelTool<FluxNamespaceInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<FluxNamespaceInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing Flux image policies in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<FluxNamespaceInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const lines: string[] = [];
            const ns = options.input.namespace;
            const displayNs = ns || 'all';

            // Image Repositories
            try {
                const repos = await listImageRepositories(ns);
                lines.push(`=== Flux Image Repositories (namespace: ${displayNs}) ===`);
                const repoHeaders = ['NAME', 'NAMESPACE', 'IMAGE', 'STATUS', 'LAST SCAN', 'AGE'];
                const repoRows = repos.map((r: any) => {
                    const conditions: FluxCondition[] = r.status?.conditions || [];
                    return [
                        r.metadata?.name || '', r.metadata?.namespace || '',
                        r.spec?.image || '', getFluxHealthStatus(conditions, r.spec?.suspend),
                        r.status?.lastScanResult?.scanTime ? formatFluxAge(r.status.lastScanResult.scanTime) : 'never',
                        formatFluxAge(r.metadata?.creationTimestamp),
                    ];
                });
                lines.push(formatTable(repoHeaders, repoRows));
                lines.push(`\nFound ${repos.length} image repositories`);
            } catch { lines.push('(ImageRepositories not available)'); }

            // Image Policies
            lines.push('');
            try {
                const policies = await listImagePolicies(ns);
                lines.push(`=== Flux Image Policies (namespace: ${displayNs}) ===`);
                const polHeaders = ['NAME', 'NAMESPACE', 'IMAGE-REPO', 'POLICY', 'LATEST', 'AGE'];
                const polRows = policies.map((p: any) => {
                    const imageRef = p.spec?.imageRepositoryRef?.name || '';
                    let policy = '';
                    if (p.spec?.policy?.semver?.range) { policy = `semver: ${p.spec.policy.semver.range}`; }
                    else if (p.spec?.policy?.numerical) { policy = 'numerical'; }
                    else if (p.spec?.policy?.alphabetical) { policy = 'alphabetical'; }
                    const latest = p.status?.latestImage || '';
                    return [
                        p.metadata?.name || '', p.metadata?.namespace || '',
                        imageRef, policy, latest, formatFluxAge(p.metadata?.creationTimestamp),
                    ];
                });
                lines.push(formatTable(polHeaders, polRows));
                lines.push(`\nFound ${policies.length} image policies`);
            } catch { lines.push('(ImagePolicies not available)'); }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return handleFluxError('listing Flux image policies', err);
        }
    }
}

// ---- diagnose_flux_kustomization ----

interface DiagnoseFluxInput { namespace: string; name: string; }

export class DiagnoseFluxKustomizationTool implements vscode.LanguageModelTool<DiagnoseFluxInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<DiagnoseFluxInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Diagnosing Flux Kustomization ${options.input.namespace}/${options.input.name}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<DiagnoseFluxInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, name } = options.input;
            const ks = await getKustomization(namespace, name);
            const conditions: FluxCondition[] = ks.status?.conditions || [];
            const health = getFluxHealthStatus(conditions, ks.spec?.suspend);
            const lines: string[] = [];

            lines.push(`=== Flux Kustomization Diagnosis: ${name} (namespace: ${namespace}) ===`);
            lines.push(`STATUS: ${health}`);
            lines.push(`PATH: ${ks.spec?.path || './'}`);
            lines.push(`REVISION: ${ks.status?.lastAppliedRevision || '<none>'}`);
            lines.push(`SUSPENDED: ${ks.spec?.suspend ? 'true' : 'false'}`);
            lines.push(`AGE: ${formatFluxAge(ks.metadata?.creationTimestamp)}`);
            lines.push('');

            lines.push('FINDINGS:');
            let findings = 0;

            // Condition analysis
            const readyMsg = getConditionMessage(conditions, 'Ready');
            if (health === 'Failed') {
                lines.push(`[CRITICAL] Kustomization is failing: ${readyMsg}`);
                findings++;
            } else if (health === 'Stalled') {
                lines.push(`[WARNING] Kustomization is stalled: ${getConditionMessage(conditions, 'Stalled')}`);
                findings++;
            } else if (health === 'Reconciling') {
                lines.push(`[INFO] Kustomization is reconciling: ${getConditionMessage(conditions, 'Reconciling')}`);
                findings++;
            } else if (health === 'Suspended') {
                lines.push('[INFO] Kustomization is suspended');
                findings++;
            }

            // Check source health
            const sourceRef = ks.spec?.sourceRef;
            if (sourceRef) {
                const srcNs = sourceRef.namespace || namespace;
                try {
                    const source = await getSource(sourceRef.kind, srcNs, sourceRef.name);
                    const srcConditions: FluxCondition[] = source.status?.conditions || [];
                    const srcHealth = getFluxHealthStatus(srcConditions, source.spec?.suspend);
                    if (srcHealth !== 'Ready') {
                        lines.push(`[WARNING] Source ${sourceRef.kind}/${sourceRef.name} is ${srcHealth}: ${getConditionMessage(srcConditions, 'Ready')}`);
                        findings++;
                    }
                } catch {
                    lines.push(`[WARNING] Could not check source ${sourceRef.kind}/${sourceRef.name}`);
                    findings++;
                }
            }

            // dependsOn
            for (const dep of ks.spec?.dependsOn || []) {
                try {
                    const depNs = dep.namespace || namespace;
                    const depKs = await getKustomization(depNs, dep.name);
                    const depConditions: FluxCondition[] = depKs.status?.conditions || [];
                    const depHealth = getFluxHealthStatus(depConditions, depKs.spec?.suspend);
                    if (depHealth !== 'Ready') {
                        lines.push(`[WARNING] Dependency ${dep.name} is ${depHealth}`);
                        findings++;
                    }
                } catch {
                    lines.push(`[WARNING] Could not check dependency ${dep.name}`);
                    findings++;
                }
            }

            if (findings === 0) {
                lines.push('  No issues found — kustomization appears healthy.');
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return handleFluxError('diagnosing Flux Kustomization', err);
        }
    }
}

// ---- diagnose_flux_helm_release ----

export class DiagnoseFluxHelmReleaseTool implements vscode.LanguageModelTool<DiagnoseFluxInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<DiagnoseFluxInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Diagnosing Flux HelmRelease ${options.input.namespace}/${options.input.name}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<DiagnoseFluxInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, name } = options.input;
            const hr = await getHelmRelease(namespace, name);
            const conditions: FluxCondition[] = hr.status?.conditions || [];
            const health = getFluxHealthStatus(conditions, hr.spec?.suspend);
            const lines: string[] = [];

            const chart = hr.spec?.chart?.spec?.chart || hr.spec?.chartRef?.name || '';
            const version = hr.spec?.chart?.spec?.version || '';

            lines.push(`=== Flux HelmRelease Diagnosis: ${name} (namespace: ${namespace}) ===`);
            lines.push(`STATUS: ${health}`);
            lines.push(`CHART: ${chart} (${version || 'latest'})`);
            lines.push(`SUSPENDED: ${hr.spec?.suspend ? 'true' : 'false'}`);
            lines.push(`AGE: ${formatFluxAge(hr.metadata?.creationTimestamp)}`);
            lines.push('');

            lines.push('FINDINGS:');
            let findings = 0;

            const readyMsg = getConditionMessage(conditions, 'Ready');
            if (health === 'Failed') {
                lines.push(`[CRITICAL] HelmRelease is failing: ${readyMsg}`);
                findings++;
            } else if (health === 'Stalled') {
                lines.push(`[WARNING] HelmRelease is stalled: ${getConditionMessage(conditions, 'Stalled')}`);
                findings++;
            } else if (health === 'Suspended') {
                lines.push('[INFO] HelmRelease is suspended');
                findings++;
            }

            // Check release history
            const history = hr.status?.history || [];
            if (history.length > 0) {
                lines.push('');
                lines.push('--- Release History ---');
                for (const snap of history.slice(0, 5)) {
                    lines.push(`  v${snap.chartVersion || '?'} - ${snap.status || ''} - ${snap.testHooks || ''}`);
                }
            }

            // Check source
            const sourceRef = hr.spec?.chart?.spec?.sourceRef;
            if (sourceRef) {
                const srcNs = sourceRef.namespace || namespace;
                try {
                    const source = await getSource(sourceRef.kind, srcNs, sourceRef.name);
                    const srcConditions: FluxCondition[] = source.status?.conditions || [];
                    const srcHealth = getFluxHealthStatus(srcConditions, source.spec?.suspend);
                    if (srcHealth !== 'Ready') {
                        lines.push(`[WARNING] Source ${sourceRef.kind}/${sourceRef.name} is ${srcHealth}`);
                        findings++;
                    }
                } catch {
                    lines.push(`[WARNING] Could not check chart source`);
                    findings++;
                }
            }

            if (findings === 0) {
                lines.push('  No issues found — helm release appears healthy.');
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return handleFluxError('diagnosing Flux HelmRelease', err);
        }
    }
}

// ---- diagnose_flux_system ----

export class DiagnoseFluxSystemTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Diagnosing Flux system health...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const lines: string[] = [];
            lines.push('=== Flux System Health ===');
            lines.push('');

            // 1. flux-system pods
            lines.push('--- Flux Controllers ---');
            try {
                const pods = await listPods('flux-system');
                const unhealthy = pods.filter(p => !isPodHealthy(p));
                if (pods.length === 0) {
                    lines.push('[WARNING] No pods in flux-system namespace — Flux may not be installed');
                } else {
                    lines.push(`  ${pods.length} pods, ${unhealthy.length} unhealthy`);
                    for (const p of unhealthy) {
                        lines.push(`  [CRITICAL] ${p.metadata?.name}: ${podPhaseReason(p)}`);
                    }
                }
            } catch {
                lines.push('  (could not list flux-system pods)');
            }

            // 2. Kustomization health
            lines.push('');
            lines.push('--- Kustomization Health ---');
            let ksReady = 0, ksFailed = 0, ksSuspended = 0, ksTotal = 0;
            try {
                const ksList = await listKustomizations();
                ksTotal = ksList.length;
                for (const ks of ksList) {
                    const h = getFluxHealthStatus(ks.status?.conditions || [], ks.spec?.suspend);
                    if (h === 'Ready') { ksReady++; }
                    else if (h === 'Suspended') { ksSuspended++; }
                    else { ksFailed++; }
                }
                lines.push(`  Total: ${ksTotal}, Ready: ${ksReady}, Failed/Stalled: ${ksFailed}, Suspended: ${ksSuspended}`);
                if (ksFailed > 0) {
                    lines.push(`[WARNING] ${ksFailed} kustomizations are not ready`);
                }
            } catch { lines.push('  (Kustomizations not available)'); }

            // 3. HelmRelease health
            lines.push('');
            lines.push('--- HelmRelease Health ---');
            let hrReady = 0, hrFailed = 0, hrSuspended = 0, hrTotal = 0;
            try {
                const hrList = await listHelmReleases();
                hrTotal = hrList.length;
                for (const hr of hrList) {
                    const h = getFluxHealthStatus(hr.status?.conditions || [], hr.spec?.suspend);
                    if (h === 'Ready') { hrReady++; }
                    else if (h === 'Suspended') { hrSuspended++; }
                    else { hrFailed++; }
                }
                lines.push(`  Total: ${hrTotal}, Ready: ${hrReady}, Failed/Stalled: ${hrFailed}, Suspended: ${hrSuspended}`);
                if (hrFailed > 0) {
                    lines.push(`[WARNING] ${hrFailed} helm releases are not ready`);
                }
            } catch { lines.push('  (HelmReleases not available)'); }

            // Warning events
            lines.push('');
            lines.push('--- Recent Warning Events ---');
            try {
                const events = await listEvents('flux-system');
                const warnings = events.filter(e => e.type === 'Warning');
                if (warnings.length === 0) {
                    lines.push('  No warning events');
                } else {
                    lines.push(`  ${warnings.length} warning events:`);
                    for (const e of warnings.slice(0, 10)) {
                        const obj = `${(e.involvedObject?.kind || '').toLowerCase()}/${e.involvedObject?.name || ''}`;
                        lines.push(`  - ${obj}: ${e.reason}: ${e.message}`);
                    }
                }
            } catch { lines.push('  (could not check events)'); }

            // Mermaid
            lines.push('');
            lines.push('FLUX TOPOLOGY:');
            lines.push('```mermaid');
            lines.push('graph TD');
            lines.push(`    FLUX[Flux System]`);
            lines.push(`    FLUX --> KS[Kustomizations: ${ksReady}/${ksTotal} ready]`);
            lines.push(`    FLUX --> HR[HelmReleases: ${hrReady}/${hrTotal} ready]`);
            lines.push('```');

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return handleFluxError('diagnosing Flux system', err);
        }
    }
}

// ---- get_flux_resource_tree ----

interface GetFluxResourceTreeInput { namespace: string; name: string; resourceKind?: string; }

export class GetFluxResourceTreeTool implements vscode.LanguageModelTool<GetFluxResourceTreeInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<GetFluxResourceTreeInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Building Flux resource tree for ${options.input.namespace}/${options.input.name}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<GetFluxResourceTreeInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, name, resourceKind } = options.input;
            const kind = resourceKind || 'Kustomization';
            const lines: string[] = [];
            const mermaidLines: string[] = ['graph TD'];

            lines.push(`=== Flux Resource Tree: ${kind}/${name} (namespace: ${namespace}) ===`);
            lines.push('');

            if (kind === 'Kustomization') {
                const ks = await getKustomization(namespace, name);
                const conditions: FluxCondition[] = ks.status?.conditions || [];
                const health = getFluxHealthStatus(conditions, ks.spec?.suspend);
                const rootId = `KS_${name.replace(/[^a-zA-Z0-9]/g, '_')}`;
                mermaidLines.push(`    ${rootId}[Kustomization: ${name} - ${health}]`);

                // Source
                const sourceRef = ks.spec?.sourceRef;
                if (sourceRef) {
                    const srcId = `SRC_${sourceRef.name.replace(/[^a-zA-Z0-9]/g, '_')}`;
                    lines.push(`Source: ${sourceRef.kind}/${sourceRef.name}`);
                    mermaidLines.push(`    ${rootId} --> ${srcId}[${sourceRef.kind}: ${sourceRef.name}]`);
                }

                // Dependencies
                for (const dep of ks.spec?.dependsOn || []) {
                    const depId = `DEP_${dep.name.replace(/[^a-zA-Z0-9]/g, '_')}`;
                    lines.push(`Depends on: ${dep.name}`);
                    mermaidLines.push(`    ${depId}[Kustomization: ${dep.name}] --> ${rootId}`);
                }

                // Managed resources from inventory
                const entries = ks.status?.inventory?.entries || [];
                if (entries.length > 0) {
                    lines.push(`\nManaged Resources (${entries.length}):`);
                    for (const entry of entries.slice(0, 20)) {
                        const id = entry.id || '';
                        lines.push(`  - ${id}`);
                    }
                    if (entries.length > 20) {
                        lines.push(`  ... and ${entries.length - 20} more`);
                    }
                }
            } else {
                // HelmRelease
                const hr = await getHelmRelease(namespace, name);
                const conditions: FluxCondition[] = hr.status?.conditions || [];
                const health = getFluxHealthStatus(conditions, hr.spec?.suspend);
                const rootId = `HR_${name.replace(/[^a-zA-Z0-9]/g, '_')}`;
                mermaidLines.push(`    ${rootId}[HelmRelease: ${name} - ${health}]`);

                const sourceRef = hr.spec?.chart?.spec?.sourceRef;
                if (sourceRef) {
                    const srcId = `SRC_${sourceRef.name.replace(/[^a-zA-Z0-9]/g, '_')}`;
                    lines.push(`Source: ${sourceRef.kind}/${sourceRef.name}`);
                    mermaidLines.push(`    ${rootId} --> ${srcId}[${sourceRef.kind}: ${sourceRef.name}]`);
                }

                const chart = hr.spec?.chart?.spec?.chart || '';
                if (chart) { lines.push(`Chart: ${chart}`); }

                for (const dep of hr.spec?.dependsOn || []) {
                    const depId = `DEP_${dep.name.replace(/[^a-zA-Z0-9]/g, '_')}`;
                    lines.push(`Depends on: ${dep.name}`);
                    mermaidLines.push(`    ${depId}[HelmRelease: ${dep.name}] --> ${rootId}`);
                }
            }

            lines.push('');
            lines.push('DEPENDENCY GRAPH:');
            lines.push('```mermaid');
            lines.push(mermaidLines.join('\n'));
            lines.push('```');

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return handleFluxError('building Flux resource tree', err);
        }
    }
}
