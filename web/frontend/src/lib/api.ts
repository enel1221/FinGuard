const API_BASE = '/api/v1';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(API_BASE + path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });
  if (resp.status === 401) {
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }
  if (!resp.ok) {
    const body = await resp.json().catch(() => ({ error: resp.statusText }));
    throw new Error(body.error || resp.statusText);
  }
  return resp.json();
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};

export interface Project {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface CostSource {
  id: string;
  projectId: string;
  type: string;
  name: string;
  config: Record<string, unknown>;
  enabled: boolean;
  lastCollectedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CostSummary {
  totalListCost: number;
  totalNetCost: number;
  totalAmortized: number;
  totalAmortizedNet: number;
  recordCount: number;
}

export interface HealthResponse {
  status: string;
  services: Record<string, string>;
}

export interface UserInfo {
  userId: string;
  email: string;
  displayName: string;
  groups?: string[];
}

export interface PluginMeta {
  name: string;
  version: string;
  description: string;
  type: string;
  topics: string[];
  routes: { method: string; path: string; description: string }[];
}

export type SubjectType = 'user' | 'group';
export type RoleType = 'viewer' | 'editor' | 'admin' | 'platform-admin';

export interface ProjectRole {
  projectId: string;
  subjectType: SubjectType;
  subjectId: string;
  role: RoleType;
}

export interface KubernetesSourceConfig {
  clusterName: string;
  opencostUrl: string;
  kubeconfigRef?: string;
}

export interface AWSSourceConfig {
  accountId: string;
  roleArn: string;
  externalId?: string;
  region: string;
  athenaBucket?: string;
  athenaRegion?: string;
  athenaDatabase?: string;
  athenaTable?: string;
  athenaWorkgroup?: string;
  curVersion?: string;
}

export interface AzureSourceConfig {
  subscriptionId: string;
  tenantId: string;
  clientId: string;
  clientSecret?: string;
  storageAccount?: string;
  storageAccessKey?: string;
  storageContainer?: string;
  containerPath?: string;
  azureCloud?: string;
}

export interface GCPSourceConfig {
  projectId: string;
  billingAccountId?: string;
  billingDataDataset?: string;
  serviceAccountKey?: string;
}
