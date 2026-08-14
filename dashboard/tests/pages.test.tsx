import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { App } from '../src/App';
import type { DashboardAPI } from '../src/api';

afterEach(() => { cleanup(); window.localStorage.clear(); });
const conversation = { id:'request-1', sessionId:'session-42', projectId:'project', client:{name:'codex',version:'1'}, model:{displayName:'gpt-test'}, protocol:'openai', method:'POST', path:'/v1/responses', statusCode:200, startedAt:'2026-08-14T10:00:00Z', completedAt:'2026-08-14T10:00:01Z', firstByteMs:120, durationMs:900, usage:{inputTokens:42,outputTokens:18,cachedTokens:10,reasoningTokens:3,precision:'exact' as const,provenance:'provider'}, cost:{currency:'USD',precision:'unavailable' as const,provenance:'no-catalog'}, messages:[{id:'m1',requestId:'request-1',sequence:0,role:'user' as const,content:'请完整分析这个问题',createdAt:'2026-08-14T10:00:00Z'},{id:'m2',requestId:'request-1',sequence:1,role:'assistant' as const,content:'这是完整模型回复',createdAt:'2026-08-14T10:00:01Z'}] };
const api: DashboardAPI = {
  loadSummary: vi.fn(async()=>({projects:2,sessions:3,activeSessions:1,events:12,precision:{exact:8,estimated:3,unavailable:1}})),
  loadSnapshot: vi.fn(async()=>({sessions:[],costs:{currency:'USD',exactMicros:0,estimatedMicros:0,unavailable:1},memories:{active:4,candidate:2,disabled:1},trends:[{date:'2026-08-14',sessions:2,events:9}],comparisonCount:0})),
  loadPrivacy: vi.fn(async()=>({capturePrompts:true,captureFileContents:false,retentionDays:30})), updatePrivacy: vi.fn(async(value)=>value), loadSession:vi.fn(async()=>({sessionId:'session-42',events:[]})),
  loadConversations:vi.fn(async()=>[conversation]), loadConversation:vi.fn(async()=>conversation), loadConnections:vi.fn(async()=>[{key:'codex',displayName:'Codex',detected:true,state:'active' as const,capability:'proxy',detail:'正在采集',updatedAt:'2026-08-14T10:00:00Z'}]),
  loadLiveAnalysis:vi.fn(async()=>({requests:1,activeSessions:1,inputTokens:42,outputTokens:18,cachedTokens:10,reasoningTokens:3,exactCostMicros:0,estimatedCostMicros:0,unknownCostCount:1,averageLatencyMs:900,limitations:[]})), deleteSession:vi.fn(async()=>undefined),
};

describe('Chinese-first live dashboard',()=>{
  it('renders real live metrics and complete conversation text',async()=>{render(<App api={api}/>);await waitFor(()=>expect(screen.getByText('看清每一次 AI 协作')).toBeVisible());expect(await screen.findByText('请完整分析这个问题')).toBeVisible();expect(screen.getByText('这是完整模型回复')).toBeVisible();expect(screen.getByText('60')).toBeVisible();});
  it('keeps all eight pages useful',async()=>{render(<App api={api}/>);await screen.findByText('请完整分析这个问题');for(const label of ['完整对话','用量与费用','项目记忆','任务对比','效率趋势','编辑器连接','本地数据']){fireEvent.click(screen.getByRole('button',{name:label}));expect(screen.getByRole('heading',{name:label})).toBeVisible();}});
  it('renders long content as text and persists language',async()=>{render(<App api={api}/>);await screen.findByText('这是完整模型回复');fireEvent.click(screen.getByRole('button',{name:'English'}));expect(window.localStorage.getItem('agent-doctor-locale')).toBe('en');expect(screen.getByRole('heading',{name:/See every AI collaboration clearly/})).toBeVisible();});
  it('requires a second action before deleting a session',async()=>{render(<App api={api}/>);await screen.findByText('请完整分析这个问题');fireEvent.click(screen.getByRole('button',{name:'本地数据'}));fireEvent.click(screen.getByRole('button',{name:'删除这条会话'}));expect(api.deleteSession).not.toHaveBeenCalled();fireEvent.click(screen.getByRole('button',{name:'再次点击，永久删除'}));await waitFor(()=>expect(api.deleteSession).toHaveBeenCalledWith('session-42'));});
});
