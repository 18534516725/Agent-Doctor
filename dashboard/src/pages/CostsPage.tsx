import { useEffect, useState } from 'react';

import type { CostIntelligence } from '../api';
import type { PageProps } from './types';

export function CostsPage({ locale, api }: PageProps) {
  const [days, setDays] = useState<7 | 30>(30);
  const [data, setData] = useState<CostIntelligence>();
  const zh = locale === 'zh';
  useEffect(() => { void api.loadCostIntelligence(days).then(setData); }, [api, days]);
  if (!data) return <div className="loading-panel"><strong>{zh ? '正在计算真实用量…' : 'Calculating usage…'}</strong></div>;
  const title = data.cost.availability === 'unavailable' ? (zh ? '费用暂不可计算' : 'Cost is not yet calculable') : data.cost.availability === 'partial' ? (zh ? '部分费用可核验' : 'Cost is partially verified') : (zh ? '费用已核验' : 'Cost is verified');
  return <div className="ledger intelligence-page">
    <div className="intelligence-heading"><div><span>{zh ? '先看金额可信度' : 'COST AVAILABILITY FIRST'}</span><h2>{title}</h2><p>{data.cost.unknownRequests ? (zh ? `${data.cost.unknownRequests} 条请求缺少价格依据，不会被伪装成 0 元。` : `${data.cost.unknownRequests} requests have no verified price and are not treated as free.`) : (zh ? `已核验金额 ${money(data.cost.exactMicros, data.cost.currency)}` : `Verified ${money(data.cost.exactMicros, data.cost.currency)}`)}</p></div><Range days={days} setDays={setDays} zh={zh} /></div>
    <section className="analysis-strip cost-composition">
      <Metric label={zh ? '未缓存输入' : 'Uncached input'} value={compact(data.usage.uncachedInputTokens, locale)} detail={zh ? '真正新增的上下文' : 'new context'} />
      <Metric label={zh ? '缓存输入' : 'Cached input'} value={compact(data.usage.cachedTokens, locale)} detail={`${zh ? '缓存率' : 'cache rate'} ${data.usage.cacheRate.toFixed(1)}%`} />
      <Metric label={zh ? '模型输出' : 'Output'} value={compact(data.usage.outputTokens, locale)} detail={zh ? '生成内容' : 'generated'} />
      <Metric label={zh ? '推理 Token' : 'Reasoning'} value={compact(data.usage.reasoningTokens, locale)} detail={zh ? '提供方可见部分' : 'provider-reported'} />
    </section>
    {data.rankings.length > 0 && <section className="insight-table"><h3>{zh ? '消耗来自哪里' : 'Where usage comes from'}</h3>{data.rankings.map((item) => <article key={`${item.dimension}-${item.label}`}><span>{dimension(item.dimension, zh)} · {item.label}</span><strong>{compact(item.uncachedInputTokens, locale)}</strong><small>{item.requests} {zh ? '次请求' : 'requests'}{item.unknownCosts ? ` · ${item.unknownCosts} ${zh ? '笔费用未知' : 'unknown costs'}` : ''}</small></article>)}</section>}
    {data.unknown.length > 0 && <details className="unknown-records"><summary>{zh ? `查看 ${data.unknown.length} 条待补价格记录` : `Inspect ${data.unknown.length} unpriced records`}</summary>{data.unknown.map((item) => <p key={item.requestId}><code>{item.requestId}</code> · {item.client} · {item.model} · {item.provenance}</p>)}</details>}
  </div>;
}

function Range({ days, setDays, zh }: { days: 7 | 30; setDays(value: 7 | 30): void; zh: boolean }) { return <label className="range-control"><span>{zh ? '统计范围' : 'Range'}</span><select value={days} onChange={(event) => setDays(Number(event.target.value) as 7 | 30)}><option value={7}>{zh ? '最近 7 天' : 'Last 7 days'}</option><option value={30}>{zh ? '最近 30 天' : 'Last 30 days'}</option></select></label>; }
function Metric({ label, value, detail }: { label: string; value: string; detail: string }) { return <article><span>{label}</span><strong>{value}</strong><small>{detail}</small></article>; }
function compact(value: number, locale: 'zh' | 'en') { if (locale === 'zh') { if (value >= 1e8) return `${(value / 1e8).toFixed(1)} 亿`; if (value >= 1e4) return `${(value / 1e4).toFixed(1)} 万`; } return new Intl.NumberFormat(locale === 'zh' ? 'zh-CN' : 'en-US', { notation: value >= 1000 ? 'compact' : 'standard', maximumFractionDigits: 1 }).format(value); }
function money(micros: number, currency: string) { return new Intl.NumberFormat('zh-CN', { style: 'currency', currency: currency || 'USD', maximumFractionDigits: 4 }).format(micros / 1_000_000); }
function dimension(value: string, zh: boolean) { return zh ? ({ project: '项目', session: '会话', client: '客户端', model: '模型' }[value] ?? value) : value; }
