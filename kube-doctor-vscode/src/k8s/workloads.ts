import * as k8s from '@kubernetes/client-node';
import { getAppsApi, getBatchApi } from './client';

export async function listDeployments(namespace: string, labelSelector?: string): Promise<k8s.V1Deployment[]> {
    const api = getAppsApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listDeploymentForAllNamespaces({ labelSelector });
        return response.items;
    }
    const response = await api.listNamespacedDeployment({ namespace, labelSelector });
    return response.items;
}

export async function getDeployment(namespace: string, name: string): Promise<k8s.V1Deployment> {
    const api = getAppsApi();
    return api.readNamespacedDeployment({ namespace, name });
}

export async function listReplicaSets(namespace: string, labelSelector?: string): Promise<k8s.V1ReplicaSet[]> {
    const api = getAppsApi();
    const response = await api.listNamespacedReplicaSet({ namespace, labelSelector });
    return response.items;
}

export async function listStatefulSets(namespace: string, labelSelector?: string): Promise<k8s.V1StatefulSet[]> {
    const api = getAppsApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listStatefulSetForAllNamespaces({ labelSelector });
        return response.items;
    }
    const response = await api.listNamespacedStatefulSet({ namespace, labelSelector });
    return response.items;
}

export async function listDaemonSets(namespace: string, labelSelector?: string): Promise<k8s.V1DaemonSet[]> {
    const api = getAppsApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listDaemonSetForAllNamespaces({ labelSelector });
        return response.items;
    }
    const response = await api.listNamespacedDaemonSet({ namespace, labelSelector });
    return response.items;
}

export async function listJobs(namespace: string, labelSelector?: string): Promise<k8s.V1Job[]> {
    const api = getBatchApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listJobForAllNamespaces({ labelSelector });
        return response.items;
    }
    const response = await api.listNamespacedJob({ namespace, labelSelector });
    return response.items;
}
