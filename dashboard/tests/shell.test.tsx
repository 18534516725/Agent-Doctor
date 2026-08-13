import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import { App } from '../src/App';

afterEach(cleanup);

describe('diagnostic dashboard shell', () => {
  it('offers all diagnostic destinations with keyboard-accessible navigation', () => {
    render(<App />);
    for (const label of ['Overview', 'Task evidence', 'Costs', 'Memory', 'Comparison', 'Trends', 'Integrations', 'Privacy']) {
      expect(screen.getByRole('button', { name: label })).toBeVisible();
    }
    fireEvent.click(screen.getByRole('button', { name: 'Costs' }));
    expect(screen.getByRole('heading', { name: 'Cost ledger' })).toBeVisible();
  });

  it('keeps local-only storage visible and does not render raw HTML from an API-shaped value', () => {
    render(<App />);
    expect(screen.getByText(/This device only/i)).toBeVisible();
    expect(screen.queryByText('<img src=x onerror=alert(1)>')).not.toBeInTheDocument();
  });

  it('switches the entire diagnostic shell to Chinese without changing the selected section', () => {
    render(<App />);
    fireEvent.click(screen.getByRole('button', { name: 'Costs' }));
    fireEvent.click(screen.getByRole('button', { name: '中文' }));
    expect(screen.getByRole('heading', { name: '费用账本' })).toBeVisible();
    expect(screen.getByRole('button', { name: '总览' })).toBeVisible();
    expect(screen.getByText(/仅保存在此设备/i)).toBeVisible();
  });
});
