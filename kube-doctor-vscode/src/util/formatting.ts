const MAX_PODS = 200;
const MAX_EVENTS = 50;
const MAX_LOG_BYTES = 50 * 1024;

export { MAX_PODS, MAX_EVENTS, MAX_LOG_BYTES };

export function formatAge(dateStr: string | Date | undefined): string {
    if (!dateStr) { return '<unknown>'; }
    const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr;
    const ms = Date.now() - date.getTime();
    const secs = Math.floor(ms / 1000);
    if (secs < 60) { return `${secs}s`; }
    const mins = Math.floor(secs / 60);
    if (mins < 60) { return `${mins}m`; }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        const remMins = mins % 60;
        return remMins > 0 ? `${hours}h${remMins}m` : `${hours}h`;
    }
    const days = Math.floor(hours / 24);
    if (days > 365) { return `${Math.floor(days / 365)}y${days % 365}d`; }
    return `${days}d`;
}

export function formatTable(headers: string[], rows: string[][]): string {
    if (rows.length === 0) { return '(none)'; }
    const widths = headers.map((h, i) => {
        let max = h.length;
        for (const row of rows) {
            if (row[i] && row[i].length > max) { max = row[i].length; }
        }
        return max;
    });
    const headerLine = headers.map((h, i) => h.padEnd(widths[i])).join('  ');
    const dataLines = rows.map(row =>
        row.map((cell, i) => (cell || '').padEnd(widths[i] || 0)).join('  ')
    );
    return [headerLine, ...dataLines].join('\n');
}

export function formatLabels(labels: Record<string, string> | undefined): string {
    if (!labels || Object.keys(labels).length === 0) { return '<none>'; }
    return Object.entries(labels).map(([k, v]) => `${k}=${v}`).join(', ');
}

export function truncateString(s: string, maxLen: number): string {
    if (s.length <= maxLen) { return s; }
    return s.slice(0, maxLen) + '\n... [output truncated]';
}

export function formatError(action: string, err: unknown): string {
    const e = err as any;
    const message = e?.body?.message || e?.response?.body?.message || e?.message || String(err);
    if (message.includes('forbidden') || message.includes('Forbidden')) {
        return `Permission denied: ${action}. Check RBAC permissions.`;
    }
    if (message.includes('not found') || message.includes('NotFound')) {
        return `Not found: ${action}`;
    }
    return `Error ${action}: ${message}`;
}
