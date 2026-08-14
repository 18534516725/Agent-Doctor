import type { LiveAnalysis } from '../api';
import type { Locale } from '../i18n';

export function LiveMetrics({ analysis, locale }: { analysis: LiveAnalysis; locale: Locale }) {
  const metrics = [
    [formatCompactNumber(analysis.requests, locale), locale === 'zh' ? '模型请求' : 'Requests'],
    [formatTokens(analysis.inputTokens + analysis.outputTokens, locale), locale === 'zh' ? '总 Token' : 'Total tokens'],
    [formatDuration(analysis.averageLatencyMs, locale), locale === 'zh' ? '平均响应' : 'Average latency'],
    [analysis.unknownCostCount ? (locale === 'zh' ? '待核验' : 'Pending') : formatCost(analysis.exactCostMicros), locale === 'zh' ? '费用依据' : 'Cost evidence'],
  ];
  return <section className="live-metrics" aria-label={locale === 'zh' ? '实时指标' : 'Live metrics'}>{metrics.map(([value, label], index) => <article key={label}><span>0{index + 1}</span><strong>{value}</strong><small>{label}</small></article>)}</section>;
}

function formatCompactNumber(value: number, locale: Locale) { return new Intl.NumberFormat(locale === 'zh' ? 'zh-CN' : 'en-US', { maximumFractionDigits: 1 }).format(value); }
export function formatTokens(value: number, locale: Locale) { if (locale === 'zh') { if (value >= 100_000_000) return `${(value / 100_000_000).toFixed(1)} 亿`; if (value >= 10_000) return `${(value / 10_000).toFixed(1)} 万`; } if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`; if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`; return formatCompactNumber(value, locale); }
export function formatDuration(milliseconds: number, locale: Locale) { const seconds = milliseconds / 1000; if (seconds < 60) return locale === 'zh' ? `${seconds.toFixed(1)} 秒` : `${seconds.toFixed(1)}s`; const minutes = Math.floor(seconds / 60); const remainder = seconds - minutes * 60; return locale === 'zh' ? `${minutes} 分 ${remainder.toFixed(1)} 秒` : `${minutes}m ${remainder.toFixed(1)}s`; }
function formatCost(micros: number) { const amount = micros / 1_000_000; return `$${amount < .01 ? amount.toFixed(4) : amount.toFixed(2)}`; }
