import { getCustomObjectsApi, getCoreApi } from './client';

// Flux API constants
const FLUX_KUSTOMIZE_GROUP = 'kustomize.toolkit.fluxcd.io';
const FLUX_KUSTOMIZE_VERSION = 'v1';
const FLUX_HELM_GROUP = 'helm.toolkit.fluxcd.io';
const FLUX_HELM_VERSION = 'v2';
const FLUX_SOURCE_GROUP = 'source.toolkit.fluxcd.io';
const FLUX_SOURCE_VERSION = 'v1';
const FLUX_SOURCE_VERSION_BETA2 = 'v1beta2';
const FLUX_IMAGE_GROUP = 'image.toolkit.fluxcd.io';
const FLUX_IMAGE_VERSION = 'v1beta2';

export interface FluxCondition {
    type: string;
    status: string;
    reason?: string;
    message?: string;
    lastTransitionTime?: string;
}

export type FluxHealthStatus = 'Ready' | 'Reconciling' | 'Stalled' | 'Failed' | 'Suspended' | 'Unknown';

export function getFluxHealthStatus(conditions: FluxCondition[], suspended?: boolean): FluxHealthStatus {
    if (suspended) { return 'Suspended'; }
    const stalled = conditions.find(c => c.type === 'Stalled' && c.status === 'True');
    if (stalled) { return 'Stalled'; }
    const reconciling = conditions.find(c => c.type === 'Reconciling' && c.status === 'True');
    if (reconciling) { return 'Reconciling'; }
    const ready = conditions.find(c => c.type === 'Ready');
    if (ready) {
        return ready.status === 'True' ? 'Ready' : 'Failed';
    }
    return 'Unknown';
}

export function getConditionMessage(conditions: FluxCondition[], condType: string): string {
    const c = conditions.find(cond => cond.type === condType);
    return c?.message || '';
}

function isFluxNotInstalled(err: any): boolean {
    const msg = err?.body?.message || err?.response?.body?.message || err?.message || String(err);
    return msg.includes('the server could not find the requested resource') ||
           msg.includes('not found') ||
           msg.includes('no matches for kind');
}

async function listFluxResources(group: string, version: string, plural: string, namespace?: string): Promise<any[]> {
    const api = getCustomObjectsApi();
    let response: any;
    if (!namespace || namespace === 'all') {
        response = await api.listClusterCustomObject({ group, version, plural });
    } else {
        response = await api.listNamespacedCustomObject({ group, version, namespace, plural });
    }
    return response.items || [];
}

async function getFluxResource(group: string, version: string, plural: string, namespace: string, name: string): Promise<any> {
    const api = getCustomObjectsApi();
    return api.getNamespacedCustomObject({ group, version, namespace, plural, name });
}

// --- Kustomizations ---

export async function listKustomizations(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_KUSTOMIZE_GROUP, FLUX_KUSTOMIZE_VERSION, 'kustomizations', namespace);
}

export async function getKustomization(namespace: string, name: string): Promise<any> {
    return getFluxResource(FLUX_KUSTOMIZE_GROUP, FLUX_KUSTOMIZE_VERSION, 'kustomizations', namespace, name);
}

// --- HelmReleases ---

export async function listHelmReleases(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_HELM_GROUP, FLUX_HELM_VERSION, 'helmreleases', namespace);
}

export async function getHelmRelease(namespace: string, name: string): Promise<any> {
    return getFluxResource(FLUX_HELM_GROUP, FLUX_HELM_VERSION, 'helmreleases', namespace, name);
}

// --- Sources ---

export async function listGitRepositories(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION, 'gitrepositories', namespace);
}

export async function listOCIRepositories(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION_BETA2, 'ocirepositories', namespace);
}

export async function listHelmRepositories(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION, 'helmrepositories', namespace);
}

export async function listHelmCharts(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION, 'helmcharts', namespace);
}

export async function listBuckets(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION_BETA2, 'buckets', namespace);
}

// --- Image Policies ---

export async function listImageRepositories(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_IMAGE_GROUP, FLUX_IMAGE_VERSION, 'imagerepositories', namespace);
}

export async function listImagePolicies(namespace?: string): Promise<any[]> {
    return listFluxResources(FLUX_IMAGE_GROUP, FLUX_IMAGE_VERSION, 'imagepolicies', namespace);
}

// --- Source lookup helpers ---

export async function getSource(kind: string, namespace: string, name: string): Promise<any> {
    switch (kind) {
        case 'GitRepository':
            return getFluxResource(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION, 'gitrepositories', namespace, name);
        case 'OCIRepository':
            return getFluxResource(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION_BETA2, 'ocirepositories', namespace, name);
        case 'HelmRepository':
            return getFluxResource(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION, 'helmrepositories', namespace, name);
        case 'Bucket':
            return getFluxResource(FLUX_SOURCE_GROUP, FLUX_SOURCE_VERSION_BETA2, 'buckets', namespace, name);
        default:
            throw new Error(`Unknown source kind: ${kind}`);
    }
}

// --- Utilities ---

export function formatFluxAge(timestamp: string | undefined): string {
    if (!timestamp) { return '<unknown>'; }
    const date = new Date(timestamp);
    const ms = Date.now() - date.getTime();
    const secs = Math.floor(ms / 1000);
    if (secs < 60) { return `${secs}s`; }
    const mins = Math.floor(secs / 60);
    if (mins < 60) { return `${mins}m`; }
    const hours = Math.floor(mins / 60);
    if (hours < 24) { return `${hours}h`; }
    const days = Math.floor(hours / 24);
    return `${days}d`;
}

export function truncateRevision(rev: string | undefined, maxLen: number = 12): string {
    if (!rev) { return '<none>'; }
    // Git SHA-like revisions: take branch@sha
    const atIdx = rev.lastIndexOf('@');
    if (atIdx > 0) {
        const branch = rev.slice(0, atIdx);
        const sha = rev.slice(atIdx + 1);
        if (sha.startsWith('sha256:')) {
            return `${branch}@${sha.slice(0, 7 + maxLen)}`;
        }
        return `${branch}@${sha.slice(0, maxLen)}`;
    }
    if (rev.length > maxLen) { return rev.slice(0, maxLen) + '...'; }
    return rev;
}

export { isFluxNotInstalled };
