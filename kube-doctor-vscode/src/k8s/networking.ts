import * as k8s from '@kubernetes/client-node';
import { getCoreApi, getNetworkingApi } from './client';

export async function listServices(namespace: string, labelSelector?: string): Promise<k8s.V1Service[]> {
    const api = getCoreApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listServiceForAllNamespaces({ labelSelector });
        return response.items;
    }
    const response = await api.listNamespacedService({ namespace, labelSelector });
    return response.items;
}

export async function listIngresses(namespace: string): Promise<k8s.V1Ingress[]> {
    const api = getNetworkingApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listIngressForAllNamespaces();
        return response.items;
    }
    const response = await api.listNamespacedIngress({ namespace });
    return response.items;
}

export async function getEndpoints(namespace: string, name: string): Promise<k8s.V1Endpoints> {
    const api = getCoreApi();
    return api.readNamespacedEndpoints({ namespace, name });
}
