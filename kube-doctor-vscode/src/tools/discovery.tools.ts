import * as vscode from 'vscode';
import { listCRDs, getAPIResources, listMutatingWebhookConfigurations, listValidatingWebhookConfigurations } from '../k8s/discovery';
import { formatAge, formatTable, formatError } from '../util/formatting';

// ---- list_crds ----

interface ListCRDsInput { groupFilter?: string; }

export class ListCRDsTool implements vscode.LanguageModelTool<ListCRDsInput> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Listing Custom Resource Definitions...' };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListCRDsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const crds = await listCRDs(options.input.groupFilter);
            const headers = ['NAME', 'GROUP', 'VERSION', 'SCOPE', 'AGE'];
            const rows = crds.map(crd => {
                const versions = (crd.spec?.versions || [])
                    .filter(v => v.served)
                    .map(v => v.storage ? `${v.name}*` : v.name)
                    .join(',');
                return [
                    crd.metadata?.name || '', crd.spec?.group || '', versions,
                    crd.spec?.scope || '', formatAge(crd.metadata?.creationTimestamp),
                ];
            });
            let output = `=== Custom Resource Definitions ===\n`;
            if (options.input.groupFilter) {
                output += `Filter: group contains '${options.input.groupFilter}'\n`;
            }
            output += `${formatTable(headers, rows)}\n\nFound ${crds.length} CRDs`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing CRDs', err))]);
        }
    }
}

// ---- get_api_resources ----

interface GetAPIResourcesInput { groupFilter?: string; }

export class GetAPIResourcesTool implements vscode.LanguageModelTool<GetAPIResourcesInput> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Getting API resources...' };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<GetAPIResourcesInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const resourceLists = await getAPIResources(options.input.groupFilter);
            const headers = ['NAME', 'API-GROUP', 'NAMESPACED', 'VERBS'];
            const rows: string[][] = [];
            for (const rl of resourceLists) {
                for (const r of rl.resources) {
                    rows.push([
                        r.name || '', rl.groupVersion,
                        r.namespaced ? 'true' : 'false',
                        (r.verbs || []).join(','),
                    ]);
                }
            }
            let output = `=== API Resources ===\n`;
            if (options.input.groupFilter) {
                output += `Filter: group contains '${options.input.groupFilter}'\n`;
            }
            output += `${formatTable(headers, rows)}\n\nFound ${rows.length} API resources`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('getting API resources', err))]);
        }
    }
}

// ---- list_webhook_configs ----

export class ListWebhookConfigsTool implements vscode.LanguageModelTool<Record<string, never>> {
    async prepareInvocation(): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: 'Listing webhook configurations...' };
    }

    async invoke(): Promise<vscode.LanguageModelToolResult> {
        try {
            const lines: string[] = [];
            lines.push('=== Webhook Configurations ===');
            lines.push('');

            // Mutating
            lines.push('--- Mutating Webhooks ---');
            try {
                const mutating = await listMutatingWebhookConfigurations();
                if (mutating.length === 0) {
                    lines.push('  (none)');
                } else {
                    for (const mwc of mutating) {
                        lines.push(`\n  ${mwc.metadata?.name}:`);
                        for (const wh of mwc.webhooks || []) {
                            const failPolicy = wh.failurePolicy || 'Ignore';
                            const timeout = wh.timeoutSeconds || 10;
                            const svc = wh.clientConfig?.service;
                            const endpoint = svc ? `${svc.namespace}/${svc.name}:${svc.port || 443}${svc.path || '/'}` : (wh.clientConfig?.url || '<unknown>');
                            lines.push(`    Webhook: ${wh.name}`);
                            lines.push(`      Endpoint: ${endpoint}`);
                            lines.push(`      Failure Policy: ${failPolicy}`);
                            lines.push(`      Timeout: ${timeout}s`);
                            for (const rule of wh.rules || []) {
                                const ops = (rule.operations || []).join(',');
                                const groups = (rule.apiGroups || []).join(',');
                                const resources = (rule.resources || []).join(',');
                                lines.push(`      Rule: ${ops} ${groups} ${resources}`);
                            }
                            if (failPolicy === 'Fail') {
                                lines.push('      [WARNING] failurePolicy=Fail — webhook outage will block matching API requests');
                            }
                        }
                    }
                }
            } catch (e) { lines.push(`  (could not list: ${e})`); }

            // Validating
            lines.push('');
            lines.push('--- Validating Webhooks ---');
            try {
                const validating = await listValidatingWebhookConfigurations();
                if (validating.length === 0) {
                    lines.push('  (none)');
                } else {
                    for (const vwc of validating) {
                        lines.push(`\n  ${vwc.metadata?.name}:`);
                        for (const wh of vwc.webhooks || []) {
                            const failPolicy = wh.failurePolicy || 'Ignore';
                            const timeout = wh.timeoutSeconds || 10;
                            const svc = wh.clientConfig?.service;
                            const endpoint = svc ? `${svc.namespace}/${svc.name}:${svc.port || 443}${svc.path || '/'}` : (wh.clientConfig?.url || '<unknown>');
                            lines.push(`    Webhook: ${wh.name}`);
                            lines.push(`      Endpoint: ${endpoint}`);
                            lines.push(`      Failure Policy: ${failPolicy}`);
                            lines.push(`      Timeout: ${timeout}s`);
                            for (const rule of wh.rules || []) {
                                const ops = (rule.operations || []).join(',');
                                const groups = (rule.apiGroups || []).join(',');
                                const resources = (rule.resources || []).join(',');
                                lines.push(`      Rule: ${ops} ${groups} ${resources}`);
                            }
                            if (failPolicy === 'Fail') {
                                lines.push('      [WARNING] failurePolicy=Fail — webhook outage will block matching API requests');
                            }
                        }
                    }
                }
            } catch (e) { lines.push(`  (could not list: ${e})`); }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing webhooks', err))]);
        }
    }
}
