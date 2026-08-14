import { useEffect, useMemo, useState } from 'react';

import type { Conversation } from '../api';
import type { PageProps } from './types';

export function ComparisonPage({ locale, conversations }: PageProps) {
  const zh = locale === 'zh';
  const options = useMemo(() => uniqueSessions(conversations), [conversations]);
  const [leftId, setLeftId] = useState('');
  const [rightId, setRightId] = useState('');
  useEffect(() => { if (!leftId && options[0]) setLeftId(options[0].id); if (!rightId && options[1]) setRightId(options[1].id); }, [options, leftId, rightId]);
  const left = options.find((item) => item.id === leftId);
  const right = options.find((item) => item.id === rightId);
  if (options.length < 2) return <section className="diagnosis-empty"><span>{zh ? '至少需要两个会话' : 'TWO SESSIONS REQUIRED'}</span><h2>{zh ? '再完成一个 AI 会话即可开始对比' : 'Complete one more AI session to compare'}</h2><p>{zh ? `当前已有 ${options.length} 个会话。这里不会要求 15 个样本才显示基础数据；15 个只用于更强的群组结论。` : `${options.length} session captured. Basic comparison starts at two; 15 is only for cohort claims.`}</p></section>;
  return <div className="comparison-workspace">
    <div className="comparison-selectors"><SessionSelect label={zh ? '会话 A' : 'Session A'} value={leftId} options={options} setValue={setLeftId} /><SessionSelect label={zh ? '会话 B' : 'Session B'} value={rightId} options={options} setValue={setRightId} /></div>
    {left && right && <section className="comparison-metrics"><h2>{zh ? '真实会话差异' : 'Observed session differences'}</h2><Metric label={zh ? '请求状态' : 'Status'} left={left.statusCode < 400 ? (zh ? '成功' : 'Success') : (zh ? '失败' : 'Failed')} right={right.statusCode < 400 ? (zh ? '成功' : 'Success') : (zh ? '失败' : 'Failed')} /><Metric label={zh ? '总耗时' : 'Duration'} left={duration(left.durationMs, zh)} right={duration(right.durationMs, zh)} delta={delta(left.durationMs, right.durationMs)} /><Metric label={zh ? '未缓存输入' : 'Uncached input'} left={number(uncached(left), locale)} right={number(uncached(right), locale)} delta={delta(uncached(left), uncached(right))} /><Metric label={zh ? '模型输出' : 'Output'} left={number(left.usage.outputTokens ?? 0, locale)} right={number(right.usage.outputTokens ?? 0, locale)} delta={delta(left.usage.outputTokens ?? 0, right.usage.outputTokens ?? 0)} /><Metric label={zh ? '缓存率' : 'Cache rate'} left={`${cacheRate(left).toFixed(1)}%`} right={`${cacheRate(right).toFixed(1)}%`} delta={`${(cacheRate(right) - cacheRate(left)).toFixed(1)} pp`} /><Metric label={zh ? '费用可信度' : 'Cost evidence'} left={precision(left.cost.precision, zh)} right={precision(right.cost.precision, zh)} /></section>}
    <p className="comparison-gate">{zh ? `这是描述性对比，不宣称哪个 AI 更好。当前 ${options.length} 个可比会话；距离 15 个样本的群组结论还差 ${Math.max(0, 15-options.length)} 个。` : `This is descriptive, not a model-quality claim. ${Math.max(0, 15-options.length)} more sessions are needed for a 15-sample cohort.`}</p>
  </div>;
}

function uniqueSessions(items: Conversation[]) { const seen = new Set<string>(); return items.filter((item) => { if (seen.has(item.sessionId)) return false; seen.add(item.sessionId); return true; }); }
function SessionSelect({ label, value, options, setValue }: { label: string; value: string; options: Conversation[]; setValue(value: string): void }) { return <label><span>{label}</span><select value={value} onChange={(event) => setValue(event.target.value)}>{options.map((item) => <option key={item.id} value={item.id}>{item.client.name} · {item.model.displayName} · {new Date(item.startedAt).toLocaleString()}</option>)}</select></label>; }
function Metric({ label, left, right, delta: change }: { label: string; left: string; right: string; delta?: string }) { return <article><strong>{label}</strong><span>{left}</span><span>{right}</span><small>{change || '—'}</small></article>; }
function uncached(item: Conversation) { return Math.max(0, (item.usage.inputTokens ?? 0) - (item.usage.cachedTokens ?? 0)); }
function cacheRate(item: Conversation) { const input = item.usage.inputTokens ?? 0; return input ? (item.usage.cachedTokens ?? 0) / input * 100 : 0; }
function delta(left: number, right: number) { if (!left) return right ? '+∞' : '0%'; return `${right >= left ? '+' : ''}${((right-left)/left*100).toFixed(1)}%`; }
function number(value: number, locale: 'zh' | 'en') { return new Intl.NumberFormat(locale === 'zh' ? 'zh-CN' : 'en-US', { notation: value >= 10000 ? 'compact' : 'standard', maximumFractionDigits: 1 }).format(value); }
function duration(ms: number, zh: boolean) { return ms >= 60000 ? `${Math.floor(ms/60000)}${zh?'分':'m'} ${Math.round(ms%60000/1000)}${zh?'秒':'s'}` : `${(ms/1000).toFixed(1)}${zh?'秒':'s'}`; }
function precision(value: string, zh: boolean) { return zh ? ({ exact:'精确',estimated:'估算',unavailable:'未知' }[value] ?? value) : value; }
