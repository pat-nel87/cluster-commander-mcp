import { getCustomObjectsApi } from './client';

export interface NodeMetrics {
    name: string;
    usage: { cpu: string; memory: string };
}

export interface PodMetrics {
    name: string;
    namespace: string;
    containers: { name: string; usage: { cpu: string; memory: string } }[];
}

export async function getNodeMetrics(): Promise<NodeMetrics[]> {
    const api = getCustomObjectsApi();
    const response = await api.listClusterCustomObject({
        group: 'metrics.k8s.io',
        version: 'v1beta1',
        plural: 'nodes',
    }) as any;
    return (response.items || []).map((item: any) => ({
        name: item.metadata?.name || '',
        usage: {
            cpu: item.usage?.cpu || '0',
            memory: item.usage?.memory || '0',
        },
    }));
}

export async function getPodMetrics(namespace?: string): Promise<PodMetrics[]> {
    const api = getCustomObjectsApi();
    let response: any;
    if (!namespace || namespace === 'all') {
        response = await api.listClusterCustomObject({
            group: 'metrics.k8s.io',
            version: 'v1beta1',
            plural: 'pods',
        });
    } else {
        response = await api.listNamespacedCustomObject({
            group: 'metrics.k8s.io',
            version: 'v1beta1',
            namespace,
            plural: 'pods',
        });
    }
    return (response.items || []).map((item: any) => ({
        name: item.metadata?.name || '',
        namespace: item.metadata?.namespace || '',
        containers: (item.containers || []).map((c: any) => ({
            name: c.name || '',
            usage: {
                cpu: c.usage?.cpu || '0',
                memory: c.usage?.memory || '0',
            },
        })),
    }));
}

/** Parse k8s resource quantity to millicores (e.g. "250m" -> 250, "1" -> 1000) */
export function parseCPU(value: string): number {
    if (value.endsWith('n')) { return Math.round(parseInt(value) / 1_000_000); }
    if (value.endsWith('u')) { return Math.round(parseInt(value) / 1_000); }
    if (value.endsWith('m')) { return parseInt(value); }
    return Math.round(parseFloat(value) * 1000);
}

/** Parse k8s resource quantity to bytes */
export function parseMemory(value: string): number {
    const units: Record<string, number> = {
        'Ki': 1024, 'Mi': 1024 ** 2, 'Gi': 1024 ** 3, 'Ti': 1024 ** 4,
        'K': 1000, 'M': 1000 ** 2, 'G': 1000 ** 3, 'T': 1000 ** 4,
        'k': 1000, 'm': 1000 ** 2, 'g': 1000 ** 3, 't': 1000 ** 4,
    };
    for (const [suffix, multiplier] of Object.entries(units)) {
        if (value.endsWith(suffix)) {
            return Math.round(parseFloat(value.slice(0, -suffix.length)) * multiplier);
        }
    }
    return parseInt(value) || 0;
}

/** Format bytes as human-readable */
export function formatBytes(bytes: number): string {
    const gi = 1024 ** 3;
    const mi = 1024 ** 2;
    const ki = 1024;
    if (bytes >= gi) { return `${(bytes / gi).toFixed(1)}Gi`; }
    if (bytes >= mi) { return `${(bytes / mi).toFixed(1)}Mi`; }
    if (bytes >= ki) { return `${(bytes / ki).toFixed(1)}Ki`; }
    return `${bytes}B`;
}
