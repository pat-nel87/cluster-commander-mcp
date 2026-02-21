import * as vscode from 'vscode';
import { listPVCs, listPVs } from '../k8s/storage';
import { formatAge, formatTable, formatError } from '../util/formatting';

// ---- list_pvcs ----

interface ListPVCsInput { namespace: string; }

export class ListPVCsTool implements vscode.LanguageModelTool<ListPVCsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListPVCsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing PVCs in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListPVCsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const pvcs = await listPVCs(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'STATUS', 'CAPACITY', 'STORAGE CLASS', 'ACCESS MODES', 'AGE'];
            const rows = pvcs.map(pvc => {
                const capacity = pvc.status?.capacity?.['storage'] || '<pending>';
                const storageClass = pvc.spec?.storageClassName || '<default>';
                const accessModes = (pvc.spec?.accessModes || []).join(',');
                return [
                    pvc.metadata?.name || '', pvc.metadata?.namespace || '',
                    pvc.status?.phase || '', capacity, storageClass, accessModes,
                    formatAge(pvc.metadata?.creationTimestamp),
                ];
            });
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== PersistentVolumeClaims (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${pvcs.length} PVCs`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing PVCs', err))]);
        }
    }
}

// ---- list_pvs ----

export class ListPVsTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Listing PersistentVolumes...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const pvs = await listPVs();
            const headers = ['NAME', 'STATUS', 'CAPACITY', 'RECLAIM POLICY', 'STORAGE CLASS', 'CLAIM', 'AGE'];
            const rows = pvs.map(pv => {
                const capacity = pv.spec?.capacity?.['storage'] || '<unknown>';
                const claim = pv.spec?.claimRef ? `${pv.spec.claimRef.namespace}/${pv.spec.claimRef.name}` : '<unbound>';
                return [
                    pv.metadata?.name || '', pv.status?.phase || '', capacity,
                    pv.spec?.persistentVolumeReclaimPolicy || '', pv.spec?.storageClassName || '',
                    claim, formatAge(pv.metadata?.creationTimestamp),
                ];
            });
            const output = `=== PersistentVolumes ===\n${formatTable(headers, rows)}\n\nFound ${pvs.length} PVs`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing PVs', err))]);
        }
    }
}
