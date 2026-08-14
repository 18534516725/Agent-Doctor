import type { ClientConnection, Conversation, DashboardAPI, LiveAnalysis, PrivacySettings, Snapshot, Summary } from '../api';
import type { Locale } from '../i18n';

export type PageProps = { locale: Locale; api: DashboardAPI; summary: Summary; snapshot: Snapshot; conversations: Conversation[]; selected?: Conversation; connections: ClientConnection[]; analysis: LiveAnalysis; privacy: PrivacySettings; setPrivacy(value: PrivacySettings): void; selectConversation(id: string): void; refresh(): void; };
