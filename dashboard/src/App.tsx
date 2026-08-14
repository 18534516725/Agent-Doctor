import { useCallback, useEffect, useMemo, useState } from 'react';

import { localAPI, type ClientConnection, type ControlLevel, type Conversation, type DashboardAPI, type GuidanceDecision, type GuidanceStatus, type LiveAnalysis, type PrivacySettings, type Snapshot, type Summary } from './api';
import { copy, initialLocale, persistLocale, routeOrder, type Locale, type Route } from './i18n';
import { useLiveUpdates, type EventSourceFactory } from './live';
import { ComparisonPage } from './pages/ComparisonPage';
import { CostsPage } from './pages/CostsPage';
import { EvidencePage } from './pages/EvidencePage';
import { IntegrationsPage } from './pages/IntegrationsPage';
import { MemoryPage } from './pages/MemoryPage';
import { OverviewPage } from './pages/OverviewPage';
import { PrivacyPage } from './pages/PrivacyPage';
import { TrendsPage } from './pages/TrendsPage';
import type { PageProps } from './pages/types';

const emptySummary: Summary = { projects: 0, sessions: 0, activeSessions: 0, events: 0, precision: { exact: 0, estimated: 0, unavailable: 0 } };
const emptySnapshot: Snapshot = { sessions: [], costs: { currency: 'USD', exactMicros: 0, estimatedMicros: 0, unavailable: 0 }, memories: { active: 0, candidate: 0, disabled: 0 }, trends: [], comparisonCount: 0 };
const emptyAnalysis: LiveAnalysis = { projectId: '', requests: 0, activeSessions: 0, inputTokens: 0, outputTokens: 0, cachedTokens: 0, reasoningTokens: 0, exactCostMicros: 0, estimatedCostMicros: 0, unknownCostCount: 0, averageLatencyMs: 0, failedRequests: 0, toolCalls: 0, tokensPerRequest: 0, cacheHitRate: 0, healthScore: 0, summary: '', findings: [], limitations: [] };
const clientCatalog = [['codex', 'Codex'], ['claude-code', 'Claude Code'], ['cursor', 'Cursor'], ['windsurf', 'Windsurf'], ['cline', 'Cline'], ['roo-code', 'Roo Code'], ['continue', 'Continue'], ['opencode', 'OpenCode'], ['aider', 'Aider'], ['cherry-studio', 'Cherry Studio'], ['generic-cli', 'Generic CLI']] as const;

export function App({ api = localAPI, liveFactory }: { api?: DashboardAPI; liveFactory?: EventSourceFactory }) {
  const [route, setRoute] = useState<Route>('overview');
  const [locale, setLocale] = useState<Locale>(initialLocale);
  const [summary, setSummary] = useState(emptySummary);
  const [snapshot, setSnapshot] = useState(emptySnapshot);
  const [analysis, setAnalysis] = useState(emptyAnalysis);
  const [guidance, setGuidance] = useState<GuidanceDecision[]>([]);
  const [guidanceStatus, setGuidanceStatus] = useState<GuidanceStatus>({ state: 'unavailable', client: '', advice: false, enforcement: false, explanation: '' });
  const [controlLevel, setControlLevel] = useState<ControlLevel>('guide');
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [connections, setConnections] = useState<ClientConnection[]>([]);
  const [privacy, setPrivacy] = useState<PrivacySettings>({ capturePrompts: true, captureFileContents: false, retentionDays: 30 });
  const [selectedID, setSelectedID] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const labels = copy[locale];

  const refresh = useCallback(() => {
    setError('');
    return Promise.all([api.loadSummary(), api.loadSnapshot(), api.loadConversations(), api.loadConnections(), api.loadLiveAnalysis(), api.loadPrivacy(), api.loadActiveGuidance()])
      .then(async ([nextSummary, nextSnapshot, nextConversations, nextConnections, nextAnalysis, nextPrivacy, nextGuidance]) => {
        setSummary(nextSummary); setSnapshot(nextSnapshot); setConversations(nextConversations); setConnections(nextConnections); setAnalysis(nextAnalysis); setPrivacy(nextPrivacy); setGuidance(nextGuidance.items); setGuidanceStatus(nextGuidance.status);
        setSelectedID((current) => current && nextConversations.some((item) => item.id === current) ? current : (nextConversations[0]?.id ?? ''));
        const projectId = nextGuidance.items[0]?.projectId || nextAnalysis.projectId || nextConversations[0]?.projectId || '';
        if (projectId) setControlLevel((await api.loadGuidanceControlLevel(projectId)).controlLevel);
      })
      .catch((reason: unknown) => setError(reason instanceof Error ? reason.message : (locale === 'zh' ? '本地服务暂不可用' : 'Local service unavailable')))
      .finally(() => setLoading(false));
  }, [api, locale]);
  useEffect(() => { void refresh(); }, [refresh]);
  const liveState = useLiveUpdates(refresh, liveFactory);
  const selected = conversations.find((item) => item.id === selectedID) ?? conversations[0];
  const mergedConnections = useMemo(() => clientCatalog.map(([key, displayName]) => connections.find((item) => item.key === key) ?? { key, displayName, detected: false, state: 'unavailable' as const, capability: '', detail: '', updatedAt: '' }), [connections]);
  const updateGuidanceControlLevel = (projectId: string, level: ControlLevel) => {
    if (!projectId) return;
    const previous = controlLevel;
    setControlLevel(level);
    void api.updateGuidanceControlLevel(projectId, level).catch((reason: unknown) => {
      setControlLevel(previous);
      setError(reason instanceof Error ? reason.message : (locale === 'zh' ? '介入级别更新失败' : 'Control level update failed'));
    });
  };
  const pageProps: PageProps = { locale, api, summary, snapshot, conversations, selected, connections: mergedConnections, analysis, guidance, guidanceStatus, controlLevel, privacy, setPrivacy, setGuidanceControlLevel: updateGuidanceControlLevel, selectConversation: setSelectedID, refresh: () => { void refresh(); } };
  const switchLocale = () => { const next = locale === 'zh' ? 'en' : 'zh'; setLocale(next); persistLocale(next); };

  return <main className="doctor-shell">
    <header className="doctor-header"><button className="wordmark" onClick={() => setRoute('overview')}><i>AD</i><span>Agent Doctor<small>{locale === 'zh' ? '本地 AI 协作观测台' : 'Local AI observatory'}</small></span></button><div className="header-actions"><div className={`live-pill state-${liveState}`}><i />{liveState === 'connected' ? labels.listening : labels.offline}</div><button className="language-toggle" onClick={switchLocale}>{labels.language}</button></div></header>
    <div className="doctor-layout"><nav className="doctor-nav" aria-label={locale === 'zh' ? '功能导航' : 'Navigation'}>{routeOrder.map((id, index) => <button key={id} className={route === id ? 'is-active' : ''} onClick={() => setRoute(id)} aria-label={labels.nav[id]}><small>{String(index + 1).padStart(2, '0')}</small><span>{labels.nav[id]}</span><i>↗</i></button>)}</nav>
      <section className="doctor-stage"><div className={`stage-intro ${route === 'overview' ? 'stage-intro-overview' : ''}`}><div><p className="eyebrow">{labels.eyebrow}</p>{route !== 'overview' && <h1>{labels.nav[route]}</h1>}<p>{route === 'overview' ? labels.lead : sectionLead(route, locale)}</p></div><div className="local-seal"><i />{labels.local}<small>SQLite v7</small></div></div>
        {error && <div className="error-banner" role="alert"><strong>{locale === 'zh' ? '连接失败' : 'Connection failed'}</strong><span>{error}</span><button onClick={() => void refresh()}>{locale === 'zh' ? '重试' : 'Retry'}</button></div>}
        {loading ? <div className="loading-panel"><i /><strong>{locale === 'zh' ? '正在读取本地数据…' : 'Loading local data…'}</strong></div> : <div className="page-enter" key={route}>{renderPage(route, pageProps)}</div>}
      </section></div>
    <footer className="doctor-footer"><strong>Agent Doctor</strong><span>{locale === 'zh' ? '完整证据，清晰边界，本地保存。' : 'Complete evidence. Clear boundaries. Local storage.'}</span><small>v1.1 · {new Date().getFullYear()}</small></footer>
  </main>;
}

function renderPage(route: Route, props: PageProps) { switch (route) { case 'overview': return <OverviewPage {...props} />; case 'task': return <EvidencePage {...props} />; case 'costs': return <CostsPage {...props} />; case 'memory': return <MemoryPage {...props} />; case 'comparison': return <ComparisonPage {...props} />; case 'trends': return <TrendsPage {...props} />; case 'integrations': return <IntegrationsPage {...props} />; case 'privacy': return <PrivacyPage {...props} />; } }
function sectionLead(route: Route, locale: Locale) { const zh: Record<Route, string> = { overview: '', task: '原始消息只作为诊断证据，按需展开查看。', costs: 'Token、延迟和金额分开标注精确、估算与未知。', memory: '管理可复用项目上下文，保留来源与状态。', comparison: '只对比真正可比的任务，拒绝虚假结论。', trends: '用真实历史观察响应速度、调用量与工作节奏。', integrations: '查看每个客户端是否已发现、已连接或正在活动。', privacy: '完整对话保存在本机 SQLite；认证凭证永不写入数据库。' }; const en: Record<Route, string> = { overview: '', task: 'Raw messages remain available as on-demand diagnostic evidence.', costs: 'Tokens, latency and amount keep exact, estimated and unknown evidence separate.', memory: 'Manage reusable project context with provenance and state.', comparison: 'Compare only genuinely matched work.', trends: 'Observe real latency, volume and work rhythms.', integrations: 'See whether each client is detected, connected or active.', privacy: 'Full conversations stay in local SQLite; transport credentials are never persisted.' }; return (locale === 'zh' ? zh : en)[route]; }
