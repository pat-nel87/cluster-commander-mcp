import * as k8s from '@kubernetes/client-node';
import { getApiExtensionsApi, getAdmissionApi, getKubeConfig } from './client';

export async function listCRDs(groupFilter?: string): Promise<k8s.V1CustomResourceDefinition[]> {
    const api = getApiExtensionsApi();
    const response = await api.listCustomResourceDefinition();
    let items = response.items;
    if (groupFilter) {
        items = items.filter(crd => crd.spec?.group?.includes(groupFilter));
    }
    return items;
}

export async function getAPIResources(groupFilter?: string): Promise<{ groupVersion: string; resources: k8s.V1APIResource[] }[]> {
    const kc = getKubeConfig();
    const client = kc.makeApiClient(k8s.ApisApi);
    const groupList = await client.getAPIVersions();
    const result: { groupVersion: string; resources: k8s.V1APIResource[] }[] = [];

    // Core API
    try {
        const coreClient = kc.makeApiClient(k8s.CoreApi);
        const coreVersions = await coreClient.getAPIVersions();
        for (const v of coreVersions.versions || []) {
            if (groupFilter && !v.includes(groupFilter)) { continue; }
            try {
                const coreApi = kc.makeApiClient(k8s.CoreV1Api);
                const resourceList = await coreApi.getAPIResources();
                const resources = (resourceList.resources || []).filter(r => !r.name.includes('/'));
                if (resources.length > 0) {
                    result.push({ groupVersion: `v1`, resources });
                }
            } catch { /* skip */ }
        }
    } catch { /* skip */ }

    // API groups
    for (const group of groupList.groups || []) {
        const gv = group.preferredVersion?.groupVersion;
        if (!gv) { continue; }
        if (groupFilter && !gv.includes(groupFilter)) { continue; }
        try {
            const cluster = kc.getCurrentCluster();
            if (!cluster?.server) { continue; }
            const url = `${cluster.server}/apis/${gv}`;
            const fetchInit = await kc.applyToFetchOptions({} as any);
            const resp = await fetch(url, fetchInit as any);
            if (resp.ok) {
                const body = await resp.json() as any;
                const resources = (body.resources || []).filter((r: any) => !r.name.includes('/'));
                if (resources.length > 0) {
                    result.push({ groupVersion: gv, resources });
                }
            }
        } catch { /* skip */ }
    }

    return result;
}

export async function listMutatingWebhookConfigurations(): Promise<k8s.V1MutatingWebhookConfiguration[]> {
    const api = getAdmissionApi();
    const response = await api.listMutatingWebhookConfiguration();
    return response.items;
}

export async function listValidatingWebhookConfigurations(): Promise<k8s.V1ValidatingWebhookConfiguration[]> {
    const api = getAdmissionApi();
    const response = await api.listValidatingWebhookConfiguration();
    return response.items;
}
