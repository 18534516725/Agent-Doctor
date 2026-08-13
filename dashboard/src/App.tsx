import { useEffect, useState } from 'react';

import { localAPI, type DashboardAPI, type PrivacySettings, type SafeEvent, type Snapshot, type Summary } from './api';

type Route = 'overview' | 'task' | 'costs' | 'memory' | 'comparison' | 'trends' | 'integrations' | 'privacy';
type Locale = 'en' | 'zh';
type PageCopy = { eyebrow: string; title: string; summary: string };
type InterfaceCopy = {
  apiUnavailable: string; evidenceUnavailable: string; settingsUnavailable: string;
  events(count: number): string; inspect(sessionID: string): string;
  unavailableZero: string; capturePrompts: string; captureFileContents: string;
  retentionDays: string; savePrivacy: string; saved: string;
  sessions(count: number): string;
};

const routeOrder: Route[] = ['overview', 'task', 'costs', 'memory', 'comparison', 'trends', 'integrations', 'privacy'];
const navigation = {
  en: { overview: 'Overview', task: 'Task evidence', costs: 'Costs', memory: 'Memory', comparison: 'Comparison', trends: 'Trends', integrations: 'Integrations', privacy: 'Privacy' },
  zh: { overview: '总览', task: '任务证据', costs: '费用', memory: '记忆', comparison: '对比', trends: '趋势', integrations: '集成', privacy: '隐私' },
} satisfies Record<Locale, Record<Route, string>>;
const content: Record<Locale, Record<Route, PageCopy>> = {
  en: {
    overview: { eyebrow: 'LOCAL DIAGNOSTIC DESK', title: 'Agent Doctor: what changed in this task?', summary: 'Your diagnostic history is stored on this device by default. Every number below keeps its evidence precision visible.' },
    task: { eyebrow: 'TASK EVIDENCE', title: 'Evidence before explanation.', summary: 'Inspect safe event metadata without importing prompts, source code or transcripts.' },
    costs: { eyebrow: 'COST LEDGER', title: 'Cost ledger', summary: 'Exact charges, estimates and unavailable values remain separate.' },
    memory: { eyebrow: 'PROJECT MEMORY', title: 'Useful context, not a transcript.', summary: 'Review active, candidate and disabled memory without displaying private content in the overview.' },
    comparison: { eyebrow: 'MATCHED COMPARISONS', title: 'Compare like with like.', summary: 'Comparisons require matched project, task type and major-version cohorts.' },
    trends: { eyebrow: 'PERSONAL BASELINE', title: 'A pattern needs enough history.', summary: 'Daily sessions and events are shown as observations, not a claim of causality.' },
    integrations: { eyebrow: 'LOCAL INTEGRATIONS', title: 'Connected at the lowest useful privilege.', summary: 'Each client uses only documented hooks, MCP, wrappers or extension interfaces.' },
    privacy: { eyebrow: 'YOUR LOCAL DATA', title: 'This device only.', summary: 'Choose retention and optional capture controls explicitly. Sensitive capture is off by default.' },
  },
  zh: {
    overview: { eyebrow: '本地诊断工作台', title: 'Agent Doctor：这次任务发生了什么？', summary: '诊断记录默认仅保存在此设备，下方每个数字都会保留证据精度。' },
    task: { eyebrow: '任务证据', title: '先看证据，再做判断。', summary: '只查看安全事件元数据，不导入提示词、源码或完整对话。' },
    costs: { eyebrow: '费用账本', title: '费用账本', summary: '已确认费用、估算费用和不可用数据分别展示。' },
    memory: { eyebrow: '项目记忆', title: '保留有用上下文，不保存整段对话。', summary: '查看已启用、候选和已停用记忆，概览不展示私密正文。' },
    comparison: { eyebrow: '匹配对比', title: '只比较真正可比的任务。', summary: '只使用相同项目、任务类型和主版本样本。' },
    trends: { eyebrow: '个人基线', title: '模式判断需要足够的历史。', summary: '按日展示会话与事件，只报告观察结果，不夸大因果。' },
    integrations: { eyebrow: '本地集成', title: '只请求实现目标所需的最小权限。', summary: '每种客户端仅使用公开 Hook、MCP、包装器或扩展接口。' },
    privacy: { eyebrow: '你的本地数据', title: '仅保存在此设备。', summary: '明确控制保留期和可选采集；敏感采集默认关闭。' },
  },
};
const interfaceCopy: Record<Locale, InterfaceCopy> = {
  en: {
    apiUnavailable: 'Local API unavailable', evidenceUnavailable: 'Evidence unavailable', settingsUnavailable: 'Settings unavailable',
    events: (count) => `${count} events`, inspect: (sessionID) => `Inspect ${sessionID}`,
    unavailableZero: 'Unavailable data is never shown as zero.', capturePrompts: 'Capture prompts',
    captureFileContents: 'Capture file contents', retentionDays: 'Retention days', savePrivacy: 'Save privacy settings',
    saved: 'Saved locally.', sessions: (count) => `${count} sessions`,
  },
  zh: {
    apiUnavailable: '本地 API 暂不可用', evidenceUnavailable: '任务证据暂不可用', settingsUnavailable: '隐私设置保存失败',
    events: (count) => `${count} 个事件`, inspect: (sessionID) => `查看 ${sessionID}`,
    unavailableZero: '数据缺失时绝不会显示成零。', capturePrompts: '采集提示词',
    captureFileContents: '采集文件内容', retentionDays: '保留天数', savePrivacy: '保存隐私设置',
    saved: '已保存到本地。', sessions: (count) => `${count} 个会话`,
  },
};

const emptySummary: Summary = { projects: 0, sessions: 0, activeSessions: 0, events: 0, precision: { exact: 0, estimated: 0, unavailable: 0 } };
const emptySnapshot: Snapshot = { sessions: [], costs: { currency: 'USD', exactMicros: 0, estimatedMicros: 0, unavailable: 0 }, memories: { active: 0, candidate: 0, disabled: 0 }, trends: [], comparisonCount: 0 };
const integrations = [
  ['Codex', 'MCP + wrapper'], ['Claude Code', 'Lifecycle hooks'], ['Cursor', 'MCP'], ['Cline', 'Hooks'],
  ['OpenCode', 'Plugin'], ['Windsurf', 'MCP'], ['Roo Code', 'Hooks'], ['Continue', 'Extension'],
  ['Aider', 'Safe wrapper'], ['Cherry Studio', 'MCP template'], ['Generic CLI', 'Safe wrapper'],
] as const;

export function App({ api = localAPI }: { api?: DashboardAPI }) {
  const [route, setRoute] = useState<Route>('overview');
  const [locale, setLocale] = useState<Locale>('en');
  const [summary, setSummary] = useState(emptySummary);
  const [snapshot, setSnapshot] = useState(emptySnapshot);
  const [privacy, setPrivacy] = useState<PrivacySettings>({ capturePrompts: false, captureFileContents: false, retentionDays: 30 });
  const [events, setEvents] = useState<SafeEvent[]>([]);
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);
  const labels = interfaceCopy[locale];

  const refresh = () => {
    setError('');
    Promise.all([api.loadSummary(), api.loadSnapshot(), api.loadPrivacy()])
      .then(([nextSummary, nextSnapshot, nextPrivacy]) => { setSummary(nextSummary); setSnapshot(nextSnapshot); setPrivacy(nextPrivacy); })
      .catch((reason: unknown) => setError(reason instanceof Error ? reason.message : labels.apiUnavailable));
  };
  useEffect(refresh, [api, labels.apiUnavailable]);

  const inspectSession = (id: string) => api.loadSession(id).then((result) => setEvents(result.events)).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : labels.evidenceUnavailable));
  const savePrivacy = () => api.updatePrivacy(privacy).then((next) => { setPrivacy(next); setSaved(true); }).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : labels.settingsUnavailable));
  const page = content[locale][route];

  return (
    <main className="doctor-shell">
      <header className="doctor-header">
        <a className="wordmark" href="#overview" aria-label={locale === 'en' ? 'Agent Doctor overview' : 'Agent Doctor 总览'} onClick={() => setRoute('overview')}>Agent <i>Doctor</i></a>
        <div className="header-actions">
          <button className="language-toggle" type="button" onClick={() => setLocale(locale === 'en' ? 'zh' : 'en')}>{locale === 'en' ? '中文' : 'English'}</button>
          <p className="local-status"><span aria-hidden="true" /> {locale === 'en' ? 'This device only' : '仅保存在此设备'}</p>
        </div>
      </header>
      <div className="doctor-layout">
        <nav className="doctor-nav" aria-label={locale === 'en' ? 'Diagnostic sections' : '诊断分区'}>
          {routeOrder.map((id, index) => <button key={id} aria-label={navigation[locale][id]} className={route === id ? 'is-active' : ''} onClick={() => setRoute(id)}><small>{String(index + 1).padStart(2, '0')}</small>{navigation[locale][id]}</button>)}
        </nav>
        <section className="doctor-stage" aria-live="polite" key={`${route}-${locale}`}>
          <p className="eyebrow">{page.eyebrow}</p><h1>{page.title}</h1><p className="lead">{page.summary}</p>
          {error && <p className="error-banner" role="alert">{error}</p>}
          <div className="reading-line" aria-hidden="true"><span /></div>
          <Page route={route} locale={locale} labels={labels} summary={summary} snapshot={snapshot} privacy={privacy} setPrivacy={setPrivacy} events={events} inspectSession={inspectSession} savePrivacy={savePrivacy} saved={saved} refresh={refresh} />
        </section>
      </div>
      <footer className="doctor-footer">Agent Doctor · {locale === 'en' ? 'local-first diagnostics for AI coding work' : '面向 AI 编程工作的本地优先诊断'} <span>{locale === 'en' ? 'Evidence is not a verdict.' : '证据不是武断结论。'}</span></footer>
    </main>
  );
}

type PageProps = { route: Route; locale: Locale; labels: InterfaceCopy; summary: Summary; snapshot: Snapshot; privacy: PrivacySettings; setPrivacy(value: PrivacySettings): void; events: SafeEvent[]; inspectSession(id: string): void; savePrivacy(): void; saved: boolean; refresh(): void };
function Page(props: PageProps) {
  const { route, locale, labels, summary, snapshot } = props;
  if (route === 'overview') return <><div className="metric-grid"><Metric value={summary.events} label={locale === 'en' ? 'events' : '事件'} /><Metric value={summary.sessions} label={`${locale === 'en' ? 'sessions' : '会话'} · ${summary.activeSessions} ${locale === 'en' ? 'active' : '进行中'}`} /><Metric value={summary.projects} label={locale === 'en' ? 'projects' : '项目'} /><Metric value={summary.precision.exact} label={locale === 'en' ? 'exact evidence' : '精确证据'} /></div><button className="action-button" onClick={props.refresh}>{locale === 'en' ? 'Refresh local evidence' : '刷新本地证据'} ↻</button></>;
  if (route === 'task') return <div className="data-stack">{snapshot.sessions.length === 0 ? <Empty text={locale === 'en' ? 'No sessions recorded yet.' : '尚无会话记录。'} labels={labels} /> : snapshot.sessions.map((session) => <article className="data-row" key={session.id}><div><strong>{session.id}</strong><span>{session.client} · {session.model} · {labels.events(session.eventCount)}</span></div><button onClick={() => props.inspectSession(session.id)} aria-label={labels.inspect(session.id)}>{locale === 'en' ? 'Inspect' : '查看'} ↗</button></article>)}{props.events.map((event) => <article className="timeline-row" key={event.eventId}><strong>{event.eventType}</strong><span>{event.precision} · {event.provenance}</span><time>{event.timestamp}</time></article>)}</div>;
  if (route === 'costs') return <div className="metric-grid"><Metric value={money(snapshot.costs.currency, snapshot.costs.exactMicros)} label={locale === 'en' ? 'exact charged' : '精确费用'} /><Metric value={money(snapshot.costs.currency, snapshot.costs.estimatedMicros)} label={locale === 'en' ? 'estimated' : '估算费用'} /><Metric value={snapshot.costs.unavailable} label={locale === 'en' ? 'unavailable records' : '不可用记录'} /></div>;
  if (route === 'memory') return <div className="metric-grid"><Metric value={snapshot.memories.active} label={locale === 'en' ? 'active' : '已启用'} join /><Metric value={snapshot.memories.candidate} label={locale === 'en' ? 'candidate' : '候选'} join /><Metric value={snapshot.memories.disabled} label={locale === 'en' ? 'disabled' : '已停用'} join /></div>;
  if (route === 'comparison') return snapshot.comparisonCount === 0 ? <Empty text={locale === 'en' ? 'No matched comparison yet. Collect at least 15 comparable tasks per cohort.' : '尚无匹配对比。每个样本组至少需要 15 个可比任务。'} labels={labels} /> : <Metric value={snapshot.comparisonCount} label={locale === 'en' ? 'saved comparisons' : '已保存对比'} />;
  if (route === 'trends') return <div className="data-stack">{snapshot.trends.length === 0 ? <Empty text={locale === 'en' ? 'No baseline yet.' : '尚未形成基线。'} labels={labels} /> : snapshot.trends.map((point) => <article className="trend-row" key={point.date}><time>{point.date}</time><div><span style={{ width: `${Math.min(100, point.events * 5)}%` }} /></div><strong>{labels.sessions(point.sessions)} · {labels.events(point.events)}</strong></article>)}</div>;
  if (route === 'integrations') return <div className="integration-grid">{integrations.map(([client, method]) => <article key={client}><span className="status-dot" /> <strong>{client}</strong><small>{method}</small></article>)}</div>;
  return <form className="privacy-form" onSubmit={(event) => { event.preventDefault(); props.savePrivacy(); }}><label><input aria-label={labels.capturePrompts} type="checkbox" checked={props.privacy.capturePrompts} onChange={(event) => props.setPrivacy({ ...props.privacy, capturePrompts: event.target.checked })} /> {labels.capturePrompts}</label><label><input aria-label={labels.captureFileContents} type="checkbox" checked={props.privacy.captureFileContents} onChange={(event) => props.setPrivacy({ ...props.privacy, captureFileContents: event.target.checked })} /> {labels.captureFileContents}</label><label>{labels.retentionDays}<input aria-label={labels.retentionDays} type="number" min="1" max="3650" value={props.privacy.retentionDays} onChange={(event) => props.setPrivacy({ ...props.privacy, retentionDays: Number(event.target.value) })} /></label><button className="action-button" type="submit">{labels.savePrivacy}</button>{props.saved && <span role="status">{labels.saved}</span>}</form>;
}

function Metric({ value, label, join = false }: { value: string | number; label: string; join?: boolean }) { return <article className="metric-card"><strong>{value}{join ? ` ${label}` : ''}</strong>{!join && <span>{label}</span>}</article>; }
function Empty({ text, labels }: { text: string; labels: InterfaceCopy }) { return <div className="empty-state"><span>○</span><strong>{text}</strong><p>{labels.unavailableZero}</p></div>; }
function money(currency: string, micros: number) { return `${currency} ${(micros / 1_000_000).toFixed(2)}`; }
