import * as k8s from '@kubernetes/client-node';
import { getCoreApi } from './client';

export async function listResourceQuotas(namespace: string): Promise<k8s.V1ResourceQuota[]> {
    const api = getCoreApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listResourceQuotaForAllNamespaces();
        return response.items;
    }
    const response = await api.listNamespacedResourceQuota({ namespace });
    return response.items;
}

export async function listLimitRanges(namespace: string): Promise<k8s.V1LimitRange[]> {
    const api = getCoreApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listLimitRangeForAllNamespaces();
        return response.items;
    }
    const response = await api.listNamespacedLimitRange({ namespace });
    return response.items;
}
