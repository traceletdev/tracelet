import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import * as os from 'os';

let diagnosticCollection: vscode.DiagnosticCollection;
let statusBarItem: vscode.StatusBarItem;
let debounceTimer: NodeJS.Timeout | undefined;

// Shapes of the JSON emitted by `tracelet lint --format json`.
interface LintResult {
  ruleId?: string;
  level?: string;
  detail?: string;
}
interface RouteStat {
  jsGzipBytes?: number;
}
interface LintStats {
  routes?: RouteStat[];
}

class TraceletCodeActionProvider implements vscode.CodeActionProvider {
  provideCodeActions(
    document: vscode.TextDocument,
    range: vscode.Range | vscode.Selection,
    context: vscode.CodeActionContext
  ): vscode.CodeAction[] {
    const actions: vscode.CodeAction[] = [];

    for (const diag of context.diagnostics) {
      if (diag.source !== 'tracelet' || !diag.code) {
        continue;
      }

      let ruleId: string | undefined;
      if (typeof diag.code === 'string') {
        ruleId = diag.code;
      } else if (typeof diag.code === 'number') {
        continue; // Skip numeric codes
      } else {
        ruleId = typeof diag.code.value === 'string' ? diag.code.value : undefined;
      }

      if (!ruleId) {
        continue;
      }

      if (ruleId === 'unoptimized-image') {
        const action = new vscode.CodeAction('Add loading="lazy"', vscode.CodeActionKind.QuickFix);
        action.diagnostics = [diag];
        action.edit = new vscode.WorkspaceEdit();
        const line = document.lineAt(range.start.line);
        const lineText = line.text;
        const imgMatch = lineText.match(/<img([^>]*)>/);
        if (imgMatch && !imgMatch[1].includes('loading')) {
          const newText = lineText.replace(/<img([^>]*)>/, '<img$1 loading="lazy">');
          action.edit.replace(document.uri, line.range, newText);
          actions.push(action);
        }
      } else if (ruleId === 'font-display') {
        const action = new vscode.CodeAction(
          'Add font-display: swap',
          vscode.CodeActionKind.QuickFix
        );
        action.diagnostics = [diag];
        action.edit = new vscode.WorkspaceEdit();
        const line = document.lineAt(range.start.line);
        const lineText = line.text;
        if (lineText.includes('@font-face')) {
          // Find the opening brace
          let braceLine = range.start.line;
          for (
            let i = range.start.line;
            i < Math.min(range.start.line + 5, document.lineCount);
            i++
          ) {
            if (document.lineAt(i).text.includes('{')) {
              braceLine = i;
              break;
            }
          }
          const nextLine = Math.min(braceLine + 1, document.lineCount - 1);
          const nextText = document.lineAt(nextLine).text;
          if (!nextText.includes('font-display')) {
            const indent = nextText.match(/^(\s*)/)?.[1] || '  ';
            action.edit.insert(
              document.uri,
              new vscode.Position(nextLine, 0),
              `${indent}font-display: swap;\n`
            );
            actions.push(action);
          }
        }
      }
    }

    return actions;
  }
}

export function activate(context: vscode.ExtensionContext) {
  diagnosticCollection = vscode.languages.createDiagnosticCollection('tracelet');
  context.subscriptions.push(diagnosticCollection);

  statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
  context.subscriptions.push(statusBarItem);

  const config = vscode.workspace.getConfiguration('tracelet');
  const debounceMs = config.get<number>('debounceMs', 600);

  // Lint on save
  context.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument(doc => {
      if (debounceTimer) {
        clearTimeout(debounceTimer);
      }
      debounceTimer = setTimeout(() => {
        lintDocument(doc);
      }, debounceMs);
    })
  );

  // Lint on type (if enabled)
  if (config.get<boolean>('enableOnType', false)) {
    context.subscriptions.push(
      vscode.workspace.onDidChangeTextDocument(e => {
        if (debounceTimer) {
          clearTimeout(debounceTimer);
        }
        debounceTimer = setTimeout(() => {
          lintDocument(e.document);
        }, debounceMs * 2);
      })
    );
  }

  // Code action provider for quick fixes
  context.subscriptions.push(
    vscode.languages.registerCodeActionsProvider(
      { scheme: 'file', pattern: '**/*.{html,tsx,jsx,css,scss}' },
      new TraceletCodeActionProvider(),
      {
        providedCodeActionKinds: [vscode.CodeActionKind.QuickFix],
      }
    )
  );

  // Commands
  context.subscriptions.push(
    vscode.commands.registerCommand('tracelet.lintChanged', () => {
      const editor = vscode.window.activeTextEditor;
      if (editor) {
        lintDocument(editor.document);
      }
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('tracelet.probeCurrentRoute', async () => {
      const url = await vscode.window.showInputBox({
        prompt: 'Enter URL to probe (e.g., http://localhost:3000)',
        placeHolder: 'http://localhost:3000',
      });
      if (url) {
        probeRoute(url);
      }
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('tracelet.openConfig', async () => {
      const configPath = findConfigFile();
      if (configPath && fs.existsSync(configPath)) {
        const doc = await vscode.workspace.openTextDocument(configPath);
        await vscode.window.showTextDocument(doc);
      } else {
        vscode.window.showInformationMessage('tracelet.config.json not found in workspace');
      }
    })
  );

  // Initial lint if workspace is available
  if (vscode.workspace.workspaceFolders) {
    setTimeout(() => {
      vscode.workspace.textDocuments.forEach(doc => {
        if (isRelevantFile(doc)) {
          lintDocument(doc);
        }
      });
    }, 1000);
  }
}

async function lintDocument(doc: vscode.TextDocument) {
  if (!isRelevantFile(doc)) {
    console.log('[tracelet] skipping file (not relevant):', doc.fileName);
    return;
  }

  const configPath = findConfigFile();
  if (!configPath) {
    console.log('[tracelet] no config file found');
    return;
  }

  const binary = getBinaryPath();
  if (!binary) {
    console.log('[tracelet] binary not found');
    return;
  }

  console.log('[tracelet] linting:', doc.fileName);

  try {
    const args = ['lint', '--format', 'json', '--file', doc.fileName, '--config', configPath];
    console.log('[tracelet] running:', binary, args.join(' '));

    // Use spawn to capture both stdout and stderr
    const result = cp.spawnSync(binary, args, {
      encoding: 'utf8',
      cwd: path.dirname(configPath),
      maxBuffer: 10 * 1024 * 1024,
      stdio: 'pipe',
    });

    const stdout = result.stdout || '';
    const stderr = result.stderr || '';

    // Log stderr if present
    if (stderr) {
      console.log('[tracelet] stderr:', stderr);
    }

    // Try to parse output even if exit code is non-zero (CLI might return JSON with errors)
    if (stdout.trim()) {
      try {
        console.log('[tracelet] result:', stdout.substring(0, 200));
        const data = JSON.parse(stdout);
        console.log('[tracelet] results count:', data.results?.length || 0);
        updateDiagnostics(doc, data.results || []);
        updateStatusBar(data.stats);
        return; // Success
      } catch (parseErr) {
        console.log('[tracelet] JSON parse error:', parseErr);
      }
    }

    // If we get here, either no output or parse failed
    if (result.status !== 0) {
      console.log('[tracelet] lint failed with exit code:', result.status);
      if (stderr) {
        vscode.window.showWarningMessage(`tracelet lint failed: ${stderr.substring(0, 100)}`);
      }
    }

    // Clear diagnostics if command failed
    diagnosticCollection.set(doc.uri, []);
  } catch (e: unknown) {
    // Log errors for debugging
    const err = e as { message?: string; stderr?: Buffer };
    const errMsg = err.message || String(e);
    const stderr = err.stderr ? err.stderr.toString() : '';
    console.log('[tracelet] lint error:', errMsg);
    if (stderr) {
      console.log('[tracelet] stderr:', stderr);
    }
    if (errMsg.includes('ENOENT') || errMsg.includes('not found')) {
      vscode.window.showWarningMessage(
        `tracelet binary not found. Set tracelet.binaryPath in settings.`
      );
    }
    // Clear diagnostics on error
    diagnosticCollection.set(doc.uri, []);
  }
}

function updateDiagnostics(doc: vscode.TextDocument, results: LintResult[]) {
  const diagnostics: vscode.Diagnostic[] = [];

  for (const r of results) {
    if (!r.ruleId || !r.detail) continue;

    const severity =
      r.level === 'error'
        ? vscode.DiagnosticSeverity.Error
        : r.level === 'warn'
          ? vscode.DiagnosticSeverity.Warning
          : vscode.DiagnosticSeverity.Information;

    // Find line number - try multiple strategies
    let line = 0;
    const lineMatch = r.detail.match(/line (\d+)/i);
    if (lineMatch) {
      line = parseInt(lineMatch[1]) - 1;
    } else {
      // For font-display, search for @font-face in the document
      if (r.ruleId === 'font-display') {
        for (let i = 0; i < doc.lineCount; i++) {
          if (doc.lineAt(i).text.includes('@font-face')) {
            line = i;
            break;
          }
        }
      }
      // For unoptimized-image, search for <img> tags
      else if (r.ruleId === 'unoptimized-image') {
        for (let i = 0; i < doc.lineCount; i++) {
          if (doc.lineAt(i).text.match(/<img/i)) {
            line = i;
            break;
          }
        }
      }
    }

    const range = doc.lineAt(Math.min(line, doc.lineCount - 1)).range;
    const diag = new vscode.Diagnostic(range, `[${r.ruleId}] ${r.detail}`, severity);
    diag.source = 'tracelet';
    diag.code = r.ruleId;

    diagnostics.push(diag);
  }

  diagnosticCollection.set(doc.uri, diagnostics);
}

function updateStatusBar(stats: LintStats) {
  if (!stats || !stats.routes) {
    statusBarItem.hide();
    return;
  }

  let totalBytes = 0;
  // ponytail: always 0 today — the status bar never surfaces an over-budget
  // count; wire this to real budget comparison if/when the extension loads config.
  const overCount = 0;
  for (const r of stats.routes) {
    totalBytes += r.jsGzipBytes || 0;
  }

  const kb = (totalBytes / 1024).toFixed(1);
  statusBarItem.text = `$(dashboard) Tracelet: ${kb}KB`;
  statusBarItem.tooltip = `Total JS: ${kb}KB`;

  if (overCount > 0) {
    statusBarItem.text += ` ($(error) ${overCount})`;
  }

  statusBarItem.show();
}

async function probeRoute(url: string) {
  const binary = getBinaryPath();
  if (!binary) {
    vscode.window.showErrorMessage(
      'tracelet binary not found. Set tracelet.binaryPath in settings.'
    );
    return;
  }

  const config = vscode.workspace.getConfiguration('tracelet');
  const profile = config.get<string>('probe.profile', 'desktop');

  try {
    vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: 'Probing route...',
        cancellable: false,
      },
      async () => {
        const result = cp.execFileSync(
          binary,
          ['probe', url, '--profile', profile, '--format', 'json'],
          {
            encoding: 'utf8',
            maxBuffer: 10 * 1024 * 1024,
          }
        );

        const data = JSON.parse(result);
        const m = data.metrics || {};
        const msg = `FCP: ${m.fcp}ms, LCP: ${m.lcp}ms, CLS: ${m.cls || 0}`;
        vscode.window.showInformationMessage(`Tracelet Probe: ${msg}`);
      }
    );
  } catch (e: unknown) {
    const err = e as { message?: string };
    vscode.window.showErrorMessage(`Probe failed: ${err.message ?? String(e)}`);
  }
}

function getBinaryPath(): string | null {
  const config = vscode.workspace.getConfiguration('tracelet');
  const customPath = config.get<string>('binaryPath', '');

  if (customPath && fs.existsSync(customPath)) {
    console.log('[tracelet] using custom binary path:', customPath);
    return customPath;
  }

  // Try common locations
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

  // Platform-specific binary paths in node_modules
  const platform = process.platform;
  const arch = process.arch;
  let binaryDir = '';
  if (platform === 'darwin') {
    binaryDir = arch === 'arm64' ? 'darwin-arm64' : 'darwin-x64';
  } else if (platform === 'linux') {
    binaryDir = arch === 'arm64' ? 'linux-arm64' : 'linux-x64';
  } else if (platform === 'win32') {
    binaryDir = arch === 'arm64' ? 'win32-arm64' : 'win32-x64';
  }
  const binaryName = platform === 'win32' ? 'tracelet.exe' : 'tracelet';

  // Common tracelet repo locations (if user built it from source)
  // Also check node_modules for npm-installed tracelet
  const possiblePaths = [
    // Check node_modules first (npm-installed)
    workspaceRoot ? path.join(workspaceRoot, 'node_modules', '.bin', 'tracelet') : null,
    workspaceRoot && binaryDir
      ? path.join(workspaceRoot, 'node_modules', 'tracelet', 'binaries', binaryDir, binaryName)
      : null,
    // Local repo builds
    workspaceRoot ? path.join(workspaceRoot, 'tracelet') : null,
    workspaceRoot ? path.join(workspaceRoot, '..', 'tracelet', 'tracelet') : null,
    workspaceRoot ? path.join(workspaceRoot, '..', '..', 'tracelet', 'tracelet') : null,
    // Check if there's a tracelet repo sibling to current workspace
    path.join(path.dirname(workspaceRoot || ''), 'tracelet', 'tracelet'),
    // Also try in common dev locations
    path.join(os.homedir(), 'Documents', 'tracelet', 'tracelet'),
    path.join(os.homedir(), 'go', 'bin', 'tracelet'),
    // Global npm install
    path.join(os.homedir(), '.npm-global', 'bin', 'tracelet'),
    // Try npx (will be handled by PATH)
    'tracelet',
  ].filter((p): p is string => p !== null);

  for (const p of possiblePaths) {
    try {
      if (fs.existsSync(p) && fs.statSync(p).isFile()) {
        // Try to execute it to verify it's a valid binary
        try {
          cp.execFileSync(p, ['lint', '--help'], {
            encoding: 'utf8',
            stdio: 'pipe',
            timeout: 1000,
          });
        } catch {
          // Help might fail, but if file exists and is executable, that's fine
        }
        console.log('[tracelet] found binary at:', p);
        return p;
      }
    } catch {
      // continue
    }
  }

  console.log('[tracelet] binary not found in:', possiblePaths);
  return null;
}

function findConfigFile(): string | null {
  const folders = vscode.workspace.workspaceFolders;
  if (!folders) {
    return null;
  }

  for (const folder of folders) {
    const configPath = path.join(folder.uri.fsPath, 'tracelet.config.json');
    if (fs.existsSync(configPath)) {
      return configPath;
    }
  }

  return null;
}

function isRelevantFile(doc: vscode.TextDocument): boolean {
  const ext = path.extname(doc.fileName).toLowerCase();
  return ['.html', '.tsx', '.jsx', '.css', '.scss'].includes(ext);
}

export function deactivate() {
  if (debounceTimer) {
    clearTimeout(debounceTimer);
  }
}
