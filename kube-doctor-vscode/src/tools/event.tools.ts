import * as vscode from 'vscode';
import { listEvents } from '../k8s/events';
import { formatAge, formatTable, formatError } from '../util/formatting';

interface GetEventsInput {
    namespace?: string;
    eventType?: string;
    involvedObject?: string;
}

export class GetEventsTool implements vscode.LanguageModelTool<GetEventsInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<GetEventsInput>
    ): Promise<vscode.PreparedToolInvocation> {
        const ns = options.input.namespace || 'all namespaces';
        return { invocationMessage: `Getting events in ${ns}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<GetEventsInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, eventType, involvedObject } = options.input;
            const selectors: string[] = [];
            if (involvedObject) { selectors.push(`involvedObject.name=${involvedObject}`); }
            if (eventType) { selectors.push(`type=${eventType}`); }
            const fieldSelector = selectors.length > 0 ? selectors.join(',') : undefined;

            const events = await listEvents(namespace || undefined, fieldSelector);

            const headers = ['TYPE', 'REASON', 'OBJECT', 'MESSAGE', 'COUNT', 'LAST SEEN'];
            const rows = events.map(e => {
                const obj = `${(e.involvedObject?.kind || '').toLowerCase()}/${e.involvedObject?.name || ''}`;
                const lastSeen = e.lastTimestamp ? formatAge(e.lastTimestamp) : formatAge(e.metadata?.creationTimestamp);
                let msg = e.message || '';
                if (msg.length > 80) { msg = msg.slice(0, 77) + '...'; }
                return [e.type || '', e.reason || '', obj, msg, `${e.count || 1}`, lastSeen];
            });

            const displayNs = namespace || 'all';
            const output = `=== Events (namespace: ${displayNs}) ===\n${formatTable(headers, rows)}\n\nFound ${events.length} events`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing events', err))]);
        }
    }
}
