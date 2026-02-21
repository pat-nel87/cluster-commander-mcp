import * as k8s from '@kubernetes/client-node';
import { getCoreApi } from './client';

export async function listNodes(): Promise<k8s.V1Node[]> {
    const api = getCoreApi();
    const response = await api.listNode();
    return response.items;
}

export function nodeStatus(node: k8s.V1Node): string {
    for (const cond of node.status?.conditions || []) {
        if (cond.type === 'Ready') {
            return cond.status === 'True' ? 'Ready' : 'NotReady';
        }
    }
    return 'Unknown';
}

export function nodeRoles(node: k8s.V1Node): string {
    const roles: string[] = [];
    for (const key of Object.keys(node.metadata?.labels || {})) {
        if (key.startsWith('node-role.kubernetes.io/')) {
            const role = key.replace('node-role.kubernetes.io/', '');
            if (role) { roles.push(role); }
        }
    }
    return roles.length > 0 ? roles.join(',') : '<none>';
}
