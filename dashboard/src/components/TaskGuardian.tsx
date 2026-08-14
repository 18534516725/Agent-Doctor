import { useState } from 'react';

import type { ControlLevel, GuidanceDecision } from '../api';
import type { Locale } from '../i18n';

type Props = {
  locale: Locale;
  guidance: GuidanceDecision[];
  controlLevel: ControlLevel;
  projectId: string;
  sessionLabel: string;
  clientLabel: string;
  onControlLevelChange(projectId: string, level: ControlLevel): void | Promise<void>;
};

const levelCopy: Record<ControlLevel, { zh: string; en: string }> = {
  observe: { zh: '只观察，不向 AI 提示', en: 'Observe without prompting the AI' },
  guide: { zh: '发现偏差时给出建议', en: 'Advise when drift is detected' },
  guard: { zh: '高风险动作可被拦截', en: 'Block supported high-risk actions' },
  autopilot: { zh: '自动执行最强保护策略', en: 'Apply the strongest supported controls' },
};

export function TaskGuardian({ locale, guidance, controlLevel, projectId, sessionLabel, clientLabel, onControlLevelChange }: Props) {
  const [showEvidence, setShowEvidence] = useState(false);
  const current = guidance[0];
  const zh = locale === 'zh';
  const state = guardianState(current?.kind, zh);

  return <section className={`task-guardian guardian-${current?.kind ?? 'quiet'}`} aria-labelledby="guardian-title">
    <div className="guardian-meta">
      <span>{zh ? '实时控制回路' : 'LIVE CONTROL LOOP'}</span>
      <div><b>{clientLabel}</b><i />{sessionLabel}<i />{projectId || (zh ? '等待项目证据' : 'Awaiting project evidence')}</div>
    </div>
    <div className="guardian-core">
      <div className="guardian-state" aria-label={zh ? '当前状态' : 'Current state'}><i /><span>{state}</span></div>
      <div className="guardian-message">
        <h2 id="guardian-title">{zh ? '任务守护中' : 'Task under guard'}</h2>
        {current ? <>
          <p className="guardian-finding">{current.finding}</p>
          <p className="guardian-instruction">{current.instruction}</p>
        </> : <>
          <p className="guardian-finding">{zh ? '当前没有需要介入的问题' : 'No issue currently needs intervention'}</p>
          <p className="guardian-instruction">{zh ? 'Agent Doctor 保持静默；任务可以按当前方向继续。' : 'Agent Doctor stays silent while the task continues on course.'}</p>
        </>}
      </div>
      <label className="guardian-control">
        <span>{zh ? '介入级别' : 'Control level'}</span>
        <select aria-label={zh ? '介入级别' : 'Control level'} value={controlLevel} disabled={!projectId} onChange={(event) => void onControlLevelChange(projectId, event.target.value as ControlLevel)}>
          <option value="observe">Observe</option><option value="guide">Guide</option><option value="guard">Guard</option><option value="autopilot">Autopilot</option>
        </select>
        <small>{levelCopy[controlLevel][locale]}</small>
      </label>
    </div>
    <div className="guardian-evidence-bar">
      <button aria-expanded={showEvidence} onClick={() => setShowEvidence((value) => !value)} disabled={!current}>
        {current ? (zh ? `${current.evidence.length} 条证据 · ${expiryLabel(current.expiresAt, locale)}` : `${current.evidence.length} evidence items · ${expiryLabel(current.expiresAt, locale)}`) : (zh ? '等待可验证证据' : 'Awaiting verifiable evidence')}
        <span>{showEvidence ? '−' : '+'}</span>
      </button>
      <span>{zh ? `最近介入 ${Math.min(guidance.length, 5)} 次` : `${Math.min(guidance.length, 5)} recent interventions`}</span>
    </div>
    <div className={`guardian-details ${showEvidence ? 'is-open' : ''}`} aria-hidden={!showEvidence}>
      <div>{current?.evidence.map((id, index) => <code key={id}>{String(index + 1).padStart(2, '0')} · {id}</code>)}</div>
      <dl>
        {current?.prohibitedActions.map((item) => <div key={item}><dt>{zh ? '避免' : 'Avoid'}</dt><dd>{item}</dd></div>)}
        {current?.verification.map((item) => <div key={item}><dt>{zh ? '验收' : 'Verify'}</dt><dd>{item}</dd></div>)}
      </dl>
    </div>
    {guidance.length > 1 && <ol className="guardian-history">{guidance.slice(1, 5).map((item) => <li key={item.decisionId}><span>{guardianState(item.kind, zh)}</span><b>{item.finding}</b><time>{new Date(item.createdAt).toLocaleTimeString(zh ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit' })}</time></li>)}</ol>}
  </section>;
}

function guardianState(kind: GuidanceDecision['kind'] | undefined, zh: boolean) {
  if (kind === 'redirect' || kind === 'advise' || kind === 'ask') return zh ? '需要纠偏' : 'REDIRECT';
  if (kind === 'verify') return zh ? '等待验收' : 'AWAITING VERIFICATION';
  if (kind === 'block') return zh ? '已阻断' : 'BLOCKED';
  return zh ? '正在推进' : 'ON TRACK';
}

function expiryLabel(value: string, locale: Locale) {
  if (!value) return locale === 'zh' ? '持续观察' : 'monitoring';
  const formatted = new Date(value).toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit' });
  return locale === 'zh' ? `${formatted} 前有效` : `valid until ${formatted}`;
}
