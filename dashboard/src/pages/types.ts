import type { ClientConnection, ControlLevel, Conversation, DashboardAPI, GuidanceDecision, LiveAnalysis, PrivacySettings, Snapshot, Summary } from '../api';
import type { Locale } from '../i18n';

export type PageProps = { locale: Locale; api: DashboardAPI; summary: Summary; snapshot: Snapshot; conversations: Conversation[]; selected?: Conversation; connections: ClientConnection[]; analysis: LiveAnalysis; guidance: GuidanceDecision[]; controlLevel: ControlLevel; privacy: PrivacySettings; setPrivacy(value: PrivacySettings): void; setGuidanceControlLevel(projectId: string, level: ControlLevel): void; selectConversation(id: string): void; refresh(): void; };
