import * as k8s from '@kubernetes/client-node';
import { getNetworkingApi } from './client';

export async function listNetworkPolicies(namespace: string): Promise<k8s.V1NetworkPolicy[]> {
    const api = getNetworkingApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listNetworkPolicyForAllNamespaces();
        return response.items;
    }
    const response = await api.listNamespacedNetworkPolicy({ namespace });
    return response.items;
}
