import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { App } from '../src/App';
afterEach(()=>window.localStorage.clear());
describe('App',()=>{it('renders the local product identity without fake zero cards',()=>{render(<App/>);expect(screen.getAllByText('Agent Doctor').length).toBeGreaterThan(0);expect(screen.getByText('正在读取本地数据…')).toBeInTheDocument();expect(screen.queryByText('0 events')).not.toBeInTheDocument();});});
