export type Locale = 'zh' | 'en';
export type Route = 'overview' | 'task' | 'costs' | 'memory' | 'comparison' | 'trends' | 'integrations' | 'privacy';

export const routeOrder: Route[] = ['overview', 'task', 'costs', 'memory', 'comparison', 'trends', 'integrations', 'privacy'];
export const copy = {
  zh: {
    nav: { overview: '实时总览', task: '完整对话', costs: '用量与费用', memory: '项目记忆', comparison: '任务对比', trends: '效率趋势', integrations: '编辑器连接', privacy: '本地数据' },
    eyebrow: 'LOCAL AI OBSERVATORY', title: '看清每一次 AI 协作', lead: '自动连接常用编辑器，完整记录对话、工具调用、Token、耗时与费用依据。所有数据只保存在你的电脑。',
    listening: '正在监听本机调用', offline: '实时连接暂时中断', noTraffic: '已经准备好，等待第一条模型请求', noTrafficDetail: '让 Codex、Claude Code 或其他客户端通过本地代理调用后，这里会立即出现完整对话与实时指标。',
    exact: '精确', estimated: '估算', unavailable: '未知', refresh: '刷新数据', language: 'English', local: '仅此设备',
  },
  en: {
    nav: { overview: 'Live overview', task: 'Conversations', costs: 'Usage & cost', memory: 'Memory', comparison: 'Comparisons', trends: 'Trends', integrations: 'Connections', privacy: 'Local data' },
    eyebrow: 'LOCAL AI OBSERVATORY', title: 'See every AI collaboration clearly', lead: 'Connect common editors and preserve complete conversations, tool calls, tokens, latency and cost evidence on this device.',
    listening: 'Listening for local requests', offline: 'Live connection interrupted', noTraffic: 'Ready for the first model request', noTrafficDetail: 'Route Codex, Claude Code or another client through the local proxy and the full conversation will appear here immediately.',
    exact: 'Exact', estimated: 'Estimated', unavailable: 'Unknown', refresh: 'Refresh data', language: '中文', local: 'This device only',
  },
} as const;

export function initialLocale(): Locale {
  let saved: string | null = null;
  try { saved = typeof window === 'undefined' ? null : window.localStorage?.getItem('agent-doctor-locale'); } catch { saved = null; }
  return saved === 'en' ? 'en' : 'zh';
}
export function persistLocale(locale: Locale) { try { if (typeof window !== 'undefined') window.localStorage?.setItem('agent-doctor-locale', locale); } catch { /* locale persistence is optional */ } }
