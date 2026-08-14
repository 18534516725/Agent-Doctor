import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { StatusRail } from '../src/components/StatusRail';
import type { ClientConnection } from '../src/api';

afterEach(cleanup);

describe('StatusRail', () => {
  it('shows only discovered clients and counts only visible connections', () => {
    const connections: ClientConnection[] = [
      {
        key: 'codex',
        displayName: 'Codex',
        detected: true,
        state: 'detected',
        capability: 'mcp',
        detail: '已发现客户端',
        updatedAt: '2026-08-14T10:00:00Z',
      },
      {
        key: 'cursor',
        displayName: 'Cursor',
        detected: true,
        state: 'connected',
        capability: 'proxy',
        detail: '已连接',
        updatedAt: '2026-08-14T10:00:00Z',
      },
      {
        key: 'windsurf',
        displayName: 'Windsurf',
        detected: false,
        state: 'unavailable',
        capability: 'none',
        detail: '未检测到本机配置',
        updatedAt: '2026-08-14T10:00:00Z',
      },
    ];

    render(<StatusRail connections={connections} locale="zh" />);

    expect(screen.getByText('1/2')).toBeVisible();
    expect(screen.getByText('Codex')).toBeVisible();
    expect(screen.getByText('Cursor')).toBeVisible();
    expect(screen.queryByText('Windsurf')).not.toBeInTheDocument();
  });
});
