import { useEffect, useState } from 'react';

import type { RequestTrends } from '../api';
import type { PageProps } from './types';

export function TrendsPage({ locale, api }: PageProps) {
  const [days, setDays] = useState<7 | 30>(30);
  const [data, setData] = useState<RequestTrends>();
  const zh = locale === 'zh';
  useEffect(() => { void api.loadRequestTrends(days).then(setData); }, [api, days]);
  if (!data) return <div className="loading-panel"><strong>{zh ? '正在生成请求趋势…' : 'Building request trends…'}</strong></div>;
  const total = data.points.reduce((sum, item) => sum + item.requests, 0);
  const failed = data.points.reduce((sum, item) => sum + item.failed, 0);
  const p50 = Math.max(0, ...data.points.map((item) => item.p50LatencyMs));
  const p95 = Math.max(0, ...data.points.map((item) => item.p95LatencyMs));
  const max = Math.max(1, ...data.points.map((item) => item.requests));
  return <div className="trend-board intelligence-page">
    <div className="intelligence-heading"><div><span>{zh ? '真实模型请求' : 'CAPTURED MODEL REQUESTS'}</span><h2>{zh ? `${total} 次请求的效率变化` : `Efficiency across ${total} requests`}</h2></div><label className="range-control"><span>{zh ? '统计范围' : 'Range'}</span><select value={days} onChange={(event) => setDays(Number(event.target.value) as 7 | 30)}><option value={7}>{zh ? '最近 7 天' : 'Last 7 days'}</option><option value={30}>{zh ? '最近 30 天' : 'Last 30 days'}</option></select></label></div>
    <section className="analysis-strip"><Metric label={zh ? '请求数' : 'Requests'} value={String(total)} detail={`${data.points.length} ${zh ? '个活跃日期' : 'active days'}`} /><Metric label={zh ? '失败率' : 'Failure rate'} value={`${total ? (failed / total * 100).toFixed(1) : '0.0'}%`} detail={`${failed} ${zh ? '次失败' : 'failed'}`} /><Metric label="P50" value={duration(p50, zh)} detail={zh ? '典型等待' : 'typical wait'} /><Metric label="P95" value={duration(p95, zh)} detail={zh ? '长尾等待' : 'tail wait'} /></section>
    <div className="trend-bars">{data.points.length === 0 ? <p>{zh ? '还没有请求数据；完成一次真实 AI 调用后这里会自动出现。' : 'No requests yet. Complete one real AI call to populate this view.'}</p> : data.points.map((point) => <article key={point.date}><time>{point.date}</time><div><i style={{ width: `${Math.max(3, point.requests / max * 100)}%` }} /></div><strong>{point.requests}</strong><small>{zh ? `失败 ${point.failureRate.toFixed(1)}% · P95 ${duration(point.p95LatencyMs, zh)} · 未缓存 ${compact(point.uncachedInputTokens)}` : `${point.failureRate.toFixed(1)}% failed · P95 ${duration(point.p95LatencyMs, zh)} · ${compact(point.uncachedInputTokens)} uncached`}</small></article>)}</div>
  </div>;
}

function Metric({ label, value, detail }: { label: string; value: string; detail: string }) { return <article><span>{label}</span><strong>{value}</strong><small>{detail}</small></article>; }
function duration(ms: number, zh: boolean) { const seconds = ms / 1000; return seconds < 60 ? `${seconds.toFixed(1)} ${zh ? '秒' : 's'}` : `${Math.floor(seconds / 60)} ${zh ? '分' : 'm'} ${(seconds % 60).toFixed(0)} ${zh ? '秒' : 's'}`; }
function compact(value: number) { return new Intl.NumberFormat('zh-CN', { notation: 'compact', maximumFractionDigits: 1 }).format(value); }
