import * as vscode from 'vscode';
import { listNetworkPolicies } from '../k8s/network_policies';
import { formatAge, formatTable, formatError } from '../util/formatting';

interface ListNetworkPoliciesInput {
    namespace: string;
}

export class ListNetworkPoliciesTool implements vscode.LanguageModelTool<ListNetworkPoliciesInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<ListNetworkPoliciesInput>
    ): Promise<vscode.PreparedToolInvocation> {
        const ns = options.input.namespace || 'all namespaces';
        return { invocationMessage: `Listing network policies in ${ns}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<ListNetworkPoliciesInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace } = options.input;
            const policies = await listNetworkPolicies(namespace);

            const headers = ['NAME', 'NAMESPACE', 'POD-SELECTOR', 'INGRESS', 'EGRESS', 'AGE'];
            const rows = policies.map(np => {
                const selector = np.spec?.podSelector?.matchLabels
                    ? Object.entries(np.spec.podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')
                    : '<all pods>';
                const policyTypes = np.spec?.policyTypes || ['Ingress'];
                return [
                    np.metadata?.name || '',
                    np.metadata?.namespace || '',
                    selector,
                    `${np.spec?.ingress?.length || 0} rules`,
                    `${np.spec?.egress?.length || 0} rules`,
                    formatAge(np.metadata?.creationTimestamp),
                ];
            });

            const displayNs = !namespace || namespace === 'all' ? 'all' : namespace;
            let output = `=== Network Policies (namespace: ${displayNs}) ===\n`;
            output += formatTable(headers, rows);
            output += `\n\nFound ${policies.length} network policies`;

            // Detail section
            if (policies.length > 0) {
                output += '\n\n--- Policy Details ---';
                for (const np of policies) {
                    output += `\n\n  ${np.metadata?.namespace}/${np.metadata?.name}:`;
                    const selector = np.spec?.podSelector?.matchLabels
                        ? Object.entries(np.spec.podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')
                        : '<all pods>';
                    output += `\n    Pod Selector: ${selector}`;
                    for (const [i, rule] of (np.spec?.ingress || []).entries()) {
                        const fromParts: string[] = [];
                        if (!rule._from || rule._from.length === 0) {
                            fromParts.push('all sources');
                        } else {
                            for (const f of rule._from) {
                                if (f.podSelector?.matchLabels) {
                                    fromParts.push(`pods: ${Object.entries(f.podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (f.namespaceSelector?.matchLabels) {
                                    fromParts.push(`namespaces: ${Object.entries(f.namespaceSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (f.ipBlock) {
                                    fromParts.push(`CIDR: ${f.ipBlock.cidr}`);
                                }
                            }
                        }
                        output += `\n    Ingress Rule ${i + 1}: from ${fromParts.join('; ')}`;
                    }
                    for (const [i, rule] of (np.spec?.egress || []).entries()) {
                        const toParts: string[] = [];
                        if (!rule.to || rule.to.length === 0) {
                            toParts.push('all destinations');
                        } else {
                            for (const t of rule.to) {
                                if (t.podSelector?.matchLabels) {
                                    toParts.push(`pods: ${Object.entries(t.podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (t.namespaceSelector?.matchLabels) {
                                    toParts.push(`namespaces: ${Object.entries(t.namespaceSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (t.ipBlock) {
                                    toParts.push(`CIDR: ${t.ipBlock.cidr}`);
                                }
                            }
                        }
                        output += `\n    Egress Rule ${i + 1}: to ${toParts.join('; ')}`;
                    }
                }
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing network policies', err))]);
        }
    }
}
