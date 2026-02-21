import * as vscode from 'vscode';
import { registerAllTools } from './tools';

export function activate(context: vscode.ExtensionContext) {
    console.log('Kube Doctor extension activated');
    registerAllTools(context);
}

export function deactivate() {}
