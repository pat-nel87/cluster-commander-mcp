import * as k8s from '@kubernetes/client-node';
import { getAutoscalingApi, getPolicyApi } from './client';

export async function listHPAs(namespace: string): Promise<k8s.V2HorizontalPodAutoscaler[]> {
    const api = getAutoscalingApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listHorizontalPodAutoscalerForAllNamespaces();
        return response.items;
    }
    const response = await api.listNamespacedHorizontalPodAutoscaler({ namespace });
    return response.items;
}

export async function listPDBs(namespace: string): Promise<k8s.V1PodDisruptionBudget[]> {
    const api = getPolicyApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listPodDisruptionBudgetForAllNamespaces();
        return response.items;
    }
    const response = await api.listNamespacedPodDisruptionBudget({ namespace });
    return response.items;
}
