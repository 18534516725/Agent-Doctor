import { useState } from 'react';

type Route = 'overview' | 'task' | 'costs' | 'memory' | 'comparison' | 'trends' | 'integrations' | 'privacy';
type Locale = 'en' | 'zh';
type PageCopy = { eyebrow: string; title: string; summary: string; action: string };

const routeOrder: Route[] = [
	'overview', 'task', 'costs', 'memory', 'comparison', 'trends', 'integrations', 'privacy',
];

const navigation: Record<Locale, Record<Route, string>> = {
  en: {
    overview: 'Overview', task: 'Task evidence', costs: 'Costs', memory: 'Memory', comparison: 'Comparison', trends: 'Trends', integrations: 'Integrations', privacy: 'Privacy',
  },
  zh: {
    overview: '总览', task: '任务证据', costs: '费用', memory: '记忆', comparison: '对比', trends: '趋势', integrations: '集成', privacy: '隐私',
  },
};

const content: Record<Locale, Record<Route, PageCopy>> = {
en: {
  overview: { eyebrow: 'LOCAL DIAGNOSTIC DESK', title: 'Agent Doctor: what changed in this task?', summary: 'Your diagnostic history is stored on this device by default. Costs, timing, validation and memory remain clearly separated by confidence.', action: 'Open latest task evidence' },
  task: { eyebrow: 'TASK EVIDENCE', title: 'Evidence before explanation.', summary: 'Inspect Git snapshots, approved validations, client events and counter-evidence without importing a transcript.', action: 'Inspect evidence timeline' },
  costs: { eyebrow: 'COST LEDGER', title: 'Cost ledger', summary: 'Exact charged amounts, local estimates and unavailable values stay separate. No hidden conversion or false zero.', action: 'Review budget policy' },
  memory: { eyebrow: 'PROJECT MEMORY', title: 'Useful context, not a transcript.', summary: 'Memory capsules show whether each fact is user-supplied, repository-derived or inferred — and can be removed locally.', action: 'Review memory sources' },
  comparison: { eyebrow: 'MATCHED COMPARISONS', title: 'Compare like with like.', summary: 'Client and model comparisons only use matched project, task type and version cohorts. Small samples remain cautionary.', action: 'View cohort limits' },
  trends: { eyebrow: 'PERSONAL BASELINE', title: 'A pattern needs enough history.', summary: 'Medians and P90 trends start only after comparable samples exist. Before that, the dashboard reports facts instead of pretending certainty.', action: 'See sample requirements' },
  integrations: { eyebrow: 'LOCAL INTEGRATIONS', title: 'Connected at the lowest useful privilege.', summary: 'Codex, Claude Code, Cursor and other clients can supply only what their documented interfaces actually expose.', action: 'Review capability matrix' },
  privacy: { eyebrow: 'YOUR LOCAL DATA', title: 'This device only.', summary: 'Diagnostic history is local by default. Export and deletion controls are explicit, and raw prompts, source code and credentials are not required.', action: 'Open data controls' },
},
zh: {
  overview: { eyebrow: '本地诊断工作台', title: 'Agent Doctor：这次任务发生了什么？', summary: '诊断记录默认仅保存在此设备。费用、耗时、验证和记忆会按证据可信度分别展示。', action: '查看最近任务证据' },
  task: { eyebrow: '任务证据', title: '先看证据，再做判断。', summary: '查看 Git 快照、已批准的验证、客户端事件和反证；不需要导入完整对话。', action: '查看证据时间线' },
  costs: { eyebrow: '费用账本', title: '费用账本', summary: '已确认费用、本地估算和不可用数据分别展示，不会隐藏换算，也不会用零伪造缺失数据。', action: '查看预算策略' },
  memory: { eyebrow: '项目记忆', title: '保留有用上下文，不保存整段对话。', summary: '每条记忆都会标记为用户提供、仓库提取或推断结果，并可在本机随时删除。', action: '查看记忆来源' },
  comparison: { eyebrow: '匹配对比', title: '只比较真正可比的任务。', summary: '客户端与模型对比只使用相同项目、任务类型和主版本的样本；样本过少时会明确提示。', action: '查看分组限制' },
  trends: { eyebrow: '个人基线', title: '模式判断需要足够的历史。', summary: '只有积累到可比样本后才展示中位数与 P90 趋势；此前只报告事实，不假装确定。', action: '查看样本要求' },
  integrations: { eyebrow: '本地集成', title: '只请求实现目标所需的最小权限。', summary: 'Codex、Claude Code、Cursor 等客户端只会提交其公开接口实际能够提供的数据。', action: '查看能力矩阵' },
  privacy: { eyebrow: '你的本地数据', title: '仅保存在此设备。', summary: '诊断记录默认本地保存，导出和删除入口清晰可见；不需要原始提示词、源码或凭证。', action: '打开数据控制' },
},
};

const signals: Record<Locale, { local: string; nav: string; known: string; visible: string; caution: string; unavailable: string; confidence: string; footer: string; limitation: string }> = {
  en: { local: 'This device only', nav: 'Diagnostic sections', known: 'Known now', visible: 'Provenance is visible', caution: 'Needs care', unavailable: 'Unavailable ≠ zero', confidence: 'Confidence stays explicit', footer: 'Agent Doctor · local-first diagnostics for AI coding work', limitation: 'Evidence is not a verdict.' },
  zh: { local: '仅保存在此设备', nav: '诊断分区', known: '当前已知', visible: '证据来源清晰可见', caution: '需要注意', unavailable: '不可用不等于零', confidence: '可信度会被明确说明', footer: 'Agent Doctor · 面向 AI 编程工作的本地优先诊断', limitation: '证据不是武断结论。' },
};

const specialSignals: Record<Locale, Record<Route, string>> = {
  en: { overview: 'Local evidence', task: 'Timeline ready', costs: 'Exact + estimated', memory: 'Source-labelled memory', comparison: 'Matched cohorts', trends: 'Sample-aware baseline', integrations: 'Verified capabilities', privacy: 'Local controls' },
  zh: { overview: '本地证据', task: '可追溯时间线', costs: '已确认 + 估算', memory: '带来源标记的记忆', comparison: '匹配样本组', trends: '样本感知基线', integrations: '已验证能力', privacy: '本地控制' },
};

export function App() {
  const [route, setRoute] = useState<Route>('overview');
  const [locale, setLocale] = useState<Locale>('en');
  const page = content[locale][route];
  const copy = signals[locale];
  return (
    <main className="doctor-shell">
      <header className="doctor-header">
        <a className="wordmark" href="#overview" aria-label="Agent Doctor overview" onClick={() => setRoute('overview')}>Agent <i>Doctor</i></a>
        <div className="header-actions">
          <button className="language-toggle" type="button" onClick={() => setLocale(locale === 'en' ? 'zh' : 'en')}>{locale === 'en' ? '中文' : 'English'}</button>
          <p className="local-status"><span aria-hidden="true" /> {copy.local}</p>
        </div>
      </header>
      <div className="doctor-layout">
        <nav className="doctor-nav" aria-label={copy.nav}>
          {routeOrder.map((id, index) => (
            <button key={id} aria-label={navigation[locale][id]} className={route === id ? 'is-active' : ''} onClick={() => setRoute(id)}>
              <small>{String(index + 1).padStart(2, '0')}</small>{navigation[locale][id]}
            </button>
          ))}
        </nav>
        <section className="doctor-stage" aria-live="polite">
          <p className="eyebrow">{page.eyebrow}</p>
          <h1>{page.title}</h1>
          <p className="lead">{page.summary}</p>
          <div className="reading-line" aria-hidden="true"><span /></div>
          <div className="evidence-grid">
            <article className="signal-panel"><p>{copy.known}</p><strong>{specialSignals[locale][route]}</strong><span>{copy.visible}</span></article>
            <article className="signal-panel muted"><p>{copy.caution}</p><strong>{route === 'trends' ? (locale === 'en' ? 'Sample size' : '样本数量') : copy.unavailable}</strong><span>{copy.confidence}</span></article>
          </div>
          <button className="action-button" type="button">{page.action} <span aria-hidden="true">↗</span></button>
        </section>
      </div>
      <footer className="doctor-footer">{copy.footer} <span>{copy.limitation}</span></footer>
    </main>
  );
}
