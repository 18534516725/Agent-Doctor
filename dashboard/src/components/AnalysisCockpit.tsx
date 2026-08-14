import type { LiveAnalysis } from '../api';
import type { Locale } from '../i18n';
import { formatDuration, formatTokens } from './LiveMetrics';

export function AnalysisCockpit({ analysis, locale }: { analysis: LiveAnalysis; locale: Locale }) {
  const zh = locale === 'zh';
  if (!analysis.requests) return <section className="diagnosis-empty"><span>{zh ? '分析引擎已就绪' : 'Analysis engine ready'}</span><h2>{zh ? '等待第一批真实调用' : 'Waiting for real activity'}</h2><p>{zh ? '当编辑器产生模型请求后，这里会自动给出健康度、风险、效率和下一步建议。' : 'Project health, risks, efficiency and next actions will appear after the first captured request.'}</p></section>;
  return <>
    <section className="diagnosis-hero" aria-label={zh ? 'Agent Doctor 项目结论' : 'Agent Doctor project diagnosis'}>
      <div className="health-orbit" style={{ '--score': analysis.healthScore } as React.CSSProperties}><strong>{analysis.healthScore}</strong><small>{zh ? '项目健康度' : 'health score'}</small></div>
      <div className="diagnosis-copy"><span>{zh ? 'Agent Doctor 结论' : 'Agent Doctor diagnosis'}</span><h2>{analysis.summary}</h2><p>{zh ? `依据 ${analysis.requests} 次请求、${analysis.toolCalls} 次工具调用与本地 SQLite 证据生成，不调用额外模型。` : `Built from ${analysis.requests} requests, ${analysis.toolCalls} tool calls and local SQLite evidence without an extra model.`}</p></div>
      <div className="diagnosis-pulse"><i /><span>{zh ? '持续分析中' : 'Analyzing live'}</span></div>
    </section>
    <section className="analysis-strip" aria-label={zh ? '效率指标' : 'Efficiency metrics'}>
      <article><span>{zh ? '失败请求' : 'Failed'}</span><strong>{analysis.failedRequests}</strong><small>{analysis.requests ? `${(analysis.failedRequests / analysis.requests * 100).toFixed(1)}%` : '0%'}</small></article>
      <article><span>{zh ? '单次上下文' : 'Tokens / request'}</span><strong>{formatTokens(analysis.tokensPerRequest, locale)}</strong><small>{zh ? '平均 Token' : 'average tokens'}</small></article>
      <article><span>{zh ? '缓存占比' : 'Cache share'}</span><strong>{analysis.cacheHitRate.toFixed(1)}%</strong><small>{zh ? `已缓存 ${formatTokens(analysis.cachedTokens, locale)} Token` : `${formatTokens(analysis.cachedTokens, locale)} cached tokens`}</small></article>
      <article><span>{zh ? '响应节奏' : 'Response pace'}</span><strong>{formatDuration(analysis.averageLatencyMs, locale)}</strong><small>{zh ? '平均耗时' : 'average latency'}</small></article>
    </section>
    <section className="findings-board">
      <div className="findings-heading"><span>{zh ? '优先级诊断' : 'Prioritized findings'}</span><strong>{analysis.findings.filter((finding) => finding.severity === 'high' || finding.severity === 'medium').length}</strong></div>
      <div className="findings-list">{analysis.findings.map((finding, index) => <article className={`finding finding-${finding.severity}`} key={finding.id}>
        <div className="finding-index">{String(index + 1).padStart(2, '0')}</div>
        <div><span>{severityLabel(finding.severity, locale)}</span><h3>{finding.title}</h3><p>{finding.description}</p></div>
        <dl><div><dt>{zh ? '证据' : 'Evidence'}</dt><dd>{finding.evidence}</dd></div><div><dt>{zh ? '下一步' : 'Next action'}</dt><dd>{finding.recommendation}</dd></div></dl>
      </article>)}</div>
    </section>
  </>;
}

function severityLabel(severity: string, locale: Locale) { const zh: Record<string, string> = { good: '正常', info: '信息', low: '关注', medium: '优先', high: '高风险' }; const en: Record<string, string> = { good: 'Healthy', info: 'Info', low: 'Watch', medium: 'Priority', high: 'High risk' }; return (locale === 'zh' ? zh : en)[severity] ?? severity; }
