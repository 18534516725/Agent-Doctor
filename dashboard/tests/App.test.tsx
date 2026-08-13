import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { App } from '../src/App';

describe('App', () => {
  it('renders the local-only product identity', () => {
    render(<App />);
    expect(screen.getByRole('heading', { name: /Agent Doctor/i })).toBeInTheDocument();
    expect(screen.getByText(/stored on this device/i)).toBeInTheDocument();
  });
});
