import { getCoreApi } from './client';
import { MAX_LOG_BYTES, truncateString } from '../util/formatting';

export async function getPodLogs(
    namespace: string,
    name: string,
    container?: string,
    tailLines?: number,
    previous?: boolean
): Promise<string> {
    const api = getCoreApi();
    const response = await api.readNamespacedPodLog({
        namespace,
        name,
        container: container || undefined,
        tailLines: tailLines || 100,
        previous: previous || false,
    });
    const logs = typeof response === 'string' ? response : String(response);
    return truncateString(logs, MAX_LOG_BYTES);
}
