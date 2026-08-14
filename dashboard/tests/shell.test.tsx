import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { App } from '../src/App';
afterEach(()=>{cleanup();window.localStorage.clear();});
describe('dashboard shell',()=>{
  it('starts in Chinese and exposes all destinations',()=>{render(<App/>);expect(screen.getByRole('heading',{name:'项目分析驾驶舱'})).toBeVisible();for(const label of ['实时总览','完整对话','用量与费用','项目记忆','任务对比','效率趋势','编辑器连接','本地数据'])expect(screen.getByRole('button',{name:label})).toBeVisible();});
  it('keeps local-only status visible',()=>{render(<App/>);expect(screen.getByText('仅此设备')).toBeVisible();});
});
