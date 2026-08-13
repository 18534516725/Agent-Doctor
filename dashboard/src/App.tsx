import { useState } from 'react';

type Route = 'overview' | 'task' | 'costs' | 'memory' | 'comparison' | 'trends' | 'integrations' | 'privacy';

const navigation: Array<{ id: Route; label: string }> = [
  { id: 'overview', label: 'Overview' }, { id: 'task', label: 'Task evidence' }, { id: 'costs', label: 'Costs' }, { id: 'memory', label: 'Memory' },
  { id: 'comparison', label: 'Comparison' }, { id: 'trends', label: 'Trends' }, { id: 'integrations', label: 'Integrations' }, { id: 'privacy', label: 'Privacy' },
];

const content: Record<Route, { eyebrow: string; title: string; summary: string; action: string }> = {
  overview: { eyebrow: 'LOCAL DIAGNOSTIC DESK', title: 'Agent Doctor: what changed in this task?', summary: 'Your diagnostic history is stored on this device by default. Costs, timing, validation and memory remain clearly separated by confidence.', action: 'Open latest task evidence' },
  task: { eyebrow: 'TASK EVIDENCE', title: 'Evidence before explanation.', summary: 'Inspect Git snapshots, approved validations, client events and counter-evidence without importing a transcript.', action: 'Inspect evidence timeline' },
  costs: { eyebrow: 'COST LEDGER', title: 'Cost ledger', summary: 'Exact charged amounts, local estimates and unavailable values stay separate. No hidden conversion or false zero.', action: 'Review budget policy' },
  memory: { eyebrow: 'PROJECT MEMORY', title: 'Useful context, not a transcript.', summary: 'Memory capsules show whether each fact is user-supplied, repository-derived or inferred — and can be removed locally.', action: 'Review memory sources' },
  comparison: { eyebrow: 'MATCHED COMPARISONS', title: 'Compare like with like.', summary: 'Client and model comparisons only use matched project, task type and version cohorts. Small samples remain cautionary.', action: 'View cohort limits' },
  trends: { eyebrow: 'PERSONAL BASELINE', title: 'A pattern needs enough history.', summary: 'Medians and P90 trends start only after comparable samples exist. Before that, the dashboard reports facts instead of pretending certainty.', action: 'See sample requirements' },
  integrations: { eyebrow: 'LOCAL INTEGRATIONS', title: 'Connected at the lowest useful privilege.', summary: 'Codex, Claude Code, Cursor and other clients can supply only what their documented interfaces actually expose.', action: 'Review capability matrix' },
  privacy: { eyebrow: 'YOUR LOCAL DATA', title: 'This device only.', summary: 'Diagnostic history is local by default. Export and deletion controls are explicit, and raw prompts, source code and credentials are not required.', action: 'Open data controls' },
};

export function App() {
  const [route, setRoute] = useState<Route>('overview');
  const page = content[route];
  return (
    <main className="doctor-shell">
      <header className="doctor-header">
        <a className="wordmark" href="#overview" aria-label="Agent Doctor overview" onClick={() => setRoute('overview')}>Agent <i>Doctor</i></a>
        <p className="local-status"><span aria-hidden="true" /> This device only</p>
      </header>
      <div className="doctor-layout">
        <nav className="doctor-nav" aria-label="Diagnostic sections">
          {navigation.map((item, index) => (
            <button key={item.id} aria-label={item.label} className={route === item.id ? 'is-active' : ''} onClick={() => setRoute(item.id)}>
              <small>{String(index + 1).padStart(2, '0')}</small>{item.label}
            </button>
          ))}
        </nav>
        <section className="doctor-stage" aria-live="polite">
          <p className="eyebrow">{page.eyebrow}</p>
          <h1>{page.title}</h1>
          <p className="lead">{page.summary}</p>
          <div className="reading-line" aria-hidden="true"><span /></div>
          <div className="evidence-grid">
            <article className="signal-panel"><p>Known now</p><strong>{route === 'costs' ? 'Exact + estimated' : 'Local evidence'}</strong><span>Provenance is visible</span></article>
            <article className="signal-panel muted"><p>Needs care</p><strong>{route === 'trends' ? 'Sample size' : 'Unavailable ≠ zero'}</strong><span>Confidence stays explicit</span></article>
          </div>
          <button className="action-button" type="button">{page.action} <span aria-hidden="true">↗</span></button>
        </section>
      </div>
      <footer className="doctor-footer">Agent Doctor · local-first diagnostics for AI coding work <span>Evidence is not a verdict.</span></footer>
    </main>
  );
}
