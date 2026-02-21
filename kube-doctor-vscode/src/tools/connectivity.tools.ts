import * as vscode from 'vscode';
import { getPod } from '../k8s/pods';
import { listNetworkPolicies } from '../k8s/network_policies';
import { formatLabels, formatError } from '../util/formatting';

interface AnalyzePodConnectivityInput {
    namespace: string;
    podName: string;
}

export class AnalyzePodConnectivityTool implements vscode.LanguageModelTool<AnalyzePodConnectivityInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<AnalyzePodConnectivityInput>
    ): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Analyzing connectivity for pod ${options.input.namespace}/${options.input.podName}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<AnalyzePodConnectivityInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, podName } = options.input;
            const pod = await getPod(namespace, podName);
            const policies = await listNetworkPolicies(namespace);
            const podLabels = pod.metadata?.labels || {};

            const lines: string[] = [];
            lines.push(`=== Pod Connectivity Analysis: ${podName} (namespace: ${namespace}) ===`);
            lines.push('');
            lines.push(`Pod Labels: ${formatLabels(podLabels)}`);
            lines.push('');

            // Find policies that select this pod
            const matching = policies.filter(np => {
                const sel = np.spec?.podSelector?.matchLabels || {};
                // Empty selector matches all pods
                if (Object.keys(sel).length === 0 && (!np.spec?.podSelector?.matchExpressions || np.spec.podSelector.matchExpressions.length === 0)) {
                    return true;
                }
                return Object.entries(sel).every(([k, v]) => podLabels[k] === v);
            });

            if (matching.length === 0) {
                lines.push('[INFO] No network policies select this pod — all traffic is allowed by default');
                lines.push('');
                lines.push('CONNECTIVITY DIAGRAM:');
                lines.push('```mermaid');
                lines.push('graph LR');
                lines.push(`    ANY[Any Source] -->|allowed| POD[Pod: ${podName}]`);
                lines.push(`    POD -->|allowed| ANY2[Any Destination]`);
                lines.push('```');
            } else {
                lines.push(`Matching Policies: ${matching.length}`);
                lines.push('');

                let hasIngress = false;
                let hasEgress = false;
                const ingressSources: string[] = [];
                const egressDests: string[] = [];

                for (const np of matching) {
                    const policyTypes = np.spec?.policyTypes || ['Ingress'];
                    if (policyTypes.includes('Ingress')) { hasIngress = true; }
                    if (policyTypes.includes('Egress')) { hasEgress = true; }

                    for (const rule of np.spec?.ingress || []) {
                        if (!rule._from || rule._from.length === 0) {
                            ingressSources.push('All Sources');
                        } else {
                            for (const f of rule._from) {
                                if (f.podSelector?.matchLabels) {
                                    ingressSources.push(`Pods: ${Object.entries(f.podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (f.namespaceSelector?.matchLabels) {
                                    ingressSources.push(`NS: ${Object.entries(f.namespaceSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (f.ipBlock) {
                                    ingressSources.push(`CIDR: ${f.ipBlock.cidr}`);
                                }
                            }
                        }
                    }

                    for (const rule of np.spec?.egress || []) {
                        if (!rule.to || rule.to.length === 0) {
                            egressDests.push('All Destinations');
                        } else {
                            for (const t of rule.to) {
                                if (t.podSelector?.matchLabels) {
                                    egressDests.push(`Pods: ${Object.entries(t.podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (t.namespaceSelector?.matchLabels) {
                                    egressDests.push(`NS: ${Object.entries(t.namespaceSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(',')}`);
                                }
                                if (t.ipBlock) {
                                    egressDests.push(`CIDR: ${t.ipBlock.cidr}`);
                                }
                            }
                        }
                    }
                }

                // Findings
                lines.push('FINDINGS:');
                for (const np of matching) {
                    lines.push(`  Policy '${np.metadata?.name}':`);
                    const policyTypes = np.spec?.policyTypes || ['Ingress'];
                    if (policyTypes.includes('Ingress') && (!np.spec?.ingress || np.spec.ingress.length === 0)) {
                        lines.push('    [WARNING] Ingress policy with no rules — all ingress DENIED');
                    }
                    if (policyTypes.includes('Egress') && (!np.spec?.egress || np.spec.egress.length === 0)) {
                        lines.push('    [WARNING] Egress policy with no rules — all egress DENIED');
                    }
                }

                // Mermaid diagram
                lines.push('');
                lines.push('CONNECTIVITY DIAGRAM:');
                lines.push('```mermaid');
                lines.push('graph LR');

                const uniqueSources = [...new Set(ingressSources)];
                const uniqueDests = [...new Set(egressDests)];

                if (hasIngress) {
                    if (uniqueSources.length > 0) {
                        uniqueSources.forEach((src, i) => {
                            lines.push(`    SRC${i}[${src}] -->|allowed| POD[Pod: ${podName}]`);
                        });
                    } else {
                        lines.push(`    BLOCKED1[All Sources] -.->|denied| POD[Pod: ${podName}]`);
                    }
                } else {
                    lines.push(`    ANY_IN[Any Source] -->|allowed| POD[Pod: ${podName}]`);
                }

                if (hasEgress) {
                    if (uniqueDests.length > 0) {
                        uniqueDests.forEach((dst, i) => {
                            lines.push(`    POD -->|allowed| DST${i}[${dst}]`);
                        });
                    } else {
                        lines.push(`    POD -.->|denied| BLOCKED2[All Destinations]`);
                    }
                } else {
                    lines.push(`    POD -->|allowed| ANY_OUT[Any Destination]`);
                }

                lines.push('```');
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('analyzing pod connectivity', err))]);
        }
    }
}
