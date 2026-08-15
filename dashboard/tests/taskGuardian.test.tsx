import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TaskGuardian } from '../src/components/TaskGuardian';
import type { GuidanceDecision } from '../src/api';

afterEach(cleanup);

const redirect: GuidanceDecision = {
  decisionId: 'decision-1', sessionId: 'session-42', projectId: 'project-1',
  kind: 'redirect', severity: 'high',
  finding: '同一个工具调用已经连续失败三次',
  instruction: '停止原样重试，先验证新的诊断假设。',
  evidence: ['event-1', 'event-2', 'event-3'], confidence: 'supported',
  prohibitedActions: ['不要再次执行相同输入。'], verification: ['运行确定性检查。'],
  evidenceFingerprint: 'sha256:evidence',
  expiresAt: '2026-08-14T10:10:00Z', createdAt: '2026-08-14T10:00:00Z',
};

describe('TaskGuardian', () => {
  it('leads with the intervention and keeps evidence collapsed', () => {
    const update = vi.fn();
    render(<TaskGuardian
      locale="zh" guidance={[redirect]} delivery={{ sessionId: 'session-42', projectId: 'project-1', client: 'codex-mcp', decisionId: 'decision-1', decisionKind: 'redirect', controlLevel: 'guide', deliveryCount: 3, deliveredAt: '2026-08-14T10:01:00Z' }} controlLevel="guide"
      status={{ state: 'active', client: 'codex', advice: true, enforcement: false, explanation: '', lastEvidenceAt: '2026-08-14T10:00:00Z' }}
      projectId="project-1" sessionLabel="session-42" clientLabel="Codex"
      onControlLevelChange={update}
    />);

    expect(screen.getAllByRole('heading')[0]).toHaveTextContent('纠偏指令已送达 AI');
    expect(screen.getByText('停止原样重试，先验证新的诊断假设。')).toBeVisible();
    const disclosure = screen.getByRole('button', { name: /3 条证据/ });
    expect(disclosure).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('event-1')).not.toBeInTheDocument();

    const labels = Array.from(screen.getByRole('combobox', { name: '介入级别' }).querySelectorAll('option')).map((item) => item.textContent);
    expect(labels).toEqual(['只观察', '自动指导']);
    expect(screen.getByText('AI 已读取 3 次')).toBeVisible();
    fireEvent.change(screen.getByRole('combobox', { name: '介入级别' }), { target: { value: 'observe' } });
    expect(update).toHaveBeenCalledWith('project-1', 'observe');
  });

  it('shows an honest quiet state without a fake score', () => {
    render(<TaskGuardian
      locale="zh" guidance={[]} delivery={null} controlLevel="guide"
      status={{ state: 'unavailable', client: '', advice: false, enforcement: false, explanation: '' }}
      projectId="project-1" sessionLabel="暂无活动任务" clientLabel="等待连接"
      onControlLevelChange={vi.fn()}
    />);
	expect(screen.getByText('尚未建立引导链路')).toBeVisible();
	expect(screen.queryByText('正在推进')).not.toBeInTheDocument();
	const labels = Array.from(screen.getByRole('combobox', { name: '介入级别' }).querySelectorAll('option')).map((item) => item.textContent);
	expect(labels).toEqual(['只观察', '自动指导']);
    expect(screen.queryByText('100')).not.toBeInTheDocument();
  });

  it('distinguishes a successful guidance read from an intervention', () => {
    render(<TaskGuardian
      locale="zh" guidance={[]} controlLevel="guide"
      delivery={{ sessionId: 'session-42', projectId: 'project-1', client: 'codex-mcp', decisionId: 'decision-quiet', decisionKind: 'continue', controlLevel: 'guide', deliveryCount: 1, deliveredAt: '2026-08-14T10:01:00Z' }}
      status={{ state: 'observing', client: 'codex', advice: true, enforcement: false, explanation: '', lastEvidenceAt: '2026-08-14T10:00:00Z' }}
      projectId="project-1" sessionLabel="session-42" clientLabel="Codex"
      onControlLevelChange={vi.fn()}
    />);
    expect(screen.getByText('AI 已读取本轮指导')).toBeVisible();
    expect(screen.getByText('AI 已读取；本轮未触发纠偏规则')).toBeVisible();
    expect(screen.getByText('AI 已读取 1 次')).toBeVisible();
  });
});
