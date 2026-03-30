// Noms CLI wrapper library
// Communicates with the noms CLI via subprocess

export interface NomsLogEntry {
  hash: string;
  parents: string;
  message: string;
  author: string;
  date: string;
  dataset: string;
}

export interface NomsBranch {
  name: string;
  head: string;
  created_at: string;
}

export interface NomsDataset {
  name: string;
  head: string;
  root_hash: string;
}

export interface NomsStatus {
  current_branch: string;
  branches: string[];
  staged: string[];
  modified: string[];
  clean: boolean;
}

export interface Remote {
  name: string;
  url: string;
}

export interface NomsError {
  error: string;
}

function isNomsError(obj: unknown): obj is NomsError {
  return typeof obj === 'object' && obj !== null && 'error' in obj;
}

const SAFE_NAME_PATTERN = /^[a-zA-Z0-9_-]+$/;
const MAX_NAME_LENGTH = 256;

function validateName(name: string, fieldName: string): void {
  if (typeof name !== 'string') {
    throw new Error(`${fieldName} must be a string`);
  }
  if (name.length === 0) {
    throw new Error(`${fieldName} cannot be empty`);
  }
  if (name.length > MAX_NAME_LENGTH) {
    throw new Error(`${fieldName} exceeds maximum length`);
  }
  if (!SAFE_NAME_PATTERN.test(name)) {
    throw new Error(`${fieldName} contains invalid characters`);
  }
}

function validateUrl(url: string): void {
  if (typeof url !== 'string') {
    throw new Error('URL must be a string');
  }
  if (url.length === 0) {
    throw new Error('URL cannot be empty');
  }
  try {
    new URL(url);
  } catch {
    throw new Error('Invalid URL format');
  }
}

function validateMessage(message: string): void {
  if (typeof message !== 'string') {
    throw new Error('Message must be a string');
  }
  if (message.length > 10000) {
    throw new Error('Message exceeds maximum length');
  }
}

export async function runNoms(args: string[], format: 'text' | 'json' = 'json'): Promise<{ stdout: string; stderr: string; code: number }> {
  const proc = Bun.spawn(['noms', ...args, '--format', format], {
    stdout: 'pipe',
    stderr: 'pipe',
  });
  
  const [stdout, stderr] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
  ]);
  
  const code = await proc.exited;
  return { stdout, stderr, code };
}

export async function getLog(limit: number = 50): Promise<NomsLogEntry[]> {
  const result = await runNoms(['query', 'noms_log']);
  if (result.code !== 0) {
    throw new Error(result.stderr || result.stdout);
  }
  
  try {
    const data = JSON.parse(result.stdout);
    return data.noms_log || [];
  } catch {
    return [];
  }
}

export async function getBranches(): Promise<NomsBranch[]> {
  const result = await runNoms(['query', 'noms_branches']);
  if (result.code !== 0) {
    throw new Error(result.stderr);
  }
  
  try {
    const data = JSON.parse(result.stdout);
    return data.branches || [];
  } catch {
    return [];
  }
}

export async function getDatasets(): Promise<NomsDataset[]> {
  const result = await runNoms(['query', 'noms_datasets']);
  if (result.code !== 0) {
    throw new Error(result.stderr);
  }
  
  try {
    const data = JSON.parse(result.stdout);
    return data.datasets || [];
  } catch {
    return [];
  }
}

export async function getStatus(): Promise<NomsStatus | null> {
  const result = await runNoms(['status']);
  if (result.code !== 0) {
    return null;
  }
  
  try {
    return JSON.parse(result.stdout);
  } catch {
    return null;
  }
}

export async function getRemotes(): Promise<Remote[]> {
  const result = await runNoms(['remote']);
  if (result.code !== 0) {
    return [];
  }
  
  try {
    return JSON.parse(result.stdout);
  } catch {
    return [];
  }
}

export async function checkoutBranch(branchName: string, create: boolean = false): Promise<boolean> {
  validateName(branchName, 'Branch name');
  const args = create ? ['checkout', '-b', branchName] : ['checkout', branchName];
  const result = await runNoms(args);
  return result.code === 0;
}

export async function createBranch(branchName: string): Promise<boolean> {
  validateName(branchName, 'Branch name');
  const result = await runNoms(['branch', '-c', branchName]);
  return result.code === 0;
}

export async function deleteBranch(branchName: string): Promise<boolean> {
  validateName(branchName, 'Branch name');
  const result = await runNoms(['branch', '-d', branchName]);
  return result.code === 0;
}

export async function addRemote(name: string, url: string): Promise<boolean> {
  validateName(name, 'Remote name');
  validateUrl(url);
  const result = await runNoms(['remote', '--add', name, url]);
  return result.code === 0;
}

export async function removeRemote(name: string): Promise<boolean> {
  validateName(name, 'Remote name');
  const result = await runNoms(['remote', '--remove', name]);
  return result.code === 0;
}

export async function push(remote: string = 'origin', branch: string = ''): Promise<boolean> {
  if (remote) validateName(remote, 'Remote name');
  if (branch) validateName(branch, 'Branch name');
  const args = branch ? ['push', remote, branch] : ['push', remote];
  const result = await runNoms(args);
  return result.code === 0;
}

export async function pull(remote: string = 'origin', branch: string = ''): Promise<boolean> {
  if (remote) validateName(remote, 'Remote name');
  if (branch) validateName(branch, 'Branch name');
  const args = branch ? ['pull', remote, branch] : ['pull', remote];
  const result = await runNoms(args);
  return result.code === 0;
}

export async function init(): Promise<boolean> {
  const result = await runNoms(['init']);
  return result.code === 0;
}

export async function commit(message: string, path: string, dataset: string): Promise<boolean> {
  validateMessage(message);
  validateName(path, 'Path');
  validateName(dataset, 'Dataset');
  const result = await runNoms(['commit', '-m', message, path, dataset]);
  return result.code === 0;
}
