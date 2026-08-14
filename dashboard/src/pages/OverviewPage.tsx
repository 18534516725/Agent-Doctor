import { useState } from 'react';
import { AnalysisCockpit } from '../components/AnalysisCockpit';
import { ConversationTimeline } from '../components/ConversationTimeline';
import { LiveMetrics } from '../components/LiveMetrics';
import { StatusRail } from '../components/StatusRail';
import type { PageProps } from './types';

export function OverviewPage(props: PageProps) {
  const { locale, analysis, connections, selected } = props;
  const [showRaw, setShowRaw] = useState(false);
  return <div className="overview-grid"><div className="overview-main"><AnalysisCockpit analysis={analysis} locale={locale} /><LiveMetrics analysis={analysis} locale={locale} /><div className="evidence-disclosure"><div><span>{locale === 'zh' ? '原始证据' : 'Raw evidence'}</span><h2>{locale === 'zh' ? '需要时再核对完整对话' : 'Inspect the full transcript on demand'}</h2><p>{locale === 'zh' ? '首页以分析结论为主，用户、模型和工具原文不会默认铺满页面。' : 'The overview prioritizes analysis; user, model and tool messages stay collapsed.'}</p></div><button aria-expanded={showRaw} onClick={() => setShowRaw((value) => !value)}>{showRaw ? (locale === 'zh' ? '收起原始对话' : 'Hide transcript') : (locale === 'zh' ? '查看原始对话' : 'View raw transcript')}</button></div>{showRaw && <ConversationTimeline conversation={selected} locale={locale} />}</div><StatusRail connections={connections} locale={locale} /></div>;
}
