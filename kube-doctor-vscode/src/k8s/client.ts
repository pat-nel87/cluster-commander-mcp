import * as k8s from '@kubernetes/client-node';

let _kubeConfig: k8s.KubeConfig | undefined;

export function getKubeConfig(): k8s.KubeConfig {
    if (!_kubeConfig) {
        _kubeConfig = new k8s.KubeConfig();
        _kubeConfig.loadFromDefault();
    }
    return _kubeConfig;
}

export function getCoreApi(): k8s.CoreV1Api {
    return getKubeConfig().makeApiClient(k8s.CoreV1Api);
}

export function getAppsApi(): k8s.AppsV1Api {
    return getKubeConfig().makeApiClient(k8s.AppsV1Api);
}

export function getNetworkingApi(): k8s.NetworkingV1Api {
    return getKubeConfig().makeApiClient(k8s.NetworkingV1Api);
}

export function getCurrentContext(): string {
    return getKubeConfig().getCurrentContext();
}

export function getRbacApi(): k8s.RbacAuthorizationV1Api {
    return getKubeConfig().makeApiClient(k8s.RbacAuthorizationV1Api);
}

export function getPolicyApi(): k8s.PolicyV1Api {
    return getKubeConfig().makeApiClient(k8s.PolicyV1Api);
}

/** Reset cached config (e.g. if user switches context). */
export function resetClient(): void {
    _kubeConfig = undefined;
}
