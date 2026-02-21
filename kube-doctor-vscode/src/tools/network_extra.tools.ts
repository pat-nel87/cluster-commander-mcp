import * as vscode from 'vscode';
import { listIngresses, getEndpoints } from '../k8s/networking';
import { formatAge, formatTable, formatError } from '../util/formatting';

// ---- list_ingresses ----

interface ListIngressesInput { namespace: string; }

export class ListIngressesTool implements vscode.LanguageModelTool<ListIngressesInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListIngressesInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing ingresses in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListIngressesInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const ingresses = await listIngresses(options.input.namespace);
            const headers = ['NAME', 'NAMESPACE', 'HOSTS', 'PATHS', 'TLS', 'AGE'];
            const rows = ingresses.map(ing => {
                const hosts: string[] = [];
                const paths: string[] = [];
                for (const rule of ing.spec?.rules || []) {
                    if (rule.host) { hosts.push(rule.host); }
                    for (const path of rule.http?.paths || []) {
                        if (path.path) { paths.push(path.path); }
                    }
                }
                return [
                    ing.metadata?.name || '', ing.metadata?.namespace || '',
                    hosts.join(',') || '*', paths.join(',') || '/',
                    (ing.spec?.tls?.length || 0) > 0 ? 'Yes' : 'No',
                    formatAge(ing.metadata?.creationTimestamp),
                ];
            });
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const output = `=== Ingresses (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${ingresses.length} ingresses`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing ingresses', err))]);
        }
    }
}

// ---- get_endpoints ----

interface GetEndpointsInput { namespace: string; name: string; }

export class GetEndpointsTool implements vscode.LanguageModelTool<GetEndpointsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<GetEndpointsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Getting endpoints for ${options.input.namespace}/${options.input.name}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<GetEndpointsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, name } = options.input;
            const endpoints = await getEndpoints(namespace, name);
            const lines: string[] = [];
            lines.push(`=== Endpoints: ${endpoints.metadata?.name} (namespace: ${endpoints.metadata?.namespace}) ===`);

            let totalAddresses = 0;
            for (const subset of endpoints.subsets || []) {
                if (subset.addresses && subset.addresses.length > 0) {
                    lines.push('\nReady Addresses:');
                    for (const addr of subset.addresses) {
                        const ref = addr.targetRef ? ` (${addr.targetRef.kind}/${addr.targetRef.name})` : '';
                        for (const port of subset.ports || []) {
                            lines.push(`  ${addr.ip}:${port.port}${ref}`);
                        }
                        totalAddresses++;
                    }
                }
                if (subset.notReadyAddresses && subset.notReadyAddresses.length > 0) {
                    lines.push('\nNot Ready Addresses:');
                    for (const addr of subset.notReadyAddresses) {
                        const ref = addr.targetRef ? ` (${addr.targetRef.kind}/${addr.targetRef.name})` : '';
                        for (const port of subset.ports || []) {
                            lines.push(`  ${addr.ip}:${port.port}${ref}`);
                        }
                    }
                }
            }

            if (totalAddresses === 0 && (!endpoints.subsets || endpoints.subsets.length === 0)) {
                lines.push('\n(no endpoints - check service selector matches pod labels)');
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting endpoints', err))]);
        }
    }
}
