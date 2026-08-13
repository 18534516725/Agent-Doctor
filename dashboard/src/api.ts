export type PrecisionCounts = { exact: number; estimated: number; unavailable: number };
export type Summary = { projects: number; sessions: number; activeSessions: number; events: number; precision: PrecisionCounts };
export type Session = { id: string; client: string; model: string; status: string; startedAt: string; eventCount: number };
export type Costs = { currency: string; exactMicros: number; estimatedMicros: number; unavailable: number };
export type Memories = { active: number; candidate: number; disabled: number };
export type TrendPoint = { date: string; sessions: number; events: number };
export type Snapshot = { sessions: Session[]; costs: Costs; memories: Memories; trends: TrendPoint[]; comparisonCount: number };
export type PrivacySettings = { capturePrompts: boolean; captureFileContents: boolean; retentionDays: number };
export type SafeEvent = { eventId: string; timestamp: string; eventType: string; provenance: string; precision: 'exact' | 'estimated' | 'unavailable' };
export type SessionEvidence = { sessionId: string; events: SafeEvent[] };

export interface DashboardAPI {
  loadSummary(): Promise<Summary>;
  loadSnapshot(): Promise<Snapshot>;
  loadPrivacy(): Promise<PrivacySettings>;
  updatePrivacy(value: PrivacySettings): Promise<PrivacySettings>;
  loadSession(id: string): Promise<SessionEvidence>;
}

const token = () => document.querySelector<HTMLMetaElement>('meta[name="agent-doctor-token"]')?.content ?? '';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token()}`, ...(init?.headers ?? {}) },
  });
  if (!response.ok) throw new Error(`Local API request failed (${response.status})`);
  return response.json() as Promise<T>;
}

export const localAPI: DashboardAPI = {
  async loadSummary() { return (await request<{ summary: Summary }>('/api/v1/dashboard/summary')).summary; },
  async loadSnapshot() { return (await request<{ snapshot: Snapshot }>('/api/v1/dashboard/snapshot')).snapshot; },
  loadPrivacy() { return request<PrivacySettings>('/api/v1/settings/privacy'); },
  updatePrivacy(value) { return request<PrivacySettings>('/api/v1/settings/privacy', { method: 'PUT', body: JSON.stringify(value) }); },
  loadSession(id) { return request<SessionEvidence>(`/api/v1/sessions/${encodeURIComponent(id)}`); },
};
