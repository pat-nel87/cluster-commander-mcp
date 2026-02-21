import * as k8s from '@kubernetes/client-node';
import { getCoreApi } from './client';

export async function listServices(namespace: string, labelSelector?: string): Promise<k8s.V1Service[]> {
    const api = getCoreApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listServiceForAllNamespaces({ labelSelector });
        return response.items;
    }
    const response = await api.listNamespacedService({ namespace, labelSelector });
    return response.items;
}
