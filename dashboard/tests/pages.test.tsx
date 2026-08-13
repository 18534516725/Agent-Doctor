import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { App } from '../src/App';
import type { DashboardAPI } from '../src/api';

afterEach(cleanup);

const api: DashboardAPI = {
  loadSummary: vi.fn(async () => ({ projects: 2, sessions: 3, activeSessions: 1, events: 12, precision: { exact: 8, estimated: 3, unavailable: 1 } })),
  loadSnapshot: vi.fn(async () => ({
    sessions: [{ id: 'session-42', client: 'codex', model: 'model-a', status: 'active', startedAt: '2026-08-13T10:00:00Z', eventCount: 7 }],
    costs: { currency: 'USD', exactMicros: 1500000, estimatedMicros: 250000, unavailable: 1 },
    memories: { active: 4, candidate: 2, disabled: 1 },
    trends: [{ date: '2026-08-13', sessions: 2, events: 9 }], comparisonCount: 0,
  })),
  loadPrivacy: vi.fn(async () => ({ capturePrompts: false, captureFileContents: false, retentionDays: 30 })),
  updatePrivacy: vi.fn(async (value) => value),
  loadSession: vi.fn(async () => ({ sessionId: 'session-42', events: [{ eventId: 'event-1', timestamp: '2026-08-13T10:01:00Z', eventType: 'validation.completed', provenance: 'client-event', precision: 'exact' as const }] })),
};

describe('complete local dashboard', () => {
  it('renders live overview values and all eight destinations', async () => {
    render(<App api={api} />);
    await waitFor(() => expect(screen.getByText('12')).toBeVisible());
    expect(screen.getByText('sessions · 1 active')).toBeVisible();
    for (const label of ['Overview', 'Task evidence', 'Costs', 'Memory', 'Comparison', 'Trends', 'Integrations', 'Privacy']) {
      expect(screen.getByRole('button', { name: label })).toBeVisible();
    }
  });

  it('shows task evidence without raw payloads', async () => {
    render(<App api={api} />);
    fireEvent.click(screen.getByRole('button', { name: 'Task evidence' }));
    await waitFor(() => expect(screen.getByText('session-42')).toBeVisible());
    fireEvent.click(screen.getByRole('button', { name: /Inspect session-42/ }));
    await waitFor(() => expect(screen.getByText('validation.completed')).toBeVisible());
    expect(document.body.textContent).not.toContain('payload');
  });

  it('renders real cost precision, memory state, trends and comparison limits', async () => {
    render(<App api={api} />);
    await waitFor(() => expect(screen.getByText('12')).toBeVisible());
    fireEvent.click(screen.getByRole('button', { name: 'Costs' }));
    expect(screen.getByText('USD 1.50')).toBeVisible();
    expect(screen.getByText('USD 0.25')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Memory' }));
    expect(screen.getByText('4 active')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Trends' }));
    expect(screen.getByText('2026-08-13')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Comparison' }));
    expect(screen.getByText(/No matched comparison yet/i)).toBeVisible();
  });

  it('shows verified integration capability boundaries and editable privacy controls', async () => {
    render(<App api={api} />);
    await waitFor(() => expect(screen.getByText('12')).toBeVisible());
    fireEvent.click(screen.getByRole('button', { name: 'Integrations' }));
    for (const client of ['Codex', 'Claude Code', 'Cursor', 'Cline', 'OpenCode', 'Windsurf', 'Roo Code', 'Continue', 'Aider', 'Cherry Studio', 'Generic CLI']) {
      expect(screen.getByText(client)).toBeVisible();
    }
    fireEvent.click(screen.getByRole('button', { name: 'Privacy' }));
    expect(screen.getByLabelText('Capture prompts')).not.toBeChecked();
    fireEvent.change(screen.getByLabelText('Retention days'), { target: { value: '45' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save privacy settings' }));
    await waitFor(() => expect(api.updatePrivacy).toHaveBeenCalledWith(expect.objectContaining({ retentionDays: 45 })));
  });

  it('localizes task, empty-state, privacy, and action copy in Chinese', async () => {
    render(<App api={api} />);
    await waitFor(() => expect(screen.getByText('12')).toBeVisible());
    fireEvent.click(screen.getByRole('button', { name: '中文' }));
    fireEvent.click(screen.getByRole('button', { name: '任务证据' }));
    expect(document.body).toHaveTextContent('7 个事件');
    expect(screen.getByRole('button', { name: '查看 session-42' })).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '对比' }));
    expect(screen.getByText('数据缺失时绝不会显示成零。')).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: '隐私' }));
    expect(screen.getByLabelText('采集提示词')).not.toBeChecked();
    expect(screen.getByLabelText('采集文件内容')).not.toBeChecked();
    expect(screen.getByLabelText('保留天数')).toHaveValue(30);
    expect(screen.getByRole('button', { name: '保存隐私设置' })).toBeVisible();
  });
});
