import { getCoreApi } from './client';

export async function listNamespaces() {
    const api = getCoreApi();
    const response = await api.listNamespace();
    return response.items;
}
