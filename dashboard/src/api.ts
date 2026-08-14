export type Precision = 'exact' | 'estimated' | 'unavailable';
export type PrecisionCounts = { exact: number; estimated: number; unavailable: number };
export type Summary = { projects: number; sessions: number; activeSessions: number; events: number; precision: PrecisionCounts };
export type Session = { id: string; client: string; model: string; status: string; startedAt: string; eventCount: number };
export type Costs = { currency: string; exactMicros: number; estimatedMicros: number; unavailable: number };
export type Memories = { active: number; candidate: number; disabled: number };
export type TrendPoint = { date: string; sessions: number; events: number };
export type Snapshot = { sessions: Session[]; costs: Costs; memories: Memories; trends: TrendPoint[]; comparisonCount: number };
export type PrivacySettings = { capturePrompts: boolean; captureFileContents: boolean; retentionDays: number };
export type SafeEvent = { eventId: string; timestamp: string; eventType: string; provenance: string; precision: Precision };
export type SessionEvidence = { sessionId: string; events: SafeEvent[] };
export type Message = { id: string; requestId: string; sequence: number; role: 'system' | 'user' | 'assistant' | 'tool'; content: string; toolName?: string; toolPayloadJson?: string; createdAt: string };
export type Conversation = { id: string; sessionId: string; projectId: string; client: { name: string; version: string }; model: { displayName: string }; protocol: string; method: string; path: string; statusCode: number; startedAt: string; completedAt?: string; firstByteMs: number; durationMs: number; usage: { inputTokens?: number; outputTokens?: number; cachedTokens?: number; reasoningTokens?: number; precision: Precision; provenance: string }; cost: { amountMicros?: number; currency: string; precision: Precision; provenance: string }; messages: Message[] };
export type ClientConnection = { key: string; displayName: string; detected: boolean; state: 'unavailable' | 'detected' | 'connected' | 'active' | 'error'; capability: string; detail: string; lastHeartbeatAt?: string; updatedAt: string };
export type FindingSeverity = 'good' | 'info' | 'low' | 'medium' | 'high';
export type AnalysisFinding = { id: string; severity: FindingSeverity; title: string; description: string; evidence: string; recommendation: string };
export type LiveAnalysis = { projectId: string; requests: number; activeSessions: number; inputTokens: number; outputTokens: number; cachedTokens: number; reasoningTokens: number; exactCostMicros: number; estimatedCostMicros: number; unknownCostCount: number; averageLatencyMs: number; failedRequests: number; toolCalls: number; tokensPerRequest: number; cacheHitRate: number; healthScore: number; summary: string; findings: AnalysisFinding[]; limitations: string[] };
export type GuidanceKind = 'continue' | 'advise' | 'redirect' | 'ask' | 'block' | 'verify';
export type GuidanceSeverity = 'info' | 'warning' | 'high' | 'critical';
export type ControlLevel = 'observe' | 'guide' | 'guard' | 'autopilot';
export type GuidanceDecision = { decisionId: string; sessionId: string; projectId: string; kind: GuidanceKind; severity: GuidanceSeverity; finding: string; evidence: string[]; confidence: string; instruction: string; prohibitedActions: string[]; verification: string[]; evidenceFingerprint: string; expiresAt: string; createdAt: string };

export interface DashboardAPI {
  loadSummary(): Promise<Summary>;
  loadSnapshot(): Promise<Snapshot>;
  loadPrivacy(): Promise<PrivacySettings>;
  updatePrivacy(value: PrivacySettings): Promise<PrivacySettings>;
  loadSession(id: string): Promise<SessionEvidence>;
  loadConversations(): Promise<Conversation[]>;
  loadConversation(id: string): Promise<Conversation>;
  loadConnections(): Promise<ClientConnection[]>;
  loadLiveAnalysis(): Promise<LiveAnalysis>;
  loadActiveGuidance(): Promise<GuidanceDecision[]>;
  loadGuidanceControlLevel(projectId: string): Promise<{ controlLevel: ControlLevel }>;
  updateGuidanceControlLevel(projectId: string, controlLevel: ControlLevel): Promise<{ controlLevel: ControlLevel }>;
  deleteSession(id: string): Promise<void>;
}

export const sessionToken = () => document.querySelector<HTMLMetaElement>('meta[name="agent-doctor-token"]')?.content ?? '';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, { ...init, headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${sessionToken()}`, ...(init?.headers ?? {}) } });
  if (!response.ok) throw new Error(`Local API request failed (${response.status})`);
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export const localAPI: DashboardAPI = {
  async loadSummary() { return (await request<{ summary: Summary }>('/api/v1/dashboard/summary')).summary; },
  async loadSnapshot() { return (await request<{ snapshot: Snapshot }>('/api/v1/dashboard/snapshot')).snapshot; },
  loadPrivacy: () => request('/api/v1/settings/privacy'),
  updatePrivacy: (value) => request('/api/v1/settings/privacy', { method: 'PUT', body: JSON.stringify(value) }),
  loadSession: (id) => request(`/api/v1/sessions/${encodeURIComponent(id)}`),
  async loadConversations() { return (await request<{ items: Conversation[] }>('/api/v1/conversations?limit=50')).items; },
  loadConversation: (id) => request(`/api/v1/conversations/${encodeURIComponent(id)}`),
  async loadConnections() { return (await request<{ items: ClientConnection[] }>('/api/v1/connections')).items; },
  loadLiveAnalysis: () => request('/api/v1/analysis/live'),
  async loadActiveGuidance() { return (await request<{ items: GuidanceDecision[] }>('/api/v1/guidance/active')).items; },
  loadGuidanceControlLevel: (projectId) => request(`/api/v1/projects/${encodeURIComponent(projectId)}/guidance`),
  updateGuidanceControlLevel: (projectId, controlLevel) => request(`/api/v1/projects/${encodeURIComponent(projectId)}/guidance`, { method: 'PUT', body: JSON.stringify({ controlLevel }) }),
  deleteSession: (id) => request(`/api/v1/sessions/${encodeURIComponent(id)}`, { method: 'DELETE' }),
};
