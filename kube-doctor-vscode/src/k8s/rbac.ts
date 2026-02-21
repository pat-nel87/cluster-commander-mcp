import * as k8s from '@kubernetes/client-node';
import { getRbacApi } from './client';

export async function listRoleBindings(namespace: string): Promise<k8s.V1RoleBinding[]> {
    const api = getRbacApi();
    const response = await api.listNamespacedRoleBinding({ namespace });
    return response.items;
}

export async function listClusterRoleBindings(): Promise<k8s.V1ClusterRoleBinding[]> {
    const api = getRbacApi();
    const response = await api.listClusterRoleBinding();
    return response.items;
}
