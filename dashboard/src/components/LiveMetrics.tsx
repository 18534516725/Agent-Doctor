import type { LiveAnalysis } from '../api';
import type { Locale } from '../i18n';

const number = new Intl.NumberFormat('zh-CN');
export function LiveMetrics({ analysis, locale }: { analysis: LiveAnalysis; locale: Locale }) {
  const metrics = [
    [number.format(analysis.requests), locale === 'zh' ? '模型请求' : 'Requests'],
    [number.format(analysis.inputTokens + analysis.outputTokens), 'Token'],
    [`${Math.round(analysis.averageLatencyMs)} ms`, locale === 'zh' ? '平均响应' : 'Average latency'],
    [analysis.unknownCostCount ? '—' : `$${(analysis.exactCostMicros / 1_000_000).toFixed(4)}`, locale === 'zh' ? '精确费用' : 'Exact cost'],
  ];
  return <section className="live-metrics" aria-label={locale === 'zh' ? '实时指标' : 'Live metrics'}>{metrics.map(([value, label], index) => <article key={label}><span>0{index + 1}</span><strong>{value}</strong><small>{label}</small></article>)}</section>;
}
