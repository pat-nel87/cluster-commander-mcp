import * as k8s from '@kubernetes/client-node';
import { getCoreApi } from './client';

export async function listPVCs(namespace: string): Promise<k8s.V1PersistentVolumeClaim[]> {
    const api = getCoreApi();
    if (!namespace || namespace === 'all') {
        const response = await api.listPersistentVolumeClaimForAllNamespaces();
        return response.items;
    }
    const response = await api.listNamespacedPersistentVolumeClaim({ namespace });
    return response.items;
}

export async function listPVs(): Promise<k8s.V1PersistentVolume[]> {
    const api = getCoreApi();
    const response = await api.listPersistentVolume();
    return response.items;
}
