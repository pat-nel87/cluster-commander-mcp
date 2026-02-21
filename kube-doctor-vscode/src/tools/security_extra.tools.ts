import * as vscode from 'vscode';
import { listRoleBindings, listClusterRoleBindings } from '../k8s/rbac';
import { listNetworkPolicies } from '../k8s/network_policies';
import { listPods } from '../k8s/pods';
import { listPDBs } from '../k8s/autoscaling';
import { listResourceQuotas } from '../k8s/quotas';
import { formatTable, formatError } from '../util/formatting';

// ---- list_rbac_bindings ----

interface ListRBACBindingsInput { namespace: string; subjectFilter?: string; }

export class ListRBACBindingsTool implements vscode.LanguageModelTool<ListRBACBindingsInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<ListRBACBindingsInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Listing RBAC bindings in ${options.input.namespace}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<ListRBACBindingsInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace, subjectFilter } = options.input;
            const headers = ['BINDING', 'SCOPE', 'ROLE', 'SUBJECT-KIND', 'SUBJECT-NAME', 'SUBJECT-NS'];
            const rows: string[][] = [];

            // Namespace role bindings
            const roleBindings = await listRoleBindings(namespace);
            for (const rb of roleBindings) {
                for (const subject of rb.subjects || []) {
                    if (subjectFilter && !subject.name?.toLowerCase().includes(subjectFilter.toLowerCase())) { continue; }
                    rows.push([
                        rb.metadata?.name || '', 'Namespace',
                        `${rb.roleRef?.kind}/${rb.roleRef?.name}`,
                        subject.kind || '', subject.name || '', subject.namespace || '',
                    ]);
                }
            }

            // Cluster role bindings
            try {
                const crbs = await listClusterRoleBindings();
                for (const crb of crbs) {
                    for (const subject of crb.subjects || []) {
                        if (subjectFilter && !subject.name?.toLowerCase().includes(subjectFilter.toLowerCase())) { continue; }
                        if (subject.kind === 'ServiceAccount' && subject.namespace && subject.namespace !== namespace) { continue; }
                        rows.push([
                            crb.metadata?.name || '', 'Cluster',
                            `${crb.roleRef?.kind}/${crb.roleRef?.name}`,
                            subject.kind || '', subject.name || '', subject.namespace || '',
                        ]);
                    }
                }
            } catch { /* best-effort */ }

            let output = `=== RBAC Bindings (namespace: ${namespace}) ===\n`;
            if (subjectFilter) { output += `Filter: ${subjectFilter}\n`; }
            output += `${formatTable(headers, rows)}\n\nFound ${rows.length} bindings`;
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(output)]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('listing RBAC bindings', err))]);
        }
    }
}

// ---- audit_namespace_security ----

interface AuditNamespaceSecurityInput { namespace: string; }

export class AuditNamespaceSecurityTool implements vscode.LanguageModelTool<AuditNamespaceSecurityInput> {
    async prepareInvocation(options: vscode.LanguageModelToolInvocationPrepareOptions<AuditNamespaceSecurityInput>): Promise<vscode.PreparedToolInvocation> {
        return { invocationMessage: `Auditing security for namespace ${options.input.namespace}...` };
    }

    async invoke(options: vscode.LanguageModelToolInvocationOptions<AuditNamespaceSecurityInput>): Promise<vscode.LanguageModelToolResult> {
        try {
            const { namespace } = options.input;
            const lines: string[] = [];
            let score = 100;
            let findings = 0;

            lines.push(`=== Namespace Security Audit: ${namespace} ===`);
            lines.push('');

            // 1. Network Policies
            lines.push('--- Network Policies ---');
            let hasNetPol = false;
            let netPolCount = 0;
            try {
                const netPols = await listNetworkPolicies(namespace);
                netPolCount = netPols.length;
                if (netPols.length === 0) {
                    lines.push('[WARNING] No network policies — all pod traffic is unrestricted');
                    score -= 20; findings++;
                } else {
                    lines.push(`  ${netPols.length} network policies defined`);
                    hasNetPol = true;
                }
            } catch { lines.push('  (could not check network policies)'); }

            // 2. PDBs
            lines.push('');
            lines.push('--- Pod Disruption Budgets ---');
            let hasPDB = false;
            let pdbCount = 0;
            try {
                const pdbs = await listPDBs(namespace);
                pdbCount = pdbs.length;
                if (pdbs.length === 0) {
                    lines.push('[INFO] No PDBs — workloads have no disruption protection');
                    score -= 5; findings++;
                } else {
                    lines.push(`  ${pdbs.length} PDBs defined`);
                    hasPDB = true;
                    for (const pdb of pdbs) {
                        if (pdb.status?.disruptionsAllowed === 0 && (pdb.status?.expectedPods || 0) > 0) {
                            lines.push(`[WARNING] PDB '${pdb.metadata?.name}' has 0 disruptions allowed`);
                            findings++;
                        }
                    }
                }
            } catch { lines.push('  (could not check PDBs)'); }

            // 3. Pod Security
            lines.push('');
            lines.push('--- Pod Security ---');
            let podsScanned = false;
            let privilegedPods = 0, rootPods = 0, noSecCtxPods = 0;
            try {
                const pods = await listPods(namespace);
                if (pods.length === 0) {
                    lines.push('  No pods in namespace');
                } else {
                    podsScanned = true;
                    for (const pod of pods) {
                        let hasAnySecCtx = !!pod.spec?.securityContext;
                        for (const c of pod.spec?.containers || []) {
                            if (!c.securityContext) {
                                if (!hasAnySecCtx) { noSecCtxPods++; }
                                continue;
                            }
                            hasAnySecCtx = true;
                            if (c.securityContext.privileged) { privilegedPods++; }
                            if (c.securityContext.runAsUser === 0) { rootPods++; }
                        }
                        if (pod.spec?.securityContext?.runAsUser === 0) { rootPods++; }
                    }
                    lines.push(`  ${pods.length} pods scanned`);
                    if (privilegedPods > 0) {
                        lines.push(`[CRITICAL] ${privilegedPods} pod(s) running in privileged mode`);
                        score -= 15; findings++;
                    }
                    if (rootPods > 0) {
                        lines.push(`[WARNING] ${rootPods} pod(s) running as root`);
                        score -= 10; findings++;
                    }
                    if (noSecCtxPods > 0) {
                        lines.push(`[INFO] ${noSecCtxPods} pod(s) with no SecurityContext`);
                        score -= 5; findings++;
                    }
                }
            } catch { lines.push('  (could not list pods)'); }

            // 4. RBAC
            lines.push('');
            lines.push('--- RBAC ---');
            let hasRBAC = false;
            let bindingCount = 0;
            try {
                const bindings = await listRoleBindings(namespace);
                bindingCount = bindings.length;
                lines.push(`  ${bindings.length} role bindings`);
                if (bindings.length > 0) { hasRBAC = true; }
            } catch { lines.push('  (could not check RBAC)'); }

            // 5. Resource Quotas
            lines.push('');
            lines.push('--- Resource Quotas ---');
            let hasQuota = false;
            let quotaCount = 0;
            try {
                const quotas = await listResourceQuotas(namespace);
                quotaCount = quotas.length;
                if (quotas.length === 0) {
                    lines.push('[INFO] No resource quotas — resource consumption is unrestricted');
                    score -= 5; findings++;
                } else {
                    lines.push(`  ${quotas.length} resource quotas defined`);
                    hasQuota = true;
                }
            } catch { lines.push('  (could not check resource quotas)'); }

            // Score
            if (score < 0) { score = 0; }
            let grade = 'A';
            if (score >= 90) { grade = 'A'; }
            else if (score >= 80) { grade = 'B'; }
            else if (score >= 70) { grade = 'C'; }
            else if (score >= 60) { grade = 'D'; }
            else { grade = 'F'; }

            lines.push('');
            lines.push('--- Overall Security Score ---');
            lines.push(`  Score: ${score}/100 (Grade: ${grade})`);
            lines.push(`  ${findings} finding(s) identified`);

            // Mermaid
            const netPolStatus = hasNetPol ? `${netPolCount} policies` : 'none';
            const pdbStatus = hasPDB ? `${pdbCount} PDBs` : 'none';
            const secStatus = podsScanned ? (privilegedPods === 0 && rootPods === 0 ? 'good' : `${privilegedPods + rootPods} issues`) : 'not scanned';
            const rbacStatus = hasRBAC ? `${bindingCount} bindings` : 'none';
            const quotaStatus = hasQuota ? `${quotaCount} quotas` : 'none';

            lines.push('\nPOLICY COVERAGE:');
            lines.push('```mermaid');
            lines.push('graph TD');
            lines.push(`    NS[Namespace: ${namespace}]`);
            lines.push(`    NS --> NP[NetworkPolicies: ${netPolStatus}]`);
            lines.push(`    NS --> PDB[PDBs: ${pdbStatus}]`);
            lines.push(`    NS --> SEC[Pod Security: ${secStatus}]`);
            lines.push(`    NS --> RBAC[RBAC: ${rbacStatus}]`);
            lines.push(`    NS --> QUOTA[Quotas: ${quotaStatus}]`);
            lines.push('```');

            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(lines.join('\n'))]);
        } catch (err) {
            return new vscode.LanguageModelToolResult([new vscode.LanguageModelTextPart(formatError('auditing namespace security', err))]);
        }
    }
}
