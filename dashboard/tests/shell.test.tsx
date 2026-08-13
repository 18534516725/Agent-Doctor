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
});
