import * as vscode from 'vscode';
import { getPod } from '../k8s/pods';
import { formatError } from '../util/formatting';

interface AnalyzePodSecurityInput {
    namespace: string;
    podName: string;
}

export class AnalyzePodSecurityTool implements vscode.LanguageModelTool<AnalyzePodSecurityInput> {
    async prepareInvocation(
        options: vscode.LanguageModelToolInvocationPrepareOptions<AnalyzePodSecurityInput>
    ): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Analyzing security for pod ${options.input.namespace}/${options.input.podName}...` };
    }

    async invoke(
        options: vscode.LanguageModelToolInvocationOptions<AnalyzePodSecurityInput>
    ): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, podName } = options.input;
            const pod = await getPod(namespace, podName);

            const lines: string[] = [];
            const actions: string[] = [];
            let findings = 0;

            lines.push(`=== Pod Security Analysis: ${podName} (namespace: ${namespace}) ===`);
            lines.push('');
            lines.push('--- Pod-Level Security ---');

            if (pod.spec?.hostNetwork) {
                lines.push('[CRITICAL] Pod uses hostNetwork — shares node\'s network namespace');
                actions.push('Remove hostNetwork unless absolutely required');
                findings++;
            }
            if (pod.spec?.hostPID) {
                lines.push('[CRITICAL] Pod uses hostPID — can see all processes on the node');
                actions.push('Remove hostPID to prevent process visibility across the node');
                findings++;
            }
            if (pod.spec?.hostIPC) {
                lines.push('[WARNING] Pod uses hostIPC — shares node\'s IPC namespace');
                actions.push('Remove hostIPC unless inter-process communication with host is required');
                findings++;
            }

            const podSC = pod.spec?.securityContext;
            if (podSC) {
                if (podSC.runAsUser === 0) {
                    lines.push('[CRITICAL] Pod runAsUser is 0 (root)');
                    actions.push('Set runAsUser to a non-zero UID (e.g., 1000)');
                    findings++;
                }
                if (podSC.runAsNonRoot === false) {
                    lines.push('[WARNING] Pod runAsNonRoot is explicitly set to false');
                    findings++;
                }
                if (podSC.seccompProfile) {
                    lines.push(`  Seccomp Profile: ${podSC.seccompProfile.type}`);
                } else {
                    lines.push('[INFO] No seccomp profile set at pod level');
                    findings++;
                }
            } else {
                lines.push('[INFO] No pod-level SecurityContext defined');
                findings++;
            }

            lines.push('');
            lines.push('--- Container Security ---');

            const allContainers = [...(pod.spec?.initContainers || []), ...(pod.spec?.containers || [])];
            for (const c of allContainers) {
                lines.push('');
                lines.push(`  Container: ${c.name}`);
                const sc = c.securityContext;
                if (!sc) {
                    lines.push(`    [WARNING] No SecurityContext defined`);
                    actions.push(`Add SecurityContext to container '${c.name}'`);
                    findings++;
                    continue;
                }

                if (sc.privileged) {
                    lines.push(`    [CRITICAL] Container runs in privileged mode`);
                    actions.push(`Remove privileged mode from container '${c.name}'`);
                    findings++;
                }
                if (sc.allowPrivilegeEscalation === undefined || sc.allowPrivilegeEscalation) {
                    lines.push(`    [WARNING] allowPrivilegeEscalation is not explicitly disabled`);
                    findings++;
                }
                if (sc.runAsUser === 0) {
                    lines.push(`    [CRITICAL] Container runAsUser is 0 (root)`);
                    findings++;
                }
                if (!sc.readOnlyRootFilesystem) {
                    lines.push(`    [INFO] readOnlyRootFilesystem is not enabled`);
                    findings++;
                }
                if (sc.capabilities) {
                    if (sc.capabilities.add && sc.capabilities.add.length > 0) {
                        const hasDangerous = sc.capabilities.add.some(
                            cap => cap === 'SYS_ADMIN' || cap === 'NET_ADMIN' || cap === 'ALL'
                        );
                        const severity = hasDangerous ? 'CRITICAL' : 'WARNING';
                        lines.push(`    [${severity}] Added capabilities: ${sc.capabilities.add.join(', ')}`);
                        findings++;
                    }
                    if (sc.capabilities.drop && sc.capabilities.drop.length > 0) {
                        lines.push(`    Dropped capabilities: ${sc.capabilities.drop.join(', ')}`);
                    }
                } else {
                    lines.push(`    [INFO] No capabilities configuration (consider dropping ALL and adding only needed)`);
                    findings++;
                }
            }

            lines.push('');
            lines.push('--- Summary ---');
            if (findings === 0) {
                lines.push('  Pod security posture looks good. No issues found.');
            } else {
                lines.push(`  ${findings} security finding(s) identified.`);
            }

            if (actions.length > 0) {
                lines.push('');
                lines.push('SUGGESTED ACTIONS:');
                actions.forEach((action, i) => {
                    lines.push(`${i + 1}. ${action}`);
                });
            }

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('analyzing pod security', err))]);
        }
    }
}
