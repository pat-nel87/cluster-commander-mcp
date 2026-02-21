import * as vscode from 'vscode';
import { ListNamespacesTool, ListNodesTool } from './cluster.tools';
import { ListPodsTool, GetPodLogsTool } from './pod.tools';
import { GetEventsTool } from './event.tools';
import { ListDeploymentsTool } from './workload.tools';
import { ListServicesTool } from './network.tools';
import { DiagnosePodTool, DiagnoseNamespaceTool, DiagnoseClusterTool, FindUnhealthyPodsTool } from './diagnostics.tools';
import { ListNetworkPoliciesTool } from './policy.tools';
import { AnalyzePodConnectivityTool } from './connectivity.tools';
import { AnalyzePodSecurityTool } from './security.tools';
import { GetWorkloadDependenciesTool } from './dependencies.tools';

export function registerAllTools(context: vscode.ExtensionContext): void {
    // Each name MUST match the "name" in package.json contributes.languageModelTools
    context.subscriptions.push(
        vscode.lm.registerTool('kube-doctor_listNamespaces', new ListNamespacesTool()),
        vscode.lm.registerTool('kube-doctor_listPods', new ListPodsTool()),
        vscode.lm.registerTool('kube-doctor_getPodLogs', new GetPodLogsTool()),
        vscode.lm.registerTool('kube-doctor_getEvents', new GetEventsTool()),
        vscode.lm.registerTool('kube-doctor_listDeployments', new ListDeploymentsTool()),
        vscode.lm.registerTool('kube-doctor_listNodes', new ListNodesTool()),
        vscode.lm.registerTool('kube-doctor_listServices', new ListServicesTool()),
        vscode.lm.registerTool('kube-doctor_diagnosePod', new DiagnosePodTool()),
        vscode.lm.registerTool('kube-doctor_diagnoseNamespace', new DiagnoseNamespaceTool()),
        vscode.lm.registerTool('kube-doctor_diagnoseCluster', new DiagnoseClusterTool()),
        vscode.lm.registerTool('kube-doctor_findUnhealthyPods', new FindUnhealthyPodsTool()),
        vscode.lm.registerTool('kube-doctor_listNetworkPolicies', new ListNetworkPoliciesTool()),
        vscode.lm.registerTool('kube-doctor_analyzePodConnectivity', new AnalyzePodConnectivityTool()),
        vscode.lm.registerTool('kube-doctor_analyzePodSecurity', new AnalyzePodSecurityTool()),
        vscode.lm.registerTool('kube-doctor_getWorkloadDependencies', new GetWorkloadDependenciesTool()),
    );
}
