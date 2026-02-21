import * as k8s from '@kubernetes/client-node';
import { getCoreApi, getAppsApi } from './client';

export interface WorkloadDependencies {
    configMaps: string[];
    secrets: string[];
    pvcs: string[];
    serviceAccount: string;
    matchingServices: string[];
}

export async function extractWorkloadDependencies(
    namespace: string,
    name: string,
    kind: string = 'Deployment'
): Promise<{ deps: WorkloadDependencies; podLabels: Record<string, string>; displayName: string }> {
    let podSpec: k8s.V1PodSpec | undefined;
    let podLabels: Record<string, string> = {};
    let displayName = '';

    const coreApi = getCoreApi();
    const appsApi = getAppsApi();

    switch (kind) {
        case 'Deployment': {
            const deploy = await appsApi.readNamespacedDeployment({ namespace, name });
            podSpec = deploy.spec?.template?.spec;
            podLabels = deploy.spec?.template?.metadata?.labels || {};
            displayName = `Deployment/${deploy.metadata?.name}`;
            break;
        }
        case 'StatefulSet': {
            const sts = await appsApi.readNamespacedStatefulSet({ namespace, name });
            podSpec = sts.spec?.template?.spec;
            podLabels = sts.spec?.template?.metadata?.labels || {};
            displayName = `StatefulSet/${sts.metadata?.name}`;
            break;
        }
        case 'Pod': {
            const pod = await coreApi.readNamespacedPod({ namespace, name });
            podSpec = pod.spec;
            podLabels = pod.metadata?.labels || {};
            displayName = `Pod/${pod.metadata?.name}`;
            break;
        }
        default:
            throw new Error(`Unsupported workload kind: ${kind}`);
    }

    if (!podSpec) {
        throw new Error(`Could not extract pod spec from ${kind}/${name}`);
    }

    const configMaps = new Set<string>();
    const secrets = new Set<string>();
    const pvcs = new Set<string>();
    const serviceAccount = podSpec.serviceAccountName || '';

    // From volumes
    for (const vol of podSpec.volumes || []) {
        if (vol.configMap) { configMaps.add(vol.configMap.name!); }
        if (vol.secret) { secrets.add(vol.secret.secretName!); }
        if (vol.persistentVolumeClaim) { pvcs.add(vol.persistentVolumeClaim.claimName!); }
        if (vol.projected) {
            for (const src of vol.projected.sources || []) {
                if (src.configMap) { configMaps.add(src.configMap.name!); }
                if (src.secret) { secrets.add(src.secret.name!); }
            }
        }
    }

    // From container env/envFrom
    const allContainers = [...(podSpec.initContainers || []), ...(podSpec.containers || [])];
    for (const c of allContainers) {
        for (const ef of c.envFrom || []) {
            if (ef.configMapRef) { configMaps.add(ef.configMapRef.name!); }
            if (ef.secretRef) { secrets.add(ef.secretRef.name!); }
        }
        for (const env of c.env || []) {
            if (env.valueFrom?.configMapKeyRef) { configMaps.add(env.valueFrom.configMapKeyRef.name!); }
            if (env.valueFrom?.secretKeyRef) { secrets.add(env.valueFrom.secretKeyRef.name!); }
        }
    }

    // Find matching services
    const matchingServices: string[] = [];
    if (Object.keys(podLabels).length > 0) {
        try {
            const svcs = await coreApi.listNamespacedService({ namespace });
            for (const svc of svcs.items) {
                const selector = svc.spec?.selector || {};
                if (Object.keys(selector).length === 0) { continue; }
                const match = Object.entries(selector).every(([k, v]) => podLabels[k] === v);
                if (match) { matchingServices.push(svc.metadata?.name || ''); }
            }
        } catch { /* best-effort */ }
    }

    return {
        deps: {
            configMaps: Array.from(configMaps),
            secrets: Array.from(secrets),
            pvcs: Array.from(pvcs),
            serviceAccount,
            matchingServices,
        },
        podLabels,
        displayName,
    };
}
