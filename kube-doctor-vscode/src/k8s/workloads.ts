import * as k8s from '@kubernetes/client-node';
import { getAppsApi } from './client';

export async function listDeployments(namespace: string, labelSelector?: string): Promise<k8s.V1Deployment[]> {
    const api = getAppsApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listDeploymentForAllNamespaces({ labelSelector });
        return response.items;
    }
    const response = await api.listNamespacedDeployment({ namespace, labelSelector });
    return response.items;
}
