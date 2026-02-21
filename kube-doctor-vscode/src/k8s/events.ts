import * as k8s from '@kubernetes/client-node';
import { getCoreApi } from './client';
import { MAX_EVENTS } from '../util/formatting';

export async function listEvents(
    namespace?: string,
    fieldSelector?: string
): Promise<k8s.CoreV1Event[]> {
    const api = getCoreApi();
    let items: k8s.CoreV1Event[];
    if (!namespace) {
        const response = await api.listEventForAllNamespaces({ fieldSelector });
        items = response.items;
    } else {
        const response = await api.listNamespacedEvent({ namespace, fieldSelector });
        items = response.items;
    }
    // Sort most recent first
    items.sort((a, b) => {
        const ta = a.lastTimestamp ? new Date(a.lastTimestamp).getTime() : (a.metadata?.creationTimestamp ? new Date(a.metadata.creationTimestamp).getTime() : 0);
        const tb = b.lastTimestamp ? new Date(b.lastTimestamp).getTime() : (b.metadata?.creationTimestamp ? new Date(b.metadata.creationTimestamp).getTime() : 0);
        return tb - ta;
    });
    return items.slice(0, MAX_EVENTS);
}

export async function getEventsForObject(namespace: string, objectName: string): Promise<k8s.CoreV1Event[]> {
    return listEvents(namespace, `involvedObject.name=${objectName}`);
}
