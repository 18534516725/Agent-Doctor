import { useState } from 'react';

import type { ControlLevel, GuidanceDecision, GuidanceDelivery, GuidanceStatus } from '../api';
import type { Locale } from '../i18n';

type Props = {
  locale: Locale;
  guidance: GuidanceDecision[];
  delivery: GuidanceDelivery | null;
  status: GuidanceStatus;
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

export function TaskGuardian({ locale, guidance, delivery, status, controlLevel, projectId, sessionLabel, clientLabel, onControlLevelChange }: Props) {
  const [showEvidence, setShowEvidence] = useState(false);
  const current = guidance[0];
  const zh = locale === 'zh';
  const state = guardianState(current?.kind, status.state, zh);
  const deliveryMatchesTask = Boolean(delivery && delivery.projectId === projectId && (!current || delivery.decisionId === current.decisionId));
  const quiet = quietState(status.state, zh, deliveryMatchesTask ? delivery : null);
  const canEnforce = status.enforcement;
  const effectiveLevel: ControlLevel = !canEnforce && (controlLevel === 'guard' || controlLevel === 'autopilot') ? 'guide' : controlLevel;
  const heading = current
    ? (deliveryMatchesTask ? (zh ? '纠偏指令已送达 AI' : 'Guidance delivered to the AI') : (zh ? '等待 AI 读取纠偏指令' : 'Waiting for the AI to fetch guidance'))
    : quiet.heading;

  return <section className={`task-guardian guardian-${current?.kind ?? 'quiet'}`} aria-labelledby="guardian-title">
    <div className="guardian-meta">
      <span>{zh ? '实时控制回路' : 'LIVE CONTROL LOOP'}</span>
      <div><b>{clientLabel}</b><i />{sessionLabel}<i />{projectId || (zh ? '等待项目证据' : 'Awaiting project evidence')}</div>
    </div>
    <div className="guardian-core">
      <div className="guardian-state" aria-label={zh ? '当前状态' : 'Current state'}><i /><span>{state}</span></div>
      <div className="guardian-message">
        <h2 id="guardian-title">{heading}</h2>
        {current ? <>
          <p className="guardian-finding">{current.finding}</p>
          <p className="guardian-instruction">{current.instruction}</p>
        </> : <>
          <p className="guardian-finding">{quiet.finding}</p>
          <p className="guardian-instruction">{quiet.instruction}</p>
        </>}
      </div>
      <label className="guardian-control">
        <span>{zh ? '介入级别' : 'Control level'}</span>
        <select aria-label={zh ? '介入级别' : 'Control level'} value={effectiveLevel} disabled={!projectId} onChange={(event) => void onControlLevelChange(projectId, event.target.value as ControlLevel)}>
          <option value="observe">{zh ? '只观察' : 'Observe only'}</option><option value="guide">{zh ? '自动指导' : 'Automatic guidance'}</option>
          {canEnforce && <><option value="guard">{zh ? '风险拦截（Claude）' : 'Risk guard (Claude)'}</option><option value="autopilot">{zh ? '自动保护（Claude）' : 'Automatic protection (Claude)'}</option></>}
        </select>
        <small>{levelCopy[effectiveLevel][locale]}</small>
        {!canEnforce && controlLevel !== effectiveLevel && <small>{zh ? `已保存的“${levelCopy[controlLevel].zh}”在 Codex 中按“自动指导”执行，无法强制拦截。` : `The saved ${controlLevel} level operates as guidance in Codex; MCP cannot enforce blocking.`}</small>}
        <small>{capabilityLabel(status, zh)}</small>
      </label>
    </div>
    <div className="guardian-evidence-bar">
      <button aria-expanded={showEvidence} onClick={() => setShowEvidence((value) => !value)} disabled={!current}>
        {current ? (zh ? `${current.evidence.length} 条证据 · ${expiryLabel(current.expiresAt, locale)}` : `${current.evidence.length} evidence items · ${expiryLabel(current.expiresAt, locale)}`) : (zh ? '等待可验证证据' : 'Awaiting verifiable evidence')}
        <span>{showEvidence ? '−' : '+'}</span>
      </button>
      <span>{deliveryMatchesTask && delivery ? (zh ? `AI 已读取 ${delivery.deliveryCount} 次` : `Read by AI ${delivery.deliveryCount} times`) : (zh ? 'AI 尚未读取本轮指导' : 'Not fetched by the AI yet')}</span>
      <span>{zh ? `实际纠偏 ${Math.min(guidance.length, 5)} 次` : `${Math.min(guidance.length, 5)} actual interventions`}</span>
    </div>
    <div className={`guardian-details ${showEvidence ? 'is-open' : ''}`} aria-hidden={!showEvidence}>
      <div>{current?.evidence.map((id, index) => <code key={id}>{String(index + 1).padStart(2, '0')} · {id}</code>)}</div>
      <dl>
        {current?.prohibitedActions.map((item) => <div key={item}><dt>{zh ? '避免' : 'Avoid'}</dt><dd>{item}</dd></div>)}
        {current?.verification.map((item) => <div key={item}><dt>{zh ? '验收' : 'Verify'}</dt><dd>{item}</dd></div>)}
      </dl>
    </div>
    {guidance.length > 1 && <ol className="guardian-history">{guidance.slice(1, 5).map((item) => <li key={item.decisionId}><span>{guardianState(item.kind, 'active', zh)}</span><b>{item.finding}</b><time>{new Date(item.createdAt).toLocaleTimeString(zh ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit' })}</time></li>)}</ol>}
  </section>;
}

function guardianState(kind: GuidanceDecision['kind'] | undefined, status: GuidanceStatus['state'], zh: boolean) {
  if (kind === 'redirect' || kind === 'advise' || kind === 'ask') return zh ? '需要纠偏' : 'REDIRECT';
  if (kind === 'verify') return zh ? '等待验收' : 'AWAITING VERIFICATION';
  if (kind === 'block') return zh ? '已阻断' : 'BLOCKED';
  if (status === 'observing') return zh ? '监听正常' : 'OBSERVING';
  if (status === 'stale') return zh ? '证据已过期' : 'STALE';
  if (status === 'error') return zh ? '读取异常' : 'ERROR';
  return zh ? '尚未连接' : 'UNAVAILABLE';
}

function quietState(status: GuidanceStatus['state'], zh: boolean, delivery: GuidanceDelivery | null) {
	if (delivery) {
		const delivered = new Date(delivery.deliveredAt).toLocaleTimeString(zh ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit' });
		return zh
			? { heading: 'AI 已读取本轮指导', finding: 'AI 已读取；本轮未触发纠偏规则', instruction: `最近于 ${delivered} 通过 Codex MCP 送达。系统会继续分析新证据。` }
			: { heading: 'The AI fetched this turn’s guidance', finding: 'Guidance was read; no correction rule fired this turn', instruction: `Last delivered through Codex MCP at ${delivered}. New evidence remains under analysis.` };
	}
	if (status === 'observing') return zh
		? { heading: '正在观察当前对话', finding: '已收到最新证据，当前未发现需要介入的问题', instruction: 'Agent Doctor 会继续监听；只有出现可验证偏差时才向 AI 给出指令。' }
		: { heading: 'Observing this task', finding: 'Recent evidence is arriving and no intervention is active', instruction: 'Agent Doctor will stay quiet until verifiable drift appears.' };
	if (status === 'stale') return zh
		? { heading: '监听已停止更新', finding: '最近证据已经过期', instruction: '继续一次 Codex 或 Claude Code 操作；如果仍无变化，请检查连接配置并重启客户端。' }
		: { heading: 'Observation is stale', finding: 'The latest evidence is no longer current', instruction: 'Run another agent action or repair the client connection.' };
	if (status === 'error') return zh
		? { heading: '引导状态读取失败', finding: '本地证据暂时无法读取', instruction: '请重试；已有对话不会因此丢失。' }
		: { heading: 'Guidance status failed', finding: 'Local evidence could not be read', instruction: 'Retry; captured conversations remain intact.' };
	return zh
		? { heading: '尚未建立引导链路', finding: '还没有收到可用于指导 AI 的运行证据', instruction: '完成安装并重启 Codex 或 Claude Code，然后发起一次真实工具调用。' }
		: { heading: 'Guidance is not connected', finding: 'No compatible runtime evidence has arrived', instruction: 'Install the integration, restart the client, and run one real tool call.' };
}

function capabilityLabel(status: GuidanceStatus, zh: boolean) {
	if (!status.advice) return zh ? '当前客户端尚不支持实时引导' : 'This client does not support live guidance yet';
	if (status.enforcement) return zh ? 'Claude Hook：可给出建议，并对受支持的高风险步骤有限拦截' : 'Claude Hook: advice plus limited enforcement for supported risks';
	return zh ? 'Codex：AI 每轮通过 MCP 主动读取指导；网页设置不能直接改写当前回复，也不能强制阻断工具调用' : 'Codex: the AI pulls guidance through MCP each turn; the dashboard cannot rewrite replies or force-block tool calls';
}

function expiryLabel(value: string, locale: Locale) {
  if (!value) return locale === 'zh' ? '持续观察' : 'monitoring';
  const formatted = new Date(value).toLocaleTimeString(locale === 'zh' ? 'zh-CN' : 'en-US', { hour: '2-digit', minute: '2-digit' });
  return locale === 'zh' ? `${formatted} 前有效` : `valid until ${formatted}`;
}
