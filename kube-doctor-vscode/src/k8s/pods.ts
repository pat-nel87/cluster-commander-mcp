import * as k8s from '@kubernetes/client-node';
import { getCoreApi } from './client';
import { MAX_PODS } from '../util/formatting';

export async function listPods(namespace: string, labelSelector?: string): Promise<k8s.V1Pod[]> {
    const api = getCoreApi();
    let items: k8s.V1Pod[];
    if (!namespace || namespace === 'all') {
        const response = await api.listPodForAllNamespaces({ labelSelector });
        items = response.items;
    } else {
        const response = await api.listNamespacedPod({ namespace, labelSelector });
        items = response.items;
    }
    return items.slice(0, MAX_PODS);
}

export async function getPod(namespace: string, name: string): Promise<k8s.V1Pod> {
    const api = getCoreApi();
    return api.readNamespacedPod({ namespace, name });
}

export function podPhaseReason(pod: k8s.V1Pod): string {
    for (const cs of pod.status?.containerStatuses || []) {
        if (cs.state?.waiting?.reason) { return cs.state.waiting.reason; }
        if (cs.state?.terminated?.reason) { return cs.state.terminated.reason; }
    }
    for (const cs of pod.status?.initContainerStatuses || []) {
        if (cs.state?.waiting?.reason) { return 'Init:' + cs.state.waiting.reason; }
        if (cs.state?.terminated?.reason) { return 'Init:' + cs.state.terminated.reason; }
    }
    return pod.status?.phase || 'Unknown';
}

export function podContainerSummary(pod: k8s.V1Pod): { ready: number; total: number; restarts: number } {
    const total = pod.spec?.containers?.length || 0;
    let ready = 0;
    let restarts = 0;
    for (const cs of pod.status?.containerStatuses || []) {
        if (cs.ready) { ready++; }
        restarts += cs.restartCount || 0;
    }
    return { ready, total, restarts };
}

export function isPodHealthy(pod: k8s.V1Pod): boolean {
    if (pod.status?.phase === 'Running') {
        for (const cs of pod.status?.containerStatuses || []) {
            if (!cs.ready || cs.state?.waiting) { return false; }
        }
        return true;
    }
    return pod.status?.phase === 'Succeeded';
}
