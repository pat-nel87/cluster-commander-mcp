import * as vscode from 'vscode';
import { listResourceQuotas } from '../k8s/quotas';
import { formatTable, formatError } from '../util/formatting';

// ---- check_resource_quotas ----

interface CheckResourceQuotasInput { namespace: string; }

export class CheckResourceQuotasTool implements vscode.LanguageModelTool<CheckResourceQuotasInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<CheckResourceQuotasInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Checking resource quotas in ${options.input.namespace || 'all namespaces'}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<CheckResourceQuotasInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const quotas = await listResourceQuotas(options.input.namespace);
            const displayNs = !options.input.namespace || options.input.namespace === 'all' ? 'all' : options.input.namespace;
            const lines: string[] = [];
            lines.push(`=== Resource Quotas (namespace: ${displayNs}) ===`);

            if (quotas.length === 0) {
                lines.push('No resource quotas defined.');
                return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
            }

            for (const quota of quotas) {
                lines.push(`\n--- ${quota.metadata?.namespace}/${quota.metadata?.name} ---`);
                const headers = ['RESOURCE', 'USED', 'HARD', 'USAGE %'];
                const rows: string[][] = [];
                const hard = quota.status?.hard || {};
                const used = quota.status?.used || {};

                for (const [resource, hardVal] of Object.entries(hard)) {
                    const usedVal = used[resource] || '0';
                    let pct = 'N/A';
                    // Simple numeric comparison for count-based quotas
                    const hardNum = parseFloat(hardVal);
                    const usedNum = parseFloat(usedVal);
                    if (!isNaN(hardNum) && hardNum > 0 && !isNaN(usedNum)) {
                        pct = `${(usedNum / hardNum * 100).toFixed(0)}%`;
                    }
                    rows.push([resource, usedVal, hardVal, pct]);
                }

                lines.push(formatTable(headers, rows));

                // Warnings
                for (const [resource, hardVal] of Object.entries(hard)) {
                    const usedVal = used[resource] || '0';
                    const hardNum = parseFloat(hardVal);
                    const usedNum = parseFloat(usedVal);
                    if (!isNaN(hardNum) && hardNum > 0 && !isNaN(usedNum)) {
                        const pct = usedNum / hardNum * 100;
                        if (pct >= 90) {
                            lines.push(`[CRITICAL] ${resource}: ${pct.toFixed(0)}% used (${usedVal}/${hardVal})`);
                        } else if (pct >= 80) {
                            lines.push(`[WARNING] ${resource}: ${pct.toFixed(0)}% used (${usedVal}/${hardVal})`);
                        }
                    }
                }
            }

            lines.push(`\nFound ${quotas.length} resource quotas`);
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('checking resource quotas', err))]);
        }
    }
}
