import * as vscode from 'vscode';
import { extractWorkloadDependencies } from '../k8s/dependencies';
import { formatError } from '../util/formatting';

interface GetWorkloadDependenciesInput {
    namespace: string;
    workloadName: string;
    workloadKind?: string;
}

export class GetWorkloadDependenciesTool implements vscode.LanguageModelTool<GetWorkloadDependenciesInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<GetWorkloadDependenciesInput>
    ): Promise<vscode.PreparedToolInvocation> {
        const kind = options.input.workloadKind || 'Deployment';
        return { invocationMessage: `Mapping dependencies for ${kind}/${options.input.workloadName}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<GetWorkloadDependenciesInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, workloadName, workloadKind } = options.input;
            const kind = workloadKind || 'Deployment';
            const { deps, displayName } = await extractWorkloadDependencies(namespace, workloadName, kind);

            const lines: string[] = [];
            lines.push(`=== Workload Dependencies: ${displayName} (namespace: ${namespace}) ===`);
            lines.push('');

            if (deps.serviceAccount) {
                lines.push(`ServiceAccount:      ${deps.serviceAccount}`);
            }

            lines.push('');
            lines.push(`ConfigMaps (${deps.configMaps.length}):`);
            if (deps.configMaps.length === 0) {
                lines.push('  (none)');
            } else {
                for (const name of deps.configMaps) {
                    lines.push(`  - ${name}`);
                }
            }

            lines.push('');
            lines.push(`Secrets (${deps.secrets.length}):`);
            if (deps.secrets.length === 0) {
                lines.push('  (none)');
            } else {
                for (const name of deps.secrets) {
                    lines.push(`  - ${name}`);
                }
            }

            lines.push('');
            lines.push(`PVCs (${deps.pvcs.length}):`);
            if (deps.pvcs.length === 0) {
                lines.push('  (none)');
            } else {
                for (const name of deps.pvcs) {
                    lines.push(`  - ${name}`);
                }
            }

            lines.push('');
            lines.push(`Matching Services (${deps.matchingServices.length}):`);
            if (deps.matchingServices.length === 0) {
                lines.push('  (none)');
            } else {
                for (const name of deps.matchingServices) {
                    lines.push(`  - ${name}`);
                }
            }

            // Mermaid dependency graph
            lines.push('');
            lines.push('DEPENDENCY GRAPH:');
            lines.push('```mermaid');
            lines.push('graph TD');
            lines.push(`    WL[${displayName}]`);

            if (deps.serviceAccount) {
                lines.push(`    WL --> SA[ServiceAccount: ${deps.serviceAccount}]`);
            }
            deps.configMaps.forEach((name, i) => {
                lines.push(`    WL --> CM${i}[ConfigMap: ${name}]`);
            });
            deps.secrets.forEach((name, i) => {
                lines.push(`    WL --> SEC${i}[Secret: ${name}]`);
            });
            deps.pvcs.forEach((name, i) => {
                lines.push(`    WL --> PVC${i}[PVC: ${name}]`);
            });
            deps.matchingServices.forEach((name, i) => {
                lines.push(`    SVC${i}[Service: ${name}] --> WL`);
            });

            lines.push('```');

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('mapping workload dependencies', err))]);
        }
    }
}
