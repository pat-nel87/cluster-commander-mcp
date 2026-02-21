import * as vscode from 'vscode';
import { listServices } from '../k8s/networking';
import { formatAge, formatTable, formatError } from '../util/formatting';

interface ListServicesInput {
    namespace: string;
}

export class ListServicesTool implements vscode.LanguageModelTool<ListServicesInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<ListServicesInput>
    ): Promise<vscode.PreparedToolInvocation> {
        const ns = options.input.namespace || 'all namespaces';
        return { invocationMessage: `Listing services in ${ns}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<ListServicesInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const services = await listServices(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'TYPE', 'CLUSTER-IP', 'PORTS', 'AGE'];
            const rows = services.map(svc => {
                const ports = (svc.spec?.ports || []).map(p => {
                    let s = `${p.port}/${p.protocol || 'TCP'}`;
                    if (p.nodePort) { s = `${p.port}:${p.nodePort}/${p.protocol || 'TCP'}`; }
                    return s;
                }).join(',');

                return [
                    svc.metadata?.name || '',
                    svc.metadata?.namespace || '',
                    svc.spec?.type || '',
                    svc.spec?.clusterIP || '',
                    ports,
                    formatAge(svc.metadata?.creationTimestamp),
                ];
            });

            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== Services (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${services.length} services`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing services', err))]);
        }
    }
}
