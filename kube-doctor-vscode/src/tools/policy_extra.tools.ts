import * as vscode from 'vscode';
import { listHPAs, listPDBs } from '../k8s/autoscaling';
import { formatAge, formatTable, formatError } from '../util/formatting';

// ---- list_hpas ----

interface ListHPAsInput { namespace: string; }

export class ListHPAsTool implements vscode.LanguageModelTool<ListHPAsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListHPAsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing HPAs in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListHPAsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const hpas = await listHPAs(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'REFERENCE', 'MIN', 'MAX', 'CURRENT', 'AGE'];
            const rows = hpas.map(hpa => {
                const minReplicas = hpa.spec?.minReplicas ?? 1;
                return [
                    hpa.metadata?.name || '', hpa.metadata?.namespace || '',
                    `${hpa.spec?.scaleTargetRef?.kind}/${hpa.spec?.scaleTargetRef?.name}`,
                    `${minReplicas}`, `${hpa.spec?.maxReplicas || 0}`,
                    `${hpa.status?.currentReplicas || 0}`,
                    formatAge(hpa.metadata?.creationTimestamp),
                ];
            });

            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            let output = `=== Horizontal Pod Autoscalers (namespace: ${displayNs}) ===\n`;
            output += `${formatTable(headers, rows)}\n\nFound ${hpas.length} HPAs`;

            // Detail
            if (hpas.length > 0) {
                output += '\n\n--- HPA Details ---';
                for (const hpa of hpas) {
                    output += `\n\n  ${hpa.metadata?.namespace}/${hpa.metadata?.name}:`;
                    for (const metric of hpa.spec?.metrics || []) {
                        if (metric.type === 'Resource' && metric.resource) {
                            let target = 'n/a';
                            if (metric.resource.target?.averageUtilization !== undefined) {
                                target = `${metric.resource.target.averageUtilization}%`;
                            } else if (metric.resource.target?.averageValue) {
                                target = metric.resource.target.averageValue;
                            }
                            output += `\n    Metric: ${metric.resource.name} (target: ${target})`;
                        } else if (metric.type === 'Pods' && metric.pods) {
                            output += `\n    Metric: ${metric.pods.metric?.name} (target avg: ${metric.pods.target?.averageValue || 'n/a'})`;
                        } else {
                            output += `\n    Metric: ${metric.type} type`;
                        }
                    }
                    for (const cond of hpa.status?.conditions || []) {
                        output += `\n    Condition: ${cond.type}=${cond.status} (${cond.reason || ''})`;
                    }
                }
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing HPAs', err))]);
        }
    }
}

// ---- list_pdbs ----

interface ListPDBsInput { namespace: string; }

export class ListPDBsTool implements vscode.LanguageModelTool<ListPDBsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListPDBsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing PDBs in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListPDBsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const pdbs = await listPDBs(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'MIN-AVAILABLE', 'MAX-UNAVAILABLE', 'CURRENT', 'EXPECTED', 'ALLOWED-DISRUPTIONS', 'AGE'];
            const rows = pdbs.map(pdb => {
                const minAvail = pdb.spec?.minAvailable !== undefined ? String(pdb.spec.minAvailable) : 'N/A';
                const maxUnavail = pdb.spec?.maxUnavailable !== undefined ? String(pdb.spec.maxUnavailable) : 'N/A';
                return [
                    pdb.metadata?.name || '', pdb.metadata?.namespace || '',
                    minAvail, maxUnavail,
                    `${pdb.status?.currentHealthy || 0}`, `${pdb.status?.expectedPods || 0}`,
                    `${pdb.status?.disruptionsAllowed || 0}`,
                    formatAge(pdb.metadata?.creationTimestamp),
                ];
            });

            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            let output = `=== Pod Disruption Budgets (namespace: ${displayNs}) ===\n`;
            output += `${formatTable(headers, rows)}\n\nFound ${pdbs.length} PDBs`;

            // Warn on zero disruptions
            for (const pdb of pdbs) {
                if (pdb.status?.disruptionsAllowed === 0 && (pdb.status?.expectedPods || 0) > 0) {
                    output += `\n\n[WARNING] PDB '${pdb.metadata?.namespace}/${pdb.metadata?.name}' has 0 disruptions allowed — voluntary disruptions will be blocked`;
                }
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing PDBs', err))]);
        }
    }
}
